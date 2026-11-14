.PHONY: all build test clean run frontend

BINARY_NAME=driftguard
BIN_DIR=bin

all: frontend build

frontend:
	@echo "Building React Cockpit..."
	cd web && npm install && npm run build

build:
	@echo "Building driftguard daemon..."
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/driftguard

test:
	@echo "Running Go tests..."
	go test -v -race ./...

clean:
	@echo "Cleaning artifacts..."
	rm -rf $(BIN_DIR) web/dist web/node_modules driftguard.db

run: build
	./$(BIN_DIR)/$(BINARY_NAME) serve --config driftguard.yaml
