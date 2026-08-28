-- 菜单归类修正（按 name 定位父目录，不硬编码 id，规避历史数据 id 漂移）：
-- 1. 数据库备份：日志管理 -> 运维管理（备份是运维动作，不是日志）
-- 2. 执行日志：日志管理 -> 任务管理（本质是定时任务执行记录），并删除"定时任务"下重复的 cron:log 按钮（id=49）
--   修复历史隐患：000010/000011 曾用 parent_id=26 挂菜单（种子中 26 是按钮 id），本迁移按 name 重新定位，新装环境同样被修正。

-- 数据库备份移到"运维管理"目录下
UPDATE menus m
JOIN (SELECT id FROM menus WHERE parent_id = 0 AND type = 1 AND name = '运维管理') p
  ON m.parent_id <> p.id
SET m.parent_id = p.id, m.sort = 2
WHERE m.type = 2 AND m.path = '/system/backup';

-- 执行日志移到"任务管理"目录下（排在"定时任务"之后）
UPDATE menus m
JOIN (SELECT id FROM menus WHERE parent_id = 0 AND type = 1 AND name = '任务管理') p
  ON m.parent_id <> p.id
SET m.parent_id = p.id, m.sort = 2
WHERE m.type = 2 AND m.path = '/system/task-logs';

-- 删除"定时任务"下重复的 cron:log 按钮及角色绑定（页面菜单 id=50 已承载同一权限点）
DELETE FROM role_menus WHERE menu_id IN (SELECT id FROM (SELECT id FROM menus WHERE parent_id <> 0 AND type = 3 AND permission = 'cron:log' AND path IS NULL) t);
DELETE FROM menus WHERE parent_id <> 0 AND type = 3 AND permission = 'cron:log' AND path IS NULL;

-- admin 角色绑定保留 id=50 菜单（000010 已写入），无新增绑定
