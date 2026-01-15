-- Extended soft delete for messages
-- Adds metadata for tracking deletion info and supporting message restoration

-- Add new columns for extended soft delete
ALTER TABLE con_test.messages
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS deleted_by UUID,
ADD COLUMN IF NOT EXISTS is_moderated_deletion BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS original_content TEXT;

-- Index for efficient cleanup job queries (find old deleted messages)
CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON con_test.messages(deleted_at)
WHERE deleted_at IS NOT NULL;

-- Index for finding messages deleted by specific user (audit purposes)
CREATE INDEX IF NOT EXISTS idx_messages_deleted_by ON con_test.messages(deleted_by)
WHERE deleted_by IS NOT NULL;

-- Comments for documentation
COMMENT ON COLUMN con_test.messages.deleted_at IS 'Timestamp when message was deleted (NULL = not deleted)';
COMMENT ON COLUMN con_test.messages.deleted_by IS 'UUID of user who deleted the message (author or moderator)';
COMMENT ON COLUMN con_test.messages.is_moderated_deletion IS 'TRUE if deleted by moderator, FALSE if deleted by author';
COMMENT ON COLUMN con_test.messages.original_content IS 'Original message content preserved for potential restoration within retention period';
