# Job Board v2 - Design Document

## Overview

A personal job board that aggregates postings from a curated list of companies and surfaces only high-quality, relevant opportunities. The goal is a zero-friction feed: every job shown is worth applying to.

This is a single-user system. Jobs are fetched from a known set of companies once per hour.

## Data Model

### raw_job

Immutable. Written once at ingestion, never modified. Source of truth for reprocessing. Deduplication happens at fetch time — if a job already exists (unique key: company + source job ID), it is not created again.

```
raw_job:
  id
  company               string
  url                   string
  raw_data              original posting blob
  discovered_at         timestamp of first ingestion
  user_status           null | interested | not_interested | tabled | applied
```

`user_status` tracks the user's relationship with the job. Independent of any pipeline run — marking a job "applied" survives prompt reruns. Null means the user hasn't interacted with it yet.

### classified_job

Represents one pass of a raw job through the classification pipeline. Multiple classified_jobs can exist for the same raw_job (one per prompt version). All pipeline state lives here.

```
classified_job:
  id
  raw_job_id            FK → raw_job
  prompt_version        identifier for the prompt used in this run
  status                pending | non_technical | filtered_location | filtered_level | filtered_relevance | dead | accepted
  created_at

  # Normalization fields (null if run didn't pass triage)
  title                 normalized role title (e.g. "Senior Infrastructure Engineer")
  locations[]           list of location objects:
    country             two-letter ISO 3166-1 code (e.g. "US", "CA", "GB")
    city                "City, State/Province" format if available, null if not
    setting             remote | hybrid | onsite
  technologies[]        lowercase strings (e.g. ["go", "rust", "k8s"])
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

### outbox_task

Pipeline orchestration. Each task represents one step of work to be done on a classified_job. Workers pick up tasks by `task_name` and `status = waiting`.

```
outbox_task:
  id
  classified_job_id     FK → classified_job
  task_name             triage | normalize | hard_filter | classify | liveness_check
  status                waiting | processing | done | failed
  created_at
  updated_at
```

When a task completes, the handler either creates the next outbox task (job advances) or sets a terminal status on the classified_job (job stops). Failed tasks can be retried by resetting status to `waiting`.

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

### domain_lock

Used by liveness workers to prevent concurrent requests to the same host.

```
domain_lock:
  domain                string (e.g. "boards.greenhouse.io")
  locked_by             worker ID or null
  locked_at             timestamp (for stale lock recovery)
```

### Separation of Concerns

- **`raw_job`** — immutable raw data + the user's relationship with the job (`user_status`).
- **`classified_job`** — one pipeline run. All pipeline state, classification output, and terminal status live here. Multiple runs can exist per raw job (one per prompt version).
- **`outbox_task`** — pipeline orchestration. Tracks what work needs to be done on a classified_job.
- **`eval_entry`** — the user's judgment on classification correctness. Linked to raw_job. Used only for eval runs when tuning prompts. Does not affect the UI feed.

If a job is miscategorized, the user doesn't reclassify it — they interact with it via `user_status` (interested, applied, etc.) and optionally flag it via `eval_entry` so the prompt can improve. The feed shows an indicator on jobs that have an eval entry so the user can see which jobs they've already reviewed.

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
   → Category not_relevant: set classified_job.status = "filtered_relevance", task done
   → Relevant: create liveness_check task
7. Liveness handler sends HEAD request to job URL
   → Dead: set classified_job.status = "dead", task done
   → Live: set classified_job.status = "accepted", task done → job appears in UI
```

Each step either advances the job (creates the next task) or terminates it (sets a terminal status on classified_job). No task is created after a terminal status.

All LLM steps start with Haiku. Any step can be upgraded to Sonnet independently — it's a config change per step, not an architectural change.

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
- You can see what you're filtering out ("12 great roles if you were open to relocating")
- Relevance stays stable across preference changes, making prompt version comparisons cleaner

Discrete buckets are used instead of a numeric score for more consistent LLM output:

| Bucket | Meaning | Feed Behavior |
|---|---|---|
| `strong_match` | Core interest area, right tech stack, compelling work. | Default feed |
| `good_match` | Most criteria met, maybe adjacent tech stack or slightly outside core categories. | Default feed |
| `partial_match` | Interesting but notable gaps — tangentially related role or outside core interests but worth a look. | "Browse maybes" view |
| `weak_match` | One interesting signal but mostly not aligned. | Hidden, accessible if searching |

Jobs with category `not_relevant` have no relevance bucket and are terminated with `classified_job.status = filtered_relevance`.

### Hard Constraint Filtering

Deterministic filters applied to the normalized output. Runs before classification to reduce LLM calls. These represent non-negotiable requirements:

- **Location**: US remote or Chicago-based
- **Level**: Filter to desired seniority range

Jobs filtered out at this step are stored with their normalization data intact — they can be surfaced if constraints change. Changing filter criteria is a free re-filter, no LLM rerun needed.

### Liveness Check

HTTP request to each job's URL to verify it hasn't been taken down or redirected. Filters out ghost postings and stale listings. Runs only on jobs that survived all prior pipeline steps (triage, normalization, filtering, classification) to minimize HTTP requests and avoid rate limiting.

See "Liveness Workers" below for execution details.

### UI

