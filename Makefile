.PHONY: proto

proto:
	PATH="$(shell go env GOPATH)/bin:$$PATH" protoc -I . \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/rates/v1/rates.proto
