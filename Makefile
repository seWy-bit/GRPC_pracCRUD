.PHONY: generate build run clean

# Переменные
PROTO_DIR=proto
PROTO_FILE=$(PROTO_DIR)/order.proto
GO_OUT=pkg/api/test
GW_OUT=$(GO_OUT)
THIRD_PARTY=third_party

# Генерация кода для gRPC и gRPC-Gateway
generate:
	protoc -I=$(PROTO_DIR) -I=$(THIRD_PARTY) \
		--go_out=$(GO_OUT) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT) \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(GW_OUT) \
		--grpc-gateway_opt=paths=source_relative \
		--grpc-gateway_opt=logtostderr=true \
		$(PROTO_FILE)

# Сборка бинарника
build:
	go build -o bin/server ./cmd/server

# Запуск сервера
run: build
	./bin/server

# Запуск в режиме разработки (без сборки)
dev:
	go run ./cmd/server

# Очистка
clean:
	rm -rf bin/
	rm -f $(GO_OUT)/*.pb.go
	rm -f $(GO_OUT)/*.gw.go

# Установка необходимых инструментов
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

# Помощь
help:
	@echo "Available targets:"
	@echo "  generate - Generate gRPC and gRPC-Gateway code"
	@echo "  build    - Build binary"
	@echo "  run      - Build and run server"
	@echo "  dev      - Run server directly (for development)"
	@echo "  clean    - Clean generated files and binaries"
	@echo "  tools    - Install required tools"