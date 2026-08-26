DELETE FROM role_menus WHERE role_id = 1 AND menu_id BETWEEN 41 AND 49;

DELETE FROM menus WHERE id BETWEEN 41 AND 49;

DROP TABLE IF EXISTS cron_task_executions;

DROP TABLE IF EXISTS cron_tasks;
