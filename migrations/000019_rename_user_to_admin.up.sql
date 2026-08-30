-- 将「用户管理」菜单及子按钮改名为「管理员管理」，与 C 端"用户"概念区分
UPDATE menus SET name = '管理员管理' WHERE id = 4;
UPDATE menus SET name = '管理员列表' WHERE id = 9;
UPDATE menus SET name = '管理员查询' WHERE id = 10;
UPDATE menus SET name = '新增管理员' WHERE id = 11;
UPDATE menus SET name = '编辑管理员' WHERE id = 12;
UPDATE menus SET name = '删除管理员' WHERE id = 13;
