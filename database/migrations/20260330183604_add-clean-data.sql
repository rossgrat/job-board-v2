-- Modify "raw_job" table
ALTER TABLE "raw_job" ADD COLUMN "clean_data" text NOT NULL DEFAULT '';
