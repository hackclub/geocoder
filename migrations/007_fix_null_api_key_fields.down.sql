-- Revert NULL field fixes (set back to NULL where they were default values)
UPDATE api_keys SET owner = NULL WHERE owner = 'noowner';
UPDATE api_keys SET app_name = NULL WHERE app_name = 'unknown';
UPDATE api_keys SET environment = NULL WHERE environment = 'unknown';
