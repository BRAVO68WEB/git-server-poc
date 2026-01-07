-- Create "ci_jobs" table
CREATE TABLE "ci_jobs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "run_id" uuid NOT NULL,
  "repository_id" uuid NOT NULL,
  "commit_sha" character varying(40) NOT NULL,
  "ref_name" character varying(255) NOT NULL,
  "ref_type" character varying(20) NOT NULL,
  "trigger_type" character varying(20) NOT NULL,
  "trigger_actor" character varying(255) NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "error" text NULL,
  "created_at" timestamptz NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "config_path" character varying(255) NULL DEFAULT '.stasis-ci.yaml',
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_ci_jobs_repository" FOREIGN KEY ("repository_id") REFERENCES "repositories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_ci_jobs_commit_sha" to table: "ci_jobs"
CREATE INDEX "idx_ci_jobs_commit_sha" ON "ci_jobs" ("commit_sha");
-- Create index "idx_ci_jobs_repository_id" to table: "ci_jobs"
CREATE INDEX "idx_ci_jobs_repository_id" ON "ci_jobs" ("repository_id");
-- Create index "idx_ci_jobs_run_id" to table: "ci_jobs"
CREATE INDEX "idx_ci_jobs_run_id" ON "ci_jobs" ("run_id");
-- Create index "idx_ci_jobs_status" to table: "ci_jobs"
CREATE INDEX "idx_ci_jobs_status" ON "ci_jobs" ("status");
-- Create "ci_artifacts" table
CREATE TABLE "ci_artifacts" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_id" uuid NOT NULL,
  "name" character varying(255) NOT NULL,
  "path" character varying(1024) NOT NULL,
  "size" bigint NOT NULL,
  "checksum" character varying(64) NOT NULL,
  "url" character varying(2048) NULL,
  "created_at" timestamptz NULL,
  "expires_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_ci_jobs_artifacts" FOREIGN KEY ("job_id") REFERENCES "ci_jobs" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_ci_artifacts_job_id" to table: "ci_artifacts"
CREATE INDEX "idx_ci_artifacts_job_id" ON "ci_artifacts" ("job_id");
-- Create "ci_job_logs" table
CREATE TABLE "ci_job_logs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "job_id" uuid NOT NULL,
  "step_name" character varying(255) NULL,
  "level" character varying(20) NOT NULL DEFAULT 'info',
  "message" text NOT NULL,
  "sequence" bigint NOT NULL,
  "timestamp" timestamptz NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_ci_jobs_logs" FOREIGN KEY ("job_id") REFERENCES "ci_jobs" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_ci_job_logs_step_name" to table: "ci_job_logs"
CREATE INDEX "idx_ci_job_logs_step_name" ON "ci_job_logs" ("step_name");
-- Create index "idx_ci_job_logs_timestamp" to table: "ci_job_logs"
CREATE INDEX "idx_ci_job_logs_timestamp" ON "ci_job_logs" ("timestamp");
-- Create index "idx_job_sequence" to table: "ci_job_logs"
CREATE INDEX "idx_job_sequence" ON "ci_job_logs" ("job_id", "sequence");
