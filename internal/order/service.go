package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service 封装订单领域的核心用例（不依赖 gRPC / HTTP），便于复用和测试。
type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// CreateOrderInput 创建订单的入参（可作为传输层 DTO 的基础）。
type CreateOrderInput struct {
	UserID    string
	VehicleID string
	DriverID  string

	BizTag   string
	Channel  string
	Pickup   string
	Dropoff  string
	Currency string

	EstimatedPrice int64
}

// ListOrdersFilter 查询条件。
type ListOrdersFilter struct {
	UserID string
	Status Status
	Offset int
	Limit  int
}

func (s *Service) CreateOrder(ctx context.Context, in CreateOrderInput) (*Order, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	if strings.TrimSpace(in.UserID) == "" {
		return nil, fmt.Errorf("user_id required")
	}

	o := &Order{
		ID:             uuid.NewString(),
		UserID:         strings.TrimSpace(in.UserID),
		VehicleID:      strings.TrimSpace(in.VehicleID),
		DriverID:       strings.TrimSpace(in.DriverID),
		Status:         StatusCreated,
		BizTag:         strings.TrimSpace(in.BizTag),
		Channel:        strings.TrimSpace(in.Channel),
		PickupAddress:  strings.TrimSpace(in.Pickup),
		DropoffAddress: strings.TrimSpace(in.Dropoff),
		EstimatedPrice: in.EstimatedPrice,
		Currency:       defaultCurrency(in.Currency),
	}

	if err := s.repo.Create(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

// UpdateStatus 根据状态机规则进行状态流转。
func (s *Service) UpdateStatus(ctx context.Context, orderID string, to Status, now time.Time) (*Order, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, fmt.Errorf("order_id required")
	}
	if to == "" {
		return nil, fmt.Errorf("target status required")
	}

	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := ApplyTransition(o, to, now); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Service) GetOrder(ctx context.Context, id string) (*Order, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListOrders(ctx context.Context, f ListOrdersFilter) ([]Order, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, fmt.Errorf("service not initialized")
	}
	return s.repo.List(ctx, strings.TrimSpace(f.UserID), f.Status, f.Offset, f.Limit)
}

func defaultCurrency(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return "CNY"
	}
	return strings.ToUpper(c)
}
