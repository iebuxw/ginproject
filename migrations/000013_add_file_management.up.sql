-- 文件管理：上传文件元数据表
CREATE TABLE IF NOT EXISTS `files` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `original_name` VARCHAR(255) NOT NULL,
  `stored_name` VARCHAR(128) NOT NULL,
  `size` BIGINT DEFAULT 0,
  `ext` VARCHAR(32) DEFAULT '',
  `uploader_id` BIGINT DEFAULT 0,
  `uploader_name` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='上传文件记录';

-- 一级目录：运维管理（现有菜单 id 最大 55，从 56 起分配）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (56, 0, '运维管理', '/system/ops-mgr', '', 1, 'el-icon-s-tools', 5, NOW(), NOW());

-- 二级菜单：文件管理（file:list 挂在页面上，与 db_backup 模式一致）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (57, 56, '文件管理', '/system/file', 'file:list', 2, 'el-icon-folder', 1, NOW(), NOW());

-- 按钮权限点
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (58, 57, '上传', '', 'file:upload', 3, '', 1, NOW(), NOW()),
       (59, 57, '下载', '', 'file:download', 3, '', 2, NOW(), NOW()),
       (60, 57, '删除', '', 'file:delete', 3, '', 3, NOW(), NOW());

-- admin 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id)
VALUES (1, 56), (1, 57), (1, 58), (1, 59), (1, 60);
