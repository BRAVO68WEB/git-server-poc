-- Modify "repositories" table
ALTER TABLE "repositories" ADD COLUMN "is_mirror" boolean NULL DEFAULT false, ADD COLUMN "mirror_url" text NULL, ADD COLUMN "last_synced_at" timestamptz NULL, ADD COLUMN "sync_status" text NULL, ADD COLUMN "sync_error" text NULL;
