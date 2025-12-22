-- Refactor file groups: Files Service owns groups, Chat Service owns mapping
-- This migration removes chat-specific logic from Files Service

-- ============================================
-- STEP 1: Drop old structures
-- ============================================

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_add_file_link_group_permissions ON con_test.file_links;
DROP TRIGGER IF EXISTS trg_create_chat_file_access_groups ON con_test.chats;

-- Drop functions
DROP FUNCTION IF EXISTS con_test.add_file_link_group_permissions();
DROP FUNCTION IF EXISTS con_test.create_chat_file_access_groups();
DROP FUNCTION IF EXISTS con_test.check_file_access(UUID, UUID);

-- Drop view
DROP VIEW IF EXISTS con_test.effective_file_permissions;

-- Drop old tables
DROP TABLE IF EXISTS con_test.file_link_group_permissions;
DROP TABLE IF EXISTS con_test.chat_file_access_groups;

-- Remove chat_id from file_links
ALTER TABLE con_test.file_links DROP COLUMN IF EXISTS chat_id;

-- ============================================
-- STEP 2: Create new group structures
-- ============================================

-- File groups (Files Service owns this)
-- Groups have permissions, not individual file_links
CREATE TABLE IF NOT EXISTS con_test.file_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    can_read BOOLEAN DEFAULT TRUE,
    can_delete BOOLEAN DEFAULT FALSE,
    can_transfer BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Group members (just user_id, no role - Files doesn't know about roles)
CREATE TABLE IF NOT EXISTS con_test.file_group_members (
    group_id UUID NOT NULL REFERENCES con_test.file_groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

-- Link file_links to groups
CREATE TABLE IF NOT EXISTS con_test.file_link_groups (
    file_link_id UUID NOT NULL REFERENCES con_test.file_links(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES con_test.file_groups(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (file_link_id, group_id)
);

-- ============================================
-- STEP 3: Indexes
-- ============================================

CREATE INDEX IF NOT EXISTS idx_file_groups_name ON con_test.file_groups(name);
CREATE INDEX IF NOT EXISTS idx_file_group_members_user_id ON con_test.file_group_members(user_id);
CREATE INDEX IF NOT EXISTS idx_file_link_groups_group_id ON con_test.file_link_groups(group_id);

-- ============================================
-- STEP 4: Function to check file access
-- ============================================

-- Returns: 'none', 'read', 'delete', 'transfer'
-- Priority: owner > group permissions > individual permissions
CREATE OR REPLACE FUNCTION con_test.check_file_access(
    p_file_link_id UUID,
    p_user_id UUID
) RETURNS TEXT AS $$
DECLARE
    v_file_link RECORD;
    v_individual_perm RECORD;
    v_group_perms RECORD;
BEGIN
    -- Get file link info
    SELECT id, uploaded_by, is_deleted
    INTO v_file_link
    FROM con_test.file_links
    WHERE id = p_file_link_id;

    IF NOT FOUND OR v_file_link.is_deleted THEN
        RETURN 'none';
    END IF;

    -- Owner always has full access
    IF v_file_link.uploaded_by = p_user_id THEN
        RETURN 'transfer';
    END IF;

    -- Check group permissions (aggregate from all groups user is member of)
    SELECT
        BOOL_OR(fg.can_read) AS can_read,
        BOOL_OR(fg.can_delete) AS can_delete,
        BOOL_OR(fg.can_transfer) AS can_transfer
    INTO v_group_perms
    FROM con_test.file_link_groups flg
    JOIN con_test.file_groups fg ON fg.id = flg.group_id
    JOIN con_test.file_group_members fgm ON fgm.group_id = fg.id
    WHERE flg.file_link_id = p_file_link_id AND fgm.user_id = p_user_id;

    IF v_group_perms.can_transfer THEN
        RETURN 'transfer';
    END IF;
    IF v_group_perms.can_delete THEN
        RETURN 'delete';
    END IF;
    IF v_group_perms.can_read THEN
        RETURN 'read';
    END IF;

    -- Check individual permissions
    SELECT can_view, can_download, can_delete
    INTO v_individual_perm
    FROM con_test.file_link_permissions
    WHERE file_link_id = p_file_link_id AND user_id = p_user_id;

    IF FOUND THEN
        IF v_individual_perm.can_delete THEN
            RETURN 'delete';
        END IF;
        IF v_individual_perm.can_download THEN
            RETURN 'read';
        END IF;
        IF v_individual_perm.can_view THEN
            RETURN 'read';
        END IF;
    END IF;

    RETURN 'none';
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- STEP 5: Helper function to get files by group
-- ============================================

CREATE OR REPLACE FUNCTION con_test.get_files_by_group(
    p_group_id UUID
) RETURNS TABLE (
    file_link_id UUID,
    file_id UUID,
    filename VARCHAR,
    original_filename VARCHAR,
    content_type VARCHAR,
    size BIGINT,
    uploaded_by UUID,
    uploaded_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        fl.id AS file_link_id,
        f.id AS file_id,
        f.filename,
        f.original_filename,
        f.content_type,
        f.size,
        fl.uploaded_by,
        fl.uploaded_at
    FROM con_test.file_link_groups flg
    JOIN con_test.file_links fl ON fl.id = flg.file_link_id
    JOIN con_test.files f ON f.id = fl.file_id
    WHERE flg.group_id = p_group_id AND fl.is_deleted = FALSE;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- STEP 6: Cleanup old groups from users schema
-- ============================================

-- Remove chat-related groups that were created by old triggers
DELETE FROM con_test.groups WHERE name LIKE 'chat_%_moderate_all' OR name LIKE 'chat_%_view_all';
