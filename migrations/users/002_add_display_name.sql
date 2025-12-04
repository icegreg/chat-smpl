-- Add display_name column to users table
ALTER TABLE con_test.users ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);
