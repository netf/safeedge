-- Seed data for development and testing

-- Create default organization
INSERT INTO organizations (id, name, created_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Organization', NOW())
ON CONFLICT (id) DO NOTHING;
