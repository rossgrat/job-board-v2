# Job Board v2 - Design Document

## Overview

A personal job board that aggregates postings from a curated list of companies and surfaces only high-quality, relevant opportunities. The goal is a zero-friction feed: every job shown is worth applying to.

This is a single-user system. Jobs are fetched from a known set of companies once per hour.

## Data Model

### company

Defines a company to fetch jobs from, including the source type and source-specific configuration.

```
company:
  id
  name                  string (e.g. "Stripe")
  fetch_type            greenhouse | gem | workday | ...
  fetch_config          jsonb (fetcher-specific config, varies by fetch_type)
  favicon_url           string (URL to company favicon, provided at company creation)
  is_active             boolean (can disable fetching without deleting)
```

Example `fetch_config` by type:

```json
// greenhouse
{ "board_slug": "stripe" }

// workday
{ "base_url": "https://nvidia.wd5.myworkdayjobs.com", "path": "/en-US/search" }

// gem
{ "api_key_env": "GEM_API_KEY", "company_id": "abc123" }
```

The scheduler iterates over active companies. The fetcher factory reads `fetch_type`, deserializes `fetch_config` into the appropriate Go struct, and returns the right fetcher implementation.

### raw_job

The `raw_data` blob is immutable — written once at ingestion, never modified. Source of truth for reprocessing. `user_status` is the only mutable field. Deduplication happens at fetch time — if a job already exists (unique key: company_id + source_job_id), it is not created again.

```
raw_job:
  id
  company_id            FK → company
  source_job_id         string (job ID from the source, unique per company)
  url                   string
  raw_data              original posting blob
  discovered_at         timestamp of first ingestion
  user_status           null | applied | tabled | rejected
```

Unique constraint: `(company_id, source_job_id)` — prevents duplicate ingestion.

`user_status` tracks the user's relationship with the job. Independent of any pipeline run — survives prompt reruns. Null means new/unreviewed.

### classified_job

Represents one pass of a raw job through the classification pipeline. Multiple classified_jobs can exist for the same raw_job (one per prompt version). All pipeline state lives here.

```
classified_job:
  id
  raw_job_id            FK → raw_job
  prompt_version        content hash of the prompt files used in this run
  status                pending | non_technical | filtered_location | filtered_level | filtered_relevance | dead | accepted
  is_current            boolean (true for the latest run per raw_job)
  created_at

  # Normalization fields (null if run didn't pass triage)
  # Locations and technologies stored in separate tables (classified_job_location, classified_job_technology)
  title                 normalized role title (e.g. "Senior Infrastructure Engineer")
  salary_min            number or null
  salary_max            number or null
  level                 junior | mid | senior | staff | principal | management | unknown
  normalized_at         timestamp (null if not yet normalized)

  # Classification fields (null if run didn't pass filtering)
  category              backend_engineer | embedded_firmware | linux_kernel_networking | other_interesting | not_relevant
  relevance             strong_match | good_match | partial_match | weak_match | null
                        (null when category is not_relevant)
  reasoning             brief explanation of category and relevance
  classified_at         timestamp (null if not yet classified)
```

`status` is `pending` while the job is in the pipeline. Once it reaches a terminal state — accepted or rejected at some step — the status is set. Jobs that don't pass triage get `non_technical` with all normalization and classification fields null. Jobs that pass normalization but fail hard constraint filtering have normalization fields populated but classification fields null.

`is_current` marks the latest run per raw_job. Only one classified_job per raw_job can have `is_current = true` — enforced by a unique partial index (`CREATE UNIQUE INDEX ON classified_job (raw_job_id) WHERE is_current = true`). When a run completes, the `is_current` flip is done inside a transaction that locks the raw_job row first:

```sql
BEGIN;
  SELECT ... FROM raw_job WHERE id = :raw_job_id FOR UPDATE;
  UPDATE classified_job SET is_current = false WHERE raw_job_id = :raw_job_id AND is_current = true;
  UPDATE classified_job SET is_current = true, status = :status WHERE id = :this_classified_job_id;
COMMIT;
```

