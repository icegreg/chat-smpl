-- Chat service schema
-- Note: Uses the same con_test schema as other services

-- Chats table
CREATE TABLE IF NOT EXISTS con_test.chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    chat_type VARCHAR(50) NOT NULL CHECK (chat_type IN ('private', 'group', 'channel')),
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Chat participants table
CREATE TABLE IF NOT EXISTS con_test.chat_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member', 'readonly')),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_chat_participant UNIQUE (chat_id, user_id)
);

-- Messages table
CREATE TABLE IF NOT EXISTS con_test.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES con_test.messages(id) ON DELETE SET NULL,
    sender_id UUID NOT NULL,
    content TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN DEFAULT FALSE
);

-- Message reactions table
CREATE TABLE IF NOT EXISTS con_test.message_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES con_test.messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    reaction VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_user_reaction UNIQUE (message_id, user_id, reaction)
);

-- Message read status table
CREATE TABLE IF NOT EXISTS con_test.message_readers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES con_test.messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    read_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_message_reader UNIQUE (message_id, user_id)
);

-- Chat favorites table
CREATE TABLE IF NOT EXISTS con_test.chat_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_chat_favorite UNIQUE (chat_id, user_id)
);

-- Polls table
CREATE TABLE IF NOT EXISTS con_test.polls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    message_id UUID REFERENCES con_test.messages(id) ON DELETE CASCADE,
    created_by UUID NOT NULL,
    question TEXT NOT NULL,
    is_multiple_choice BOOLEAN DEFAULT FALSE,
    is_anonymous BOOLEAN DEFAULT FALSE,
    is_finished BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    finished_at TIMESTAMP WITH TIME ZONE
);

-- Poll options table
CREATE TABLE IF NOT EXISTS con_test.poll_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES con_test.polls(id) ON DELETE CASCADE,
    text VARCHAR(255) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

-- Poll votes table
CREATE TABLE IF NOT EXISTS con_test.poll_votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES con_test.polls(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES con_test.poll_options(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    voted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_poll_vote UNIQUE (poll_id, option_id, user_id)
);

-- Archived chats table
CREATE TABLE IF NOT EXISTS con_test.archived_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    archived_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_archived_chat UNIQUE (chat_id, user_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_chats_created_by ON con_test.chats(created_by);
CREATE INDEX IF NOT EXISTS idx_chats_chat_type ON con_test.chats(chat_type);
CREATE INDEX IF NOT EXISTS idx_chat_participants_chat_id ON con_test.chat_participants(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_participants_user_id ON con_test.chat_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON con_test.messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON con_test.messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_parent_id ON con_test.messages(parent_id);
CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON con_test.messages(sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_reactions_message_id ON con_test.message_reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_message_readers_message_id ON con_test.message_readers(message_id);
CREATE INDEX IF NOT EXISTS idx_polls_chat_id ON con_test.polls(chat_id);
CREATE INDEX IF NOT EXISTS idx_poll_votes_poll_id ON con_test.poll_votes(poll_id);
CREATE INDEX IF NOT EXISTS idx_archived_chats_user_id ON con_test.archived_chats(user_id);

-- Trigger for chats table
DROP TRIGGER IF EXISTS update_chats_updated_at ON con_test.chats;
CREATE TRIGGER update_chats_updated_at
    BEFORE UPDATE ON con_test.chats
    FOR EACH ROW
    EXECUTE FUNCTION con_test.update_updated_at_column();
