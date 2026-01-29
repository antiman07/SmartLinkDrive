package order

import (
	"context"
	"strings"
	"time"

	orderpb "github.com/SmartLinkDrive/SmartLinkDrive/internal/api/proto/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type GRPCServer struct {
	orderpb.UnimplementedOrderServiceServer

	svc *Service
}

func NewGRPCServer(db *gorm.DB) *GRPCServer {
	repo := NewRepo(db)
	return &GRPCServer{
		svc: NewService(repo),
	}
}

func (s *GRPCServer) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	in := CreateOrderInput{
		UserID:         strings.TrimSpace(req.UserId),
		VehicleID:      strings.TrimSpace(req.VehicleId),
		DriverID:       strings.TrimSpace(req.DriverId),
		BizTag:         strings.TrimSpace(req.BizTag),
		Channel:        strings.TrimSpace(req.Channel),
		Pickup:         strings.TrimSpace(req.PickupAddress),
		Dropoff:        strings.TrimSpace(req.DropoffAddress),
		EstimatedPrice: req.EstimatedPrice,
		Currency:       strings.TrimSpace(req.Currency),
	}
	o, err := s.svc.CreateOrder(ctx, in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &orderpb.CreateOrderResponse{Order: toPBOrder(o)}, nil
}

func (s *GRPCServer) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	id := strings.TrimSpace(req.Id)
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	o, err := s.svc.GetOrder(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &orderpb.GetOrderResponse{Order: toPBOrder(o)}, nil
}

func (s *GRPCServer) UpdateOrderStatus(ctx context.Context, req *orderpb.UpdateOrderStatusRequest) (*orderpb.UpdateOrderStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	id := strings.TrimSpace(req.Id)
	st := strings.TrimSpace(req.Status)
	if id == "" || st == "" {
		return nil, status.Error(codes.InvalidArgument, "id/status required")
	}
	o, err := s.svc.UpdateStatus(ctx, id, Status(st), time.Now())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		// 对状态机非法流转，返回 FailedPrecondition
		if strings.Contains(err.Error(), "invalid order status transition") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &orderpb.UpdateOrderStatusResponse{Order: toPBOrder(o)}, nil
}

func (s *GRPCServer) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
	f := ListOrdersFilter{}
	if req != nil {
		f.UserID = strings.TrimSpace(req.UserId)
		if st := strings.TrimSpace(req.Status); st != "" {
			f.Status = Status(st)
		}
		page := int(req.Page)
		size := int(req.PageSize)
		if page <= 0 {
			page = 1
		}
		if size <= 0 || size > 200 {
			size = 20
		}
		f.Offset = (page - 1) * size
		f.Limit = size
	}

	orders, total, err := s.svc.ListOrders(ctx, f)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*orderpb.Order, 0, len(orders))
	for i := range orders {
		o := orders[i]
		out = append(out, toPBOrder(&o))
	}
	return &orderpb.ListOrdersResponse{Orders: out, Total: total}, nil
}

func toPBOrder(o *Order) *orderpb.Order {
	if o == nil {
		return nil
	}
	var acceptedAt, startedAt, completedAt, canceledAt int64
	if o.AcceptedAt != nil {
		acceptedAt = o.AcceptedAt.Unix()
	}
	if o.StartedAt != nil {
		startedAt = o.StartedAt.Unix()
	}
	if o.CompletedAt != nil {
		completedAt = o.CompletedAt.Unix()
	}
	if o.CanceledAt != nil {
		canceledAt = o.CanceledAt.Unix()
	}
	return &orderpb.Order{
		Id:             o.ID,
		UserId:         o.UserID,
		VehicleId:      o.VehicleID,
		DriverId:       o.DriverID,
		Status:         string(o.Status),
		BizTag:         o.BizTag,
		Channel:        o.Channel,
		PickupAddress:  o.PickupAddress,
		DropoffAddress: o.DropoffAddress,
		EstimatedPrice: o.EstimatedPrice,
		FinalPrice:     o.FinalPrice,
		Currency:       o.Currency,
		CreatedAt:      o.CreatedAt.Unix(),
		UpdatedAt:      o.UpdatedAt.Unix(),
		AcceptedAt:     acceptedAt,
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		CanceledAt:     canceledAt,
	}
}
