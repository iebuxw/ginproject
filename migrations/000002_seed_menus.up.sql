-- 一级目录
INSERT IGNORE INTO menus (id, parent_id, name, icon, path, type, permission, sort, status) VALUES
(1, 0, '系统管理', 'el-icon-setting', '/system', 1, '', 1, 1),
(2, 0, '日志管理', 'el-icon-document', '/system/log-mgr', 1, '', 2, 1),
(3, 0, '数据字典', 'el-icon-notebook-2', '/system/dict-type', 1, '', 3, 1);

-- 系统管理子菜单
INSERT IGNORE INTO menus (id, parent_id, name, icon, path, type, permission, sort, status) VALUES
(4, 1, '用户管理', 'el-icon-user', '/system/user', 2, '', 1, 1),
(5, 1, '角色管理', 'el-icon-s-custom', '/system/role', 2, '', 2, 1),
(6, 1, '菜单管理', 'el-icon-menu', '/system/menu', 2, '', 3, 1);

-- 日志管理子菜单
INSERT IGNORE INTO menus (id, parent_id, name, icon, path, type, permission, sort, status) VALUES
(7, 2, '操作日志', 'el-icon-document', '/system/log', 2, '', 1, 1),
(8, 2, '登录日志', 'el-icon-document-checked', '/system/login-log', 2, '', 2, 1);

-- 用户管理按钮
INSERT IGNORE INTO menus (id, parent_id, name, type, permission, sort, status) VALUES
(9, 4, '用户列表', 3, 'user:list', 1, 1),
(10, 4, '用户查询', 3, 'user:query', 2, 1),
(11, 4, '用户新增', 3, 'user:add', 3, 1),
(12, 4, '用户编辑', 3, 'user:edit', 4, 1),
(13, 4, '用户删除', 3, 'user:delete', 5, 1);

-- 角色管理按钮
INSERT IGNORE INTO menus (id, parent_id, name, type, permission, sort, status) VALUES
(14, 5, '角色列表', 3, 'role:list', 1, 1),
(15, 5, '角色查询', 3, 'role:query', 2, 1),
(16, 5, '角色新增', 3, 'role:add', 3, 1),
(17, 5, '角色编辑', 3, 'role:edit', 4, 1),
(18, 5, '角色删除', 3, 'role:delete', 5, 1);

-- 菜单管理按钮
INSERT IGNORE INTO menus (id, parent_id, name, type, permission, sort, status) VALUES
(19, 6, '菜单列表', 3, 'menu:list', 1, 1),
(20, 6, '菜单查询', 3, 'menu:query', 2, 1),
(21, 6, '菜单新增', 3, 'menu:add', 3, 1),
(22, 6, '菜单编辑', 3, 'menu:edit', 4, 1),
(23, 6, '菜单删除', 3, 'menu:delete', 5, 1);

-- 操作日志按钮
INSERT IGNORE INTO menus (id, parent_id, name, type, permission, sort, status) VALUES
(24, 7, '日志列表', 3, 'log:list', 1, 1),
(25, 7, '日志导出', 3, 'log:export', 2, 1);

-- 登录日志按钮
INSERT IGNORE INTO menus (id, parent_id, name, type, permission, sort, status) VALUES
(26, 8, '日志列表', 3, 'login-log:list', 1, 1);

-- 数据字典按钮
INSERT IGNORE INTO menus (id, parent_id, name, type, permission, sort, status) VALUES
(27, 3, '字典列表', 3, 'dict:list', 1, 1),
(28, 3, '字典查询', 3, 'dict:query', 2, 1),
(29, 3, '字典新增', 3, 'dict:add', 3, 1),
(30, 3, '字典编辑', 3, 'dict:edit', 4, 1),
(31, 3, '字典删除', 3, 'dict:delete', 5, 1);
