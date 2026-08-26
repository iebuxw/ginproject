DELETE FROM cron_tasks WHERE name = '日志清理' AND url LIKE '%logs/cleanup%';
