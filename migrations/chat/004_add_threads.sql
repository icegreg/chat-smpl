-- Migration: Add threads support
-- Threads allow conversations within a chat, including:
-- - Reply threads (started from a message)
-- - Standalone threads (created independently)
-- - System threads (activity logs, events)

-- Threads table
CREATE TABLE IF NOT EXISTS con_test.threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,

    -- Link to parent message (for reply-threads)
    parent_message_id UUID REFERENCES con_test.messages(id) ON DELETE SET NULL,

    -- Thread type: 'user' for discussions, 'system' for activity logs
    thread_type VARCHAR(20) NOT NULL DEFAULT 'user'
        CHECK (thread_type IN ('user', 'system')),

    -- Title (for standalone and system threads)
    title VARCHAR(255),

    -- Counters for UI
    message_count INT NOT NULL DEFAULT 0,
    last_message_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    created_by UUID,  -- NULL for system threads
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_archived BOOLEAN DEFAULT FALSE,

    -- Participant restriction (FALSE = inherit from chat)
    restricted_participants BOOLEAN DEFAULT FALSE
);

-- Indexes for threads
CREATE INDEX IF NOT EXISTS idx_threads_chat_id ON con_test.threads(chat_id);
CREATE INDEX IF NOT EXISTS idx_threads_parent_message_id ON con_test.threads(parent_message_id) WHERE parent_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_threads_type ON con_test.threads(thread_type);
CREATE INDEX IF NOT EXISTS idx_threads_last_message ON con_test.threads(last_message_at DESC) WHERE last_message_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_threads_chat_type ON con_test.threads(chat_id, thread_type);

-- Thread participants (for restricted threads)
CREATE TABLE IF NOT EXISTS con_test.thread_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES con_test.threads(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_thread_participant UNIQUE (thread_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_thread_participants_thread_id ON con_test.thread_participants(thread_id);
CREATE INDEX IF NOT EXISTS idx_thread_participants_user_id ON con_test.thread_participants(user_id);

-- Add thread_id column to messages
ALTER TABLE con_test.messages
ADD COLUMN IF NOT EXISTS thread_id UUID REFERENCES con_test.threads(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON con_test.messages(thread_id) WHERE thread_id IS NOT NULL;

-- Add is_system column to messages for system messages
ALTER TABLE con_test.messages
ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT FALSE;

-- Function to update thread counters when a message is added
CREATE OR REPLACE FUNCTION con_test.update_thread_counters()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.thread_id IS NOT NULL THEN
        UPDATE con_test.threads
        SET
            message_count = message_count + 1,
            last_message_at = NEW.created_at,
            updated_at = NOW()
        WHERE id = NEW.thread_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for updating thread counters
DROP TRIGGER IF EXISTS trigger_update_thread_counters ON con_test.messages;
CREATE TRIGGER trigger_update_thread_counters
    AFTER INSERT ON con_test.messages
    FOR EACH ROW
    WHEN (NEW.thread_id IS NOT NULL)
    EXECUTE FUNCTION con_test.update_thread_counters();

-- Function to decrement thread counters when a message is deleted
CREATE OR REPLACE FUNCTION con_test.decrement_thread_counters()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.thread_id IS NOT NULL THEN
        UPDATE con_test.threads
        SET
            message_count = GREATEST(0, message_count - 1),
            updated_at = NOW()
        WHERE id = OLD.thread_id;
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger for decrementing thread counters
DROP TRIGGER IF EXISTS trigger_decrement_thread_counters ON con_test.messages;
CREATE TRIGGER trigger_decrement_thread_counters
    AFTER DELETE ON con_test.messages
    FOR EACH ROW
    WHEN (OLD.thread_id IS NOT NULL)
    EXECUTE FUNCTION con_test.decrement_thread_counters();
