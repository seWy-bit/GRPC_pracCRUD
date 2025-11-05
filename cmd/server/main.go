package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"prac_service/internal/config"
	"prac_service/internal/interceptor"
	"prac_service/internal/logger"
	"prac_service/internal/service"
	"prac_service/pkg/api/test"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func startHTTPServer(ctx context.Context, cfg *config.Config) *http.Server {
	mux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := test.RegisterOrderServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", cfg.GRPCPort), opts)
	if err != nil {
		logger.Log.Errorf("Failed to register HTTP handlers: %v", err)
		return nil
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	go func() {
		logger.Log.Infof("HTTP gateway starting on port %d", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Errorf("HTTP server error: %v", err)
		}
	}()

	return httpServer
}
func startGRPCServer(cfg *config.Config) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.LoggingInterceptor()),
	)

	orderService := service.NewOrderService()
	test.RegisterOrderServiceServer(grpcServer, orderService)

	reflection.Register(grpcServer)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.GRPCPort)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			logger.Log.Errorf("failed to listen on gRPC port: %v", err)

			return
		}

		logger.Log.Infof("gRPC server starting on port %d", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Errorf("gRPC server error: %v", err)
		}
	}()

	return grpcServer
}

func waitForShutdown(grpcServer *grpc.Server, httpServer *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Log.Info("Shutdown signal received")

	if grpcServer != nil {
		logger.Log.Info("Gracefully stopping gRPC server...")
		grpcServer.GracefulStop()
	}

	if httpServer != nil {
		logger.Log.Info("Gracefully stopping HTTP server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Log.Errorf("HTTP shutdown error: %v", err)
		}
	}

	logger.Log.Info("Server Stopped")
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := logger.Init(cfg.LogLevel); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer := startGRPCServer(cfg)

	httpServer := startHTTPServer(ctx, cfg)

	waitForShutdown(grpcServer, httpServer)
}
