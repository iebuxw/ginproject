-- 修复 000019：id 与迁移种子不同，实际用户管理菜单是 id=2，不是 id=4
-- 1. 回滚 000019 改错的记录
UPDATE menus SET name = '菜单管理' WHERE id = 4;
UPDATE menus SET name = '编辑管理员' WHERE id = 9;
UPDATE menus SET name = '删除管理员' WHERE id = 10;
UPDATE menus SET name = '角色列表' WHERE id = 11;
UPDATE menus SET name = '角色查询' WHERE id = 12;
UPDATE menus SET name = '角色新增' WHERE id = 13;

-- 2. 改正确的记录：用户管理菜单 id=2 及其子按钮 id=6,7,8
UPDATE menus SET name = '管理员管理' WHERE id = 2;
UPDATE menus SET name = '管理员列表' WHERE id = 6;
UPDATE menus SET name = '管理员查询' WHERE id = 7;
UPDATE menus SET name = '新增管理员' WHERE id = 8;
