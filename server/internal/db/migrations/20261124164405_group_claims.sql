-- Create "group_claims" table
CREATE TABLE `group_claims` (
  `id` text NOT NULL,
  `code` text NOT NULL,
  `user_id` text NOT NULL,
  `expires_at` datetime NOT NULL,
  `used_at` datetime NULL,
  `group_id` text NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`group_id`) REFERENCES `wa_groups` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "group_claims_code" to table: "group_claims"
CREATE UNIQUE INDEX `group_claims_code` ON `group_claims` (`code`);
-- Create index "idx_group_claims_user" to table: "group_claims"
CREATE INDEX `idx_group_claims_user` ON `group_claims` (`user_id`);
