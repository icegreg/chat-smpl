-- Migration: 008_conference_threads.sql
-- Description: Add support for conference threads (linked to voice conferences)

-- Добавляем новый тип треда 'conference' для мероприятий
-- Сначала удаляем старый constraint
ALTER TABLE con_test.threads DROP CONSTRAINT IF EXISTS threads_thread_type_check;

-- Создаем новый constraint с типом 'conference'
ALTER TABLE con_test.threads
ADD CONSTRAINT threads_thread_type_check
    CHECK (thread_type IN ('user', 'system', 'conference'));

-- Добавляем связь с конференцией
ALTER TABLE con_test.threads
ADD COLUMN IF NOT EXISTS conference_id UUID;

COMMENT ON COLUMN con_test.threads.conference_id IS 'ID конференции для conference-type тредов';

-- Индекс для быстрого поиска треда по conference_id
CREATE INDEX IF NOT EXISTS idx_threads_conference_id
    ON con_test.threads(conference_id)
    WHERE conference_id IS NOT NULL;

-- Уникальный индекс: одна конференция = один тред
CREATE UNIQUE INDEX IF NOT EXISTS idx_threads_conference_unique
    ON con_test.threads(conference_id)
    WHERE conference_id IS NOT NULL;

-- Функция для создания или получения conference треда
CREATE OR REPLACE FUNCTION con_test.get_or_create_conference_thread(
    p_chat_id UUID,
    p_conference_id UUID,
    p_title VARCHAR(255)
)
RETURNS con_test.threads AS $$
DECLARE
    v_thread con_test.threads;
BEGIN
    -- Пытаемся найти существующий тред для этой конференции
    SELECT * INTO v_thread
    FROM con_test.threads
    WHERE conference_id = p_conference_id;

    -- Если не нашли, создаем новый
    IF v_thread.id IS NULL THEN
        INSERT INTO con_test.threads (
            chat_id,
            conference_id,
            thread_type,
            title,
            created_by
        ) VALUES (
            p_chat_id,
            p_conference_id,
            'conference',
            p_title,
            NULL  -- системный тред, без создателя
        )
        RETURNING * INTO v_thread;
    END IF;

    RETURN v_thread;
END;
$$ LANGUAGE plpgsql;

-- Функция для получения треда конференции
CREATE OR REPLACE FUNCTION con_test.get_conference_thread(
    p_conference_id UUID
)
RETURNS con_test.threads AS $$
DECLARE
    v_thread con_test.threads;
BEGIN
    SELECT * INTO v_thread
    FROM con_test.threads
    WHERE conference_id = p_conference_id;

    RETURN v_thread;
END;
$$ LANGUAGE plpgsql;
