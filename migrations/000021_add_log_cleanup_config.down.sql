DELETE FROM menus WHERE path = '/system/log-setting';
DELETE FROM system_settings WHERE setting_key IN ('log_cleanup_days', 'log_cleanup_scope');
