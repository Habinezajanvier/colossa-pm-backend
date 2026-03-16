ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_full_name VARCHAR(255) NOT NULL DEFAULT '';
 
-- Remove the default after adding so future inserts must provide it
ALTER TABLE audit_logs ALTER COLUMN user_full_name DROP DEFAULT;