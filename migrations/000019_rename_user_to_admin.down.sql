-- 回滚：恢复「用户管理」菜单及子按钮原名
UPDATE menus SET name = '用户管理' WHERE name = '管理员管理' AND path = '/system/user';
UPDATE menus SET name = '用户列表' WHERE name = '管理员列表' AND permission = 'user:list';
UPDATE menus SET name = '用户查询' WHERE name = '管理员查询' AND permission = 'user:query';
UPDATE menus SET name = '用户新增' WHERE name = '新增管理员' AND permission = 'user:add';
UPDATE menus SET name = '用户编辑' WHERE name = '编辑管理员' AND permission = 'user:edit';
UPDATE menus SET name = '用户删除' WHERE name = '删除管理员' AND permission = 'user:delete';
