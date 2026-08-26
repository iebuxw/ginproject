-- 将现有日志清理任务改为使用预定义命令模式
UPDATE cron_tasks
SET command = 'clean_logs',
    url = '',
    method = 'POST',
    headers = '',
    body = ''
WHERE url LIKE '%logs/cleanup%';
