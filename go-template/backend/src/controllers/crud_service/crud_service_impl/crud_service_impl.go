package crud_service_impl

import (
	"context"

	"cargonex-backend/src/controllers/crud_service"
	"cargonex-backend/src/database/store"
)

type NewCRUDServiceImpl struct {
	Store  *store.Store
	Config store.CRUDConfig
}

type service struct {
	store *store.Store
	cfg   store.CRUDConfig
}

func NewCRUDService(params NewCRUDServiceImpl) crud_service.CRUDService {
	return &service{store: params.Store, cfg: params.Config}
}

func (s *service) CreateOwned(ctx context.Context, body map[string]any, currentUserID int) (map[string]any, error) {
	return s.store.CreateOwned(ctx, s.cfg, body, currentUserID)
}

func (s *service) ListOwned(ctx context.Context, currentUserID int, page int, limit int) ([]map[string]any, store.Meta, error) {
	return s.store.ListOwned(ctx, s.cfg, currentUserID, page, limit)
}

func (s *service) GetOwned(ctx context.Context, id int, currentUserID int) (map[string]any, error) {
	return s.store.GetOwned(ctx, s.cfg, id, currentUserID)
}

func (s *service) UpdateOwned(ctx context.Context, id int, currentUserID int, body map[string]any) (map[string]any, error) {
	return s.store.UpdateOwned(ctx, s.cfg, id, currentUserID, body)
}

func (s *service) DeleteOwned(ctx context.Context, id int, currentUserID int) error {
	return s.store.DeleteOwned(ctx, s.cfg, id, currentUserID)
}
