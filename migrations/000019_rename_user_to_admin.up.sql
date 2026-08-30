-- 将「用户管理」菜单及子按钮改名为「管理员管理」，与 C 端"用户"概念区分
UPDATE menus SET name = '管理员管理' WHERE name = '用户管理';
UPDATE menus SET name = '管理员列表' WHERE name = '用户列表';
UPDATE menus SET name = '管理员查询' WHERE name = '用户查询';
UPDATE menus SET name = '新增管理员' WHERE name = '用户新增';
UPDATE menus SET name = '编辑管理员' WHERE name = '用户编辑';
UPDATE menus SET name = '删除管理员' WHERE name = '用户删除';
