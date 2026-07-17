package deduction_service

import (
	"context"

	"cargonex-backend/src/database/store"
)

type DeductionService interface {
	ApplyTourDeductionStatus(ctx context.Context, tourID string, deductionType string, currentUserID int, resetResources bool) error
	DeleteDeduction(ctx context.Context, cfg store.CRUDConfig, deductionID int, currentUserID int) error
}
