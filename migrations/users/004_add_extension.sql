-- Add extension field for SIP/Verto phone numbers
-- Extension range: 10000-99999

-- Enable pgcrypto for gen_random_bytes
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE con_test.users ADD COLUMN IF NOT EXISTS extension VARCHAR(10) UNIQUE;
ALTER TABLE con_test.users ADD COLUMN IF NOT EXISTS sip_password VARCHAR(64);

-- Create sequence for auto-assigning extensions
CREATE SEQUENCE IF NOT EXISTS con_test.user_extension_seq
    START WITH 10000
    INCREMENT BY 1
    MINVALUE 10000
    MAXVALUE 99999
    NO CYCLE;

-- Index for fast extension lookup
CREATE INDEX idx_users_extension ON con_test.users(extension) WHERE extension IS NOT NULL;

-- Function to get next available extension
CREATE OR REPLACE FUNCTION con_test.get_next_extension()
RETURNS VARCHAR(10) AS $$
DECLARE
    next_ext VARCHAR(10);
BEGIN
    next_ext := nextval('con_test.user_extension_seq')::VARCHAR;
    RETURN next_ext;
END;
$$ LANGUAGE plpgsql;

-- Function to generate SIP password (random 16-char hex string)
CREATE OR REPLACE FUNCTION con_test.generate_sip_password()
RETURNS VARCHAR(64) AS $$
BEGIN
    RETURN encode(gen_random_bytes(16), 'hex');
END;
$$ LANGUAGE plpgsql;

-- Assign extensions and SIP passwords to existing users (except guests)
DO $$
DECLARE
    user_record RECORD;
BEGIN
    FOR user_record IN
        SELECT id FROM con_test.users
        WHERE extension IS NULL AND role != 'guest'
        ORDER BY created_at
    LOOP
        UPDATE con_test.users
        SET extension = con_test.get_next_extension(),
            sip_password = con_test.generate_sip_password()
        WHERE id = user_record.id;
    END LOOP;
END $$;
