-- 1. Default Tenant
INSERT OR IGNORE INTO "tenant" ("id", "name") VALUES (1, 'Main Tenant');

-- 2. Seeded System Roles (created_by is NULL)
INSERT OR IGNORE INTO "role" ("id", "name", "created_by") VALUES 
(1, 'admin', NULL),
(2, 'member', NULL),
(3, 'viewer', NULL);

-- 3. Seeded System User (Created after role 1 and tenant 1 exist)
INSERT OR IGNORE INTO "user" (
  "id",
  "tenant_id",
  "role_id",
  "name",
  "email",
  "password_hash"
) VALUES ( 
  1,
  1,
  1,
  'SYSTEM',
  'system@main_tenant.com',
  NULL
);

-- 4. Permissions
INSERT OR IGNORE INTO "permission" ("id", "code", "description") VALUES
(1, 'lists:read',    'View shopping lists'),
(2, 'lists:create',  'Create new shopping lists'),
(3, 'lists:delete',  'Delete shopping lists'),
(4, 'items:create',  'Add items to a list'),
(5, 'items:check',   'Tick or untick items'),
(6, 'roles:manage',  'Create and manage custom roles');

-- 5. Role -> Permission Mappings
-- Admin gets all permissions (1..6)
INSERT OR IGNORE INTO "role_permission" ("role_id", "permission_id") VALUES
(1, 1), (1, 2), (1, 3), (1, 4), (1, 5), (1, 6);

-- Member can view, create lists, and manage items (cannot delete lists or manage roles)
INSERT OR IGNORE INTO "role_permission" ("role_id", "permission_id") VALUES
(2, 1), (2, 2), (2, 4), (2, 5);

-- Viewer can only view lists (cannot create or edit)
INSERT OR IGNORE INTO "role_permission" ("role_id", "permission_id") VALUES
(3, 1);
