APP_NAME ?= app
BIN_DIR ?= bin
IMAGE_NAME ?= grinex-app:latest

.PHONY: build test docker-build run lint proto

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/app

test:
	go test ./...

docker-build:
	docker build -t $(IMAGE_NAME) .

run:
	go run ./cmd/app

lint:
	golangci-lint run ./...

proto:
	PATH="$(shell go env GOPATH)/bin:$$PATH" protoc -I . \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/rates/v1/rates.proto