The row lock serializes concurrent runs for the same raw_job. The unique index acts as a safety net.

### outbox_task

Pipeline orchestration. Each task represents one step of work to be done on a classified_job. Workers pick up tasks by `task_name` and `status = waiting`.

```
outbox_task:
  id
  classified_job_id     FK → classified_job (ON DELETE CASCADE)
  task_name             triage | normalize | hard_filter | classify | liveness_check
  status                waiting | processing | done | failed
  retry_count           int (default 0)
  max_retries           int (default 3)
  not_before            timestamp (null = ready now, set on retry for backoff)
  created_at
  updated_at
```

When a task completes, the handler either creates the next outbox task (job advances) or sets a terminal status on the classified_job (job stops). On failure, retry_count is incremented and status is set back to `waiting` with a `not_before` backoff (30s, 2m, 10m). After max_retries, status is set to `failed` for manual inspection.

### eval_entry

Stores the user's judgment on a job's classification — either confirming the model was correct or providing a correction. Linked to the raw job, not a specific run, because the user's opinion about a role doesn't change between prompt versions. Used only for prompt tuning eval runs, never affects the UI feed or classification data.

```
eval_entry:
  id
  raw_job_id            FK → raw_job
  expected_category     backend_engineer | embedded_firmware | linux_kernel_networking | other_interesting | not_relevant
  expected_relevance    strong_match | good_match | partial_match | weak_match | null
                        (null when expected_category is not_relevant)
  created_at            timestamp
```


### filter

Defines hard constraint filtering rules applied to normalized job data. All active filters are ANDed together. The `in`/`not_in` operators provide OR logic within a single field.

```
filter:
  id
  field                 location_country | location_city | location_setting | level
  operator              equals | not_equals | contains | in | not_in
  value                 text (single value for equals/not_equals/contains, JSON array for in/not_in)
  is_active             boolean
```

Example filters:

| field | operator | value |
|---|---|---|
| location_country | equals | US |
| location_setting | in | ["remote", "hybrid"] |
| level | not_in | ["junior", "management"] |

### classified_job_location

Normalized location data extracted by the LLM. Separate table for clean SQL filtering.

```
classified_job_location:
  id
  classified_job_id     FK → classified_job (ON DELETE CASCADE)
  country               text (ISO 3166-1 two-letter code)
  city                  text (null if not specified)
  setting               text (remote | hybrid | onsite)
```

### classified_job_technology

Technologies extracted by the LLM. Separate table for queryability.

```
classified_job_technology:
  id
  classified_job_id     FK → classified_job (ON DELETE CASCADE)
  name                  text (lowercase, e.g. "go", "rust", "k8s")
```

### Separation of Concerns

- **`raw_job`** — raw data (immutable) + the user's relationship with the job (`user_status`, mutable).
- **`classified_job`** — one pipeline run. All pipeline state, classification output, and terminal status live here. Multiple runs can exist per raw job (one per prompt version).
- **`outbox_task`** — pipeline orchestration. Tracks what work needs to be done on a classified_job.
- **`eval_entry`** — the user's judgment on classification correctness. Linked to raw_job. Used only for eval runs when tuning prompts. Does not affect the UI feed.

If a job is miscategorized, the user doesn't reclassify it — they interact with it via `user_status` (applied, tabled, rejected) and optionally flag it via `eval_entry` so the prompt can improve. The feed shows an indicator on jobs that have an eval entry so the user can see which jobs they've already reviewed.

### Normalization Notes

- **Locations**: Parsed from all parts of the posting (title, location field, description body). A single job may have multiple locations with different settings (e.g. "remote in US, hybrid in Chicago"). The LLM handles all input variations ("United States", "US", "USA", "U.S.", etc.) and always outputs the same ISO format.
- **Technologies**: Extracted as requirements or preferences from the description. Generic tools (git, linux) excluded unless core to the role.
- **Level**: Mapped to a fixed scale. The LLM infers level from title, years of experience, and scope of responsibilities. "unknown" is used when the posting genuinely doesn't specify — better than a forced guess.
- **Salary**: Extracted if present, null if not. Many postings omit salary.

