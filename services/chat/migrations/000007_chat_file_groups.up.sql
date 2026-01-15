-- Chat file groups: mapping between chats and file groups
-- Chat Service owns this mapping, Files Service owns the groups

-- Group type enum
CREATE TYPE con_test.chat_file_group_type AS ENUM ('moderate', 'read');

-- Chat to file group mapping
-- Each chat can have multiple groups (typically moderate_all and read_all)
CREATE TABLE IF NOT EXISTS con_test.chat_file_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    group_id UUID NOT NULL,  -- References file_groups in Files Service (no FK - different service)
    group_type con_test.chat_file_group_type NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_chat_group UNIQUE (chat_id, group_type)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_chat_file_groups_chat_id ON con_test.chat_file_groups(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_file_groups_group_id ON con_test.chat_file_groups(group_id);

-- Chat file links: tracks which files are attached to which chat
-- This is the source of truth for "files in chat"
CREATE TABLE IF NOT EXISTS con_test.chat_file_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    file_link_id UUID NOT NULL,  -- References file_links in Files Service (no FK - different service)
    attached_by UUID NOT NULL,
    attached_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_chat_file_link UNIQUE (chat_id, file_link_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_chat_file_links_chat_id ON con_test.chat_file_links(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_file_links_file_link_id ON con_test.chat_file_links(file_link_id);
CREATE INDEX IF NOT EXISTS idx_chat_file_links_attached_by ON con_test.chat_file_links(attached_by);

-- Helper view: get file group IDs for a chat
CREATE OR REPLACE VIEW con_test.chat_file_groups_view AS
SELECT
    c.id AS chat_id,
    c.name AS chat_name,
    cfg_mod.group_id AS moderate_group_id,
    cfg_read.group_id AS read_group_id
FROM con_test.chats c
LEFT JOIN con_test.chat_file_groups cfg_mod
    ON cfg_mod.chat_id = c.id AND cfg_mod.group_type = 'moderate'
LEFT JOIN con_test.chat_file_groups cfg_read
    ON cfg_read.chat_id = c.id AND cfg_read.group_type = 'read';
