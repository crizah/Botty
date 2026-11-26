-- Create "wa_messages" table
CREATE TABLE `wa_messages` (
  `id` text NOT NULL,
  `group_id` text NOT NULL,
  `message_id` text NOT NULL,
  `sender_wa_id` text NOT NULL,
  `sender_name` text NOT NULL DEFAULT '',
  `text` text NOT NULL,
  `quoted_id` text NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`group_id`) REFERENCES `wa_groups` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "wa_messages_group_id_message_id" to table: "wa_messages"
CREATE UNIQUE INDEX `wa_messages_group_id_message_id` ON `wa_messages` (`group_id`, `message_id`);
-- Create index "idx_wa_messages_created" to table: "wa_messages"
CREATE INDEX `idx_wa_messages_created` ON `wa_messages` (`created_at`);
