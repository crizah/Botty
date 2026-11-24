-- Drop index "idx_users_email_nocase" from table: "users"
DROP INDEX `idx_users_email_nocase`;
-- Drop index "idx_users_username_nocase" from table: "users"
DROP INDEX `idx_users_username_nocase`;
-- Create index "idx_users_email_lower" to table: "users"
CREATE UNIQUE INDEX `idx_users_email_lower` ON `users` ((lower(email)));
-- Create index "idx_users_username_lower" to table: "users"
CREATE UNIQUE INDEX `idx_users_username_lower` ON `users` ((lower(username)));
