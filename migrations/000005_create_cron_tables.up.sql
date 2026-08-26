CREATE TABLE IF NOT EXISTS cron_tasks (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    url VARCHAR(255) NOT NULL,
    method VARCHAR(8) NOT NULL DEFAULT 'GET',
    headers TEXT,
    body TEXT,
    cron VARCHAR(32) NOT NULL,
    timeout INT NOT NULL DEFAULT 30,
    status TINYINT NOT NULL DEFAULT 1,
    remark VARCHAR(255) DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS cron_task_executions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id BIGINT UNSIGNED NOT NULL,
    `trigger` VARCHAR(16) NOT NULL,
    status TINYINT NOT NULL,
    http_status INT,
    response TEXT,
    error_msg VARCHAR(255) DEFAULT '',
    duration_ms INT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_cron_task_executions_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 注：现有库菜单 id 实际已用到 34（历史数据与 000002 种子不同），故从 41 起分配。
-- 一级目录：任务管理
INSERT IGNORE INTO menus (id, parent_id, name, icon, path, type, permission, sort, status) VALUES
(41, 0, '任务管理', 'el-icon-alarm-clock', '/system/task-mgr', 1, '', 4, 1);

-- 二级菜单：定时任务
INSERT IGNORE INTO menus (id, parent_id, name, icon, path, type, permission, sort, status) VALUES
(42, 41, '定时任务', 'el-icon-alarm-clock', '/system/task', 2, '', 1, 1);

-- 定时任务按钮
INSERT IGNORE INTO menus (id, parent_id, name, type, permission, sort, status) VALUES
(43, 42, '任务列表', 3, 'cron:list', 1, 1),
(44, 42, '任务查询', 3, 'cron:query', 2, 1),
(45, 42, '任务新增', 3, 'cron:add', 3, 1),
(46, 42, '任务编辑', 3, 'cron:edit', 4, 1),
(47, 42, '任务删除', 3, 'cron:delete', 5, 1),
(48, 42, '立即执行', 3, 'cron:run', 6, 1),
(49, 42, '执行日志', 3, 'cron:log', 7, 1);

-- admin 角色（role_id=1）绑定新菜单
INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES
(1, 41), (1, 42), (1, 43), (1, 44), (1, 45), (1, 46), (1, 47), (1, 48), (1, 49);
