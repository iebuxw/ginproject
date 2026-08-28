-- 回滚：恢复错位的菜单编辑/删除按钮及角色绑定（按 000014 之前的历史状态）

INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, status)
VALUES (22, 6, '菜单编辑', NULL, 'menu:edit', 3, '', 4, 1),
       (23, 6, '菜单删除', NULL, 'menu:delete', 3, '', 5, 1);

INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (1, 22), (1, 23);
