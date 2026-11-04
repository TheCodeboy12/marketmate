package db

import (
	"database/sql"
	_ "embed"
)

//go:embed ../../db/rbac-practice.sql
var schemaSQL string

//go:embed ../../db/seed.sql
var seedSQL string

func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(seedSQL); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
