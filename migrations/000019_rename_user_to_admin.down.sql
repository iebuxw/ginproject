-- 回滚：恢复 000019 改错的记录（实际 id 与迁移种子不同）
UPDATE menus SET name = '菜单管理' WHERE id = 4;
UPDATE menus SET name = '用户编辑' WHERE id = 9;
UPDATE menus SET name = '用户删除' WHERE id = 10;
UPDATE menus SET name = '角色列表' WHERE id = 11;
UPDATE menus SET name = '角色查询' WHERE id = 12;
UPDATE menus SET name = '角色新增' WHERE id = 13;
