-- Drop triggers
DROP TRIGGER IF EXISTS update_groups_updated_at ON con_test.groups;
DROP TRIGGER IF EXISTS update_users_updated_at ON con_test.users;

-- Drop function
DROP FUNCTION IF EXISTS con_test.update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS con_test.idx_group_members_user_id;
DROP INDEX IF EXISTS con_test.idx_group_members_group_id;
DROP INDEX IF EXISTS con_test.idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS con_test.idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS con_test.idx_users_role;
DROP INDEX IF EXISTS con_test.idx_users_username;
DROP INDEX IF EXISTS con_test.idx_users_email;

-- Drop tables
DROP TABLE IF EXISTS con_test.group_members;
DROP TABLE IF EXISTS con_test.groups;
DROP TABLE IF EXISTS con_test.refresh_tokens;
DROP TABLE IF EXISTS con_test.users;
