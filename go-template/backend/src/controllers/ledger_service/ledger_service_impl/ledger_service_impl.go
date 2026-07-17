package ledger_service_impl

import (
	"context"

	"cargonex-backend/src/controllers/ledger_service"
	"cargonex-backend/src/database/store"
)

type NewLedgerServiceImpl struct {
	Store *store.Store
}

type service struct {
	store *store.Store
}

func NewLedgerService(params NewLedgerServiceImpl) ledger_service.LedgerService {
	return &service{store: params.Store}
}

func (s *service) ListLedgers(ctx context.Context, query store.LedgerQuery, currentUserID int) (map[string]any, error) {
	return s.store.ListLedgers(ctx, query, currentUserID)
}

func (s *service) GetLedger(ctx context.Context, id string, currentUserID int) (map[string]any, error) {
	return s.store.GetLedgerByID(ctx, id, currentUserID)
}
