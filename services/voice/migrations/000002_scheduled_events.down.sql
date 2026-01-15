-- Rollback scheduled events migration
DROP FUNCTION IF EXISTS con_test.get_pending_reminders(TIMESTAMP WITH TIME ZONE);
DROP FUNCTION IF EXISTS con_test.get_chat_conferences(UUID, BOOLEAN);
DROP FUNCTION IF EXISTS con_test.get_user_scheduled_conferences(UUID, BOOLEAN, INT, INT);
DROP FUNCTION IF EXISTS con_test.can_change_conference_role(con_test.conference_role, con_test.conference_role, con_test.conference_role);
DROP FUNCTION IF EXISTS con_test.map_chat_role_to_conference_role(VARCHAR);
DROP TRIGGER IF EXISTS trg_update_rsvp_counts ON con_test.conference_participants;
DROP FUNCTION IF EXISTS con_test.update_conference_rsvp_counts();
DROP TABLE IF EXISTS con_test.conference_reminders;
DROP INDEX IF EXISTS idx_conf_participants_role;
DROP INDEX IF EXISTS idx_conf_participants_rsvp;
ALTER TABLE con_test.conference_participants
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS rsvp_status,
    DROP COLUMN IF EXISTS rsvp_at;
DROP TABLE IF EXISTS con_test.conference_recurrence;
DROP INDEX IF EXISTS idx_conferences_event_type;
DROP INDEX IF EXISTS idx_conferences_series_id;
DROP INDEX IF EXISTS idx_conferences_scheduled_at;
ALTER TABLE con_test.conferences
    DROP COLUMN IF EXISTS event_type,
    DROP COLUMN IF EXISTS scheduled_at,
    DROP COLUMN IF EXISTS series_id,
    DROP COLUMN IF EXISTS accepted_count,
    DROP COLUMN IF EXISTS declined_count;
DROP TYPE IF EXISTS con_test.recurrence_frequency;
DROP TYPE IF EXISTS con_test.rsvp_status;
DROP TYPE IF EXISTS con_test.conference_role;
DROP TYPE IF EXISTS con_test.event_type;
