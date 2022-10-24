include .env

.PHONY: migrate-up migrate-down test locs

migrate-up:
	migrate -path migrations -database ${POSTGRES_URL} --verbose up

migrate-down:
	migrate -path migrations -database ${POSTGRES_URL} --verbose down

test:
	go test -cover

locs:
	docker run --rm -v "${PWD}":/workdir hhatto/gocloc . --exclude-ext=xml --by-file

