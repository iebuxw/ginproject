-- 回滚：恢复 000020 的修改
UPDATE menus SET name = '用户管理' WHERE id = 2;
UPDATE menus SET name = '用户列表' WHERE id = 6;
UPDATE menus SET name = '用户查询' WHERE id = 7;
UPDATE menus SET name = '用户新增' WHERE id = 8;

UPDATE menus SET name = '管理员管理' WHERE id = 4;
UPDATE menus SET name = '管理员列表' WHERE id = 9;
UPDATE menus SET name = '管理员查询' WHERE id = 10;
UPDATE menus SET name = '新增管理员' WHERE id = 11;
UPDATE menus SET name = '编辑管理员' WHERE id = 12;
UPDATE menus SET name = '删除管理员' WHERE id = 13;
