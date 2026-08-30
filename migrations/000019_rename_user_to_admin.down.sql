-- 回滚：恢复「用户管理」菜单及子按钮原名
UPDATE menus SET name = '用户管理' WHERE id = 4;
UPDATE menus SET name = '用户列表' WHERE id = 9;
UPDATE menus SET name = '用户查询' WHERE id = 10;
UPDATE menus SET name = '用户新增' WHERE id = 11;
UPDATE menus SET name = '用户编辑' WHERE id = 12;
UPDATE menus SET name = '用户删除' WHERE id = 13;
