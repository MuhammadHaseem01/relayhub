package router

import (
	"errors"
	"net/http"
	"time"

	"cargonex-backend/src/database/store"
)

func (s *Server) handleCreateVehicle(w http.ResponseWriter, r *http.Request, user currentUser) {
	body, err := decodeJSON(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	status, ok := requestedStatus(body)
	if !ok {
		status = "Avaliable"
	}
	body["status"] = status
	if v, ok := body["statusDateTime"]; !ok || v == nil || v == "" {
		body["statusDateTime"] = "now()"
	}
	cfg := vehicleStoreConfig()
	if writeValidationErrors(w, validateBody(body, cfg.CreateColumns, validationCreate, "statuses")) {
		return
	}
	item, err := s.resourceService.CreateWithStatusHistory(r.Context(), cfg, store.HistoryVehicle, body, user.ID)
	if err != nil {
		writeError(w, errorStatus(err), "Vehicle creation failed")
		return
	}
	items, err := s.resourceService.AttachHistories(r.Context(), store.HistoryVehicle, []map[string]any{item}, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Vehicle creation failed")
		return
	}
	item = items[0]
	writeCreated(w, map[string]any{"message": "Vehicle created successfully", "vehicle": item})
}

func (s *Server) handleListVehicles(w http.ResponseWriter, r *http.Request, user currentUser) {
	page, limit, ok := pagination(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid pagination query")
		return
	}
	items, meta, err := s.resourceService.ListWithHistories(r.Context(), vehicleStoreConfig(), store.HistoryVehicle, user.ID, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Request failed")
		return
	}
	writeOK(w, map[string]any{"success": true, "vehicles": items, "meta": meta})
}

func (s *Server) handleGetVehicle(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	item, err := s.resourceService.GetWithHistories(r.Context(), vehicleStoreConfig(), store.HistoryVehicle, id, user.ID, false)
	if err != nil {
		writeError(w, errorStatus(err), "Vehicle not found")
		return
	}
	writeOK(w, item)
}

func (s *Server) handleUpdateVehicle(w http.ResponseWriter, r *http.Request, user currentUser) {
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
	status, hasStatus := requestedStatus(body)
	if hasStatus {
		body["status"] = status
		if _, ok := body["statusDateTime"]; !ok {
			body["statusDateTime"] = "now()"
		}
	}
	if writeValidationErrors(w, validateBody(body, vehicleStoreConfig().UpdateColumns, validationUpdate, "statuses")) {
		return
	}
	item, err := s.resourceService.UpdateWithStatusHistory(r.Context(), vehicleStoreConfig(), store.HistoryVehicle, id, user.ID, body, hasStatus, status)
	if err != nil {
		writeResourceUpdateError(w, err, "Vehicle")
		return
	}
	items, err := s.resourceService.AttachHistories(r.Context(), store.HistoryVehicle, []map[string]any{item}, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Vehicle update failed")
		return
	}
	item = items[0]
	writeOK(w, map[string]any{"message": "Vehicle updated successfully", "vehicle": item})
}

func (s *Server) handleDeleteVehicle(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	if err := s.resourceService.DeleteOwned(r.Context(), vehicleStoreConfig(), id, user.ID); err != nil {
		writeError(w, errorStatus(err), "Vehicle not found")
		return
	}
	writeOK(w, map[string]any{"message": "Vehicle deleted successfully"})
}

func (s *Server) handleCreateDriver(w http.ResponseWriter, r *http.Request, user currentUser) {
	body, err := decodeJSON(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	status, ok := requestedStatus(body)
	if !ok {
		status = "Avaliable"
	}
	body["status"] = status
	if v, ok := body["statusDateTime"]; !ok || v == nil || v == "" {
		body["statusDateTime"] = "now()"
	}
	if writeValidationErrors(w, validateBody(body, driverStoreConfig().CreateColumns, validationCreate, "statuses")) {
		return
	}
	item, err := s.resourceService.CreateWithStatusHistory(r.Context(), driverStoreConfig(), store.HistoryDriver, body, user.ID)
	if err != nil {
		writeError(w, errorStatus(err), "Driver creation failed")
		return
	}
	items, err := s.resourceService.AttachHistories(r.Context(), store.HistoryDriver, []map[string]any{item}, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Driver creation failed")
		return
	}
	item = items[0]
	writeCreated(w, map[string]any{"message": "Driver created successfully", "driver": item})
}

func (s *Server) handleListDrivers(w http.ResponseWriter, r *http.Request, user currentUser) {
	page, limit, ok := pagination(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid pagination query")
		return
	}
	items, meta, err := s.resourceService.ListWithHistories(r.Context(), driverStoreConfig(), store.HistoryDriver, user.ID, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Request failed")
		return
	}
	writeOK(w, map[string]any{"success": true, "drivers": items, "data": items, "meta": meta})
}

func (s *Server) handleUpdateDriver(w http.ResponseWriter, r *http.Request, user currentUser) {
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
	status, hasStatus := requestedStatus(body)
	if hasStatus {
		body["status"] = status
		if _, ok := body["statusDateTime"]; !ok {
			body["statusDateTime"] = "now()"
		}
	}
	if writeValidationErrors(w, validateBody(body, driverStoreConfig().UpdateColumns, validationUpdate, "statuses")) {
		return
	}
	item, err := s.resourceService.UpdateWithStatusHistory(r.Context(), driverStoreConfig(), store.HistoryDriver, id, user.ID, body, hasStatus, status)
	if err != nil {
		writeResourceUpdateError(w, err, "Driver")
		return
	}
	items, err := s.resourceService.AttachHistories(r.Context(), store.HistoryDriver, []map[string]any{item}, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Driver update failed")
		return
	}
	item = items[0]
	writeOK(w, map[string]any{"message": "Driver updated successfully", "driver": item})
}

func (s *Server) handleDeleteDriver(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	if err := s.resourceService.DeleteOwned(r.Context(), driverStoreConfig(), id, user.ID); err != nil {
		writeError(w, errorStatus(err), "Driver not found")
		return
	}
	writeOK(w, map[string]any{"message": "Driver deleted successfully"})
}

func (s *Server) withResourceHistories(r *http.Request, kind store.StatusHistoryKind, items []map[string]any) ([]map[string]any, error) {
	return s.withResourceHistoriesMode(r, kind, items, true)
}

func (s *Server) withResourceHistoriesMode(r *http.Request, kind store.StatusHistoryKind, items []map[string]any, ensureToday bool) ([]map[string]any, error) {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, idFromItem(item))
		if ensureToday {
			_ = s.store.EnsureTodayStatusHistory(r.Context(), kind, idFromItem(item), statusFromItem(item))
		}
	}
	histories, err := s.store.StatusHistories(r.Context(), kind, ids)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		normalizeResourceDisplay(item)
		item["statusHistory"] = histories[idFromItem(item)]
	}
	return items, nil
}

