DELETE FROM dict_data WHERE dict_type_id BETWEEN 1 AND 4;
DELETE FROM dict_types WHERE id BETWEEN 1 AND 4;
DELETE FROM user_roles WHERE user_id = 1 AND role_id = 1;
DELETE FROM role_menus WHERE role_id = 1;
DELETE FROM roles WHERE id = 1;
DELETE FROM users WHERE id = 1;
