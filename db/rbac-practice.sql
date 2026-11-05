-- 1. Tenant Table 
CREATE TABLE IF NOT EXISTS "tenant" ( "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"name" VARCHAR NOT NULL
);

-- 2. Permission Table
CREATE TABLE IF NOT EXISTS "permission" (
	"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"code" VARCHAR NOT NULL UNIQUE,
	"description" VARCHAR
);

-- 3. Role Table (created_by is NULL for system-seeded roles)
CREATE TABLE IF NOT EXISTS "role" (
	"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"name" VARCHAR NOT NULL UNIQUE,
	"created_by" INTEGER,
	FOREIGN KEY ("created_by") REFERENCES "user"("id")
	ON UPDATE NO ACTION ON DELETE SET NULL
);

-- 4. Role-Permission Junction Table
CREATE TABLE IF NOT EXISTS "role_permission" (
	"role_id" INTEGER NOT NULL,
	"permission_id" INTEGER NOT NULL,
	PRIMARY KEY("role_id", "permission_id"),
	FOREIGN KEY ("role_id") REFERENCES "role"("id")
	ON UPDATE NO ACTION ON DELETE CASCADE,
	FOREIGN KEY ("permission_id") REFERENCES "permission"("id")
	ON UPDATE NO ACTION ON DELETE CASCADE
);

-- 5. User Table
CREATE TABLE IF NOT EXISTS "user" (
	"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"tenant_id" INTEGER NOT NULL,
	"role_id" INTEGER NOT NULL,
	"name" VARCHAR NOT NULL,
	"email" VARCHAR NOT NULL UNIQUE,
	"password_hash" TEXT,
	FOREIGN KEY ("tenant_id") REFERENCES "tenant"("id")
	ON UPDATE NO ACTION ON DELETE CASCADE,
	FOREIGN KEY ("role_id") REFERENCES "role"("id")
	ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_tenant_email"
ON "user" ("tenant_id", "email");

-- 6. Shopping List Table
CREATE TABLE IF NOT EXISTS "shopping_list" (
	"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"tenant_id" INTEGER NOT NULL,
	"name" VARCHAR NOT NULL,
	"created_by" INTEGER NOT NULL,
	FOREIGN KEY ("created_by") REFERENCES "user"("id")
	ON UPDATE NO ACTION ON DELETE CASCADE,
	FOREIGN KEY ("tenant_id") REFERENCES "tenant"("id")
	ON UPDATE NO ACTION ON DELETE CASCADE
);

-- 7. Shopping List Item Table
CREATE TABLE IF NOT EXISTS "shopping_list_item" (
	"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	"shopping_list_id" INTEGER NOT NULL,
	"name" VARCHAR NOT NULL,
	"amount_to_buy" INTEGER NOT NULL,
	"created_by" INTEGER NOT NULL,
	"ticked" BOOLEAN NOT NULL DEFAULT false,
	FOREIGN KEY ("shopping_list_id") REFERENCES "shopping_list"("id")
	ON UPDATE NO ACTION ON DELETE CASCADE,
	FOREIGN KEY ("created_by") REFERENCES "user"("id")
	ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_item_list_name"
ON "shopping_list_item" ("shopping_list_id", "name");
