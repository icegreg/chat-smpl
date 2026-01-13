-- Health Service: Additional system users for voice/conference testing
-- These users are added to conferences during golden message voice tests

-- System user 2 for voice health checks
INSERT INTO con_test.users (id, username, email, password_hash, role, display_name, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    '__system_health_2__',
    'health2@system.internal',
    '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX',
    'user',
    'System Health Check 2',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- System user 3 for voice health checks
INSERT INTO con_test.users (id, username, email, password_hash, role, display_name, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000004',
    '__system_health_3__',
    'health3@system.internal',
    '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX',
    'user',
    'System Health Check 3',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Add system users 2 and 3 to health check chat (for any future tests)
INSERT INTO con_test.chat_participants (chat_id, user_id, role, joined_at)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000003',
    'member',
    NOW()
) ON CONFLICT (chat_id, user_id) DO NOTHING;

INSERT INTO con_test.chat_participants (chat_id, user_id, role, joined_at)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000004',
    'member',
    NOW()
) ON CONFLICT (chat_id, user_id) DO NOTHING;
