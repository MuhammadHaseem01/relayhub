package router

import (
	"errors"
	"net/http"
	"regexp"

	"cargonex-backend/src/database/store"
)

func (s *Server) handleDeleteTour(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	cfg := tourStoreConfig()
	if err := s.tourService.DeleteTour(r.Context(), cfg, id, user.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, errorStatus(err), "Tour not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Tour deletion failed")
		return
	}
	writeOK(w, map[string]any{"message": "Tour deleted successfully"})
}

// handleUpdateTour is a smart dispatcher for PATCH /api/tour/{id}.
//   - If {id} is numeric → normal CRUD update (delegated to the registered CRUD handler).
//   - If {id} is a non-numeric string (e.g. a petrol pump slug) → bulk petrol pump
//     payment status update, mirroring NestJS tourService.updatePetrolPumpPaymentStatus().
func (s *Server) handleUpdateTour(w http.ResponseWriter, r *http.Request, user currentUser) {
	rawID := r.PathValue("id")

	if isNumeric(rawID) {
		// Normal numeric tour update — delegate to the CRUD handler wired in server.go
		// The CRUD patch handler reads the id from the path itself, so we just call it.
		s.handleCRUDUpdateTour(w, r, user)
		return
	}

	// Non-numeric id → petrol pump bulk update
	body, err := decodeJSON(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate: only paymentStatus allowed in this mode
	for key := range body {
		if key != "paymentStatus" {
			writeError(w, http.StatusBadRequest, "Valid tour id is required")
			return
		}
	}
	paymentStatus, _ := body["paymentStatus"].(string)
	if paymentStatus == "" {
		writeError(w, http.StatusBadRequest, "Valid tour id is required")
		return
	}

	// Normalize the slug exactly like NestJS toSourceId()
	sourceID := toSlug(rawID)
	if sourceID == "" {
		writeError(w, http.StatusBadRequest, "Valid petrol pump source id is required")
		return
	}

	result, err := s.tourService.UpdatePetrolPumpPaymentStatus(r.Context(), sourceID, paymentStatus, user.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Petrol pump ledger source not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Tour update failed")
		return
	}

	writeOK(w, map[string]any{
		"message":            "Petrol pump payment status updated successfully",
		"updatedTours":       result.UpdatedTours,
		"updatedFuelDetails": result.UpdatedFuelDetails,
	})
}

var numericRe = regexp.MustCompile(`^\d+$`)

func isNumeric(s string) bool {
	return numericRe.MatchString(s)
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func toSlug(s string) string {
	out := slugRe.ReplaceAllString(s, "-")
	out = regexp.MustCompile(`^-+|-+$`).ReplaceAllString(out, "")
	return out
}

func tourStoreConfig() store.CRUDConfig {
	return store.CRUDConfig{
		Table:         `"Tour"`,
		SelectColumns: []string{"id", "tourName", "driver", "vehicle", "client", "startLocation", "endLocation", "startDate", "time", "expectedEndDate", "actualEndDate", "actualEndTime", "freightAmount", "advanceAmount", "otherCharges", "paymentStatus", "partialReceivedPayment", "expenses", "fuelDetails", "loadType", "cargoWeight", "vehicleType", "status", "notes", "createdAt", "updatedAt", "createdById"},
		UpdateColumns: []store.Column{
			{JSON: "tourName", DB: "tourName"}, {JSON: "driver", DB: "driver", JSONString: true}, {JSON: "vehicle", DB: "vehicle", JSONString: true}, {JSON: "client", DB: "client", JSONString: true},
			{JSON: "startLocation", DB: "startLocation", JSONB: true}, {JSON: "endLocation", DB: "endLocation", JSONB: true},
			{JSON: "startDate", DB: "startDate"}, {JSON: "time", DB: "time"}, {JSON: "expectedEndDate", DB: "expectedEndDate"},
			{JSON: "actualEndDate", DB: "actualEndDate"}, {JSON: "actualEndTime", DB: "actualEndTime"},
			{JSON: "freightAmount", DB: "freightAmount"}, {JSON: "advanceAmount", DB: "advanceAmount"}, {JSON: "otherCharges", DB: "otherCharges"},
			{JSON: "paymentStatus", DB: "paymentStatus", Cast: "TripPaymentStatus"}, {JSON: "partialReceivedPayment", DB: "partialReceivedPayment"},
			{JSON: "expenses", DB: "expenses", JSONB: true}, {JSON: "fuelDetails", DB: "fuelDetails", JSONB: true},
			{JSON: "loadType", DB: "loadType"}, {JSON: "cargoWeight", DB: "cargoWeight"}, {JSON: "vehicleType", DB: "vehicleType", Cast: "VehicleType"},
			{JSON: "status", DB: "status", Cast: "TourStatus"}, {JSON: "notes", DB: "notes"},
		},
	}
}

// handleCRUDUpdateTour handles a standard numeric-ID PATCH /api/tour/{id} request,
// replicating the full CRUD PATCH logic with tour-specific AfterUpdate hooks
// (ledger sync + resource reset on Completed).
func (s *Server) handleCRUDUpdateTour(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	body, err := decodeJSON(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	stripTourUpdateReadOnlyFields(body)
	keepOnlyTourCompletionStatus(body)
	if err := s.populateTourRelationNames(r.Context(), user.ID, body); err != nil {
		writeError(w, http.StatusInternalServerError, "Tour update failed")
		return
	}
	cfg := tourStoreConfig()
	if writeValidationErrors(w, validateBody(body, cfg.UpdateColumns, validationUpdate)) {
		return
	}
	item, err := s.tourService.UpdateTour(r.Context(), cfg, id, user.ID, body)
	if err != nil {
		st := errorStatus(err)
		msg := "Tour update failed"
		switch st {
		case http.StatusNotFound:
			msg = "Tour not found"
		case http.StatusBadRequest:
			msg = "At least one field is required"
		}
		writeError(w, st, msg)
		return
	}
	writeOK(w, map[string]any{"message": "Tour updated successfully", "tour": item})
}

func keepOnlyTourCompletionStatus(body map[string]any) {
	status, _ := body["status"].(string)
	if status != "Completed" {
		return
	}
	for key := range body {
		if key != "status" {
			delete(body, key)
		}
	}
}

func stripTourUpdateReadOnlyFields(body map[string]any) {
	for _, key := range []string{
		"id", "_id", "__v", "createdAt", "updatedAt", "createdBy", "createdById",
		"organizationId", "organizationName", "tourNumber", "totalFuelExpenses",
		"totalExpenses", "Profit", "profit", "data", "message", "success",
	} {
		delete(body, key)
	}
}
