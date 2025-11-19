package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"prac_service/internal/logger"
	"prac_service/pkg/api/test"
)

type OrderService struct {
	test.UnimplementedOrderServiceServer
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewOrderService(db *pgxpool.Pool, redis *redis.Client) *OrderService {
	return &OrderService{
		db:    db,
		redis: redis,
	}
}

const (
	orderCacheKey      = "order:%s"
	ordersListCacheKey = "orders:list"
	cacheTTL           = 5 * time.Minute
)

func (s *OrderService) CreateOrder(ctx context.Context, req *test.CreateOrderRequest) (*test.CreateOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Item == "" {
		return nil, status.Error(codes.InvalidArgument, "название товара не может быть пустым")
	}

	if req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "количество товара должно быть положительным числом")
	}

	id := uuid.New().String()

	query := `INSERT INTO orders (id, item, quantity) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(ctx, query, id, req.Item, req.Quantity)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create order: %v", err))
	}

	go s.invalidateOrdersListCache(context.Background())

	return &test.CreateOrderResponse{Id: id}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *test.GetOrderRequest) (*test.GetOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id заказа не может быть пустым")
	}

	cacheKey := fmt.Sprintf(orderCacheKey, req.Id)
	cachedOrder, err := s.getOrderFromCache(ctx, cacheKey)
	if err == nil && cachedOrder != nil {
		logger.Log.Debugw("Order retrieved from cache", "order_id", req.Id)
		return &test.GetOrderResponse{Order: cachedOrder}, nil
	}

	var order test.Order
	query := `SELECT id, item, quantity FROM orders where id = $1`
	err = s.db.QueryRow(ctx, query, req.Id).Scan(&order.Id, &order.Item, &order.Quantity)
	if err != nil {
		return nil, status.Error(codes.NotFound, "заказ не найден")
	}

	go s.cacheOrder(context.Background(), cacheKey, &order)

	return &test.GetOrderResponse{Order: &order}, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, req *test.UpdateOrderRequest) (*test.UpdateOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "ID заказа не может быть пустым")
	}

	if req.Item == "" {
		return nil, status.Error(codes.InvalidArgument, "название товара не может быть пустым")
	}

	if req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "количество товара должно быть положительным числом")
	}

	query := `UPDATE orders SET item = $1, quantity = $2 WHERE id = $3`
	result, err := s.db.Exec(ctx, query, req.Item, req.Quantity, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update order: %v", err))
	}

	if result.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "заказ не найден")
	}

	var order test.Order
	getQuery := `SELECT id, item, quantity FROM orders WHERE id = $1`
	err = s.db.QueryRow(ctx, getQuery, req.Id).Scan(&order.Id, &order.Item, &order.Quantity)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get updated order: %v", err))
	}

	go s.invalidateOrderCache(context.Background(), req.Id)
	go s.invalidateOrdersListCache(context.Background())

	return &test.UpdateOrderResponse{Order: &order}, nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, req *test.DeleteOrderRequest) (*test.DeleteOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "ID заказа не может быть пустым")
	}

	query := `DELETE FROM orders WHERE id = $1`
	result, err := s.db.Exec(ctx, query, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete order: %v", err))
	}

	if result.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "заказ не найден")
	}

	go s.invalidateOrderCache(context.Background(), req.Id)
	go s.invalidateOrdersListCache(context.Background())

	return &test.DeleteOrderResponse{Success: true}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *test.ListOrdersRequest) (*test.ListOrdersResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cachedOrders, err := s.getOrdersListFromCache(ctx)
	if err == nil && cachedOrders != nil {
		logger.Log.Debugw("Orders list retrieved from cache")
		return cachedOrders, nil
	}

	query := `SELECT id, item, quantity FROM orders ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list orders: %v", err))
	}
	defer rows.Close()

	var orders []*test.Order
	for rows.Next() {
		var order test.Order
		if err := rows.Scan(&order.Id, &order.Item, &order.Quantity); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to scan order: %v", err))
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("error iterating orders: %v", err))
	}

	response := &test.ListOrdersResponse{Orders: orders}

	go s.cacheOrdersList(context.Background(), response)

	return response, nil
}

func (s *OrderService) getOrderFromCache(ctx context.Context, cacheKey string) (*test.Order, error) {
	data, err := s.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		logger.Log.Warnw("Failed to get order from cache", "key", cacheKey, "error", err)
		return nil, err
	}

	var order test.Order
	if err := json.Unmarshal([]byte(data), &order); err != nil {
		logger.Log.Warnw("Failed to unmarshal order from cache", "key", cacheKey, "error", err)
		return nil, err
	}

	return &order, nil
}

func (s *OrderService) cacheOrder(ctx context.Context, cacheKey string, order *test.Order) {
	data, err := json.Marshal(order)
	if err != nil {
		logger.Log.Warnw("Failed to marshal order for cache", "order_id", order.Id, "error", err)
		return
	}

	if err := s.redis.Set(ctx, cacheKey, data, cacheTTL).Err(); err != nil {
		logger.Log.Warnw("Failed to cache order", "order_id", order.Id, "error", err)
	}
}

func (s *OrderService) invalidateOrderCache(ctx context.Context, orderID string) {
	cacheKey := fmt.Sprintf(orderCacheKey, orderID)
	if err := s.redis.Del(ctx, cacheKey).Err(); err != nil {
		logger.Log.Warnw("Failed to invalidate order cache", "order_id", orderID, "error", err)
	} else {
		logger.Log.Debugw("Order cache invalidated", "order_id", orderID)
	}
}

func (s *OrderService) getOrdersListFromCache(ctx context.Context) (*test.ListOrdersResponse, error) {
	data, err := s.redis.Get(ctx, ordersListCacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		logger.Log.Warnw("Failed to get orders list from cache", "error", err)
		return nil, err
	}

	var response test.ListOrdersResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		logger.Log.Warnw("Failed to unmarshal orders list from cache", "error", err)
		return nil, err
	}

	return &response, nil
}

func (s *OrderService) cacheOrdersList(ctx context.Context, response *test.ListOrdersResponse) {
	data, err := json.Marshal(response)
	if err != nil {
		logger.Log.Warnw("Failed to marshal orders list for cache", "error", err)
		return
	}

	if err := s.redis.Set(ctx, ordersListCacheKey, data, cacheTTL).Err(); err != nil {
		logger.Log.Warnw("Failed to cache orders list", "error", err)
	}
}

func (s *OrderService) invalidateOrdersListCache(ctx context.Context) {
	if err := s.redis.Del(ctx, ordersListCacheKey).Err(); err != nil {
		logger.Log.Warnw("Failed to invalidate orders list cache", "error", err)
	} else {
		logger.Log.Debugw("Orders list cache invalidated")
	}
}
