package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type StatusHistoryKind string

const (
	HistoryDriver  StatusHistoryKind = "driver"
	HistoryVehicle StatusHistoryKind = "vehicle"
)

type statusHistoryConfig struct {
	Table     string
	OwnerCol  string
	StatusCol string
}

func (s *Store) EnsureTodayStatusHistory(ctx context.Context, kind StatusHistoryKind, ownerID int, status string) error {
	cfg := historyConfig(kind)
	start := time.Now().UTC().Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)
	var id int
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT "id" FROM %s WHERE %s = $1 AND "date" >= $2 AND "date" < $3 LIMIT 1`, cfg.Table, cfg.OwnerCol), ownerID, start, end).Scan(&id)
	if err == nil {
		return nil
	}
	return s.CreateStatusHistory(ctx, kind, ownerID, status, time.Now().UTC())
}

func (s *Store) CreateStatusHistory(ctx context.Context, kind StatusHistoryKind, ownerID int, status string, date time.Time) error {
	cfg := historyConfig(kind)
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (%s, "status", "date") VALUES ($1, $2::%s, $3)`, cfg.Table, cfg.OwnerCol, cfg.StatusCol), ownerID, status, date)
	return err
}

func (s *Store) StatusHistories(ctx context.Context, kind StatusHistoryKind, ownerIDs []int) (map[int][]map[string]any, error) {
	out := map[int][]map[string]any{}
	if len(ownerIDs) == 0 {
		return out, nil
	}
	cfg := historyConfig(kind)
	args := make([]any, len(ownerIDs))
	placeholders := make([]string, len(ownerIDs))
	for i, id := range ownerIDs {
		args[i] = id
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	rows, err := s.queryMaps(ctx, fmt.Sprintf(`SELECT "id", %s, "status"::TEXT, "date" FROM %s WHERE %s IN (%s) ORDER BY "date" ASC, "id" ASC`, cfg.OwnerCol, cfg.Table, cfg.OwnerCol, strings.Join(placeholders, ", ")), args...)
	if err != nil {
		return nil, err
	}
	ownerKey := strings.Trim(cfg.OwnerCol, `"`)
	for _, row := range rows {
		ownerID := intFromAny(row[ownerKey])
		out[ownerID] = append(out[ownerID], map[string]any{
			"id":     row["id"],
			"status": displayStatus(fmt.Sprint(row["status"])),
			"date":   row["date"],
		})
	}
	return out, nil
}

func historyConfig(kind StatusHistoryKind) statusHistoryConfig {
	if kind == HistoryDriver {
		return statusHistoryConfig{Table: `"DriverStatusHistory"`, OwnerCol: `"driverId"`, StatusCol: `"DriverStatus"`}
	}
	return statusHistoryConfig{Table: `"VehicleStatusHistory"`, OwnerCol: `"vehicleId"`, StatusCol: `"VehicleStatus"`}
}

func displayStatus(status string) string {
	if status == "Available" {
		return "Avaliable"
	}
	return status
}

func dbStatus(status string) string {
	if status == "Available" {
		return "Avaliable"
	}
	return status
}
