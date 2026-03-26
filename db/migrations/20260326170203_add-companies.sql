-- Create "company" table
CREATE TABLE "company" (
  "id" uuid NOT NULL,
  "name" text NOT NULL,
  "fetch_type" text NOT NULL,
  "fetch_config" jsonb NOT NULL,
  "favicon_url" text NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
