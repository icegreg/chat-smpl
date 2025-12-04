-- Users service schema
CREATE SCHEMA IF NOT EXISTS con_test;

-- Users table
CREATE TABLE IF NOT EXISTS con_test.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user' CHECK (role IN ('owner', 'moderator', 'user', 'guest')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS con_test.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES con_test.users(id) ON DELETE CASCADE,
    token VARCHAR(512) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_token UNIQUE (token)
);

-- Groups table
CREATE TABLE IF NOT EXISTS con_test.groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Group members table
CREATE TABLE IF NOT EXISTS con_test.group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES con_test.groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES con_test.users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_group_member UNIQUE (group_id, user_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON con_test.users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON con_test.users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON con_test.users(role);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON con_test.refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON con_test.refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON con_test.group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON con_test.group_members(user_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION con_test.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for users table
DROP TRIGGER IF EXISTS update_users_updated_at ON con_test.users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON con_test.users
    FOR EACH ROW
    EXECUTE FUNCTION con_test.update_updated_at_column();

-- Trigger for groups table
DROP TRIGGER IF EXISTS update_groups_updated_at ON con_test.groups;
CREATE TRIGGER update_groups_updated_at
    BEFORE UPDATE ON con_test.groups
    FOR EACH ROW
    EXECUTE FUNCTION con_test.update_updated_at_column();
