-- Modify "raw_job" table
ALTER TABLE "raw_job" ALTER COLUMN "clean_data" DROP DEFAULT;
-- Create "filter_group" table
CREATE TABLE "filter_group" (
  "id" uuid NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id")
);
-- Create "filter_condition" table
CREATE TABLE "filter_condition" (
  "id" uuid NOT NULL,
  "filter_group_id" uuid NOT NULL,
  "field" text NOT NULL,
  "operator" text NOT NULL,
  "value" text NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "filter_condition_filter_group_id_fkey" FOREIGN KEY ("filter_group_id") REFERENCES "filter_group" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
