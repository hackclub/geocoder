-- Add owner, app_name, and environment fields to api_keys table
ALTER TABLE api_keys ADD COLUMN owner VARCHAR(255);
ALTER TABLE api_keys ADD COLUMN app_name VARCHAR(255);
ALTER TABLE api_keys ADD COLUMN environment VARCHAR(255);

-- Add indexes for better querying
CREATE INDEX idx_api_keys_owner ON api_keys(owner);
CREATE INDEX idx_api_keys_app_name ON api_keys(app_name);
CREATE INDEX idx_api_keys_environment ON api_keys(environment);
