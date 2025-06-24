include .env
export

.PHONY: db-up db-down migrate-up migrate-down migrate-create migrate-force migrate-version

db-up:
	cd go && docker-compose up -d postgres

db-down:
	cd go && docker-compose down

.PHONY: migrate-up migrate-down migrate-create migrate-force migrate-version

migrate-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is not set. Please set it in .env file"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is not set. Please set it in .env file"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" down

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=migration_name"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(name)

migrate-force:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is not set. Please set it in .env file"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" force $(version)

migrate-version:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is not set. Please set it in .env file"; exit 1; fi
	migrate -path migrations -database "$(DATABASE_URL)" version