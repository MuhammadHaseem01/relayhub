package maintenance_service

import (
	"context"
)

type MaintenanceService interface {
	SyncLedger(ctx context.Context, maintenanceID int, currentUserID int) error
	DeleteLedger(ctx context.Context, maintenanceID int, currentUserID int) error
}
