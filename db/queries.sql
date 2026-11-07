-- ========================================================
-- 1. AUTH & RBAC (The Most Important Query for Permissions)
-- ========================================================
-- name: ListUsers :many
SELECT
    id,
    tenant_id,
    role_id,
    name,
    email
FROM
    user
LIMIT
    ?;

-- name: GetUserByEmail :one
SELECT
    id,
    tenant_id,
    role_id,
    name,
    email,
    password_hash
FROM
    user
WHERE
    email = ?
LIMIT
    1;

-- name: GetUserById :one
SELECT
    id,
    tenant_id,
    role_id,
    name,
    email
FROM
    user
WHERE
    id = ?
LIMIT
    1;

-- name: CreateUser :one
INSERT INTO
    user (
        tenant_id,
        role_id,
        name,
        email,
        password_hash
    )
VALUES
    (?, ?, ?, ?, ?)
RETURNING
    id,
    tenant_id,
    role_id,
    name,
    email;

-- name: UpdateUser :one
UPDATE
    user
SET
    name = ?,
    email = ?,
    role_id = ?
WHERE
    id = ?
RETURNING
    id,
    tenant_id,
    role_id,
    name,
    email;

-- name: UpdateUserPassword :exec
UPDATE
    user
SET
    password_hash = ?
WHERE
    id = ?;

-- name: DeleteUser :exec
DELETE FROM
    user
WHERE
    id = ?;

-- name: GetUserPermissions :many
-- Resolves: User -> Role -> Permission Codes in ONE query!
SELECT
    p.code
FROM
    user u
    JOIN role_permission rp ON u.role_id = rp.role_id
    JOIN permission p ON rp.permission_id = p.id
WHERE
    u.id = ?;

-- name: GetRoleByID :one
SELECT
    id,
    name,
    created_by
FROM
    role
WHERE
    id = ?;

-- ========================================================
-- 2. SHOPPING LISTS
-- ========================================================
-- name: ListShoppingListsByTenant :many
SELECT
    id,
    tenant_id,
    name,
    created_by
FROM
    shopping_list
WHERE
    tenant_id = ?
ORDER BY
    id DESC;

-- name: GetShoppingListByID :one
SELECT
    id,
    tenant_id,
    name,
    created_by
FROM
    shopping_list
WHERE
    id = ?
    AND tenant_id = ?;

-- name: CreateShoppingList :one
INSERT INTO
    shopping_list (tenant_id, name, created_by)
VALUES
    (?, ?, ?)
RETURNING
    id,
    tenant_id,
    name,
    created_by;

-- name: DeleteShoppingList :exec
DELETE FROM
    shopping_list
WHERE
    id = ?
    AND tenant_id = ?;

-- ========================================================
-- 3. ITEMS
-- ========================================================
-- name: ListItemsByListID :many
SELECT
    id,
    shopping_list_id,
    name,
    amount_to_buy,
    category,
    created_by,
    ticked,
    prefferd_supermarket_id
FROM
    shopping_list_item
WHERE
    shopping_list_id = ?
ORDER BY
    id ASC;

-- name: CreateItem :one
INSERT INTO
    shopping_list_item (
        shopping_list_id,
        name,
        amount_to_buy,
        created_by,
        ticked,
        category,
        prefferd_supermarket_id
    )
VALUES
    (?, ?, ?, ?, false, ?, ?)
RETURNING
    id,
    shopping_list_id,
    name,
    amount_to_buy,
    created_by,
    ticked,
    category,
    prefferd_supermarket_id;

-- name: ToggleItem :exec
UPDATE
    shopping_list_item
SET
    ticked = ?
WHERE
    id = ?;

-- name: DeleteItem :exec
DELETE FROM
    shopping_list_item
WHERE
    id = ?;

-- ========================================================
-- 4. SUPERMARKETS
-- ========================================================
-- name: ListSupermarkets :many
SELECT
    id,
    name,
    latitude,
    longitude
FROM
    supermarket
ORDER BY
    name ASC;

-- name: GetSupermarketByID :one
SELECT
    id,
    name,
    latitude,
    longitude
FROM
    supermarket
WHERE
    id = ?
LIMIT
    1;

-- name: CreateSupermarket :one
INSERT INTO
    supermarket (name, latitude, longitude)
VALUES
    (?, ?, ?)
RETURNING
    id,
    name,
    latitude,
    longitude;

-- name: UpdateSupermarket :one
UPDATE
    supermarket
SET
    name = ?,
    latitude = ?,
    longitude = ?
WHERE
    id = ?
RETURNING
    id,
    name,
    latitude,
    longitude;

-- name: DeleteSupermarket :exec
DELETE FROM
    supermarket
WHERE
    id = ?;
