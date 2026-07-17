package crud_service

import (
	"context"

	"cargonex-backend/src/database/store"
)

type CRUDService interface {
	CreateOwned(ctx context.Context, body map[string]any, currentUserID int) (map[string]any, error)
	ListOwned(ctx context.Context, currentUserID int, page int, limit int) ([]map[string]any, store.Meta, error)
	GetOwned(ctx context.Context, id int, currentUserID int) (map[string]any, error)
	UpdateOwned(ctx context.Context, id int, currentUserID int, body map[string]any) (map[string]any, error)
	DeleteOwned(ctx context.Context, id int, currentUserID int) error
}
