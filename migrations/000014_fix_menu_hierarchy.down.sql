-- 回滚菜单归类修正：恢复 000014 之前的挂载关系与 cron:log 按钮

-- 执行日志移回"日志管理"目录下（parent_id=26 为 000010 原值）
UPDATE menus
SET parent_id = 26, sort = 3
WHERE type = 2 AND path = '/system/task-logs';

-- 数据库备份移回"日志管理"目录下（parent_id=26 为 000011 原值）
UPDATE menus
SET parent_id = 26, sort = 4
WHERE type = 2 AND path = '/system/backup';

-- 恢复"定时任务"下的 cron:log 按钮（id=49）及 admin 绑定
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (49, 42, '执行日志', NULL, 'cron:log', 3, '', 7, NOW(), NOW());

INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (1, 49);
