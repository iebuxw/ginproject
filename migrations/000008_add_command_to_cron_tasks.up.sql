ALTER TABLE cron_tasks ADD COLUMN command VARCHAR(64) DEFAULT '' AFTER name;
ALTER TABLE cron_tasks ADD INDEX idx_command (command);
