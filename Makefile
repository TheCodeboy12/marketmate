.PHONY: sqlc generate

sqlc:
	docker run --rm -v $(CURDIR):/src -w /src sqlc/sqlc generate

generate: sqlc

export JWT_SECRET ?= supersecretpassword
export DB_PATH ?= app.db?_pragma=foreign_keys(1)
.PHONY: devrun devclean
devrun: sqlc
	go run .
devclean:
	rm -f app.db

