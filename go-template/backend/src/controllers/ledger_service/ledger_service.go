package ledger_service

import (
	"context"

	"cargonex-backend/src/database/store"
)

type LedgerService interface {
	ListLedgers(ctx context.Context, query store.LedgerQuery, currentUserID int) (map[string]any, error)
	GetLedger(ctx context.Context, id string, currentUserID int) (map[string]any, error)
}
