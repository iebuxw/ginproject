DELETE FROM role_menus WHERE menu_id IN (51, 52, 53, 54, 55);
DELETE FROM menus WHERE id IN (51, 52, 53, 54, 55);
DELETE FROM cron_tasks WHERE id IN (10, 11);
DROP TABLE IF EXISTS `db_backups`;
