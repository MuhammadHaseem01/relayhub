package user_service

import (
	"context"
)

type UserService interface {
	Login(ctx context.Context, email string, password string) (map[string]any, error)
	Register(ctx context.Context, body map[string]any, currentUserID int) (map[string]any, error)
	ListUsers(ctx context.Context, currentUserID int, page int, limit int) (map[string]any, error)
	GetUser(ctx context.Context, id int, currentUserID int) (map[string]any, error)
	UpdateUser(ctx context.Context, id int, currentUserID int, body map[string]any) (map[string]any, error)
	DeleteUser(ctx context.Context, id int, currentUserID int) (map[string]any, error)
}
