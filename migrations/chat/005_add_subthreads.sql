-- Migration: Add subthreads support and cascading permissions
-- Subthreads allow nested conversations within threads
-- Permissions cascade from parent: chat -> thread -> subthread

-- Add parent_thread_id to threads table for subthreads
ALTER TABLE con_test.threads
ADD COLUMN IF NOT EXISTS parent_thread_id UUID REFERENCES con_test.threads(id) ON DELETE CASCADE;

-- Add depth column to track nesting level (0 = top-level thread, 1 = subthread, etc.)
ALTER TABLE con_test.threads
ADD COLUMN IF NOT EXISTS depth INT NOT NULL DEFAULT 0;

-- Index for parent thread lookups
CREATE INDEX IF NOT EXISTS idx_threads_parent_thread_id
ON con_test.threads(parent_thread_id)
WHERE parent_thread_id IS NOT NULL;

-- Index for depth-based queries
CREATE INDEX IF NOT EXISTS idx_threads_depth ON con_test.threads(depth);

-- Composite index for chat + depth queries (listing top-level threads vs subthreads)
CREATE INDEX IF NOT EXISTS idx_threads_chat_depth ON con_test.threads(chat_id, depth);

-- Function to check cascading thread access
-- Returns TRUE if user has access to the thread (checks parent chain up to chat)
-- Permission logic:
--   1. If thread has restricted_participants=TRUE, user must be in thread_participants
--   2. If thread has restricted_participants=FALSE (or check passes), check parent:
--      a. If parent_thread_id is set, recursively check parent thread
--      b. If no parent_thread_id, check chat_participants
CREATE OR REPLACE FUNCTION con_test.check_thread_access(
    p_thread_id UUID,
    p_user_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    v_thread RECORD;
    v_has_access BOOLEAN := FALSE;
BEGIN
    -- Get thread info
    SELECT t.id, t.chat_id, t.parent_thread_id, t.restricted_participants, t.thread_type
    INTO v_thread
    FROM con_test.threads t
    WHERE t.id = p_thread_id;

    -- Thread not found
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- System threads are visible to all chat participants
    IF v_thread.thread_type = 'system' THEN
        RETURN EXISTS (
            SELECT 1 FROM con_test.chat_participants
            WHERE chat_id = v_thread.chat_id AND user_id = p_user_id
        );
    END IF;

    -- Check thread-level restriction first
    IF v_thread.restricted_participants THEN
        -- User must be explicitly in thread_participants
        IF NOT EXISTS (
            SELECT 1 FROM con_test.thread_participants
            WHERE thread_id = p_thread_id AND user_id = p_user_id
        ) THEN
            RETURN FALSE;
        END IF;
    END IF;

    -- Check parent access (cascading up)
    IF v_thread.parent_thread_id IS NOT NULL THEN
        -- Recursively check parent thread
        RETURN con_test.check_thread_access(v_thread.parent_thread_id, p_user_id);
    ELSE
        -- No parent thread, check chat participants
        RETURN EXISTS (
            SELECT 1 FROM con_test.chat_participants
            WHERE chat_id = v_thread.chat_id AND user_id = p_user_id
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

-- View for threads with access check (useful for queries)
CREATE OR REPLACE VIEW con_test.threads_with_access AS
SELECT
    t.*,
    tp.user_id AS accessible_to_user_id
FROM con_test.threads t
CROSS JOIN LATERAL (
    SELECT DISTINCT cp.user_id
    FROM con_test.chat_participants cp
    WHERE cp.chat_id = t.chat_id
) tp
WHERE con_test.check_thread_access(t.id, tp.user_id);

-- Function to get thread permission source (where the permission comes from)
-- Returns: 'thread' if explicitly in thread_participants
--          'parent_thread' if from parent thread (with parent_id)
--          'chat' if from chat participants
CREATE OR REPLACE FUNCTION con_test.get_thread_permission_source(
    p_thread_id UUID,
    p_user_id UUID
) RETURNS TABLE(source VARCHAR, source_id UUID) AS $$
DECLARE
    v_thread RECORD;
BEGIN
    -- Get thread info
    SELECT t.id, t.chat_id, t.parent_thread_id, t.restricted_participants, t.thread_type
    INTO v_thread
    FROM con_test.threads t
    WHERE t.id = p_thread_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Check if explicitly in thread_participants
    IF EXISTS (
        SELECT 1 FROM con_test.thread_participants
        WHERE thread_id = p_thread_id AND user_id = p_user_id
    ) THEN
        source := 'thread';
        source_id := p_thread_id;
        RETURN NEXT;
        RETURN;
    END IF;

    -- Check parent
    IF v_thread.parent_thread_id IS NOT NULL THEN
        -- Recursive call for parent
        RETURN QUERY
        SELECT * FROM con_test.get_thread_permission_source(v_thread.parent_thread_id, p_user_id);
        RETURN;
    END IF;

    -- Check chat
    IF EXISTS (
        SELECT 1 FROM con_test.chat_participants
        WHERE chat_id = v_thread.chat_id AND user_id = p_user_id
    ) THEN
        source := 'chat';
        source_id := v_thread.chat_id;
        RETURN NEXT;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Trigger to set depth on insert
CREATE OR REPLACE FUNCTION con_test.set_thread_depth()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_thread_id IS NULL THEN
        NEW.depth := 0;
    ELSE
        SELECT depth + 1 INTO NEW.depth
        FROM con_test.threads
        WHERE id = NEW.parent_thread_id;

        -- Default to 1 if parent not found (shouldn't happen with FK)
        IF NEW.depth IS NULL THEN
            NEW.depth := 1;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_set_thread_depth ON con_test.threads;
CREATE TRIGGER trigger_set_thread_depth
    BEFORE INSERT ON con_test.threads
    FOR EACH ROW
    EXECUTE FUNCTION con_test.set_thread_depth();

-- Add constraint to prevent too deep nesting (max 5 levels)
ALTER TABLE con_test.threads
ADD CONSTRAINT check_thread_max_depth CHECK (depth <= 5);
