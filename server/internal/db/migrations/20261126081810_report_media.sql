-- Create "report_media" table
CREATE TABLE `report_media` (
  `id` text NOT NULL,
  `report_id` text NOT NULL,
  `message_id` text NOT NULL DEFAULT '',
  `file` text NOT NULL,
  `mime` text NOT NULL,
  `size` integer NOT NULL,
  `created_at` datetime NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_report_media_report" to table: "report_media"
CREATE INDEX `idx_report_media_report` ON `report_media` (`report_id`);
