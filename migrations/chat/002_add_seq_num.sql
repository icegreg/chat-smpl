-- Add sequence number to messages for reliable sync after reconnect
-- seq_num is unique per chat and auto-incremented

-- Add seq_num column
ALTER TABLE con_test.messages ADD COLUMN IF NOT EXISTS seq_num BIGINT;

-- Create a sequence table to track per-chat sequence numbers
CREATE TABLE IF NOT EXISTS con_test.chat_sequences (
    chat_id UUID PRIMARY KEY REFERENCES con_test.chats(id) ON DELETE CASCADE,
    last_seq_num BIGINT NOT NULL DEFAULT 0
);

-- Function to get next sequence number for a chat (atomic)
CREATE OR REPLACE FUNCTION con_test.get_next_seq_num(p_chat_id UUID)
RETURNS BIGINT AS $$
DECLARE
    v_seq_num BIGINT;
BEGIN
    INSERT INTO con_test.chat_sequences (chat_id, last_seq_num)
    VALUES (p_chat_id, 1)
    ON CONFLICT (chat_id) DO UPDATE SET last_seq_num = con_test.chat_sequences.last_seq_num + 1
    RETURNING last_seq_num INTO v_seq_num;

    RETURN v_seq_num;
END;
$$ LANGUAGE plpgsql;

-- Index for efficient sync queries
CREATE INDEX IF NOT EXISTS idx_messages_chat_seq ON con_test.messages(chat_id, seq_num);

-- Backfill existing messages with seq_num (ordered by sent_at)
DO $$
DECLARE
    chat RECORD;
    msg RECORD;
    seq BIGINT;
BEGIN
    FOR chat IN SELECT DISTINCT chat_id FROM con_test.messages LOOP
        seq := 0;
        FOR msg IN SELECT id FROM con_test.messages WHERE chat_id = chat.chat_id ORDER BY sent_at LOOP
            seq := seq + 1;
            UPDATE con_test.messages SET seq_num = seq WHERE id = msg.id;
        END LOOP;
        -- Update sequence table
        INSERT INTO con_test.chat_sequences (chat_id, last_seq_num)
        VALUES (chat.chat_id, seq)
        ON CONFLICT (chat_id) DO UPDATE SET last_seq_num = seq;
    END LOOP;
END $$;

-- Make seq_num NOT NULL after backfill
ALTER TABLE con_test.messages ALTER COLUMN seq_num SET NOT NULL;
