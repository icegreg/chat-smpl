-- Add channel_uuid to conference_participants for tracking FreeSWITCH channel events
ALTER TABLE con_test.conference_participants
ADD COLUMN IF NOT EXISTS channel_uuid VARCHAR(255);

-- Create index for fast lookup on CHANNEL_HANGUP events
CREATE INDEX IF NOT EXISTS idx_conf_participants_channel_uuid
ON con_test.conference_participants(channel_uuid)
WHERE channel_uuid IS NOT NULL;

-- Add comment
COMMENT ON COLUMN con_test.conference_participants.channel_uuid IS 'FreeSWITCH channel Unique-ID for tracking CHANNEL_HANGUP events';
