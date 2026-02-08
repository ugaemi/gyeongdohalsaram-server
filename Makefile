.PHONY: build run test lint clean

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
	rm -rf $(BUILD_DIR)

.DEFAULT_GOAL := build
