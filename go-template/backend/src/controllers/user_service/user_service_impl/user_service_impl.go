package user_service_impl

import (
	"context"

	"cargonex-backend/src/auth"
	"cargonex-backend/src/controllers/user_service"
	"cargonex-backend/src/database/store"
)

type NewUserServiceImpl struct {
	Store      *store.Store
	AuthSecret string
}

type service struct {
	store      *store.Store
	authSecret string
}

func NewUserService(params NewUserServiceImpl) user_service.UserService {
	return &service{store: params.Store, authSecret: params.AuthSecret}
}

func (s *service) Login(ctx context.Context, email string, password string) (map[string]any, error) {
	user, err := s.store.LoginUser(ctx, email, password)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"message": "Login Successfully",
		"success": true,
		"Token":   auth.CreateToken(user.ID, s.authSecret),
	}, nil
}

func (s *service) Register(ctx context.Context, body map[string]any, currentUserID int) (map[string]any, error) {
	if err := s.store.RegisterUser(ctx, body, currentUserID); err != nil {
		return nil, err
	}
	return map[string]any{"message": "User registered successfully"}, nil
}

func (s *service) ListUsers(ctx context.Context, currentUserID int, page int, limit int) (map[string]any, error) {
	users, meta, err := s.store.ListUsers(ctx, currentUserID, page, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"users": users, "meta": meta}, nil
}

func (s *service) GetUser(ctx context.Context, id int, currentUserID int) (map[string]any, error) {
	user, err := s.store.GetAccessibleUser(ctx, id, currentUserID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": true, "user": user}, nil
}

func (s *service) UpdateUser(ctx context.Context, id int, currentUserID int, body map[string]any) (map[string]any, error) {
	return s.store.UpdateUser(ctx, id, currentUserID, body)
}

func (s *service) DeleteUser(ctx context.Context, id int, currentUserID int) (map[string]any, error) {
	if err := s.store.DeleteUser(ctx, id, currentUserID); err != nil {
		return nil, err
	}
	return map[string]any{"message": "User deleted successfully"}, nil
}
