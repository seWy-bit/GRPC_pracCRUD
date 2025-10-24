package service

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"prac_service/pkg/api/test"
)

type OrderService struct {
	test.UnimplementedOrderServiceServer
	mu     *sync.RWMutex
	orders map[string]*test.Order
}

func NewOrderService() *OrderService {
	return &OrderService{
		mu:     &sync.RWMutex{},
		orders: make(map[string]*test.Order),
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

	order := &test.Order{
		Id:       id,
		Item:     req.Item,
		Quantity: req.Quantity,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[id] = order

	return &test.CreateOrderResponse{Id: id}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *test.GetOrderRequest) (*test.GetOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id заказа не может быть пустым")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	order, exists := s.orders[req.Id]

	if !exists {
		return nil, status.Error(codes.NotFound, "заказ не найден")
	}

	return &test.GetOrderResponse{Order: order}, nil
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

	s.mu.Lock()
	defer s.mu.Unlock()
	order, exists := s.orders[req.Id]

	if !exists {
		return nil, status.Error(codes.NotFound, "заказ не найден")
	}

	order.Item = req.Item
	order.Quantity = req.Quantity

	return &test.UpdateOrderResponse{Order: order}, nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, req *test.DeleteOrderRequest) (*test.DeleteOrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "ID заказа не может быть пустым")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.orders[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "заказ не найден")
	}

	delete(s.orders, req.Id)

	return &test.DeleteOrderResponse{Success: true}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *test.ListOrdersRequest) (*test.ListOrdersResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	orderList := make([]*test.Order, 0, len(s.orders))

	for _, order := range s.orders {
		orderList = append(orderList, order)
	}

	return &test.ListOrdersResponse{Orders: orderList}, nil
}
