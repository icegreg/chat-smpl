-- Health Service: System User and Chat for golden message testing
-- This migration creates the system user and chat used by health-service
-- for end-to-end message delivery testing.

-- System user for health checks (uses fixed UUID for consistency)
INSERT INTO con_test.users (id, username, email, password_hash, role, display_name, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '__system_health__',
    'health@system.internal',
    '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX',
    'user',
    'System Health Check',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- System chat for health checks
INSERT INTO con_test.chats (id, name, chat_type, created_by, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '__system_health_chat__',
    'private',
    '00000000-0000-0000-0000-000000000001',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Add system user as participant in the health check chat
INSERT INTO con_test.chat_participants (chat_id, user_id, role, joined_at)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'admin',
    NOW()
) ON CONFLICT (chat_id, user_id) DO NOTHING;

-- Create index for cleaning up old health check messages (optional maintenance)
-- Health check messages contain '__health_check__' prefix
CREATE INDEX IF NOT EXISTS idx_messages_health_check
ON con_test.messages (chat_id, sent_at)
WHERE chat_id = '00000000-0000-0000-0000-000000000002';
