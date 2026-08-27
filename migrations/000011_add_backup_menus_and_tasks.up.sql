-- 建表
CREATE TABLE IF NOT EXISTS `db_backups` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `filename` VARCHAR(255) NOT NULL,
  `file_size` BIGINT DEFAULT 0,
  `trigger_type` VARCHAR(20) DEFAULT '',
  `status` TINYINT DEFAULT 0 COMMENT '0=成功 1=失败',
  `type` VARCHAR(20) DEFAULT 'backup',
  `remark` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据库备份记录';

-- 菜单种子
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (51, 26, '数据库备份', '/system/backup', 'db_backup:list', 2, 'el-icon-download', 4, NOW(), NOW());

INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (52, 51, '新增', '', 'db_backup:add', 3, '', 1, NOW(), NOW()),
       (53, 51, '恢复', '', 'db_backup:restore', 3, '', 2, NOW(), NOW()),
       (54, 51, '删除', '', 'db_backup:delete', 3, '', 3, NOW(), NOW()),
       (55, 51, '下载', '', 'db_backup:download', 3, '', 4, NOW(), NOW());

-- 角色菜单绑定
INSERT IGNORE INTO role_menus (role_id, menu_id)
VALUES (1, 51), (1, 52), (1, 53), (1, 54), (1, 55);

-- 定时任务种子
INSERT INTO cron_tasks (id, name, command, cron, timeout, status, remark)
VALUES
(10, '数据库备份', 'backup_db', '0 0 2 * * *', 300, 1, '每天凌晨2点自动备份数据库'),
(11, '清理过期备份', 'clean_backup', '0 0 4 * * *', 60, 1, '每天凌晨4点清理90天前备份');
