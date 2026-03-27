-- Create "raw_job" table
CREATE TABLE "raw_job" (
  "id" uuid NOT NULL,
  "company_id" uuid NOT NULL,
  "source_job_id" text NOT NULL,
  "url" text NOT NULL,
  "raw_data" jsonb NOT NULL,
  "discovered_at" timestamptz NOT NULL DEFAULT now(),
  "user_status" text NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "raw_job_company_id_source_job_id_key" UNIQUE ("company_id", "source_job_id"),
  CONSTRAINT "raw_job_company_id_fkey" FOREIGN KEY ("company_id") REFERENCES "company" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "classified_job" table
CREATE TABLE "classified_job" (
  "id" uuid NOT NULL,
  "raw_job_id" uuid NOT NULL,
  "prompt_version" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending',
  "is_current" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "title" text NULL,
  "salary_min" integer NULL,
  "salary_max" integer NULL,
  "level" text NULL,
  "normalized_at" timestamptz NULL,
  "category" text NULL,
  "relevance" text NULL,
  "reasoning" text NULL,
  "classified_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "classified_job_raw_job_id_fkey" FOREIGN KEY ("raw_job_id") REFERENCES "raw_job" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "classified_job_one_current_per_raw_job" to table: "classified_job"
CREATE UNIQUE INDEX "classified_job_one_current_per_raw_job" ON "classified_job" ("raw_job_id") WHERE (is_current = true);
-- Create "classified_job_location" table
CREATE TABLE "classified_job_location" (
  "id" uuid NOT NULL,
  "classified_job_id" uuid NOT NULL,
  "country" text NOT NULL,
  "city" text NULL,
  "setting" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "classified_job_location_classified_job_id_fkey" FOREIGN KEY ("classified_job_id") REFERENCES "classified_job" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "classified_job_technology" table
CREATE TABLE "classified_job_technology" (
  "id" uuid NOT NULL,
  "classified_job_id" uuid NOT NULL,
  "name" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "classified_job_technology_classified_job_id_fkey" FOREIGN KEY ("classified_job_id") REFERENCES "classified_job" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "outbox_task" table
CREATE TABLE "outbox_task" (
  "id" uuid NOT NULL,
  "classified_job_id" uuid NOT NULL,
  "task_name" text NOT NULL,
  "status" text NOT NULL DEFAULT 'waiting',
  "retry_count" integer NOT NULL DEFAULT 0,
  "max_retries" integer NOT NULL DEFAULT 3,
  "not_before" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "outbox_task_classified_job_id_fkey" FOREIGN KEY ("classified_job_id") REFERENCES "classified_job" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
