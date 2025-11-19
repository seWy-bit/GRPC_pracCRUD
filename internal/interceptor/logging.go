package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"prac_service/internal/logger"
)

func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		logger.Log.Infow("gRPC request started",
			"method", info.FullMethod,
			"request", req,
			"timestamp", startTime.Format("15:04:05"),
		)

		resp, err := handler(ctx, req)
		duration := time.Since(startTime)

		logFields := []interface{}{
			"method", info.FullMethod,
			"duration", duration,
		}

		if err != nil {
			st, _ := status.FromError(err)
			logFields = append(logFields, "error", err.Error(), "error_code", st.Code().String())
			logger.Log.Errorw("gRPC request failed", logFields...)
		} else {
			if info.FullMethod == "/api.OrderService/GetOrder" || info.FullMethod == "/api.OrderService/ListOrders" {
				logFields = append(logFields, "cached", "possible")
			}
			logger.Log.Infow("gRPC request completed", logFields...)
		}

		return resp, err
	}
}
