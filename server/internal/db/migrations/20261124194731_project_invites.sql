-- Create "project_invites" table
CREATE TABLE `project_invites` (
  `id` text NOT NULL,
  `project_id` text NOT NULL,
  `email` text NOT NULL,
  `role` text NOT NULL,
  `invited_by` text NOT NULL,
  `expires_at` datetime NOT NULL,
  `accepted_at` datetime NULL,
  `revoked_at` datetime NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`invited_by`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `1` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CHECK (role IN ('editor', 'viewer'))
);
-- Create index "idx_project_invites_project" to table: "project_invites"
CREATE INDEX `idx_project_invites_project` ON `project_invites` (`project_id`);
