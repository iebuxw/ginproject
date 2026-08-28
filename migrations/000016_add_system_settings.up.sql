-- 系统配置表（key-value 模式）
CREATE TABLE IF NOT EXISTS `system_settings` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `setting_key` VARCHAR(64) NOT NULL,
  `setting_value` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_setting_key` (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置';

-- 默认配置
INSERT IGNORE INTO system_settings (setting_key, setting_value) VALUES ('site_name', '后台管理系统');

-- 二级菜单：系统配置（挂在系统管理 parent_id=1 下）
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (61, 1, '系统配置', '/system/setting', 'setting:list', 2, 'el-icon-s-tools', 4, NOW(), NOW());

-- 按钮权限点
INSERT IGNORE INTO menus (id, parent_id, name, path, permission, type, icon, sort, created_at, updated_at)
VALUES (62, 61, '读取', '', 'setting:read', 3, '', 1, NOW(), NOW()),
       (63, 61, '保存', '', 'setting:save', 3, '', 2, NOW(), NOW());

-- admin 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id)
VALUES (1, 61), (1, 62), (1, 63);
