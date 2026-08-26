-- 恢复为自定义 HTTP 模式
UPDATE cron_tasks
SET command = '',
    url = 'http://go-app:8000/api/logs/cleanup?days=30',
    method = 'POST',
    headers = '{"X-Cleanup-Secret":"__LOG_CLEANUP_SECRET__"}',
    body = ''
WHERE command = 'clean_logs';
