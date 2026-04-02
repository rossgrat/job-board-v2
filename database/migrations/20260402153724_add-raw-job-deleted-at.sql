-- Modify "raw_job" table
ALTER TABLE "raw_job" ADD COLUMN "deleted_at" timestamptz NULL;
