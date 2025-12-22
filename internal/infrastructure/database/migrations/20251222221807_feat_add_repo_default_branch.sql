-- Modify "repositories" table
ALTER TABLE "repositories" ADD COLUMN "default_branch" text NULL DEFAULT 'main';
