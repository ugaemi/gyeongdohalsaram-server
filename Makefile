.PHONY: build run test lint clean docs

BINARY_NAME=server
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v -race ./...

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR) docs/asyncapi

docs:
	npx --yes @asyncapi/cli generate fromTemplate asyncapi.yaml @asyncapi/html-template -o docs/asyncapi --force-write

.DEFAULT_GOAL := build
