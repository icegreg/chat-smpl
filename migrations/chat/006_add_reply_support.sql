-- Add reply support for messages
-- Allows a message to reply to one or more other messages

-- Table for message reply relationships (many-to-many)
CREATE TABLE IF NOT EXISTS con_test.message_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES con_test.messages(id) ON DELETE CASCADE,
    reply_to_message_id UUID NOT NULL REFERENCES con_test.messages(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_message_reply UNIQUE (message_id, reply_to_message_id)
);

-- Index for finding what messages a given message replies to
CREATE INDEX IF NOT EXISTS idx_message_replies_message_id ON con_test.message_replies(message_id);

-- Index for finding what messages reply to a given message
CREATE INDEX IF NOT EXISTS idx_message_replies_reply_to ON con_test.message_replies(reply_to_message_id);

-- Comment for documentation
COMMENT ON TABLE con_test.message_replies IS 'Tracks which messages are replies to other messages. Supports multiple replies per message.';
COMMENT ON COLUMN con_test.message_replies.message_id IS 'The message that contains the reply';
COMMENT ON COLUMN con_test.message_replies.reply_to_message_id IS 'The message being replied to';
