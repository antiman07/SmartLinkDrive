package vehicle

import (
	"context"
	"strings"

	vehiclepb "github.com/SmartLinkDrive/SmartLinkDrive/internal/api/proto/vehicle"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type GRPCServer struct {
	vehiclepb.UnimplementedVehicleServiceServer
	repo *Repo
}

func NewGRPCServer(db *gorm.DB) *GRPCServer {
	return &GRPCServer{repo: NewRepo(db)}
}

func (s *GRPCServer) UpsertVehicle(ctx context.Context, req *vehiclepb.UpsertVehicleRequest) (*vehiclepb.UpsertVehicleResponse, error) {
	if req == nil || req.GetVehicle() == nil {
		return nil, status.Error(codes.InvalidArgument, "vehicle required")
	}
	in := req.GetVehicle()

	plate := strings.TrimSpace(in.GetPlateNumber())
	if plate == "" {
		return nil, status.Error(codes.InvalidArgument, "plate_number required")
	}

	id := strings.TrimSpace(in.GetId())
	if id == "" {
		id = uuid.NewString()
	}
	st := strings.TrimSpace(in.GetStatus())
	if st == "" {
		st = "available"
	}

	v := &Vehicle{
		ID:          id,
		PlateNumber: plate,
		VIN:         strings.TrimSpace(in.GetVin()),
		Model:       strings.TrimSpace(in.GetModel()),
		OwnerID:     strings.TrimSpace(in.GetOwnerId()),
		Status:      st,
	}
	if err := s.repo.Upsert(ctx, v); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// read back to get timestamps if DB sets them
	latest, err := s.repo.FindByID(ctx, v.ID)
	if err != nil {
		// 如果查询失败，仍返回写入的内容（时间戳可能为空）
		latest = v
	}

	return &vehiclepb.UpsertVehicleResponse{Vehicle: toPB(latest)}, nil
}

func (s *GRPCServer) GetVehicle(ctx context.Context, req *vehiclepb.GetVehicleRequest) (*vehiclepb.GetVehicleResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	v, err := s.repo.FindByID(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "vehicle not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &vehiclepb.GetVehicleResponse{Vehicle: toPB(v)}, nil
}

func (s *GRPCServer) ListVehicles(ctx context.Context, req *vehiclepb.ListVehiclesRequest) (*vehiclepb.ListVehiclesResponse, error) {
	owner := ""
	page := 1
	size := 20
	if req != nil {
		owner = strings.TrimSpace(req.GetOwnerId())
		if req.GetPage() > 0 {
			page = int(req.GetPage())
		}
		if req.GetPageSize() > 0 && req.GetPageSize() <= 200 {
			size = int(req.GetPageSize())
		}
	}
	offset := (page - 1) * size
	vs, total, err := s.repo.List(ctx, owner, offset, size)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*vehiclepb.Vehicle, 0, len(vs))
	for i := range vs {
		v := vs[i]
		out = append(out, toPB(&v))
	}
	return &vehiclepb.ListVehiclesResponse{Vehicles: out, Total: total}, nil
}

func toPB(v *Vehicle) *vehiclepb.Vehicle {
	if v == nil {
		return nil
	}
	return &vehiclepb.Vehicle{
		Id:          v.ID,
		PlateNumber: v.PlateNumber,
		Vin:         v.VIN,
		Model:       v.Model,
		OwnerId:     v.OwnerID,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt.Unix(),
		UpdatedAt:   v.UpdatedAt.Unix(),
	}
}
