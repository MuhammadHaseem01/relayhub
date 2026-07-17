package router

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"cargonex-backend/src/controllers/crud_service/crud_service_impl"
	"cargonex-backend/src/database/store"
)

func registerMaintenanceUpdateAliases(mux *http.ServeMux, requireAuth authWrapper, cfg crudConfig) {
	storeCfg := store.CRUDConfig{Table: cfg.Table, SelectColumns: cfg.SelectColumns, CreateColumns: cfg.CreateColumns, UpdateColumns: cfg.UpdateColumns}
	service := crud_service_impl.NewCRUDService(crud_service_impl.NewCRUDServiceImpl{
		Store:  cfg.Store,
		Config: storeCfg,
	})
	handle := func(w http.ResponseWriter, r *http.Request, user currentUser, id int) {
		body, err := decodeJSON(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if writeValidationErrors(w, validateBody(body, cfg.UpdateColumns, validationUpdate, "id")) {
			return
		}
		item, err := service.UpdateOwned(r.Context(), id, user.ID, body)
		if err != nil {
			status := errorStatus(err)
			message := cfg.UpdateFailedMessage
			switch status {
			case http.StatusNotFound:
				message = cfg.NotFoundMessage
			case http.StatusBadRequest:
				message = "At least one field is required"
			}
			writeError(w, status, message)
			return
		}
		if cfg.AfterUpdate != nil {
			if err := cfg.AfterUpdate(r, user, item); err != nil {
				writeError(w, http.StatusInternalServerError, cfg.UpdateFailedMessage)
				return
			}
		}
		writeOK(w, map[string]any{"message": cfg.UpdateMessage, cfg.ItemKey: item})
	}
	mux.HandleFunc("PATCH /api/vehicle-maintenance/update/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request, user currentUser) {
		id, ok := parseID(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "Valid vehicle maintenance id is required")
			return
		}
		handle(w, r, user, id)
	}))
	mux.HandleFunc("PATCH /api/vehicle-maintenance/update", requireAuth(func(w http.ResponseWriter, r *http.Request, user currentUser) {
		body, err := decodeJSON(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		r.Body = newBody(body)
		idFloat, ok := body["id"].(float64)
		if !ok || idFloat <= 0 || idFloat != float64(int(idFloat)) {
			writeError(w, http.StatusBadRequest, "Valid vehicle maintenance id is required")
			return
		}
		handle(w, r, user, int(idFloat))
	}))
}

func newBody(body map[string]any) io.ReadCloser {
	raw, _ := json.Marshal(body)
	return io.NopCloser(bytes.NewReader(raw))
}