## Target Roles

- Backend Engineer
- Embedded Systems / Firmware Engineer
- Linux / OS / Kernel / Networking Engineer
- Open-ended: any niche or low-level engineering role that involves deep technical work (compiler engineering, language runtimes, standard libraries, database internals, etc.)

## Pipeline

A single real-time pipeline processes all jobs — both new hourly fetches and initial company ingestion. Each job flows through as individual outbox tasks.

A scheduler in the worker service ticks on a configurable interval (default: once per hour). On each tick, it sends off a goroutine per company to fetch new jobs. On initial company addition, all existing jobs are fetched and enter the pipeline.

### Pipeline Flow

```
1. Scheduler triggers a fetch for a company
2. Fetch handler loads all jobs from the company's source
   → For each job: if raw_job doesn't exist (dedup by company + source job ID), create raw_job
   → Create classified_job (status = pending) + triage outbox task
3. Triage handler sends raw data to LLM (Haiku)
   → Non-technical: set classified_job.status = "non_technical", task done
   → Technical: create normalize task
4. Normalize handler sends raw data to LLM (Haiku), writes normalization fields on classified_job
   → Create hard_filter task
5. Hard filter handler applies deterministic filters to normalized data
   → Filtered: set classified_job.status = "filtered_location" | "filtered_level", task done
   → Passed: create classify task
6. Classify handler sends raw description to LLM (Haiku), writes classification fields on classified_job
   → strong_match, good_match, or partial_match: create liveness_check task
   → weak_match or not_relevant: set classified_job.status = "filtered_relevance", task done
7. Liveness handler sends GET request (with Range header) to job URL
   → Dead: set classified_job.status = "dead", task done
   → Live: set classified_job.status = "accepted", task done → job appears in UI
```

Each step either advances the job (creates the next task) or terminates it (sets a terminal status on classified_job). No task is created after a terminal status.

All LLM steps start with Haiku. Any step can be upgraded to Sonnet independently — it's a config change per step, not an architectural change.

### Prompt Storage

Prompts are text files in the repo, one per LLM step:

```
prompts/
  triage.txt
  normalize.txt
  classify.txt
```

The worker reads prompt files on startup. The `prompt_version` stored on each `classified_job` is a content hash of the prompt files used in that run. If any prompt changes between deploys, new runs get a new version identifier.

Prompt changes are made via code, committed to git, and deployed. Eval runs are executed via CLI before committing — change the file locally, run the eval, check results, then commit and deploy if satisfied.

### Core Business Logic

```
triage(raw_data) → boolean (is this a technical role?)
normalize(raw_data) → normalized fields (title, locations, technologies, salary, level)
filter(normalized_fields, constraints) → boolean
classify(raw_data, prompt) → category + relevance
check_liveness(url) → boolean
```

Deduplication is handled at fetch time (insert-if-not-exists on raw_job), not as a pipeline step.

### Triage (LLM)

Each job is sent to the LLM with the full raw data (title, description, location, etc.) and a simple prompt:

```
You are a job posting classifier. Determine whether this job posting
is for a technical engineering role.

Technical roles include: software engineering, firmware, embedded systems,
systems programming, infrastructure, DevOps/SRE, networking, kernel/OS,
security engineering, compiler engineering, database internals,
machine learning engineering, data engineering, and similar roles
where the primary work is building or maintaining technical systems.

Non-technical roles include: sales, marketing, HR, recruiting, finance,
legal, product management, design, project management, executive
leadership, customer support, operations, and similar roles where
the primary work is not engineering.

When ambiguous, err on the side of "true" — it is better to let a
borderline role through than to reject something potentially relevant.

Return JSON: { "is_technical": boolean }
```

