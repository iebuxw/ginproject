-- admin 用户（密码: 123456）
INSERT IGNORE INTO users (id, username, password, status) VALUES
(1, 'admin', '$2a$10$sdQRCxdtXiEVCrQYCr1ii.6KBCLsQySWH.TKAjeqCHttz5e7b1lHO', 1);

-- 超级管理员角色
INSERT IGNORE INTO roles (id, name, code, description, status) VALUES
(1, '超级管理员', 'admin', '系统超级管理员', 1);

-- 关联角色-菜单（全部31个菜单）
INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES
(1,1),(1,2),(1,3),(1,4),(1,5),(1,6),(1,7),(1,8),(1,9),(1,10),
(1,11),(1,12),(1,13),(1,14),(1,15),(1,16),(1,17),(1,18),(1,19),(1,20),
(1,21),(1,22),(1,23),(1,24),(1,25),(1,26),(1,27),(1,28),(1,29),(1,30),(1,31);

-- 关联用户-角色
INSERT IGNORE INTO user_roles (user_id, role_id) VALUES (1, 1);

-- 字典类型
INSERT IGNORE INTO dict_types (id, name, code, description, status) VALUES
(1, '用户状态', 'user_status', '用户启用/禁用状态', 1),
(2, '性别', 'gender', '用户性别', 1),
(3, '操作日志方法', 'http_method', 'HTTP请求方法', 1),
(4, '系统通知', 'sys_notice', '系统通知类型', 1);

-- 字典数据
INSERT IGNORE INTO dict_data (id, dict_type_id, label, value, sort, status, remark) VALUES
(1, 1, '启用', '1', 1, 1, '用户已启用'),
(2, 1, '禁用', '0', 2, 1, '用户已禁用'),
(3, 2, '男', '1', 1, 1, ''),
(4, 2, '女', '2', 2, 1, ''),
(5, 2, '未知', '0', 3, 1, ''),
(6, 3, 'GET', 'GET', 1, 1, ''),
(7, 3, 'POST', 'POST', 2, 1, ''),
(8, 3, 'PUT', 'PUT', 3, 1, ''),
(9, 3, 'DELETE', 'DELETE', 4, 1, ''),
(10, 4, '系统通知', '1', 1, 1, ''),
(11, 4, '用户通知', '2', 2, 1, '');
