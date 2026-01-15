-- Scheduled events migration
-- Schema: con_test

-- Event types enum
DO $$ BEGIN
    CREATE TYPE con_test.event_type AS ENUM (
        'adhoc',
        'adhoc_chat',
        'scheduled',
        'recurring'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Conference roles enum
DO $$ BEGIN
    CREATE TYPE con_test.conference_role AS ENUM (
        'originator',
        'moderator',
        'speaker',
        'assistant',
        'participant'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- RSVP status enum
DO $$ BEGIN
    CREATE TYPE con_test.rsvp_status AS ENUM (
        'pending',
        'accepted',
        'declined'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Recurrence frequency enum
DO $$ BEGIN
    CREATE TYPE con_test.recurrence_frequency AS ENUM (
        'daily',
        'weekly',
        'biweekly',
        'monthly'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Add new columns to conferences table
ALTER TABLE con_test.conferences
ADD COLUMN IF NOT EXISTS event_type con_test.event_type DEFAULT 'adhoc',
ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS series_id UUID,
ADD COLUMN IF NOT EXISTS accepted_count INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS declined_count INT NOT NULL DEFAULT 0;

-- Indexes for scheduled events
CREATE INDEX IF NOT EXISTS idx_conferences_scheduled_at ON con_test.conferences(scheduled_at)
WHERE scheduled_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_conferences_series_id ON con_test.conferences(series_id)
WHERE series_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_conferences_event_type ON con_test.conferences(event_type);

-- Recurrence rules table
CREATE TABLE IF NOT EXISTS con_test.conference_recurrence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conference_id UUID NOT NULL REFERENCES con_test.conferences(id) ON DELETE CASCADE,
    frequency con_test.recurrence_frequency NOT NULL,
    days_of_week INT[] DEFAULT '{}',
    day_of_month INT,
    until_date TIMESTAMP WITH TIME ZONE,
    occurrence_count INT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conf_recurrence_conference ON con_test.conference_recurrence(conference_id);

-- Add new columns to conference_participants table
ALTER TABLE con_test.conference_participants
ADD COLUMN IF NOT EXISTS role con_test.conference_role DEFAULT 'participant',
ADD COLUMN IF NOT EXISTS rsvp_status con_test.rsvp_status DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS rsvp_at TIMESTAMP WITH TIME ZONE;

-- Index for RSVP status
CREATE INDEX IF NOT EXISTS idx_conf_participants_rsvp ON con_test.conference_participants(rsvp_status);
CREATE INDEX IF NOT EXISTS idx_conf_participants_role ON con_test.conference_participants(role);

-- Scheduled reminders table (for push and in-app notifications)
CREATE TABLE IF NOT EXISTS con_test.conference_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conference_id UUID NOT NULL REFERENCES con_test.conferences(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    remind_at TIMESTAMP WITH TIME ZONE NOT NULL,
    minutes_before INT NOT NULL DEFAULT 15,
    sent BOOLEAN NOT NULL DEFAULT FALSE,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reminders_remind_at ON con_test.conference_reminders(remind_at)
WHERE sent = FALSE;

CREATE INDEX IF NOT EXISTS idx_reminders_conference ON con_test.conference_reminders(conference_id);
CREATE INDEX IF NOT EXISTS idx_reminders_user ON con_test.conference_reminders(user_id);

-- Trigger function to update accepted/declined counts
CREATE OR REPLACE FUNCTION con_test.update_conference_rsvp_counts()
RETURNS TRIGGER AS $$
DECLARE
    v_conf_id UUID;
BEGIN
    -- Determine which conference to update
    IF TG_OP = 'DELETE' THEN
        v_conf_id := OLD.conference_id;
    ELSE
        v_conf_id := NEW.conference_id;
    END IF;

    -- Update the counts
    UPDATE con_test.conferences SET
        accepted_count = (
            SELECT COUNT(*) FROM con_test.conference_participants
            WHERE conference_id = v_conf_id
            AND rsvp_status = 'accepted'
        ),
        declined_count = (
            SELECT COUNT(*) FROM con_test.conference_participants
            WHERE conference_id = v_conf_id
            AND rsvp_status = 'declined'
        ),
        updated_at = NOW()
    WHERE id = v_conf_id;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for RSVP count updates
DROP TRIGGER IF EXISTS trg_update_rsvp_counts ON con_test.conference_participants;
CREATE TRIGGER trg_update_rsvp_counts
AFTER INSERT OR UPDATE OF rsvp_status OR DELETE ON con_test.conference_participants
FOR EACH ROW EXECUTE FUNCTION con_test.update_conference_rsvp_counts();

-- Function to map chat role to conference role
CREATE OR REPLACE FUNCTION con_test.map_chat_role_to_conference_role(p_chat_role VARCHAR)
RETURNS con_test.conference_role AS $$
BEGIN
    CASE p_chat_role
        WHEN 'owner' THEN RETURN 'originator'::con_test.conference_role;
        WHEN 'admin' THEN RETURN 'originator'::con_test.conference_role;
        WHEN 'moderator' THEN RETURN 'moderator'::con_test.conference_role;
        ELSE RETURN 'participant'::con_test.conference_role;
    END CASE;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to check if actor can change target's role
CREATE OR REPLACE FUNCTION con_test.can_change_conference_role(
    p_actor_role con_test.conference_role,
    p_target_role con_test.conference_role,
    p_new_role con_test.conference_role
) RETURNS BOOLEAN AS $$
BEGIN
    -- Originator can change anyone's role
    IF p_actor_role = 'originator' THEN
        RETURN TRUE;
    END IF;

    -- Moderator can change participant, speaker, assistant roles
    -- But cannot change originator or moderator, and cannot assign originator or moderator
    IF p_actor_role = 'moderator' THEN
        IF p_target_role IN ('originator', 'moderator') THEN
            RETURN FALSE;
        END IF;
        IF p_new_role IN ('originator', 'moderator') THEN
            RETURN FALSE;
        END IF;
        RETURN TRUE;
    END IF;

    -- Others cannot change roles
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to get upcoming scheduled conferences for a user
CREATE OR REPLACE FUNCTION con_test.get_user_scheduled_conferences(
    p_user_id UUID,
    p_upcoming_only BOOLEAN DEFAULT TRUE,
    p_limit INT DEFAULT 50,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    chat_id UUID,
    created_by UUID,
    status VARCHAR(50),
    event_type con_test.event_type,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    series_id UUID,
    accepted_count INT,
    declined_count INT,
    participant_count BIGINT,
    user_role con_test.conference_role,
    user_rsvp con_test.rsvp_status,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        c.id,
        c.name,
        c.chat_id,
        c.created_by,
        c.status,
        c.event_type,
        c.scheduled_at,
        c.series_id,
        c.accepted_count,
        c.declined_count,
        (SELECT COUNT(*) FROM con_test.conference_participants cp
         WHERE cp.conference_id = c.id) AS participant_count,
        cp.role AS user_role,
        cp.rsvp_status AS user_rsvp,
        c.created_at
    FROM con_test.conferences c
    INNER JOIN con_test.conference_participants cp ON c.id = cp.conference_id AND cp.user_id = p_user_id
    WHERE c.event_type IN ('scheduled', 'recurring')
      AND c.status IN ('scheduled', 'active')
      AND (NOT p_upcoming_only OR c.scheduled_at >= NOW())
    ORDER BY c.scheduled_at ASC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Function to get conferences for a specific chat (for widget)
CREATE OR REPLACE FUNCTION con_test.get_chat_conferences(
    p_chat_id UUID,
    p_upcoming_only BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    chat_id UUID,
    created_by UUID,
    status VARCHAR(50),
    event_type con_test.event_type,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    accepted_count INT,
    declined_count INT,
    participant_count BIGINT,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.name,
        c.chat_id,
        c.created_by,
        c.status,
        c.event_type,
        c.scheduled_at,
        c.accepted_count,
        c.declined_count,
        (SELECT COUNT(*) FROM con_test.conference_participants cp
         WHERE cp.conference_id = c.id) AS participant_count,
        c.created_at
    FROM con_test.conferences c
    WHERE c.chat_id = p_chat_id
      AND c.status IN ('scheduled', 'active')
      AND (NOT p_upcoming_only OR c.scheduled_at >= NOW() OR c.event_type IN ('adhoc', 'adhoc_chat'))
    ORDER BY
        CASE WHEN c.event_type IN ('adhoc', 'adhoc_chat') AND c.status = 'active' THEN 0 ELSE 1 END,
        c.scheduled_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to get pending reminders
CREATE OR REPLACE FUNCTION con_test.get_pending_reminders(p_now TIMESTAMP WITH TIME ZONE)
RETURNS TABLE (
    id UUID,
    conference_id UUID,
    user_id UUID,
    remind_at TIMESTAMP WITH TIME ZONE,
    minutes_before INT,
    conference_name VARCHAR(255),
    scheduled_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        r.id,
        r.conference_id,
        r.user_id,
        r.remind_at,
        r.minutes_before,
        c.name AS conference_name,
        c.scheduled_at
    FROM con_test.conference_reminders r
    INNER JOIN con_test.conferences c ON r.conference_id = c.id
    WHERE r.sent = FALSE
      AND r.remind_at <= p_now
      AND c.status = 'scheduled';
END;
$$ LANGUAGE plpgsql;
