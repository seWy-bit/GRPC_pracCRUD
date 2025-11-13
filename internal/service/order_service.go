package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"prac_service/pkg/api/test"
)

type OrderService struct {
	test.UnimplementedOrderServiceServer
	db *pgxpool.Pool
}

func NewOrderService(db *pgxpool.Pool) *OrderService {
	return &OrderService{
		db: db,
	}
}

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

	return &test.CreateOrderResponse{Id: id}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *test.GetOrderRequest) (*test.GetOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id заказа не может быть пустым")
	}

	var order test.Order
	query := `SELECT id, item, quantity FROM orders where id = $1`
	err := s.db.QueryRow(ctx, query, req.Id).Scan(&order.Id, &order.Item, &order.Quantity)
	if err != nil {
		return nil, status.Error(codes.NotFound, "заказ не найден")
	}

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

	return &test.DeleteOrderResponse{Success: true}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *test.ListOrdersRequest) (*test.ListOrdersResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
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

	return &test.ListOrdersResponse{Orders: orders}, nil
}
