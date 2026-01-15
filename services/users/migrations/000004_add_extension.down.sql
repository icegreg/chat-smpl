DROP FUNCTION IF EXISTS con_test.generate_sip_password();
DROP FUNCTION IF EXISTS con_test.get_next_extension();
DROP INDEX IF EXISTS con_test.idx_users_extension;
DROP SEQUENCE IF EXISTS con_test.user_extension_seq;
ALTER TABLE con_test.users DROP COLUMN IF EXISTS sip_password;
ALTER TABLE con_test.users DROP COLUMN IF EXISTS extension;
