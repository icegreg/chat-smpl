-- Migration: 004_conference_history.sql
-- Description: Add moderator actions logging and thread linking for conferences

-- Таблица логирования действий модераторов
CREATE TABLE IF NOT EXISTS con_test.conference_moderator_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conference_id UUID NOT NULL REFERENCES con_test.conferences(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL,
    target_user_id UUID,
    action_type VARCHAR(50) NOT NULL CHECK (action_type IN (
        'mute', 'unmute', 'kick', 'role_change', 'start_recording', 'stop_recording'
    )),
    details JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE con_test.conference_moderator_actions IS 'Лог действий модераторов в конференциях';
COMMENT ON COLUMN con_test.conference_moderator_actions.actor_id IS 'ID пользователя, выполнившего действие';
COMMENT ON COLUMN con_test.conference_moderator_actions.target_user_id IS 'ID пользователя, на которого направлено действие (NULL для conference-level действий)';
COMMENT ON COLUMN con_test.conference_moderator_actions.action_type IS 'Тип действия: mute, unmute, kick, role_change, start_recording, stop_recording';
COMMENT ON COLUMN con_test.conference_moderator_actions.details IS 'Дополнительные данные (например, {"old_role": "participant", "new_role": "moderator"})';

-- Индексы для conference_moderator_actions
CREATE INDEX IF NOT EXISTS idx_conf_mod_actions_conference
    ON con_test.conference_moderator_actions(conference_id);
CREATE INDEX IF NOT EXISTS idx_conf_mod_actions_actor
    ON con_test.conference_moderator_actions(actor_id);
CREATE INDEX IF NOT EXISTS idx_conf_mod_actions_target
    ON con_test.conference_moderator_actions(target_user_id)
    WHERE target_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conf_mod_actions_created
    ON con_test.conference_moderator_actions(created_at DESC);

-- Связь конференции с тредом чата
ALTER TABLE con_test.conferences
    ADD COLUMN IF NOT EXISTS thread_id UUID;

COMMENT ON COLUMN con_test.conferences.thread_id IS 'ID системного треда в чате для логирования активности мероприятия';

CREATE INDEX IF NOT EXISTS idx_conferences_thread_id
    ON con_test.conferences(thread_id)
    WHERE thread_id IS NOT NULL;

-- Функция для получения истории конференций чата с полными данными участников
CREATE OR REPLACE FUNCTION con_test.get_chat_conference_history(
    p_chat_id UUID,
    p_limit INT DEFAULT 20,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    status VARCHAR(50),
    event_type VARCHAR(50),
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    participant_count BIGINT,
    thread_id UUID,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.name,
        c.status,
        c.event_type,
        c.started_at,
        c.ended_at,
        COUNT(DISTINCT cp.user_id) AS participant_count,
        c.thread_id,
        c.created_at
    FROM con_test.conferences c
    LEFT JOIN con_test.conference_participants cp ON cp.conference_id = c.id
    WHERE c.chat_id = p_chat_id
      AND c.status IN ('ended', 'active')
    GROUP BY c.id
    ORDER BY c.started_at DESC NULLS LAST, c.created_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Функция для получения всех сессий участников конференции (включая повторные входы)
CREATE OR REPLACE FUNCTION con_test.get_conference_participant_sessions(
    p_conference_id UUID
)
RETURNS TABLE (
    user_id UUID,
    username VARCHAR(255),
    display_name VARCHAR(255),
    avatar_url VARCHAR(512),
    joined_at TIMESTAMP WITH TIME ZONE,
    left_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50),
    role VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        cp.user_id,
        u.username,
        u.display_name,
        u.avatar_url,
        cp.joined_at,
        cp.left_at,
        cp.status,
        cp.role
    FROM con_test.conference_participants cp
    LEFT JOIN con_test.users u ON u.id = cp.user_id
    WHERE cp.conference_id = p_conference_id
    ORDER BY cp.user_id, cp.joined_at;
END;
$$ LANGUAGE plpgsql;

-- Функция для получения действий модераторов с именами пользователей
CREATE OR REPLACE FUNCTION con_test.get_conference_moderator_actions(
    p_conference_id UUID
)
RETURNS TABLE (
    id UUID,
    actor_id UUID,
    actor_username VARCHAR(255),
    actor_display_name VARCHAR(255),
    target_user_id UUID,
    target_username VARCHAR(255),
    target_display_name VARCHAR(255),
    action_type VARCHAR(50),
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        cma.id,
        cma.actor_id,
        actor.username AS actor_username,
        actor.display_name AS actor_display_name,
        cma.target_user_id,
        target.username AS target_username,
        target.display_name AS target_display_name,
        cma.action_type,
        cma.details,
        cma.created_at
    FROM con_test.conference_moderator_actions cma
    LEFT JOIN con_test.users actor ON actor.id = cma.actor_id
    LEFT JOIN con_test.users target ON target.id = cma.target_user_id
    WHERE cma.conference_id = p_conference_id
    ORDER BY cma.created_at ASC;
END;
$$ LANGUAGE plpgsql;
