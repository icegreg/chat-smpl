-- Voice service tables migration
-- Schema: con_test

-- Conferences table
CREATE TABLE IF NOT EXISTS con_test.conferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    chat_id UUID REFERENCES con_test.chats(id) ON DELETE SET NULL,
    freeswitch_name VARCHAR(255) NOT NULL UNIQUE,
    created_by UUID NOT NULL,  -- No FK to users (soft delete pattern)
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    max_members INT NOT NULL DEFAULT 10,
    is_private BOOLEAN NOT NULL DEFAULT false,
    recording_path VARCHAR(512),
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for conferences
CREATE INDEX IF NOT EXISTS idx_conferences_chat_id ON con_test.conferences(chat_id);
CREATE INDEX IF NOT EXISTS idx_conferences_status ON con_test.conferences(status);
CREATE INDEX IF NOT EXISTS idx_conferences_created_by ON con_test.conferences(created_by);
CREATE INDEX IF NOT EXISTS idx_conferences_freeswitch_name ON con_test.conferences(freeswitch_name);

-- Trigger for updated_at
DROP TRIGGER IF EXISTS update_conferences_updated_at ON con_test.conferences;
CREATE TRIGGER update_conferences_updated_at
    BEFORE UPDATE ON con_test.conferences
    FOR EACH ROW
    EXECUTE FUNCTION con_test.update_updated_at_column();

-- Conference participants table
CREATE TABLE IF NOT EXISTS con_test.conference_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conference_id UUID NOT NULL REFERENCES con_test.conferences(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,  -- No FK to users (soft delete pattern)
    fs_member_id VARCHAR(255),  -- FreeSWITCH member ID
    status VARCHAR(50) NOT NULL DEFAULT 'connecting',
    is_muted BOOLEAN NOT NULL DEFAULT false,
    is_deaf BOOLEAN NOT NULL DEFAULT false,
    is_speaking BOOLEAN NOT NULL DEFAULT false,
    joined_at TIMESTAMP WITH TIME ZONE,
    left_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Unique constraint: user can rejoin after leaving (left_at distinguishes sessions)
CREATE UNIQUE INDEX IF NOT EXISTS idx_conf_participants_unique
    ON con_test.conference_participants(conference_id, user_id)
    WHERE left_at IS NULL;

-- Indexes for conference_participants
CREATE INDEX IF NOT EXISTS idx_conf_participants_conference ON con_test.conference_participants(conference_id);
CREATE INDEX IF NOT EXISTS idx_conf_participants_user ON con_test.conference_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_conf_participants_status ON con_test.conference_participants(status);

-- Calls table (1-on-1 calls)
CREATE TABLE IF NOT EXISTS con_test.calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    caller_id UUID NOT NULL,  -- No FK to users (soft delete pattern)
    callee_id UUID NOT NULL,  -- No FK to users (soft delete pattern)
    chat_id UUID REFERENCES con_test.chats(id) ON DELETE SET NULL,
    conference_id UUID REFERENCES con_test.conferences(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'initiated',
    fs_call_uuid VARCHAR(255),  -- FreeSWITCH call UUID
    duration INT NOT NULL DEFAULT 0,  -- Duration in seconds
    end_reason VARCHAR(255),
    started_at TIMESTAMP WITH TIME ZONE,
    answered_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for calls
CREATE INDEX IF NOT EXISTS idx_calls_caller ON con_test.calls(caller_id);
CREATE INDEX IF NOT EXISTS idx_calls_callee ON con_test.calls(callee_id);
CREATE INDEX IF NOT EXISTS idx_calls_status ON con_test.calls(status);
CREATE INDEX IF NOT EXISTS idx_calls_chat ON con_test.calls(chat_id);
CREATE INDEX IF NOT EXISTS idx_calls_conference ON con_test.calls(conference_id);
CREATE INDEX IF NOT EXISTS idx_calls_created_at ON con_test.calls(created_at DESC);

-- Verto credentials table (temporary tokens for Verto authentication)
CREATE TABLE IF NOT EXISTS con_test.verto_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,  -- One active credential per user
    login VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for verto_credentials
CREATE INDEX IF NOT EXISTS idx_verto_creds_expires ON con_test.verto_credentials(expires_at);
CREATE INDEX IF NOT EXISTS idx_verto_creds_login ON con_test.verto_credentials(login);

-- Call history view (denormalized for quick access)
CREATE OR REPLACE VIEW con_test.call_history AS
SELECT
    c.id,
    c.caller_id,
    c.callee_id,
    c.chat_id,
    c.status,
    c.duration,
    c.end_reason,
    c.started_at,
    c.answered_at,
    c.ended_at,
    c.created_at,
    caller.username AS caller_username,
    caller.display_name AS caller_display_name,
    callee.username AS callee_username,
    callee.display_name AS callee_display_name
FROM con_test.calls c
LEFT JOIN con_test.users caller ON c.caller_id = caller.id
LEFT JOIN con_test.users callee ON c.callee_id = callee.id;

-- Function to clean up expired verto credentials
CREATE OR REPLACE FUNCTION con_test.cleanup_expired_verto_credentials()
RETURNS void AS $$
BEGIN
    DELETE FROM con_test.verto_credentials
    WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to get active conferences for a user (as participant or creator)
CREATE OR REPLACE FUNCTION con_test.get_user_active_conferences(p_user_id UUID)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    chat_id UUID,
    freeswitch_name VARCHAR(255),
    created_by UUID,
    status VARCHAR(50),
    max_members INT,
    is_private BOOLEAN,
    participant_count BIGINT,
    started_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        c.id,
        c.name,
        c.chat_id,
        c.freeswitch_name,
        c.created_by,
        c.status,
        c.max_members,
        c.is_private,
        (SELECT COUNT(*) FROM con_test.conference_participants cp
         WHERE cp.conference_id = c.id AND cp.status = 'joined') AS participant_count,
        c.started_at,
        c.created_at
    FROM con_test.conferences c
    LEFT JOIN con_test.conference_participants cp ON c.id = cp.conference_id
    WHERE c.status = 'active'
      AND (c.created_by = p_user_id OR (cp.user_id = p_user_id AND cp.left_at IS NULL))
    ORDER BY c.created_at DESC;
END;
$$ LANGUAGE plpgsql;
