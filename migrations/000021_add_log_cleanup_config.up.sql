-- 日志清理策略配置
INSERT IGNORE INTO system_settings (setting_key, setting_value) VALUES
('log_cleanup_days', '180'),
('log_cleanup_scope', '["operation","login"]');

-- 日志设置菜单（父级：日志管理）
INSERT INTO menus (name, path, parent_id, type, permission, icon, sort_order)
SELECT '日志设置', '/system/log-setting', id, 2, 'log:setting', 'el-icon-setting', 99
FROM menus WHERE path = '/system/log' AND type = 1;
