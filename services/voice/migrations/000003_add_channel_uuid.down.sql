-- Rollback channel_uuid migration
DROP INDEX IF EXISTS idx_conf_participants_channel_uuid;
ALTER TABLE con_test.conference_participants DROP COLUMN IF EXISTS channel_uuid;
