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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
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

	dbPool, err := connectToDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	redisClient, err := connectToRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	if err := runMigrations(cfg); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer := startGRPCServer(cfg, dbPool, redisClient)

	httpServer := startHTTPServer(ctx, cfg)

	waitForShutdown(grpcServer, httpServer, dbPool, redisClient)
}

func connectToDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.GetDBDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.HealthCheckPeriod = 1 * time.Minute
	poolConfig.MaxConnLifetime = 2 * time.Hour

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Successfully connected to database")
	return dbPool, nil
}

func connectToRedis(cfg *config.Config) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.Log.Info("Successfully connected to Redis")
	return redisClient, nil
}

func runMigrations(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.GetDBDSN())
	if err != nil {
		return fmt.Errorf("failed to parse database config for migrations: %w", err)
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool for migrations: %w", err)
	}
	defer dbPool.Close()

	var tableExists bool
	checkTableQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'orders'
		)`

	err = dbPool.QueryRow(ctx, checkTableQuery).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}

	if !tableExists {
		logger.Log.Info("Creating orders table...")

		createTableQuery := `
			CREATE TABLE orders (
				id UUID PRIMARY KEY,
				item TEXT NOT NULL,
				quantity INTEGER NOT NULL CHECK (quantity > 0),
				created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
			);
			
			CREATE INDEX idx_orders_created_at ON orders(created_at);`

		_, err = dbPool.Exec(ctx, createTableQuery)
		if err != nil {
			return fmt.Errorf("failed to create orders table: %w", err)
		}

		logger.Log.Info("Orders table created successfully")
	} else {
		logger.Log.Info("Orders table already exists")
	}

	return nil
}

func startGRPCServer(cfg *config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.LoggingInterceptor()),
	)

	orderService := service.NewOrderService(dbPool, redisClient)
	test.RegisterOrderServiceServer(grpcServer, orderService)

	reflection.Register(grpcServer)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.GRPCPort)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			logger.Log.Errorf("Failed to listen on gRPC port: %v", err)
			return
		}

		logger.Log.Infof("gRPC server starting on port %d", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Errorf("gRPC server error: %v", err)
		}
	}()

	return grpcServer
}

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

func waitForShutdown(grpcServer *grpc.Server, httpServer *http.Server, dbPool *pgxpool.Pool, redisClient *redis.Client) {
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
			logger.Log.Errorf("HTTP server shutdown error: %v", err)
		}
	}

	if dbPool != nil {
		logger.Log.Info("Closing database connections...")
		dbPool.Close()
	}

	if redisClient != nil {
		logger.Log.Info("Closing Redis connection...")
		redisClient.Close()
	}

	logger.Log.Info("Server Stopped")
}