The triage is a cheap bouncer — its job is to reject obvious mismatches, not make close calls. False positives are cheap (an extra normalization call). False negatives are expensive (a good job is never seen).

**Default model**: Haiku. **Upgrade path**: Sonnet if triage quality is insufficient.

### Normalization (LLM)

Each job that passes triage is sent to the LLM with the full raw data. The LLM extracts structured fields used for deterministic filtering in the next step.

Output:

```json
{
  "title": "Senior Infrastructure Engineer",
  "locations": [
    { "country": "US", "city": null, "setting": "remote" },
    { "country": "US", "city": "Chicago, IL", "setting": "hybrid" }
  ],
  "technologies": ["go", "rust", "kubernetes", "postgresql"],
  "salary_min": 180000,
  "salary_max": 220000,
  "level": "senior"
}
```

Key benefits of LLM normalization:
- Handles unstructured location data buried in descriptions, not just structured fields
- Normalizes messy data (titles, locations, salary formats) into consistent structured output
- Extracts information from all parts of the posting (title, location field, description body)

**Default model**: Haiku. **Upgrade path**: Sonnet if normalization quality is insufficient.

### Classification (LLM)

Jobs that pass hard constraint filtering are sent to the LLM with the **raw description only** (not normalized data — classification judges role fit independently from normalization). Category and relevance are independent assessments:

- **Category**: "What kind of technical role is this?" All jobs at this stage are technical (triage confirmed this). `not_relevant` means it's technical but not a type of engineering the user cares about (e.g. QA automation, data analyst).
- **Relevance**: "How well does this role match my interests?" Only set when category is not `not_relevant`. A role can be `other_interesting` + `strong_match` (unexpected category, great fit) or `backend_engineer` + `weak_match` (right category, wrong tech stack).

Output:

```json
{
  "category": "backend_engineer",
  "relevance": "strong_match",
  "reasoning": "Core infrastructure role focused on backend services in Go and Rust with distributed systems work"
}
```

Example of a `not_relevant` result (technical, but not a role the user cares about):

```json
{
  "category": "not_relevant",
  "relevance": null,
  "reasoning": "QA automation role — technical but outside core interests"
}
```

Key benefits of LLM classification:
- Understands the spirit of the role specification, not just literal keywords
- Surfaces interesting roles outside explicitly defined categories ("other_interesting")
- Preferences are expressed as a prompt — editable text, no retraining
- Only runs on jobs that passed hard constraint filtering — fewer LLM calls

**Default model**: Haiku. **Upgrade path**: Sonnet if classification quality is insufficient, especially for the "other_interesting" bucket where nuance matters most.

### Category and Relevance

**Category** describes the type of technical role:

| Category | Meaning |
|---|---|
| `backend_engineer` | Backend services, APIs, distributed systems |
| `embedded_firmware` | Embedded systems, firmware, hardware-adjacent |
| `linux_kernel_networking` | Linux, OS, kernel, networking |
| `other_interesting` | Niche/low-level engineering not in the above categories (compilers, database internals, etc.) |
| `not_relevant` | Technical role but not a type of engineering the user cares about |

**Relevance** represents **role fit only** — how well the role matches interests, tech stack, and preferences. Only set when category is not `not_relevant`. It intentionally excludes hard constraints like location and level, which are handled by deterministic filtering.

This separation means:
- Changing location or level preferences is a free re-filter, not an LLM rerun
- Relevance stays stable across preference changes, making prompt version comparisons cleaner

Discrete buckets are used instead of a numeric score for more consistent LLM output:

| Bucket | Meaning | Feed Behavior |
|---|---|---|
| `strong_match` | Core interest area, right tech stack, compelling work. | Dashboard |
| `good_match` | Most criteria met, maybe adjacent tech stack or slightly outside core categories. | Dashboard |
| `partial_match` | Interesting but notable gaps — tangentially related role or outside core interests but worth a look. | Filter screen |
| `weak_match` | One interesting signal but mostly not aligned. | Filter screen |

