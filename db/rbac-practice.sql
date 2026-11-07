-- 1. Tenant Table
CREATE TABLE IF NOT EXISTS "tenant" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "name" VARCHAR NOT NULL,
    "domain" VARCHAR NOT NULL UNIQUE
);

-- 2. Permission Table
CREATE TABLE IF NOT EXISTS "permission" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "code" VARCHAR NOT NULL UNIQUE,  --example: users:read
    "description" VARCHAR
);

-- 3. Role Table (created_by is NULL for system-seeded roles)
CREATE TABLE IF NOT EXISTS "role" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "name" VARCHAR NOT NULL UNIQUE,
    "tenant_id" INTEGER NOT NULL,
    "created_by" INTEGER,
    FOREIGN KEY ("created_by") REFERENCES "user"("id") ON UPDATE NO ACTION ON DELETE
    SET
        NULL,
        FOREIGN KEY(tenant_id) REFERENCES tenant(id)
);

-- 4. Role-Permission Junction Table
CREATE TABLE IF NOT EXISTS "role_permission" (
    "role_id" INTEGER NOT NULL,
    "permission_id" INTEGER NOT NULL,
    PRIMARY KEY ("role_id", "permission_id"),
    FOREIGN KEY ("role_id") REFERENCES "role"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("permission_id") REFERENCES "permission"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- 5. User Table
CREATE TABLE IF NOT EXISTS "user" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "tenant_id" INTEGER NOT NULL,
    "role_id" INTEGER NOT NULL,
    "name" VARCHAR NOT NULL,
    "email" VARCHAR NOT NULL UNIQUE,
    "password_hash" TEXT,
    FOREIGN KEY ("tenant_id") REFERENCES "tenant"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("role_id") REFERENCES "role"("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE VIEW user_public AS
SELECT
    id,
    tenant_id,
    role_id,
    name,
    email
FROM
    user;

CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_tenant_email" ON "user" ("tenant_id", "email");

-- 6. Shopping List Table
CREATE TABLE IF NOT EXISTS "shopping_list" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "tenant_id" INTEGER NOT NULL,
    "name" VARCHAR NOT NULL,
    "created_by" INTEGER NOT NULL,
    FOREIGN KEY ("created_by") REFERENCES "user"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("tenant_id") REFERENCES "tenant"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- 7. Supermarket Table
CREATE TABLE IF NOT EXISTS "supermarket" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "name" VARCHAR NOT NULL,
    "latitude" REAL NOT NULL CHECK (latitude BETWEEN -90.0 AND 90.0),
    "longitude" REAL NOT NULL CHECK (longitude BETWEEN -180.0 AND 180.0)
);

-- 8. Shopping List Item Table
CREATE TABLE IF NOT EXISTS "shopping_list_item" (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "shopping_list_id" INTEGER NOT NULL,
    "name" VARCHAR NOT NULL,
    "amount_to_buy" INTEGER NOT NULL DEFAULT 0 CHECK (amount_to_buy >= 0),
    "category" VARCHAR,
    "created_by" INTEGER NOT NULL,
    "ticked" BOOLEAN NOT NULL DEFAULT false,
    "prefferd_supermarket_id" INTEGER,
    FOREIGN KEY ("shopping_list_id") REFERENCES "shopping_list"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("created_by") REFERENCES "user"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("prefferd_supermarket_id") REFERENCES "supermarket"("id") ON UPDATE NO ACTION ON DELETE
    SET
        NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_item_list_name" ON "shopping_list_item" ("shopping_list_id", "name");
