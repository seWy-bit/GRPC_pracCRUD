.PHONY: generate build run clean

# Генерация gRPC кода
generate:
	protoc --go_out=. --go-grpc_out=. proto/order.proto

# Сборка бинарника
build:
	go build -o bin/server ./cmd/server

# Запуск сервера
run: build
	./bin/server

# Запуск без сборки (для разработки)
dev:
	go run ./cmd/server

# Очистка
clean:
	rm -rf bin/