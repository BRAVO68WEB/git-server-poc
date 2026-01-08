-- Rename a column from "is_mirror" to "mirror_enabled"
ALTER TABLE "repositories" RENAME COLUMN "is_mirror" TO "mirror_enabled";
-- Rename a column from "mirror_url" to "mirror_direction"
ALTER TABLE "repositories" RENAME COLUMN "mirror_url" TO "mirror_direction";
-- Modify "repositories" table
ALTER TABLE "repositories" ALTER COLUMN "sync_status" SET DEFAULT 'idle', ADD COLUMN "upstream_url" text NULL, ADD COLUMN "upstream_username" text NULL, ADD COLUMN "upstream_password" text NULL, ADD COLUMN "downstream_url" text NULL, ADD COLUMN "downstream_username" text NULL, ADD COLUMN "downstream_password" text NULL, ADD COLUMN "sync_interval" bigint NULL DEFAULT 3600;
