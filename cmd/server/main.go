package main

import (
	"log"
	"net"
	"prac_service/internal/service"
	"prac_service/pkg/api/test"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

	grpcServer := grpc.NewServer()

	orderService := service.NewOrderService()
	test.RegisterOrderServiceServer(grpcServer, orderService)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}
}
