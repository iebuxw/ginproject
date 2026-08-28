-- 修复历史错位权限点：id=22「菜单编辑」(menu:edit)、id=23「菜单删除」(menu:delete)
-- 当前 parent_id=6（用户管理下的「用户列表」按钮），应挂在「菜单管理」页面下。
-- 「菜单管理」页面下已有同权限点按钮（id=19/20，000002 种子），故本迁移直接删除错位的 22/23 及其角色绑定。
-- （按 name+permission 定位，不硬编码父 id，规避历史数据 id 漂移；22/23 本身按 id 删除为历史遗留数据。）

-- 确认目标页面存在同权限点后才执行删除（子查询包一层派生表，绕开 MySQL 1093 限制）
DELETE FROM role_menus WHERE menu_id IN (22, 23)
  AND EXISTS (SELECT 1 FROM (SELECT 1 FROM menus WHERE type = 2 AND path = '/system/menu') t);
DELETE FROM menus WHERE id IN (22, 23)
  AND EXISTS (SELECT 1 FROM (SELECT 1 FROM menus WHERE type = 2 AND path = '/system/menu') t);