Jobs with category `not_relevant` have no relevance bucket and are terminated with `classified_job.status = filtered_relevance`.

### Hard Constraint Filtering

Deterministic filters applied to the normalized output. Runs before classification to reduce LLM calls. These represent non-negotiable requirements:

- **Location**: US remote or Chicago-based
- **Level**: Filter to desired seniority range

Jobs filtered out at this step are stored with their normalization data intact — they can be surfaced if constraints change. Changing filter criteria is a free re-filter, no LLM rerun needed.

### Liveness Check

HTTP request to each job's URL to verify it hasn't been taken down or redirected. Filters out ghost postings and stale listings. Runs only on jobs that survived all prior pipeline steps (triage, normalization, filtering, classification) to minimize HTTP requests and avoid rate limiting.

See "Rate Limiting" and "Liveness Check Details" in the Pipeline Execution Model section.

### UI

Three screens plus a job detail page and a modal. Auth is a single password (see Tech Stack).

#### Job Listing Component

Reused across the dashboard and filter screen. Each listing shows:
- Title (links to external job posting URL), company name + favicon
- Location, level, salary (if available), technologies
- Category and relevance bucket
- Discovered date
- User status badge (applied, tabled, or nothing if new)
- Eval flag indicator (if an eval entry exists)
- Action button to open the status/eval modal

#### Dashboard

The primary screen. Shows `strong_match` and `good_match` jobs where `classified_job.is_current = true AND classified_job.status = accepted AND raw_job.user_status != 'rejected'`.

Sorted by:
1. User status: null (new) first, then applied, then tabled
2. Relevance: `strong_match` before `good_match`
3. `discovered_at` newest first

New unreviewed jobs float to the top.

#### Filter Screen

All classified jobs across all relevance buckets and user statuses, including `weak_match`, `not_relevant`, and rejected. Includes both `accepted` and `filtered_relevance` statuses. Three filters:
- **Relevance**: strong_match, good_match, partial_match, weak_match
- **User status**: new, applied, tabled, rejected
- **Company**: dropdown of active companies

Same sort order as dashboard. Same job listing component. This is where you browse `partial_match`/`weak_match` jobs, review rejected jobs, and spot-check classifications.

#### Company Screen

List of all companies. Each shows:
- Company name and favicon
- Active/inactive toggle (controls whether the scheduler fetches jobs for this company)

#### Job Detail Page

Full dump of all data for a single job. Accessed via a link on the job listing. Shows:
- All raw_job fields including the full `raw_data` blob (unstructured description, raw JSON, etc.)
- All classified_job fields from the current run (normalization + classification)
- LLM reasoning
- Pipeline status and timestamps
- Prompt version used

This is a debugging/inspection screen — everything about the job in one place.

#### Status/Eval Modal

Triggered from the action button on any job listing (dashboard or filter screen). Two sections:

**User status**: Set to applied, tabled, or rejected. Can also clear back to null (new).

**Eval flagging**: Confirm the classification is correct, or provide what the correct category and relevance should be. Creates an eval entry linked to the raw_job. Does not change the classified_job or where the job appears in the feed.

```
Job: "Storage Engine Engineer" at CockroachDB
Model says:  category = not_relevant, relevance = null
User flags:  category = other_interesting, relevance = strong_match
→ eval_entry created, classified_job unchanged, user can set status to "applied"
```

#### Prompt Editing

Not in the UI. Prompts are edited via code changes to the prompt text files in the repo. Eval runs are executed via CLI before committing and deploying prompt changes.

## Pipeline Execution Model

Postgres acts as the task queue via the outbox pattern. No external message queue infrastructure is needed. One worker type handles all tasks.

### Workers

General-purpose workers pick up individual outbox tasks and route to the appropriate handler.

