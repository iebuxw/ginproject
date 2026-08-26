-- 执行日志菜单（挂在"任务管理"目录下，id=41）
INSERT IGNORE INTO menus (id, parent_id, name, path, component, permission, type, icon, sort, created_at, updated_at)
VALUES (50, 41, '执行日志', 'task-logs', 'task/logs', 'cron:log', 2, 'document', 2, NOW(), NOW());

-- admin 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (1, 50);
