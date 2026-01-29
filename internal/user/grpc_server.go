package user

import (
	"context"
	"strings"
	"time"

	userpb "github.com/SmartLinkDrive/SmartLinkDrive/internal/api/proto/user"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/auth"
	"github.com/SmartLinkDrive/SmartLinkDrive/internal/common/config"
	commonserver "github.com/SmartLinkDrive/SmartLinkDrive/internal/common/server"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type GRPCServer struct {
	userpb.UnimplementedUserServiceServer

	repo    *Repo
	authCfg config.AuthConfig
}

func NewGRPCServer(db *gorm.DB, authCfg config.AuthConfig) *GRPCServer {
	return &GRPCServer{
		repo:    NewRepo(db),
		authCfg: authCfg,
	}
}

func (s *GRPCServer) RegisterUser(ctx context.Context, req *userpb.RegisterUserRequest) (*userpb.RegisterUserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	username := strings.TrimSpace(req.GetUsername())
	password := req.GetPassword()
	if username == "" || password == "" {
		return nil, status.Error(codes.InvalidArgument, "username/password required")
	}

	// check existence
	if _, err := s.repo.FindByUsername(ctx, username); err == nil {
		return nil, status.Error(codes.AlreadyExists, "username already exists")
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, status.Error(codes.Internal, err.Error())
	}

	salt, err := GenerateSaltHex()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	hash, err := HashPassword(password, salt)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	u := &User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Nickname:     strings.TrimSpace(req.GetNickname()),
		Phone:        strings.TrimSpace(req.GetPhone()),
		Email:        strings.TrimSpace(req.GetEmail()),
		Roles:        RolesJoin([]string{"user"}),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &userpb.RegisterUserResponse{
		User: toPBUser(u, false),
	}, nil
}

func (s *GRPCServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	username := strings.TrimSpace(req.GetUsername())
	password := req.GetPassword()
	if username == "" || password == "" {
		return nil, status.Error(codes.InvalidArgument, "username/password required")
	}

	u, err := s.repo.FindByUsername(ctx, username)
	if err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !VerifyPassword(password, u.PasswordSalt, u.PasswordHash) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, exp, err := auth.GenerateAccessToken(s.authCfg, u.ID, u.RolesSlice(), 24*time.Hour)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &userpb.LoginResponse{
		AccessToken: token,
		ExpiresAt:   exp.Unix(),
		User:        toPBUser(u, false),
	}, nil
}

func (s *GRPCServer) GetProfile(ctx context.Context, _ *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	ai, ok := commonserver.AuthFromContext(ctx)
	if !ok || strings.TrimSpace(ai.Subject) == "" {
		return nil, status.Error(codes.Unauthenticated, "missing auth")
	}
	u, err := s.repo.FindByID(ctx, ai.Subject)
	if err == gorm.ErrRecordNotFound {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.GetProfileResponse{User: toPBUser(u, false)}, nil
}

func (s *GRPCServer) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	page := int(req.GetPage())
	size := int(req.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 200 {
		size = 20
	}
	offset := (page - 1) * size
	users, total, err := s.repo.List(ctx, offset, size)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*userpb.User, 0, len(users))
	for i := range users {
		u := users[i]
		out = append(out, toPBUser(&u, false))
	}
	return &userpb.ListUsersResponse{Users: out, Total: total}, nil
}

func toPBUser(u *User, includePasswordHash bool) *userpb.User {
	if u == nil {
		return nil
	}
	pwh := ""
	if includePasswordHash {
		pwh = u.PasswordHash
	}
	return &userpb.User{
		Id:           u.ID,
		Username:     u.Username,
		PasswordHash: pwh,
		Nickname:     u.Nickname,
		Phone:        u.Phone,
		Email:        u.Email,
		Roles:        u.RolesSlice(),
		CreatedAt:    u.CreatedAt.Unix(),
		UpdatedAt:    u.UpdatedAt.Unix(),
	}
}
