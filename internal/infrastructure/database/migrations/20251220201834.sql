-- Create "tokens" table
CREATE TABLE "tokens" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(255) NOT NULL,
  "user_id" uuid NOT NULL,
  "token" text NOT NULL,
  "scope" text[] NULL,
  "expires_at" timestamptz NULL,
  "last_used" timestamptz NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_tokens_user_id" to table: "tokens"
CREATE INDEX "idx_tokens_user_id" ON "tokens" ("user_id");
-- Create "users" table
CREATE TABLE "users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "username" character varying(255) NOT NULL,
  "email" character varying(255) NOT NULL,
  "o_id_c_subject" character varying(255) NULL,
  "o_id_c_issuer" character varying(255) NULL,
  "is_admin" boolean NULL DEFAULT false,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_oidc_subject_issuer" to table: "users"
CREATE UNIQUE INDEX "idx_oidc_subject_issuer" ON "users" ("o_id_c_subject", "o_id_c_issuer");
-- Create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX "idx_users_email" ON "users" ("email");
-- Create index "idx_users_username" to table: "users"
CREATE UNIQUE INDEX "idx_users_username" ON "users" ("username");
-- Create "repositories" table
CREATE TABLE "repositories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "owner_id" uuid NOT NULL,
  "is_private" boolean NULL DEFAULT false,
  "description" text NULL,
  "git_path" text NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_repositories_owner" FOREIGN KEY ("owner_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_repositories_git_path" to table: "repositories"
CREATE UNIQUE INDEX "idx_repositories_git_path" ON "repositories" ("git_path");
-- Create "ssh_keys" table
CREATE TABLE "ssh_keys" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "title" character varying(255) NOT NULL,
  "public_key" text NOT NULL,
  "fingerprint" character varying(255) NOT NULL,
  "key_type" character varying(50) NOT NULL,
  "last_used_at" timestamptz NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_ssh_keys_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_ssh_keys_fingerprint" to table: "ssh_keys"
CREATE UNIQUE INDEX "idx_ssh_keys_fingerprint" ON "ssh_keys" ("fingerprint");
-- Create index "idx_ssh_keys_user_id" to table: "ssh_keys"
CREATE INDEX "idx_ssh_keys_user_id" ON "ssh_keys" ("user_id");
