-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_users" table
CREATE TABLE `new_users` (
  `id` text NOT NULL,
  `email` text NOT NULL,
  `username` text NULL,
  `password_hash` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`)
);
-- Copy rows from old table "users" to new temporary table "new_users"
INSERT INTO `new_users` (`id`, `email`, `username`, `password_hash`, `created_at`) SELECT `id`, `email`, `username`, `password_hash`, `created_at` FROM `users`;
-- Drop "users" table after copying rows
DROP TABLE `users`;
-- Rename temporary table "new_users" to "users"
ALTER TABLE `new_users` RENAME TO `users`;
-- Create index "idx_users_email_nocase" to table: "users"
CREATE UNIQUE INDEX `idx_users_email_nocase` ON `users` (`email`);
-- Create index "idx_users_username_nocase" to table: "users"
CREATE UNIQUE INDEX `idx_users_username_nocase` ON `users` (`username`);
-- Create "new_wa_groups" table
CREATE TABLE `new_wa_groups` (
  `id` text NOT NULL,
  `jid` text NOT NULL,
  `name` text NOT NULL,
  `owner_user_id` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`owner_user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "wa_groups" to new temporary table "new_wa_groups"
INSERT INTO `new_wa_groups` (`id`, `jid`, `name`, `owner_user_id`, `created_at`) SELECT `id`, `jid`, `name`, `owner_user_id`, `created_at` FROM `wa_groups`;
-- Drop "wa_groups" table after copying rows
DROP TABLE `wa_groups`;
-- Rename temporary table "new_wa_groups" to "wa_groups"
ALTER TABLE `new_wa_groups` RENAME TO `wa_groups`;
-- Create index "wa_groups_jid" to table: "wa_groups"
CREATE UNIQUE INDEX `wa_groups_jid` ON `wa_groups` (`jid`);
-- Create index "idx_wa_groups_owner" to table: "wa_groups"
CREATE INDEX `idx_wa_groups_owner` ON `wa_groups` (`owner_user_id`);
-- Create "new_wa_group_members" table
CREATE TABLE `new_wa_group_members` (
  `id` text NOT NULL,
  `group_id` text NOT NULL,
  `wa_id` text NOT NULL,
  `phone` text NULL,
  `push_name` text NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`group_id`) REFERENCES `wa_groups` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "wa_group_members" to new temporary table "new_wa_group_members"
INSERT INTO `new_wa_group_members` (`id`, `group_id`, `wa_id`, `phone`, `push_name`, `created_at`) SELECT `id`, `group_id`, `wa_id`, `phone`, `push_name`, `created_at` FROM `wa_group_members`;
-- Drop "wa_group_members" table after copying rows
DROP TABLE `wa_group_members`;
-- Rename temporary table "new_wa_group_members" to "wa_group_members"
ALTER TABLE `new_wa_group_members` RENAME TO `wa_group_members`;
-- Create index "wa_group_members_group_id_wa_id" to table: "wa_group_members"
CREATE UNIQUE INDEX `wa_group_members_group_id_wa_id` ON `wa_group_members` (`group_id`, `wa_id`);
-- Create "new_projects" table
CREATE TABLE `new_projects` (
  `id` text NOT NULL,
  `name` text NOT NULL,
  `description` text NOT NULL DEFAULT '',
  `listen_mode` text NOT NULL DEFAULT 'bot_only',
  `created_by` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  `updated_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CHECK (listen_mode IN ('bot_only', 'all_texts'))
);
-- Copy rows from old table "projects" to new temporary table "new_projects"
INSERT INTO `new_projects` (`id`, `name`, `description`, `listen_mode`, `created_by`, `created_at`, `updated_at`) SELECT `id`, `name`, `description`, `listen_mode`, `created_by`, `created_at`, `updated_at` FROM `projects`;
-- Drop "projects" table after copying rows
DROP TABLE `projects`;
-- Rename temporary table "new_projects" to "projects"
ALTER TABLE `new_projects` RENAME TO `projects`;
-- Create "new_tasks" table
CREATE TABLE `new_tasks` (
  `id` text NOT NULL,
  `project_id` text NOT NULL,
  `number` integer NOT NULL,
  `title` text NOT NULL,
  `description` text NOT NULL DEFAULT '',
  `status` text NOT NULL DEFAULT 'todo',
  `created_by_user_id` text NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  `updated_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`created_by_user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `1` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CHECK (status IN ('todo', 'done'))
);
-- Copy rows from old table "tasks" to new temporary table "new_tasks"
INSERT INTO `new_tasks` (`id`, `project_id`, `number`, `title`, `description`, `status`, `created_by_user_id`, `created_at`, `updated_at`) SELECT `id`, `project_id`, `number`, `title`, `description`, `status`, `created_by_user_id`, `created_at`, `updated_at` FROM `tasks`;
-- Drop "tasks" table after copying rows
DROP TABLE `tasks`;
-- Rename temporary table "new_tasks" to "tasks"
ALTER TABLE `new_tasks` RENAME TO `tasks`;
-- Create index "tasks_project_id_number" to table: "tasks"
CREATE UNIQUE INDEX `tasks_project_id_number` ON `tasks` (`project_id`, `number`);
-- Create "new_reports" table
CREATE TABLE `new_reports` (
  `id` text NOT NULL,
  `task_id` text NOT NULL,
  `group_id` text NULL,
  `reporter_member_id` text NULL,
  `message_id` text NOT NULL DEFAULT '',
  `message_text` text NOT NULL DEFAULT '',
  `assigned_by_member_id` text NULL,
  `status` text NOT NULL DEFAULT 'todo',
  `closed_by_member_id` text NULL,
  `closed_by_user_id` text NULL,
  `closed_via` text NULL,
  `closed_at` datetime NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  `updated_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`closed_by_user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `1` FOREIGN KEY (`closed_by_member_id`) REFERENCES `wa_group_members` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `2` FOREIGN KEY (`assigned_by_member_id`) REFERENCES `wa_group_members` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `3` FOREIGN KEY (`reporter_member_id`) REFERENCES `wa_group_members` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `4` FOREIGN KEY (`group_id`) REFERENCES `wa_groups` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `5` FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CHECK (status IN ('todo', 'done')),
  CHECK (closed_via IN ('dashboard', 'tag', 'keyword'))
);
-- Copy rows from old table "reports" to new temporary table "new_reports"
INSERT INTO `new_reports` (`id`, `task_id`, `group_id`, `reporter_member_id`, `message_id`, `message_text`, `assigned_by_member_id`, `status`, `closed_by_member_id`, `closed_by_user_id`, `closed_via`, `closed_at`, `created_at`, `updated_at`) SELECT `id`, `task_id`, `group_id`, `reporter_member_id`, `message_id`, `message_text`, `assigned_by_member_id`, `status`, `closed_by_member_id`, `closed_by_user_id`, `closed_via`, `closed_at`, `created_at`, `updated_at` FROM `reports`;
-- Drop "reports" table after copying rows
DROP TABLE `reports`;
-- Rename temporary table "new_reports" to "reports"
ALTER TABLE `new_reports` RENAME TO `reports`;
-- Create index "idx_reports_task" to table: "reports"
CREATE INDEX `idx_reports_task` ON `reports` (`task_id`);
-- Create "new_comments" table
CREATE TABLE `new_comments` (
  `id` text NOT NULL,
  `task_id` text NOT NULL,
  `report_id` text NULL,
  `author_user_id` text NULL,
  `body` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`author_user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `1` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `2` FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "comments" to new temporary table "new_comments"
INSERT INTO `new_comments` (`id`, `task_id`, `report_id`, `author_user_id`, `body`, `created_at`) SELECT `id`, `task_id`, `report_id`, `author_user_id`, `body`, `created_at` FROM `comments`;
-- Drop "comments" table after copying rows
DROP TABLE `comments`;
-- Rename temporary table "new_comments" to "comments"
ALTER TABLE `new_comments` RENAME TO `comments`;
-- Create index "idx_comments_task" to table: "comments"
CREATE INDEX `idx_comments_task` ON `comments` (`task_id`);
-- Create "new_audit_logs" table
CREATE TABLE `new_audit_logs` (
  `id` text NOT NULL,
  `project_id` text NOT NULL,
  `task_id` text NULL,
  `report_id` text NULL,
  `actor_user_id` text NULL,
  `actor_kind` text NOT NULL DEFAULT 'user',
  `action` text NOT NULL,
  `old_value` text NOT NULL DEFAULT '',
  `new_value` text NOT NULL DEFAULT '',
  `detail` text NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`actor_user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `1` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `2` FOREIGN KEY (`task_id`) REFERENCES `tasks` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `3` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CHECK (actor_kind IN ('user', 'bot', 'system'))
);
-- Copy rows from old table "audit_logs" to new temporary table "new_audit_logs"
INSERT INTO `new_audit_logs` (`id`, `project_id`, `task_id`, `report_id`, `actor_user_id`, `actor_kind`, `action`, `old_value`, `new_value`, `detail`, `created_at`) SELECT `id`, `project_id`, `task_id`, `report_id`, `actor_user_id`, `actor_kind`, `action`, `old_value`, `new_value`, `detail`, `created_at` FROM `audit_logs`;
-- Drop "audit_logs" table after copying rows
DROP TABLE `audit_logs`;
-- Rename temporary table "new_audit_logs" to "audit_logs"
ALTER TABLE `new_audit_logs` RENAME TO `audit_logs`;
-- Create index "idx_audit_project" to table: "audit_logs"
CREATE INDEX `idx_audit_project` ON `audit_logs` (`project_id`, `created_at`);
-- Create index "idx_audit_task" to table: "audit_logs"
CREATE INDEX `idx_audit_task` ON `audit_logs` (`task_id`, `created_at`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
