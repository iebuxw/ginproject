-- 执行日志菜单（挂在"日志管理"目录下，parent_id=26）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (50, 26, '执行日志', '/system/task-logs', 'cron:log', 2, 'el-icon-s-logs', 3, NOW(), NOW());

-- admin 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (1, 50);
