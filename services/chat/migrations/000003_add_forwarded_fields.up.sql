-- Add forwarded message fields to messages table
-- These fields track when a message is forwarded from another chat

ALTER TABLE con_test.messages
ADD COLUMN IF NOT EXISTS forwarded_from_message_id UUID REFERENCES con_test.messages(id) ON DELETE SET NULL;

ALTER TABLE con_test.messages
ADD COLUMN IF NOT EXISTS forwarded_from_chat_id UUID;

-- Index for finding forwarded messages
CREATE INDEX IF NOT EXISTS idx_messages_forwarded_from ON con_test.messages(forwarded_from_message_id)
WHERE forwarded_from_message_id IS NOT NULL;
