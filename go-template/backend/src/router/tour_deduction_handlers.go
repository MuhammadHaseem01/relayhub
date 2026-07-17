package router

import (
	"net/http"

	"cargonex-backend/src/database/store"
)

func (s *Server) handleDeleteTourDeduction(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	cfg := tourDeductionStoreConfig()
	if err := s.deductionService.DeleteDeduction(r.Context(), cfg, id, user.ID); err != nil {
		if errorStatus(err) == http.StatusNotFound {
			writeError(w, errorStatus(err), "Tour deduction not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Tour deduction deletion failed")
		return
	}
	writeOK(w, map[string]any{"message": "Tour deduction deleted successfully"})
}

func tourDeductionStoreConfig() store.CRUDConfig {
	return store.CRUDConfig{Table: `"TourDeduction"`, SelectColumns: []string{"id", "tourId", "deductedAmount", "deductionType", "deductionDate", "reason", "status", "createdAt", "updatedAt", "createdById"}, CreateColumns: []store.Column{{JSON: "tourId", DB: "tourId", Required: true}, {JSON: "deductedAmount", DB: "deductedAmount", Required: true}, {JSON: "deductionType", DB: "deductionType", Cast: "DeductionType", DefaultFromJSON: "status"}, {JSON: "deductionDate", DB: "deductionDate", Required: true}, {JSON: "reason", DB: "reason", Required: true}, {JSON: "status", DB: "status", Required: true, Cast: "DeductionType"}}, UpdateColumns: []store.Column{{JSON: "tourId", DB: "tourId"}, {JSON: "deductedAmount", DB: "deductedAmount"}, {JSON: "deductionType", DB: "deductionType", Cast: "DeductionType"}, {JSON: "deductionDate", DB: "deductionDate"}, {JSON: "reason", DB: "reason"}, {JSON: "status", DB: "status", Cast: "DeductionType"}}}
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
