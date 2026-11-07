-- ========================================================
-- 1. AUTH & RBAC (The Most Important Query for Permissions)
-- ========================================================
-- name: ListUsers :many
SELECT
    *
FROM
    user_public
WHERE
    tenant_id = ?
ORDER BY
    id ASC
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
    AND tenant_id = ?
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
    name = coalesce(?, name),
    email = coalesce(?, email),
    role_id = coalesce(?, role_id)
WHERE
    id = ?
    AND tenant_id = ?
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
    id = ?
    AND tenant_id = ?;

-- name: DeleteUser :exec
DELETE FROM
    user
WHERE
    id = ?
    AND tenant_id = ?;

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
    tenant_id,
    created_by
FROM
    role
WHERE
    id = ?
    AND tenant_id = ?;

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
    sli.id,
    sli.shopping_list_id,
    sli.name,
    sli.amount_to_buy,
    sli.category,
    sli.created_by,
    sli.ticked,
    sli.prefferd_supermarket_id
FROM
    shopping_list_item sli
    JOIN shopping_list sl ON sli.shopping_list_id = sl.id
WHERE
    sli.shopping_list_id = ?
    AND sl.tenant_id = ?
ORDER BY
    sli.id ASC;

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
    (
        (
            SELECT
                sl.id
            FROM
                shopping_list sl
            WHERE
                sl.id = ?
                AND sl.tenant_id = ?
        ),
        ?,
        ?,
        ?,
        false,
        ?,
        ?
    )
RETURNING
    *;

-- name: ToggleItem :exec
UPDATE
    shopping_list_item
SET
    ticked = ?
WHERE
    shopping_list_item.id = ?
    AND EXISTS (
        SELECT
            1
        FROM
            shopping_list sl
        WHERE
            sl.id = shopping_list_item.shopping_list_id
            AND sl.tenant_id = ?
    );

-- name: UpdateListItem :one
UPDATE
    shopping_list_item
SET
    name = coalesce(?, name),
    amount_to_buy = coalesce(?, amount_to_buy),
    category = coalesce(?, category),
    prefferd_supermarket_id = coalesce(?, prefferd_supermarket_id)
WHERE
    shopping_list_item.id = ?
    AND EXISTS (
        SELECT
            1
        FROM
            shopping_list sl
        WHERE
            sl.id = shopping_list_item.shopping_list_id
            AND sl.tenant_id = ?
    )
RETURNING
    *;

-- name: DeleteItem :exec
DELETE FROM
    shopping_list_item
WHERE
    shopping_list_item.id = ?
    AND EXISTS (
        SELECT
            1
        FROM
            shopping_list sl
        WHERE
            sl.id = shopping_list_item.shopping_list_id
            AND sl.tenant_id = ?
    );

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
