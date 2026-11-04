.PHONY: sqlc generate

sqlc:
	docker run --rm -v $(CURDIR):/src -w /src sqlc/sqlc generate

generate: sqlc