```
Worker:
  1. Poll outbox_task for tasks with status = "waiting" AND (not_before IS NULL OR not_before < now())
  2. Claim a task atomically (SELECT FOR UPDATE SKIP LOCKED, set status = "processing")
  3. Route to handler based on task_name (triage, normalize, hard_filter, classify, liveness_check)
  4. Handler executes, writes results, creates next task or sets terminal status on classified_job
  5. Set task status = "done" (or handle failure — increment retry_count, set not_before for backoff)
  6. Repeat
```

Multiple workers run as goroutines in the worker process. The number of workers is configurable.

### Rate Limiting

Two in-process rate limiters (using `golang.org/x/time/rate`) control external request rates:

**LLM rate limiter**: Shared across all workers. Limits requests to the Anthropic API to stay within tier limits. Workers call `limiter.Wait(ctx)` before each LLM call.

**Per-domain liveness rate limiters**: A map of domain → rate limiter, created on demand. Default: 1 request every 2-3 seconds per domain. When a worker picks up a liveness_check task, it acquires a token from that domain's limiter before making the GET request. If two workers grab liveness tasks for the same domain, the rate limiter serializes the HTTP calls.

```go
llmLimiter     *rate.Limiter           // single shared limiter for Anthropic API
domainLimiters sync.Map                // domain string → *rate.Limiter
```

Note: domain is the URL host, not the company. Multiple companies may share a domain (e.g. `boards.greenhouse.io`) and a single company may have jobs across multiple domains.

### Liveness Check Details

Liveness checks use GET with a `Range: bytes=0-0` header to avoid downloading full pages. This works universally — some sites block HEAD requests but accept GET. The task retry logic handles unreachable domains: after max_retries failures, set classified_job.status = "dead".

### Restart Recovery

Two mechanisms handle in-flight work when the service restarts:

**Graceful shutdown**: On SIGTERM, stop picking up new tasks and wait for in-flight tasks to complete before exiting. Handles clean restarts (deploys, config changes).

**Stale task recovery**: On startup, find any outbox tasks stuck in `processing` state longer than a timeout (e.g. 5 minutes) and reset them to `waiting`. Handles crashes and OOM kills.

Both mechanisms are needed — graceful shutdown for the normal case, stale recovery as a fallback for the crash case.

### Liveness Recheck

A scheduled task runs twice per day. It queues `liveness_check` outbox tasks for classified_jobs where `is_current = true AND status IN ('accepted', 'dead') AND raw_job.user_status NOT IN ('applied', 'rejected')`.

Jobs that were `applied` or `rejected` are skipped — their liveness status doesn't affect the user. Dead jobs that pass the recheck are flipped back to `accepted` and reappear in the feed. Accepted jobs that fail are flipped to `dead` and disappear.

### Cleanup

A scheduled task (daily ticker) in the worker process. Handles two types of housekeeping:

1. **Old classified_job runs**: Delete where `is_current = false AND status != 'pending' AND created_at < now() - 7 days`. The `ON DELETE CASCADE` foreign key on outbox_task automatically removes associated tasks.
2. **Completed outbox_tasks**: Delete where `status = 'done' AND updated_at < now() - 7 days`. Catches completed tasks on current runs that won't be cascade-deleted. Failed tasks are kept for debugging until manually cleared.

## Reruns

Prompts, filter criteria, or triage rules can be edited and jobs reprocessed. Reruns create new `classified_job` rows — old rows are untouched, enabling comparison across versions. `user_status` lives on raw_job and is unaffected by reruns.

A rerun starts at the step that changed. Steps before the starting step are skipped — their output is either not needed or copied from the previous run.

| Change | Rerun from | Copy from previous run | Eligible jobs |
|---|---|---|---|
| Triage prompt | triage | Nothing | All raw jobs |
| Normalization prompt | normalize | Nothing | Jobs that passed triage on previous run |
| Filter criteria | hard_filter | Normalization fields | Jobs that were normalized on previous run |
| Classification prompt | classify | Normalization fields | Jobs that passed filtering on previous run |

For classification reruns, further filtering is applied: only reprocess jobs that were `partial_match` or above on the previous prompt version. Jobs classified `weak_match` or with category `not_relevant` are safe to skip — no reasonable prompt change targeting engineering roles would make them relevant.

