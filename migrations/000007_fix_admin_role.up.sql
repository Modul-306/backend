-- Force fix for admin user role
UPDATE users SET role = 'platform_admin' WHERE email = 'admin@cattlehof.ch';
