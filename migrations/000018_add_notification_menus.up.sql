-- 二级菜单：消息中心（所有登录用户可见，permission 留空走 JWT 层校验）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (70, 1, '消息中心', '/system/notification', '', 2, 'el-icon-bell', 5, NOW(), NOW());

-- 二级菜单：消息发送（管理端）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (71, 1, '消息发送', '/system/notification-send', 'notification:send', 2, 'el-icon-s-promotion', 6, NOW(), NOW());

-- 按钮权限点（挂在消息发送下）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (72, 71, '列表', '', 'notification:list', 3, '', 1, NOW(), NOW()),
       (73, 71, '删除', '', 'notification:delete', 3, '', 2, NOW(), NOW());

-- admin 角色绑定（消息中心所有角色都要见：绑定到现有全部角色）
INSERT IGNORE INTO role_menus (role_id, menu_id)
SELECT r.id, 70 FROM roles r WHERE r.status = 1;
INSERT IGNORE INTO role_menus (role_id, menu_id)
VALUES (1, 71), (1, 72), (1, 73);
