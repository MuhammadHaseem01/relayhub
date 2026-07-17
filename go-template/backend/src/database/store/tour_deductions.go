package store

import (
	"context"
	"strconv"
	"strings"
)

func (s *Store) ApplyTourDeductionStatus(ctx context.Context, tourIDText string, deductionType string, currentUserID int, resetResources bool) error {
	tourID := TourIDFromText(tourIDText)
	if tourID == 0 {
		return ErrBadRequest
	}
	status := TourStatusFromDeduction(deductionType)
	rows, err := s.queryMaps(ctx, `UPDATE "Tour" SET "status" = $1::"TourStatus", "updatedAt" = NOW() WHERE "id" = $2 AND "createdById" = $3 RETURNING "id", "driver", "vehicle", "status", "createdById"`, status, tourID, currentUserID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrNotFound
	}
	if resetResources {
		return s.ResetTourAssignedResources(ctx, rows[0], currentUserID)
	}
	return nil
}

func (s *Store) SetTourCompletedFromDeduction(ctx context.Context, tourIDText string, currentUserID int) error {
	tourID := TourIDFromText(tourIDText)
	if tourID == 0 {
		return ErrBadRequest
	}
	result, err := s.db.ExecContext(ctx, `UPDATE "Tour" SET "status" = $1::"TourStatus", "updatedAt" = NOW() WHERE "id" = $2 AND "createdById" = $3`, "Completed", tourID, currentUserID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return nil
}

func TourStatusFromDeduction(deductionType string) string {
	if deductionType == "Late Delivery" || deductionType == "Late_Delivery" {
		return "Late Delivery"
	}
	return "Cancelled"
}

func TourIDFromText(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parts := strings.Split(value, "-")
	last := parts[len(parts)-1]
	id, err := strconv.Atoi(last)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}
