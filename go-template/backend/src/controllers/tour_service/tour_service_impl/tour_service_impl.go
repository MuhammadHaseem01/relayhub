package tour_service_impl

import (
	"context"
	"fmt"

	"cargonex-backend/src/controllers/tour_service"
	"cargonex-backend/src/database/store"
)

type NewTourServiceImpl struct {
	Store *store.Store
}

type service struct {
	store *store.Store
}

func NewTourService(params NewTourServiceImpl) tour_service.TourService {
	return &service{store: params.Store}
}

func (s *service) SyncLedgers(ctx context.Context, tourID int, currentUserID int) error {
	return s.store.ReplaceTourLedgers(ctx, tourID, currentUserID)
}

func (s *service) ResetAssignedResources(ctx context.Context, tour map[string]any, currentUserID int) error {
	return s.store.ResetTourAssignedResources(ctx, tour, currentUserID)
}

func (s *service) UpdateTour(ctx context.Context, cfg store.CRUDConfig, tourID int, currentUserID int, body map[string]any) (map[string]any, error) {
	item, err := s.store.UpdateOwned(ctx, cfg, tourID, currentUserID, body)
	if err != nil {
		return nil, err
	}
	if err := s.SyncLedgers(ctx, idFromItem(item), currentUserID); err != nil {
		return nil, err
	}
	if item["status"] == "Completed" {
		if err := s.ResetAssignedResources(ctx, item, currentUserID); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *service) DeleteTour(ctx context.Context, cfg store.CRUDConfig, tourID int, currentUserID int) error {
	tour, err := s.store.GetOwned(ctx, cfg, tourID, currentUserID)
	if err != nil {
		return err
	}
	if err := s.store.DeleteOwned(ctx, cfg, tourID, currentUserID); err != nil {
		return err
	}
	if err := s.store.DeleteReferenceLedgers(ctx, "tour", tourID, currentUserID); err != nil {
		return err
	}
	if tour["status"] == "In Progress" || tour["status"] == "Pre-Planned" {
		return s.ResetAssignedResources(ctx, tour, currentUserID)
	}
	return nil
}

func (s *service) UpdatePetrolPumpPaymentStatus(ctx context.Context, sourceID string, paymentStatus string, currentUserID int) (store.PetrolPumpUpdateResult, error) {
	return s.store.UpdatePetrolPumpPaymentStatus(ctx, sourceID, paymentStatus, currentUserID)
}

func idFromItem(item map[string]any) int {
	switch v := item["id"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var id int
		_, _ = fmt.Sscan(v, &id)
		return id
	}
	return 0
}
