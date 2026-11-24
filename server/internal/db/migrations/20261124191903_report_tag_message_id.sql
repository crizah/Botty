-- Add column "tag_message_id" to table: "reports"
ALTER TABLE `reports` ADD COLUMN `tag_message_id` text NOT NULL DEFAULT '';
