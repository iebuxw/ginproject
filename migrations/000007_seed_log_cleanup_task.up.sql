-- 种子：日志清理定时任务（替代手工页面配置，保证各环境上线即有）
--
-- 原页面手工任务参数（配置参考）：
--   名称：日志定时清理
--   URL：http://go-app:8000/api/logs/cleanup?secret=<LOG_CLEANUP_SECRET>&days=30
--   Method：POST，Cron：0 0 3 * * *，Timeout：30，备注：每天凌晨3点清理30天前日志
-- 种子任务差异：
--   - secret 从 URL 移到 Headers（占位符 __LOG_CLEANUP_SECRET__，应用启动时替换为 .env 实际值），避免密钥落 nginx 访问日志
--   - 先删后插保证唯一：各环境恰好一条规范清理任务

DELETE FROM cron_tasks WHERE url LIKE '%logs/cleanup%';

INSERT INTO cron_tasks (name, url, method, headers, body, cron, timeout, status, remark) VALUES
('日志清理', 'http://go-app:8000/api/logs/cleanup?days=30', 'POST',
 '{"X-Cleanup-Secret":"__LOG_CLEANUP_SECRET__"}', '', '0 0 3 * * *', 30, 1, '每天凌晨3点清理30天前日志');
