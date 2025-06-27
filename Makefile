include .env
export

.PHONY: db-up db-down db-wipe migrate-up migrate-down migrate-create migrate-force migrate-version migrate-undo-version

db-up:
	cd go && docker-compose up -d postgres

db-down:
	cd go && docker-compose down

# completely wipe the Postgres container + its volumes
db-wipe:
	cd go && docker-compose down -v

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

migrate-undo-version:
	@if [ -z "$(DATABASE_URL)" ]; then \
	  echo "DATABASE_URL is not set. Please export it or add it to your .env file."; \
	  exit 1; \
	fi
	@if [ -z "$(VERSION)" ]; then \
	  echo "VERSION is not set. Usage: make migrate-undo-version VERSION=<version>"; \
	  exit 1; \
	fi
	# strip leading zeros for arithmetic
	@NUM=$$(echo $(VERSION) | sed 's/^0*//'); \
	TARG=$$(expr $$NUM - 1); \
	if [ $$TARG -lt 0 ]; then \
	  echo "Cannot undo version $(VERSION) (no prior version)"; exit 1; \
	fi; \
	echo "Reverting migration $(VERSION), migrating down to version $$TARG..."; \
	migrate -path migrations -database "$(DATABASE_URL)" goto $$TARG


.PHONY: grpcui
# Start gRPC UI
grpcui:
	@echo "Starting grpcui..."
	grpcui -plaintext localhost:8080