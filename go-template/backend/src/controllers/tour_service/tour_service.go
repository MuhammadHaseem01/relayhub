package tour_service

import (
	"context"

	"cargonex-backend/src/database/store"
)

type TourService interface {
	SyncLedgers(ctx context.Context, tourID int, currentUserID int) error
	ResetAssignedResources(ctx context.Context, tour map[string]any, currentUserID int) error
	UpdateTour(ctx context.Context, cfg store.CRUDConfig, tourID int, currentUserID int, body map[string]any) (map[string]any, error)
	DeleteTour(ctx context.Context, cfg store.CRUDConfig, tourID int, currentUserID int) error
	UpdatePetrolPumpPaymentStatus(ctx context.Context, sourceID string, paymentStatus string, currentUserID int) (store.PetrolPumpUpdateResult, error)
}