Each rerun creates new `classified_job` rows + the appropriate starting outbox task. These flow through the same workers as new jobs. Each classified_job is self-contained — normalization fields are copied into the new row rather than referencing the previous run.

### Evaluation Set

An optional quality assurance tool built passively during normal use. When the user flags a misclassified job (or confirms a correct classification), it's added to the eval set with the expected category and relevance.

The eval set is not training data — the system works without it. It's a test suite for prompt changes:

1. Edit the classification prompt
2. Run the eval set (~100-200 jobs) against the new prompt (~$0.20 with Haiku, ~$1.60 with Sonnet)
3. Compare results vs expected — see what improved, what regressed
4. If satisfied, commit the prompt change and optionally rerun eligible jobs

This protects against the "fix one, break three" problem when tuning prompts. The eval set grows naturally from flagging during normal review — no dedicated labeling effort required.

## Cost Analysis

All LLM steps default to Haiku. Any step can be independently upgraded to Sonnet. All API calls use the standard (real-time) API. Batch API is a future optimization (see Future Work).

### Model Pricing

| Model | Input | Output |
|---|---|---|
| Haiku 4.5 | $0.80 / MTok | $4.00 / MTok |
| Sonnet 4.6 | $3.00 / MTok | $15.00 / MTok |

### Per-Job Cost (All Haiku)

| Step | Estimated Cost |
|---|---|
| Triage | ~$0.001 |
| Normalization | ~$0.001 |
| Classification | ~$0.001 |
| All three (if job passes all steps) | ~$0.003 |

### New Company Ingestion (5,000 jobs, all Haiku)

```
5,000 jobs → Triage: ~$5
  → ~2,000 survivors → Normalization: ~$2
  → ~1,200 pass filtering → Classification: ~$1.20
  → Total: ~$8.20
```

### Ongoing (~50 jobs/day, all Haiku)

```
50 jobs → Triage: ~$0.05
  → ~20 survivors → Normalization: ~$0.02
  → ~12 pass filtering → Classification: ~$0.012
  → Total: ~$0.08/day → ~$2.40/month
```

### Prompt Rerun (5,000 jobs, partial_match+ only)

Only reprocess jobs that were `partial_match` or above on the previous prompt version. `weak_match` and `not_relevant` category jobs are safe to skip. Triage, normalization, and filtering are skipped — normalization data is copied from the previous run.

```
~800 eligible jobs → Haiku classification: ~$0.80
```

### Cost with Sonnet Upgrade (classification step only)

If classification is upgraded to Sonnet while triage and normalization remain on Haiku:

| Scenario | All Haiku | Classification on Sonnet |
|---|---|---|
| New company (5,000 jobs) | ~$8.20 | ~$16.80 |
| Ongoing (50 jobs/day) | ~$2.40/mo | ~$5.40/mo |
| Prompt rerun (800 jobs) | ~$0.80 | ~$6.40 |

### Cost Optimization

- **Pipeline filtering before classification**: Triage, normalization, and hard constraint filtering all reduce the number of jobs that reach the classification step.
- **Prompt caching**: System prompt is identical across calls per step. Caching reduces input cost for that portion by 90%.
- **Relevance-based rerun filtering**: Skip `weak_match` and `not_relevant` jobs on previous prompt versions during reruns.
- **Independent model upgrades**: Start with Haiku everywhere, upgrade individual steps to Sonnet only where quality demands it.

## Future Work

### Batch Pipeline

A dedicated batch pipeline would use the Anthropic Batch API (50% off all tokens) for bulk operations: new company ingestion, prompt reruns, and any large reprocessing jobs. It would share the same core business logic as the real-time pipeline but use a different orchestration model — collecting jobs upfront and submitting them as batch API calls, then polling for completion.

