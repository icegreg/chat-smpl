-- Chat-based file access groups
-- Provides scalable permissions for files in chats

-- Add chat_id to file_links to know which chat a file belongs to
ALTER TABLE con_test.file_links
ADD COLUMN IF NOT EXISTS chat_id UUID REFERENCES con_test.chats(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_file_links_chat_id ON con_test.file_links(chat_id);

-- Chat file access groups - links chats to their permission groups
-- Each chat has two groups: moderate_all (full access) and view_all (read access)
CREATE TABLE IF NOT EXISTS con_test.chat_file_access_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES con_test.chats(id) ON DELETE CASCADE,
    moderate_group_id UUID NOT NULL REFERENCES con_test.groups(id) ON DELETE CASCADE,
    view_group_id UUID NOT NULL REFERENCES con_test.groups(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_chat_file_access_groups UNIQUE (chat_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_file_access_groups_chat_id ON con_test.chat_file_access_groups(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_file_access_groups_moderate_group ON con_test.chat_file_access_groups(moderate_group_id);
CREATE INDEX IF NOT EXISTS idx_chat_file_access_groups_view_group ON con_test.chat_file_access_groups(view_group_id);

-- Group-based permissions for file links
-- Allows granting access to groups of users instead of individuals
CREATE TABLE IF NOT EXISTS con_test.file_link_group_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_link_id UUID NOT NULL REFERENCES con_test.file_links(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES con_test.groups(id) ON DELETE CASCADE,
    can_view BOOLEAN DEFAULT TRUE,
    can_download BOOLEAN DEFAULT TRUE,
    can_delete BOOLEAN DEFAULT FALSE,
    -- Files created after this timestamp are accessible to group members
    -- NULL means all files are accessible (for moderators)
    valid_from TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_file_link_group_permission UNIQUE (file_link_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_file_link_group_permissions_file_link_id ON con_test.file_link_group_permissions(file_link_id);
CREATE INDEX IF NOT EXISTS idx_file_link_group_permissions_group_id ON con_test.file_link_group_permissions(group_id);

-- Function to create file access groups for a chat
CREATE OR REPLACE FUNCTION con_test.create_chat_file_access_groups()
RETURNS TRIGGER AS $$
DECLARE
    moderate_group_id UUID;
    view_group_id UUID;
BEGIN
    -- Create moderate_all group
    INSERT INTO con_test.groups (id, name, description)
    VALUES (
        gen_random_uuid(),
        'chat_' || NEW.id || '_moderate_all',
        'Full access to all files in chat ' || NEW.name
    )
    RETURNING id INTO moderate_group_id;

    -- Create view_all group
    INSERT INTO con_test.groups (id, name, description)
    VALUES (
        gen_random_uuid(),
        'chat_' || NEW.id || '_view_all',
        'View access to files in chat ' || NEW.name
    )
    RETURNING id INTO view_group_id;

    -- Link groups to chat
    INSERT INTO con_test.chat_file_access_groups (chat_id, moderate_group_id, view_group_id)
    VALUES (NEW.id, moderate_group_id, view_group_id);

    -- Add creator to moderate group
    INSERT INTO con_test.group_members (group_id, user_id)
    VALUES (moderate_group_id, NEW.created_by);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-create groups when a chat is created
DROP TRIGGER IF EXISTS trg_create_chat_file_access_groups ON con_test.chats;
CREATE TRIGGER trg_create_chat_file_access_groups
AFTER INSERT ON con_test.chats
FOR EACH ROW
EXECUTE FUNCTION con_test.create_chat_file_access_groups();

-- Function to add group permissions when a file is uploaded to a chat
CREATE OR REPLACE FUNCTION con_test.add_file_link_group_permissions()
RETURNS TRIGGER AS $$
DECLARE
    access_groups RECORD;
BEGIN
    -- Only process if chat_id is set
    IF NEW.chat_id IS NOT NULL THEN
        -- Get the access groups for this chat
        SELECT moderate_group_id, view_group_id INTO access_groups
        FROM con_test.chat_file_access_groups
        WHERE chat_id = NEW.chat_id;

        IF FOUND THEN
            -- Add moderate group permission (full access, no time restriction)
            INSERT INTO con_test.file_link_group_permissions (file_link_id, group_id, can_view, can_download, can_delete, valid_from)
            VALUES (NEW.id, access_groups.moderate_group_id, TRUE, TRUE, TRUE, NULL)
            ON CONFLICT (file_link_id, group_id) DO NOTHING;

            -- Add view group permission (view/download only, no time restriction for now)
            INSERT INTO con_test.file_link_group_permissions (file_link_id, group_id, can_view, can_download, can_delete, valid_from)
            VALUES (NEW.id, access_groups.view_group_id, TRUE, TRUE, FALSE, NULL)
            ON CONFLICT (file_link_id, group_id) DO NOTHING;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-add group permissions when a file link is created with chat_id
DROP TRIGGER IF EXISTS trg_add_file_link_group_permissions ON con_test.file_links;
CREATE TRIGGER trg_add_file_link_group_permissions
AFTER INSERT ON con_test.file_links
FOR EACH ROW
EXECUTE FUNCTION con_test.add_file_link_group_permissions();

-- View to check effective permissions (combining individual + group permissions)
CREATE OR REPLACE VIEW con_test.effective_file_permissions AS
SELECT
    fl.id AS file_link_id,
    fl.chat_id,
    fl.uploaded_at AS file_uploaded_at,
    u.id AS user_id,
    cp.joined_at AS user_joined_at,
    cp.role AS chat_role,
    -- Check if user has individual permission
    COALESCE(flp.can_view, FALSE) AS individual_can_view,
    COALESCE(flp.can_download, FALSE) AS individual_can_download,
    COALESCE(flp.can_delete, FALSE) AS individual_can_delete,
    -- Check if user is in moderate group (no time restriction)
    EXISTS (
        SELECT 1 FROM con_test.chat_file_access_groups cfag
        JOIN con_test.group_members gm ON gm.group_id = cfag.moderate_group_id
        WHERE cfag.chat_id = fl.chat_id AND gm.user_id = u.id
    ) AS is_moderator,
    -- Check if user is in view group
    EXISTS (
        SELECT 1 FROM con_test.chat_file_access_groups cfag
        JOIN con_test.group_members gm ON gm.group_id = cfag.view_group_id
        WHERE cfag.chat_id = fl.chat_id AND gm.user_id = u.id
    ) AS is_viewer
FROM con_test.file_links fl
CROSS JOIN con_test.users u
LEFT JOIN con_test.chat_participants cp ON cp.chat_id = fl.chat_id AND cp.user_id = u.id
LEFT JOIN con_test.file_link_permissions flp ON flp.file_link_id = fl.id AND flp.user_id = u.id
WHERE fl.is_deleted = FALSE;

-- Function to check if user can access a file link
-- Returns: 'none', 'view', 'download', 'delete'
CREATE OR REPLACE FUNCTION con_test.check_file_access(
    p_file_link_id UUID,
    p_user_id UUID
) RETURNS TEXT AS $$
DECLARE
    v_file_link RECORD;
    v_is_uploader BOOLEAN;
    v_has_individual_perm RECORD;
    v_is_moderator BOOLEAN;
    v_is_viewer BOOLEAN;
    v_user_joined_at TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Get file link info
    SELECT id, chat_id, uploaded_by, uploaded_at, is_deleted
    INTO v_file_link
    FROM con_test.file_links
    WHERE id = p_file_link_id;

    IF NOT FOUND OR v_file_link.is_deleted THEN
        RETURN 'none';
    END IF;

    -- Check if user is the uploader (always has full access)
    v_is_uploader := v_file_link.uploaded_by = p_user_id;
    IF v_is_uploader THEN
        RETURN 'delete';
    END IF;

    -- Check individual permissions
    SELECT can_view, can_download, can_delete
    INTO v_has_individual_perm
    FROM con_test.file_link_permissions
    WHERE file_link_id = p_file_link_id AND user_id = p_user_id;

    IF FOUND AND v_has_individual_perm.can_delete THEN
        RETURN 'delete';
    END IF;

    -- If file is not in a chat, only individual permissions apply
    IF v_file_link.chat_id IS NULL THEN
        IF FOUND AND v_has_individual_perm.can_download THEN
            RETURN 'download';
        ELSIF FOUND AND v_has_individual_perm.can_view THEN
            RETURN 'view';
        ELSE
            RETURN 'none';
        END IF;
    END IF;

    -- Check group-based permissions for chat files

    -- Check if user is in moderate group (full access to all files)
    SELECT EXISTS (
        SELECT 1 FROM con_test.chat_file_access_groups cfag
        JOIN con_test.group_members gm ON gm.group_id = cfag.moderate_group_id
        WHERE cfag.chat_id = v_file_link.chat_id AND gm.user_id = p_user_id
    ) INTO v_is_moderator;

    IF v_is_moderator THEN
        RETURN 'delete';
    END IF;

    -- Check if user is in view group
    SELECT EXISTS (
        SELECT 1 FROM con_test.chat_file_access_groups cfag
        JOIN con_test.group_members gm ON gm.group_id = cfag.view_group_id
        WHERE cfag.chat_id = v_file_link.chat_id AND gm.user_id = p_user_id
    ) INTO v_is_viewer;

    IF v_is_viewer THEN
        -- Get user's join date to check time-based access
        SELECT joined_at INTO v_user_joined_at
        FROM con_test.chat_participants
        WHERE chat_id = v_file_link.chat_id AND user_id = p_user_id;

        -- If user joined before file was uploaded, they can access it
        -- (Or if we don't have join date, allow access)
        IF v_user_joined_at IS NULL OR v_file_link.uploaded_at >= v_user_joined_at THEN
            RETURN 'download';
        ELSE
            -- User joined after file was uploaded - no access via view_all
            -- Fall through to check individual permissions
            NULL;
        END IF;
    END IF;

    -- Fall back to individual permissions
    IF FOUND AND v_has_individual_perm.can_download THEN
        RETURN 'download';
    ELSIF FOUND AND v_has_individual_perm.can_view THEN
        RETURN 'view';
    END IF;

    RETURN 'none';
END;
$$ LANGUAGE plpgsql;

-- Create file access groups for existing chats
DO $$
DECLARE
    chat_record RECORD;
    moderate_group_id UUID;
    view_group_id UUID;
BEGIN
    FOR chat_record IN SELECT id, name, created_by FROM con_test.chats LOOP
        -- Check if groups already exist
        IF NOT EXISTS (SELECT 1 FROM con_test.chat_file_access_groups WHERE chat_id = chat_record.id) THEN
            -- Create moderate_all group
            INSERT INTO con_test.groups (id, name, description)
            VALUES (
                gen_random_uuid(),
                'chat_' || chat_record.id || '_moderate_all',
                'Full access to all files in chat ' || chat_record.name
            )
            RETURNING id INTO moderate_group_id;

            -- Create view_all group
            INSERT INTO con_test.groups (id, name, description)
            VALUES (
                gen_random_uuid(),
                'chat_' || chat_record.id || '_view_all',
                'View access to files in chat ' || chat_record.name
            )
            RETURNING id INTO view_group_id;

            -- Link groups to chat
            INSERT INTO con_test.chat_file_access_groups (chat_id, moderate_group_id, view_group_id)
            VALUES (chat_record.id, moderate_group_id, view_group_id);

            -- Add creator to moderate group
            INSERT INTO con_test.group_members (group_id, user_id)
            VALUES (moderate_group_id, chat_record.created_by)
            ON CONFLICT DO NOTHING;
        END IF;
    END LOOP;
END $$;
