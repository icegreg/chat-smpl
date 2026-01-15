-- Rollback voice init migration
DROP FUNCTION IF EXISTS con_test.get_user_active_conferences(UUID);
DROP FUNCTION IF EXISTS con_test.cleanup_expired_verto_credentials();
DROP VIEW IF EXISTS con_test.call_history;
DROP TABLE IF EXISTS con_test.verto_credentials;
DROP TABLE IF EXISTS con_test.calls;
DROP TABLE IF EXISTS con_test.conference_participants;
DROP TABLE IF EXISTS con_test.conferences;
