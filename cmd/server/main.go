package main

import (
	"fmt"
	"log"
	"net"

	"prac_service/internal/config"
	"prac_service/internal/interceptor"
	"prac_service/internal/logger"
	"prac_service/internal/service"
	"prac_service/pkg/api/test"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := logger.Init(cfg.LogLevel); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Sync()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.LoggingInterceptor()),
	)

	orderService := service.NewOrderService()
	test.RegisterOrderServiceServer(grpcServer, orderService)

	log.Printf("Server starting on port %d", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