func (s *Server) vehicleWithHistory(r *http.Request, item map[string]any) (map[string]any, error) {
	items, err := s.withResourceHistories(r, store.HistoryVehicle, []map[string]any{item})
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func (s *Server) driverWithHistory(r *http.Request, item map[string]any) (map[string]any, error) {
	items, err := s.withResourceHistories(r, store.HistoryDriver, []map[string]any{item})
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func (s *Server) vehicleWithHistoryNoEnsure(r *http.Request, item map[string]any) (map[string]any, error) {
	items, err := s.withResourceHistoriesMode(r, store.HistoryVehicle, []map[string]any{item}, false)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func (s *Server) driverWithHistoryNoEnsure(r *http.Request, item map[string]any) (map[string]any, error) {
	items, err := s.withResourceHistoriesMode(r, store.HistoryDriver, []map[string]any{item}, false)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

func requestedStatus(body map[string]any) (string, bool) {
	if v, ok := body["status"].(string); ok && v != "" {
		return displayStatusForHTTP(v), true
	}
	if v, ok := body["statuses"].(string); ok && v != "" {
		return displayStatusForHTTP(v), true
	}
	return "", false
}

func statusFromItem(item map[string]any) string {
	s, _ := item["status"].(string)
	return dbStatusForHTTP(s)
}
func displayStatusForHTTP(status string) string {
	if status == "Available" {
		return "Avaliable"
	}
	return status
}
func dbStatusForHTTP(status string) string {
	if status == "Available" {
		return "Avaliable"
	}
	return status
}

func normalizeResourceDisplay(item map[string]any) {
	item["status"] = displayStatusForHTTP(statusFromItem(item))
}

func timeFromItem(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case string:
		if v == "now()" {
			return time.Now().UTC()
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02", v); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}

func writeResourceUpdateError(w http.ResponseWriter, err error, resource string) {
	if errors.Is(err, store.ErrEmptyUpdate) {
		writeError(w, http.StatusBadRequest, "At least one field is required")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, resource+" not found")
		return
	}
	writeError(w, http.StatusInternalServerError, resource+" update failed")
}

func vehicleStoreConfig() store.CRUDConfig {
	return store.CRUDConfig{Table: `"Vehicle"`, SelectColumns: []string{"id", "vehicleNumber", "mtag", "vehicleType", "status", "statusDateTime", "createdAt", "updatedAt"}, CreateColumns: []store.Column{{JSON: "vehicleNumber", DB: "vehicleNumber", Required: true, String: true}, {JSON: "mtag", DB: "mtag", Default: "0", String: true}, {JSON: "vehicleType", DB: "vehicleType", Required: true, Cast: "VehicleType"}, {JSON: "status", DB: "status", Cast: "VehicleStatus", Default: "Avaliable"}, {JSON: "statusDateTime", DB: "statusDateTime", Default: "now()", DateTime: true}}, UpdateColumns: []store.Column{{JSON: "vehicleNumber", DB: "vehicleNumber", String: true}, {JSON: "mtag", DB: "mtag", String: true}, {JSON: "vehicleType", DB: "vehicleType", Cast: "VehicleType"}, {JSON: "status", DB: "status", Cast: "VehicleStatus"}, {JSON: "statusDateTime", DB: "statusDateTime", DateTime: true}}}
}

func driverStoreConfig() store.CRUDConfig {
	return store.CRUDConfig{Table: `"Driver"`, SelectColumns: []string{"id", "fullName", "phoneNumber", "driverLicenseNumber", "licenseType", "licenseExpiry", "status", "statusDateTime", "email", "address", "emergencyContactName", "emergencyContactPhone", "createdAt", "updatedAt"}, CreateColumns: []store.Column{{JSON: "fullName", DB: "fullName", Required: true}, {JSON: "phoneNumber", DB: "phoneNumber", Required: true}, {JSON: "driverLicenseNumber", DB: "driverLicenseNumber", Required: true}, {JSON: "licenseType", DB: "licenseType", Required: true, Cast: "LicenseType"}, {JSON: "licenseExpiry", DB: "licenseExpiry", Required: true, DateTime: true}, {JSON: "status", DB: "status", Cast: "DriverStatus", Default: "Avaliable"}, {JSON: "statusDateTime", DB: "statusDateTime", Default: "now()", DateTime: true}, {JSON: "email", DB: "email", Lowercase: true, EmptyStringWhenMissing: true}, {JSON: "address", DB: "address", Required: true}, {JSON: "emergencyContactName", DB: "emergencyContactName", Required: true}, {JSON: "emergencyContactPhone", DB: "emergencyContactPhone", Required: true}}, UpdateColumns: []store.Column{{JSON: "fullName", DB: "fullName"}, {JSON: "phoneNumber", DB: "phoneNumber"}, {JSON: "driverLicenseNumber", DB: "driverLicenseNumber"}, {JSON: "licenseType", DB: "licenseType", Cast: "LicenseType"}, {JSON: "licenseExpiry", DB: "licenseExpiry", DateTime: true}, {JSON: "status", DB: "status", Cast: "DriverStatus"}, {JSON: "statusDateTime", DB: "statusDateTime", DateTime: true}, {JSON: "email", DB: "email", Lowercase: true}, {JSON: "address", DB: "address"}, {JSON: "emergencyContactName", DB: "emergencyContactName"}, {JSON: "emergencyContactPhone", DB: "emergencyContactPhone"}}}
}