```
Batch pipeline:
  1. Collect all jobs (full company fetch, or query eligible jobs for rerun)
  2. Dedup (bulk)
  3. Triage (Batch API) — reject non-technical roles
  4. Normalization (Batch API) — extract structured fields
  5. Hard constraint filtering (bulk)
  6. Classification (Batch API) — survivors only
  7. Liveness check
  8. → UI
```

A batch orchestrator would manage this end-to-end — triggered explicitly for new company additions or prompt reruns. Batch API results are returned within 24 hours (usually much faster). The async nature is acceptable since batch operations aren't time-sensitive.

**Cost savings**: A 5,000-job company ingestion would drop from ~$8.20 to ~$4.10. Prompt reruns from ~$0.80 to ~$0.40.

**When to build**: When the cost of new company ingestion or frequent prompt reruns becomes a concern. At current volume and frequency, the simplicity of a single real-time pipeline outweighs the savings.

## Tech Stack

### Language & Runtime

- **Go** — both services
- **Postgres** — relational database

### Two Deployables

**Worker**: Fetches jobs on a schedule, executes the pipeline via goroutines, polls the outbox for tasks. No HTTP server.

**UI/API**: Serves the web UI and API. Handles user interactions (status changes, eval flagging).

Both connect to the same Postgres instance. Configuration is per-service via Viper (YAML), with some overlap in config variables (database connection, etc.). Secrets (API keys, tokens, passwords) live in environment variables.

### LLM

- **Anthropic API** via `github.com/anthropics/anthropic-sdk-go`
- Default model: Haiku 4.5 for all LLM steps
- Any step independently upgradeable to Sonnet 4.6

### Web / UI

- **chi** — HTTP router with middleware support
- **templ** — type-safe HTML templating, compiles to Go
- **htmx** — partial page updates via HTML attributes, no JavaScript framework
- **Plain CSS** — no CSS framework

### Database

- **pgx** — Postgres driver
- **sqlc** — generates type-safe Go from SQL queries
- **Atlas** — database migrations

### Auth

Single-password authentication. Middleware checks for a cookie against a hashed password from config. No sessions, no usernames, no user table. On correct password, sets a long-lived cookie. Login page is a single password field.

### Testing

- **testcontainers** — integration tests against real Postgres
- Standard Go `testing` package

### Infrastructure

- **Docker Compose** — container orchestration (worker + API + Postgres)
- **Caddy** — reverse proxy, HTTPS termination (shared external network)
- **GitHub Actions** — CI/CD, builds Docker images on push to main
- **GHCR** — Docker image registry
- **SSH + Makefile** — deployment to remote server (same pattern as steam-deck-stock-alerts)

### Deployment

Same approach as steam-deck-stock-alerts:
1. Push to `main` triggers GitHub Actions
2. Actions builds and pushes Docker images to GHCR
3. `make deploy` SCPs docker-compose and config to remote server, pulls images, restarts

## Approaches Considered and Rejected

| Approach | Why Not |
|---|---|
| Static title keyword pre-filter | Non-technical roles have the same title variation problem as technical roles ("Client Success Partner" vs "Account Manager" vs "CSM"). Keyword list can never be comprehensive. Haiku triage handles this semantically with no maintenance. Could revisit if cost becomes an issue. |
| Dynamic keyword filter (LLM-populated) | Feedback loop where LLM scores populate title keywords. Adds coupling between pipeline steps, prompt version changes invalidate keywords, and same-title-different-description problem (title is too coarse a granularity for learned filtering). |
| Keyword matching only | Brittle, no semantic understanding, can't handle "jobs I haven't thought of yet" |
| TF-IDF + cosine similarity | No semantic understanding, synonyms are invisible |
| Embedding similarity | Needs reference examples, less interpretable, no reasoning |
| Traditional ML (SVM, Naive Bayes) | Needs labeled training data, rigid categories, can't generalize to shifting preferences |
| Fine-tuned small model (BERT) | Same training data problem, overkill for this volume, requires ML expertise |
| Manual review only | Defeats the zero-friction goal |
