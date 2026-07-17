package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Store) ResetTourAssignedResources(ctx context.Context, tour map[string]any, currentUserID int) error {
	statusDateTime := time.Now().UTC()
	if driverID, err := s.ResolveAssignedDriverID(ctx, tour, currentUserID); err != nil {
		return err
	} else if driverID > 0 {
		result, err := s.db.ExecContext(ctx, `UPDATE "Driver" SET "status" = $1::"DriverStatus", "statusDateTime" = $2, "updatedAt" = $2 WHERE "id" = $3 AND "createdById" = $4`, "Avaliable", statusDateTime, driverID, currentUserID)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected > 0 {
			_ = s.CreateStatusHistory(ctx, HistoryDriver, driverID, "Avaliable", statusDateTime)
		}
	}
	if vehicleID, err := s.ResolveAssignedVehicleID(ctx, tour, currentUserID); err != nil {
		return err
	} else if vehicleID > 0 {
		result, err := s.db.ExecContext(ctx, `UPDATE "Vehicle" SET "status" = $1::"VehicleStatus", "statusDateTime" = $2, "updatedAt" = $2 WHERE "id" = $3 AND "createdById" = $4`, "Avaliable", statusDateTime, vehicleID, currentUserID)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected > 0 {
			_ = s.CreateStatusHistory(ctx, HistoryVehicle, vehicleID, "Avaliable", statusDateTime)
		}
	}
	return nil
}

func (s *Store) ResolveAssignedDriverID(ctx context.Context, tour map[string]any, currentUserID int) (int, error) {
	selected := namedValueFromStored(tour["driver"])
	if selected.ID != "" {
		if id, ok := numericStringID(selected.ID); ok {
			return id, nil
		}
	}
	if selected.Name == "" {
		return 0, nil
	}
	var id int
	err := s.db.QueryRowContext(ctx, `SELECT "id" FROM "Driver" WHERE "fullName" = $1 AND "createdById" = $2 LIMIT 1`, selected.Name, currentUserID).Scan(&id)
	if err != nil {
		return 0, nil
	}
	return id, nil
}

func (s *Store) ResolveAssignedVehicleID(ctx context.Context, tour map[string]any, currentUserID int) (int, error) {
	selected := namedValueFromStored(tour["vehicle"])
	if selected.ID != "" {
		if id, ok := numericStringID(selected.ID); ok {
			return id, nil
		}
	}
	if selected.Name == "" {
		return 0, nil
	}
	var id int
	err := s.db.QueryRowContext(ctx, `SELECT "id" FROM "Vehicle" WHERE "vehicleNumber" = $1 AND "createdById" = $2 LIMIT 1`, selected.Name, currentUserID).Scan(&id)
	if err != nil {
		return 0, nil
	}
	return id, nil
}

type storedNamedValue struct {
	ID   string
	Name string
}

func namedValueFromStored(value any) storedNamedValue {
	raw := strings.TrimSpace(fmt.Sprint(value))
	if raw == "" || raw == "<nil>" {
		return storedNamedValue{}
	}
	var parsed map[string]any
	if json.Unmarshal([]byte(raw), &parsed) == nil {
		id := rawString(parsed["id"])
		name := rawString(parsed["name"])
		if id != "" || name != "" {
			return storedNamedValue{ID: id, Name: name}
		}
	}
	return storedNamedValue{ID: raw, Name: raw}
}

func numericStringID(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	id, err := strconv.Atoi(value)
	return id, err == nil && id > 0
}

func rawString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	}
	return ""
}

// PetrolPumpUpdateResult holds the result of a bulk petrol pump payment status update.
type PetrolPumpUpdateResult struct {
	UpdatedTours       int
	UpdatedFuelDetails int
}

// UpdatePetrolPumpPaymentStatus updates paymentStatus on all fuelDetails entries
// whose petrolPumpName slug matches sourceID, across all tours owned by currentUserID.
// Mirrors NestJS tourService.updatePetrolPumpPaymentStatus().
func (s *Store) UpdatePetrolPumpPaymentStatus(ctx context.Context, sourceID string, paymentStatus string, currentUserID int) (PetrolPumpUpdateResult, error) {
	if err := s.EnsureLedgerTable(ctx); err != nil {
		return PetrolPumpUpdateResult{}, err
	}

	// Load all tours for this user (only what we need)
	tours, err := s.queryMaps(ctx, `SELECT "id", "fuelDetails", "createdAt", "updatedAt", "createdById" FROM "Tour" WHERE "createdById" = $1`, currentUserID)
	if err != nil {
		return PetrolPumpUpdateResult{}, err
	}

	var result PetrolPumpUpdateResult

	for _, tour := range tours {
		tourID := intFromAny(tour["id"])

		// Parse fuelDetails JSON array
		var fuelDetails []map[string]any
		if raw, ok := tour["fuelDetails"].(string); ok && raw != "" {
			_ = json.Unmarshal([]byte(raw), &fuelDetails)
		}

		tourUpdated := false
		updatedCount := 0

		for i, fuel := range fuelDetails {
			pumpName, _ := fuel["petrolPumpName"].(string)
			fuelSlug := toSourceID(pumpName)
			if fuelSlug != sourceID {
				continue
			}
			// Update the paymentStatus in this fuel entry
			fuelDetails[i]["paymentStatus"] = paymentStatus
			tourUpdated = true
			updatedCount++
		}

		if !tourUpdated {
			continue
		}

		// Marshal updated fuelDetails back to JSON
		updatedJSON, err := json.Marshal(fuelDetails)
		if err != nil {
			return result, err
		}

		_, err = s.db.ExecContext(ctx,
			`UPDATE "Tour" SET "fuelDetails" = $1, "updatedAt" = NOW() WHERE "id" = $2 AND "createdById" = $3`,
			string(updatedJSON), tourID, currentUserID,
		)
		if err != nil {
			return result, err
		}

		// Sync ledgers for this tour
		_ = s.ReplaceTourLedgers(ctx, tourID, currentUserID)

		result.UpdatedTours++
		result.UpdatedFuelDetails += updatedCount
	}

	if result.UpdatedTours == 0 {
		return result, ErrNotFound
	}

	return result, nil
}
