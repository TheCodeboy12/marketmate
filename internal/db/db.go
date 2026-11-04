package db

import (
	"database/sql"
	_ "embed"

	db_internal "rbac-testing/db"

	_ "modernc.org/sqlite" // <--- registers "sqlite"
)

func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(db_internal.SchemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(db_internal.SeedSQL); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
