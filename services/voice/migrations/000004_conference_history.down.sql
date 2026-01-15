-- Rollback conference history migration
DROP FUNCTION IF EXISTS con_test.get_conference_moderator_actions(UUID);
DROP FUNCTION IF EXISTS con_test.get_conference_participant_sessions(UUID);
DROP FUNCTION IF EXISTS con_test.get_chat_conference_history(UUID, INT, INT);
DROP INDEX IF EXISTS idx_conferences_thread_id;
ALTER TABLE con_test.conferences DROP COLUMN IF EXISTS thread_id;
DROP INDEX IF EXISTS idx_conf_mod_actions_created;
DROP INDEX IF EXISTS idx_conf_mod_actions_target;
DROP INDEX IF EXISTS idx_conf_mod_actions_actor;
DROP INDEX IF EXISTS idx_conf_mod_actions_conference;
DROP TABLE IF EXISTS con_test.conference_moderator_actions;
