package resource_service

import (
	"context"

	"cargonex-backend/src/database/store"
)

type ResourceService interface {
	CreateWithStatusHistory(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, body map[string]any, currentUserID int) (map[string]any, error)
	UpdateWithStatusHistory(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, id int, currentUserID int, body map[string]any, statusChanged bool, status string) (map[string]any, error)
	ListWithHistories(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, currentUserID int, page int, limit int) ([]map[string]any, store.Meta, error)
	GetWithHistories(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, id int, currentUserID int, ensureToday bool) (map[string]any, error)
	AttachHistories(ctx context.Context, kind store.StatusHistoryKind, items []map[string]any, ensureToday bool) ([]map[string]any, error)
	DeleteOwned(ctx context.Context, cfg store.CRUDConfig, id int, currentUserID int) error
}
