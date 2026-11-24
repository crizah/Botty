-- Create "users" table
CREATE TABLE `users` (
  `id` text NULL,
  `email` text NOT NULL,
  `username` text NULL,
  `password_hash` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`)
);
-- Create index "users_email" to table: "users"
CREATE UNIQUE INDEX `users_email` ON `users` (`email`);
-- Create index "users_username" to table: "users"
CREATE UNIQUE INDEX `users_username` ON `users` (`username`);
-- Create "wa_groups" table
CREATE TABLE `wa_groups` (
  `id` text NULL,
  `jid` text NOT NULL,
  `name` text NOT NULL,
  `owner_user_id` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`owner_user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "wa_groups_jid" to table: "wa_groups"
CREATE UNIQUE INDEX `wa_groups_jid` ON `wa_groups` (`jid`);
-- Create index "idx_wa_groups_owner" to table: "wa_groups"
CREATE INDEX `idx_wa_groups_owner` ON `wa_groups` (`owner_user_id`);
-- Create "wa_group_members" table
CREATE TABLE `wa_group_members` (
  `id` text NULL,
  `group_id` text NOT NULL,
  `wa_id` text NOT NULL,
  `phone` text NULL,
  `push_name` text NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`group_id`) REFERENCES `wa_groups` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "wa_group_members_group_id_wa_id" to table: "wa_group_members"
CREATE UNIQUE INDEX `wa_group_members_group_id_wa_id` ON `wa_group_members` (`group_id`, `wa_id`);
-- Create "projects" table
CREATE TABLE `projects` (
  `id` text NULL,
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
-- Create "project_members" table
CREATE TABLE `project_members` (
  `project_id` text NOT NULL,
  `user_id` text NOT NULL,
  `role` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`project_id`, `user_id`),
  CONSTRAINT `0` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `1` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CHECK (role IN ('owner', 'editor', 'viewer'))
);
-- Create index "idx_project_members_user" to table: "project_members"
CREATE INDEX `idx_project_members_user` ON `project_members` (`user_id`);
-- Create "project_groups" table
CREATE TABLE `project_groups` (
  `project_id` text NOT NULL,
  `group_id` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`project_id`, `group_id`),
  CONSTRAINT `0` FOREIGN KEY (`group_id`) REFERENCES `wa_groups` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `1` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "project_groups_group_id" to table: "project_groups"
CREATE UNIQUE INDEX `project_groups_group_id` ON `project_groups` (`group_id`);
-- Create "tag_allowlist" table
CREATE TABLE `tag_allowlist` (
  `project_id` text NOT NULL,
  `group_id` text NOT NULL,
  `member_id` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`project_id`, `group_id`, `member_id`),
  CONSTRAINT `0` FOREIGN KEY (`project_id`, `group_id`) REFERENCES `project_groups` (`project_id`, `group_id`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `1` FOREIGN KEY (`member_id`) REFERENCES `wa_group_members` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "tasks" table
CREATE TABLE `tasks` (
  `id` text NULL,
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
-- Create index "tasks_project_id_number" to table: "tasks"
CREATE UNIQUE INDEX `tasks_project_id_number` ON `tasks` (`project_id`, `number`);
-- Create "reports" table
CREATE TABLE `reports` (
  `id` text NULL,
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
-- Create index "idx_reports_task" to table: "reports"
CREATE INDEX `idx_reports_task` ON `reports` (`task_id`);
-- Create "comments" table
CREATE TABLE `comments` (
  `id` text NULL,
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
-- Create index "idx_comments_task" to table: "comments"
CREATE INDEX `idx_comments_task` ON `comments` (`task_id`);
-- Create "audit_logs" table
CREATE TABLE `audit_logs` (
  `id` text NULL,
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
-- Create index "idx_audit_project" to table: "audit_logs"
CREATE INDEX `idx_audit_project` ON `audit_logs` (`project_id`, `created_at`);
-- Create index "idx_audit_task" to table: "audit_logs"
CREATE INDEX `idx_audit_task` ON `audit_logs` (`task_id`, `created_at`);
