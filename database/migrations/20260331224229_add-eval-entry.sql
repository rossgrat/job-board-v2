-- Create "eval_entry" table
CREATE TABLE "eval_entry" (
  "id" uuid NOT NULL,
  "raw_job_id" uuid NOT NULL,
  "expected_category" text NOT NULL,
  "expected_relevance" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "eval_entry_raw_job_id_key" UNIQUE ("raw_job_id"),
  CONSTRAINT "eval_entry_raw_job_id_fkey" FOREIGN KEY ("raw_job_id") REFERENCES "raw_job" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
