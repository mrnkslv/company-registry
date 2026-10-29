MIGRATION_DIR       := ./migrations

# Example: make goose-create name=init
goose-create:
	goose -v -dir $(MIGRATION_DIR) create $(name) sql

goose-up:
	goose up -dir $(MIGRATION_DIR)

goose-down:
	goose down-to 0 -dir $(MIGRATION_DIR)

lint:
	golangci-lint run