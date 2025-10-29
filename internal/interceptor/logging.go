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

		if err != nil {
			st, _ := status.FromError(err)
			logger.Log.Errorw("gRPC request failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err.Error(),
				"error_code", st.Code().String(),
			)
		} else {
			logger.Log.Infow("gRPC request completed",
				"method", info.FullMethod,
				"duration", duration,
				"response", resp,
			)
		}

		return resp, err
	}
}
