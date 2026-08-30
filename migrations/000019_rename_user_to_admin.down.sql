-- 回滚：恢复「用户管理」菜单及子按钮原名
UPDATE menus SET name = '用户管理' WHERE name = '管理员管理';
UPDATE menus SET name = '用户列表' WHERE name = '管理员列表';
UPDATE menus SET name = '用户查询' WHERE name = '管理员查询';
UPDATE menus SET name = '用户新增' WHERE name = '新增管理员';
UPDATE menus SET name = '用户编辑' WHERE name = '编辑管理员';
UPDATE menus SET name = '用户删除' WHERE name = '删除管理员';
