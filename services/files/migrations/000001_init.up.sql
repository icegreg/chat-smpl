-- Files service schema
-- Based on arch/FILE_SYSTEM_SERVICE_ARCHITECTURE.puml

-- Files table (central storage)
CREATE TABLE IF NOT EXISTS con_test.files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    file_path VARCHAR(512) NOT NULL,
    uploaded_by UUID NOT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'deleted')),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- File links table (context-based permissions)
CREATE TABLE IF NOT EXISTS con_test.file_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES con_test.files(id) ON DELETE CASCADE,
    uploaded_by UUID NOT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE
);

-- File link permissions table
CREATE TABLE IF NOT EXISTS con_test.file_link_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_link_id UUID NOT NULL REFERENCES con_test.file_links(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    can_view BOOLEAN DEFAULT TRUE,
    can_download BOOLEAN DEFAULT TRUE,
    can_delete BOOLEAN DEFAULT FALSE,
    CONSTRAINT unique_file_link_permission UNIQUE (file_link_id, user_id)
);

-- File share links table (public sharing)
CREATE TABLE IF NOT EXISTS con_test.file_share_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES con_test.files(id) ON DELETE CASCADE,
    token VARCHAR(64) UNIQUE NOT NULL,
    password VARCHAR(255),
    max_downloads INT,
    download_count INT DEFAULT 0,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE
);

-- Message file attachments table
CREATE TABLE IF NOT EXISTS con_test.message_file_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    file_link_id UUID NOT NULL REFERENCES con_test.file_links(id) ON DELETE CASCADE,
    sort_order INT NOT NULL DEFAULT 0,
    CONSTRAINT unique_message_attachment UNIQUE (message_id, file_link_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_files_uploaded_by ON con_test.files(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_files_status ON con_test.files(status);
CREATE INDEX IF NOT EXISTS idx_files_content_type ON con_test.files(content_type);
CREATE INDEX IF NOT EXISTS idx_file_links_file_id ON con_test.file_links(file_id);
CREATE INDEX IF NOT EXISTS idx_file_links_uploaded_by ON con_test.file_links(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_file_links_is_deleted ON con_test.file_links(is_deleted);
CREATE INDEX IF NOT EXISTS idx_file_link_permissions_file_link_id ON con_test.file_link_permissions(file_link_id);
CREATE INDEX IF NOT EXISTS idx_file_link_permissions_user_id ON con_test.file_link_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_file_share_links_file_id ON con_test.file_share_links(file_id);
CREATE INDEX IF NOT EXISTS idx_file_share_links_token ON con_test.file_share_links(token);
CREATE INDEX IF NOT EXISTS idx_file_share_links_is_active ON con_test.file_share_links(is_active);
CREATE INDEX IF NOT EXISTS idx_message_file_attachments_message_id ON con_test.message_file_attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_message_file_attachments_file_link_id ON con_test.message_file_attachments(file_link_id);
