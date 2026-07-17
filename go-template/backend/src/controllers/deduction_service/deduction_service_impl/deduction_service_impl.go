package deduction_service_impl

import (
	"context"

	"cargonex-backend/src/controllers/deduction_service"
	"cargonex-backend/src/database/store"
)

type NewDeductionServiceImpl struct {
	Store *store.Store
}

type service struct {
	store *store.Store
}

func NewDeductionService(params NewDeductionServiceImpl) deduction_service.DeductionService {
	return &service{store: params.Store}
}

func (s *service) ApplyTourDeductionStatus(ctx context.Context, tourID string, deductionType string, currentUserID int, resetResources bool) error {
	return s.store.ApplyTourDeductionStatus(ctx, tourID, deductionType, currentUserID, resetResources)
}

func (s *service) DeleteDeduction(ctx context.Context, cfg store.CRUDConfig, deductionID int, currentUserID int) error {
	deduction, err := s.store.GetOwned(ctx, cfg, deductionID, currentUserID)
	if err != nil {
		return err
	}
	if err := s.store.DeleteOwned(ctx, cfg, deductionID, currentUserID); err != nil {
		return err
	}
	tourID, _ := deduction["tourId"].(string)
	return s.store.SetTourCompletedFromDeduction(ctx, tourID, currentUserID)
}
