// Package db is for the db
package db

import _ "embed"

//go:embed rbac-practice.sql
var SchemaSQL string

//go:embed seed.sql
var SeedSQL string
