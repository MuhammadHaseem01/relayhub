GO_CACHE ?= /tmp/cargonex-go-cache
GO_MOD_CACHE ?= /tmp/cargonex-go-mod
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/nextroutex?sslmode=disable
PORT ?= 5000
BUILD_PATH ?= ./build
COMPOSE_FILE ?= dev/docker-compose.yml
MIGRATE_COMPOSE_FILE ?= dev/docker-compose.migrate.yml

.PHONY: fmt
fmt:
	gofmt -w main.go src cmd

.PHONY: tidy
tidy:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go mod tidy

.PHONY: test
test:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test ./...

.PHONY: build
build:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go build -o $(BUILD_PATH)/cargonex .

.PHONY: run
run:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) DATABASE_URL="$(DATABASE_URL)" PORT=$(PORT) go run .

.PHONY: run-side-by-side
run-side-by-side:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) DATABASE_URL="$(DATABASE_URL)" PORT=5001 go run .

.PHONY: migrate
migrate:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate

.PHONY: seed
seed:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed

.PHONY: create-org-user
create-org-user:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) DATABASE_URL="$(DATABASE_URL)" go run ./cmd/createorguser

.PHONY: contract
contract:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go run ./cmd/contracttest

.PHONY: docker_up
docker_up:
	docker-compose -f $(COMPOSE_FILE) up --build -d

.PHONY: docker_down
docker_down:
	docker-compose -f $(COMPOSE_FILE) down

.PHONY: migrate_docker
migrate_docker:
	docker-compose -f $(MIGRATE_COMPOSE_FILE) run --rm migrate

.PHONY: clean
clean:
	rm -rf $(BUILD_PATH)
