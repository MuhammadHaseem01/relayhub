package resource_service_impl

import (
	"context"
	"fmt"
	"time"

	"cargonex-backend/src/controllers/resource_service"
	"cargonex-backend/src/database/store"
)

type NewResourceServiceImpl struct {
	Store *store.Store
}

type service struct {
	store *store.Store
}

func NewResourceService(params NewResourceServiceImpl) resource_service.ResourceService {
	return &service{store: params.Store}
}

func (s *service) CreateWithStatusHistory(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, body map[string]any, currentUserID int) (map[string]any, error) {
	item, err := s.store.CreateOwned(ctx, cfg, body, currentUserID)
	if err != nil {
		return nil, err
	}
	if err := s.store.CreateStatusHistory(ctx, kind, idFromItem(item), statusFromItem(item), timeFromItem(item["statusDateTime"])); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *service) UpdateWithStatusHistory(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, id int, currentUserID int, body map[string]any, statusChanged bool, status string) (map[string]any, error) {
	item, err := s.store.UpdateOwned(ctx, cfg, id, currentUserID, body)
	if err != nil {
		return nil, err
	}
	if statusChanged {
		if err := s.store.CreateStatusHistory(ctx, kind, id, dbStatus(status), timeFromItem(item["statusDateTime"])); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *service) ListWithHistories(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, currentUserID int, page int, limit int) ([]map[string]any, store.Meta, error) {
	items, meta, err := s.store.ListOwned(ctx, cfg, currentUserID, page, limit)
	if err != nil {
		return nil, store.Meta{}, err
	}
	items, err = s.AttachHistories(ctx, kind, items, true)
	return items, meta, err
}

func (s *service) GetWithHistories(ctx context.Context, cfg store.CRUDConfig, kind store.StatusHistoryKind, id int, currentUserID int, ensureToday bool) (map[string]any, error) {
	item, err := s.store.GetOwned(ctx, cfg, id, currentUserID)
	if err != nil {
		return nil, err
	}
	items, err := s.AttachHistories(ctx, kind, []map[string]any{item}, ensureToday)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func (s *service) AttachHistories(ctx context.Context, kind store.StatusHistoryKind, items []map[string]any, ensureToday bool) ([]map[string]any, error) {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		id := idFromItem(item)
		ids = append(ids, id)
		if ensureToday {
			_ = s.store.EnsureTodayStatusHistory(ctx, kind, id, statusFromItem(item))
		}
	}
	histories, err := s.store.StatusHistories(ctx, kind, ids)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		normalizeResourceDisplay(item)
		item["statusHistory"] = histories[idFromItem(item)]
	}
	return items, nil
}

func (s *service) DeleteOwned(ctx context.Context, cfg store.CRUDConfig, id int, currentUserID int) error {
	return s.store.DeleteOwned(ctx, cfg, id, currentUserID)
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

func timeFromItem(value any) time.Time {
	if s, ok := value.(string); ok && s == "now()" {
		return time.Now().UTC()
	}
	if t, ok := value.(time.Time); ok {
		return t
	}
	return time.Now().UTC()
}

func statusFromItem(item map[string]any) string {
	status, _ := item["status"].(string)
	return dbStatus(status)
}

func dbStatus(status string) string {
	if status == "Available" {
		return "Avaliable"
	}
	return status
}

func normalizeResourceDisplay(item map[string]any) {
	item["status"] = dbStatus(statusFromItem(item))
}
