-- Create "report_duplicates" table
CREATE TABLE `report_duplicates` (
  `id` text NOT NULL,
  `report_id` text NOT NULL,
  `message_id` text NOT NULL DEFAULT '',
  `tag_message_id` text NOT NULL DEFAULT '',
  `message_text` text NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_report_duplicates_report" to table: "report_duplicates"
CREATE INDEX `idx_report_duplicates_report` ON `report_duplicates` (`report_id`);
