-- Initialize database schema
-- This file is executed first by docker-entrypoint-initdb.d

CREATE SCHEMA IF NOT EXISTS con_test;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Function to update updated_at timestamp (used by all services)
CREATE OR REPLACE FUNCTION con_test.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
