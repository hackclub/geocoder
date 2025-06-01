-- Update NULL owner, app_name, and environment fields in api_keys table
UPDATE api_keys SET owner = 'noowner' WHERE owner IS NULL;
UPDATE api_keys SET app_name = 'unknown' WHERE app_name IS NULL;
UPDATE api_keys SET environment = 'unknown' WHERE environment IS NULL;
