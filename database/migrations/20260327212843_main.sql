-- Modify "classified_job" table
ALTER TABLE "classified_job" DROP COLUMN "prompt_version", ADD COLUMN "classification_prompt_version" text NULL;
