MIGRATE_CMD = go run ./cmd/migrate

.PHONY: migrate-up migrate-down migrate-steps migrate-version migrate-force migrate-create

## Run all pending migrations
migrate-up:
	$(MIGRATE_CMD) up

## Roll back all migrations
migrate-down:
	$(MIGRATE_CMD) down

## Roll back last migration: make migrate-steps n=-1
migrate-steps:
	$(MIGRATE_CMD) steps $(n)

## Print current migration version
migrate-version:
	$(MIGRATE_CMD) version

## Force-set version (use to fix dirty state): make migrate-force v=3
migrate-force:
	$(MIGRATE_CMD) force $(v)

## Create a new migration: make migrate-create name=add_posts_table
migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=<migration_name>"; exit 1; fi
	$(eval NEXT := $(shell ls migrations/sql | grep "\.up\.sql" | wc -l | xargs -I{} expr {} + 1))
	$(eval PADDED := $(shell printf "%06d" $(NEXT)))
	touch migrations/sql/$(PADDED)_$(name).up.sql
	touch migrations/sql/$(PADDED)_$(name).down.sql
	@echo "created migrations/sql/$(PADDED)_$(name).up.sql"
	@echo "created migrations/sql/$(PADDED)_$(name).down.sql"