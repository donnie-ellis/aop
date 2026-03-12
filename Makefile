.PHONY: build-api build-controller build-agent build-all test test-api test-controller test-agent lint clean

# Build outputs
BIN_DIR := bin

# Build all binaries
build-all: build-api build-controller build-agent

build-api:
	@echo "Building API server..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/api ./api/cmd/api

build-controller:
	@echo "Building controller..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/controller ./controller/cmd/controller

build-agent:
	@echo "Building agent..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/agent ./agent/cmd/agent

# Test
test:
	go test github.com/donnie-ellis/aop/pkg/... \
		github.com/donnie-ellis/aop/api/... \
		github.com/donnie-ellis/aop/controller/... \
		github.com/donnie-ellis/aop/agent/...

test-api:
	go test ./api/...

test-controller:
	go test ./controller/...

test-agent:
	go test ./agent/...

# Lint
lint:
	golangci-lint run ./api/... ./controller/... ./agent/... ./pkg/...

# Migrations
migrate-up:
	migrate -path ./migrations -database "$$AOP_DB_URL" up

migrate-down:
	migrate -path ./migrations -database "$$AOP_DB_URL" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $$name

# Clean
clean:
	@rm -rf $(BIN_DIR)
	@echo "Cleaned."
