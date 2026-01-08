-- Add sync_schedule column for cron expression support
ALTER TABLE repositories ADD COLUMN sync_schedule VARCHAR(100);

-- Add index for repositories with cron schedules
CREATE INDEX idx_repositories_sync_schedule ON repositories(sync_schedule) WHERE sync_schedule IS NOT NULL AND sync_schedule != '';
