-- Add avatar_url column to users table
ALTER TABLE con_test.users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(512);