**Main feed**: Jobs with the latest `classified_job.status = accepted` per raw_job, sorted by relevance bucket (`strong_match` first, then `good_match`), then by `discovered_at` (newest first within each bucket). Each job shows:
- Title, company, locations, technologies, level, salary
- Category and relevance bucket
- Model's reasoning
- Eval flag indicator (if an eval entry exists)
- Two actions:
  - **Status button**: Set user status (interested, not interested, tabled, applied) → writes to `raw_job.user_status`
  - **Flag button**: Confirm or correct the classification → writes to `eval_entry`

Jobs with a `user_status` set move to a separate "My Jobs" view.

**Additional views**:
- **"Browse maybes"**: `partial_match` jobs that passed all filters
- **"Filtered out"**: Jobs rejected by hard constraints, with their normalization data intact — title, location, level visible but no classification (useful for spotting overly aggressive filters)
- **"My Jobs"**: Jobs the user has interacted with via `user_status` (interested, applied, tabled, etc.)

**Review screen**: All classified jobs, filterable by relevance bucket, category, and company. Shows the model's reasoning for each classification. Used for spot-checking results and building the eval set.

**Eval flagging**: On any job, the user can flag the classification — either confirming it's correct or providing what the correct category and relevance should be. This creates an eval entry and does not change the model's classification or where the job appears in the feed.

```
Job: "Storage Engine Engineer" at CockroachDB
Model says:  category = not_relevant, relevance = null
User flags:  category = other_interesting, relevance = strong_match
→ eval_entry created, classified_job unchanged, user can set raw_job.user_status to "interested"
```

**Prompt editor**: Edit the classification prompt directly. After editing, run the eval set against the new prompt to check for regressions before committing to a full rerun.

## Pipeline Execution Model

Postgres acts as the task queue via the outbox pattern. No external message queue infrastructure is needed. Two worker types process tasks.

### General Workers

Pick up individual outbox tasks and route to the appropriate handler.

```
General worker:
  1. Poll outbox_task for tasks with status = "waiting" and task_name != "liveness_check"
  2. Claim a task (set status = "processing")
  3. Route to handler based on task_name (triage, normalize, hard_filter, classify)
  4. Handler executes, writes results, creates next task or sets terminal status on classified_job
  5. Set task status = "done"
  6. Repeat
```

Multiple general workers can run in parallel. Steps like triage, normalization, filtering, and classification have no rate limit concerns, so concurrency is safe. Each worker makes real-time LLM API calls directly.

### Liveness Workers

Pick up only `liveness_check` tasks. Multiple liveness workers run concurrently, each claiming a domain exclusively to prevent concurrent requests to the same host.

```
Liveness worker:
  1. Query outbox_task for liveness_check tasks with status = "waiting"
  2. Group by domain, find a domain that is not locked
  3. Claim the domain (atomic lock via domain_lock table)
  4. Grab all waiting liveness tasks for that domain
  5. Process sequentially with a polite delay between requests (default: 2-3 seconds)
  6. For each task: HEAD request to URL, set classified_job.status = "accepted" or "dead", mark task done
  7. Release the domain lock
  8. Repeat from 1
  9. If no domains available, back off and poll less frequently
```

#### Domain Locking

Lock acquisition is atomic: `UPDATE domain_lock SET locked_by = :me, locked_at = now() WHERE domain = :domain AND locked_by IS NULL`. If another worker holds the lock, the update affects zero rows and the worker moves on to the next available domain.

This scales naturally — with 50 companies and 10 liveness workers, each worker cycles through available domains. If one domain has many jobs, one worker stays on it while others handle the rest.

Note: domain is the URL host, not the company. Multiple companies may share a domain (e.g. `boards.greenhouse.io`) and a single company may have jobs across multiple domains.

#### Rate Limit Management

Each liveness worker controls its own pace per domain — default one request every 2-3 seconds. Per-domain overrides can be configured if some hosts are more or less generous.

If a domain is completely unreachable (not rate-limited, just down), a max retry count on the task prevents infinite loops. After N failures, set classified_job.status = "dead" and move on.

### Restart Recovery

Two mechanisms handle in-flight work when the service restarts:

**Graceful shutdown**: On SIGTERM, stop picking up new tasks and wait for in-flight tasks to complete before exiting. Handles clean restarts (deploys, config changes).

**Stale task recovery**: On startup, find any outbox tasks stuck in `processing` state longer than a timeout (e.g. 5 minutes) and reset them to `waiting`. Handles crashes and OOM kills. Also release any domain locks where `locked_at` exceeds the timeout.

Both mechanisms are needed — graceful shutdown for the normal case, stale recovery as a fallback for the crash case.

## Prompt Tuning and Reruns

The classification prompt (role preferences, scoring criteria) can be edited and jobs reprocessed to evaluate changes. To support this:

- **Reruns create new `classified_job` rows** with the new prompt version — old rows are untouched, enabling comparison across versions
- **Comparison across versions** — diff classified_jobs for the same raw_job to see what changed
- **User status is preserved** — `user_status` lives on raw_job, unaffected by reruns
- **Relevance-based filtering on reruns** — only reprocess jobs that were `partial_match` or above on the previous prompt version. Jobs classified `weak_match` or with category `not_relevant` are safe to skip — no reasonable prompt change targeting engineering roles would make them relevant. This cuts rerun cost by ~60%.
- Rerun creates new `classified_job` rows + `classify` outbox tasks for eligible jobs, which flow through the same workers as new jobs. Triage, normalization, and filtering are skipped — these jobs already passed those steps on their initial run. Normalization data can be copied from the previous run.

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
