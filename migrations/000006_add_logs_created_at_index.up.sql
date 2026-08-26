-- 日志按 created_at 清理/查询索引（MySQL 不支持 CREATE INDEX IF NOT EXISTS，迁移只执行一次，幂等性由 golang-migrate 保证）
CREATE INDEX idx_operation_logs_created_at ON operation_logs (created_at);
CREATE INDEX idx_login_logs_created_at ON login_logs (created_at);
