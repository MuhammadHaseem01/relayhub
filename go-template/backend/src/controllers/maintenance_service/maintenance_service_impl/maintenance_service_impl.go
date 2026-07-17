package maintenance_service_impl

import (
	"context"

	"cargonex-backend/src/controllers/maintenance_service"
	"cargonex-backend/src/database/store"
)

type NewMaintenanceServiceImpl struct {
	Store *store.Store
}

type service struct {
	store *store.Store
}

func NewMaintenanceService(params NewMaintenanceServiceImpl) maintenance_service.MaintenanceService {
	return &service{store: params.Store}
}

func (s *service) SyncLedger(ctx context.Context, maintenanceID int, currentUserID int) error {
	return s.store.ReplaceVehicleMaintenanceLedger(ctx, maintenanceID, currentUserID)
}

func (s *service) DeleteLedger(ctx context.Context, maintenanceID int, currentUserID int) error {
	return s.store.DeleteReferenceLedgers(ctx, "vehicleMaintenance", maintenanceID, currentUserID)
}
