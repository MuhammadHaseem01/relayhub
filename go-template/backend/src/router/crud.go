package router

import (
	"fmt"
	"net/http"

	"cargonex-backend/src/controllers/crud_service/crud_service_impl"
	"cargonex-backend/src/database/store"
)

type authWrapper func(func(http.ResponseWriter, *http.Request, currentUser)) http.HandlerFunc

type crudConfig struct {
	CollectionKey       string
	ItemKey             string
	Table               string
	CreateMessage       string
	UpdateMessage       string
	DeleteMessage       string
	ListMessage         string
	IncludeData         bool
	NotFoundMessage     string
	CreateFailedMessage string
	UpdateFailedMessage string
	DeleteFailedMessage string
	CreateColumns       []store.Column
	UpdateColumns       []store.Column
	SelectColumns       []string
	Store               *store.Store
	BeforeCreate        func(*http.Request, currentUser, map[string]any) error
	AfterCreate         func(*http.Request, currentUser, map[string]any) error
	AfterUpdate         func(*http.Request, currentUser, map[string]any) error
	AfterDelete         func(*http.Request, currentUser, int) error
	DisableUpdate       bool
	DisableDelete       bool
	TransformList       func(*http.Request, currentUser, []map[string]any) ([]map[string]any, error)
}

func registerCRUD(mux *http.ServeMux, base string, requireAuth authWrapper, cfg crudConfig) {
	storeCfg := store.CRUDConfig{Table: cfg.Table, SelectColumns: cfg.SelectColumns, CreateColumns: cfg.CreateColumns, UpdateColumns: cfg.UpdateColumns}
	service := crud_service_impl.NewCRUDService(crud_service_impl.NewCRUDServiceImpl{
		Store:  cfg.Store,
		Config: storeCfg,
	})

	mux.HandleFunc("POST "+base+"/add", requireAuth(func(w http.ResponseWriter, r *http.Request, user currentUser) {
		body, err := decodeJSON(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if cfg.BeforeCreate != nil {
			if err := cfg.BeforeCreate(r, user, body); err != nil {
				writeError(w, http.StatusInternalServerError, cfg.CreateFailedMessage)
				return
			}
		}
		if writeValidationErrors(w, validateBody(body, cfg.CreateColumns, validationCreate)) {
			return
		}
		item, err := service.CreateOwned(r.Context(), body, user.ID)
		if err != nil {
			status := errorStatus(err)
			message := cfg.CreateFailedMessage
			if status == http.StatusBadRequest {
				message = "Invalid request body"
			}
			writeError(w, status, message)
			return
		}
		if cfg.AfterCreate != nil {
			if err := cfg.AfterCreate(r, user, item); err != nil {
				writeError(w, http.StatusInternalServerError, cfg.CreateFailedMessage)
				return
			}
		}
		writeCreated(w, map[string]any{"message": cfg.CreateMessage, cfg.ItemKey: item})
	}))

	mux.HandleFunc("GET "+base, requireAuth(func(w http.ResponseWriter, r *http.Request, user currentUser) {
		page, limit, ok := pagination(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "Invalid pagination query")
			return
		}
		items, meta, err := service.ListOwned(r.Context(), user.ID, page, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Request failed")
			return
		}
		if cfg.TransformList != nil {
			items, err = cfg.TransformList(r, user, items)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Request failed")
				return
			}
		}
		response := map[string]any{cfg.CollectionKey: items, "meta": meta}
		if cfg.IncludeData {
			response["data"] = items
		}
		if cfg.ListMessage != "" {
			response["message"] = cfg.ListMessage
		}
		writeOK(w, response)
	}))

	mux.HandleFunc("GET "+base+"/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request, user currentUser) {
		id, ok := parseID(r)
		if !ok {
			writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
			return
		}
		item, err := service.GetOwned(r.Context(), id, user.ID)
		if err != nil {
			writeError(w, errorStatus(err), cfg.NotFoundMessage)
			return
		}
		writeOK(w, item)
	}))

	if !cfg.DisableUpdate {
		mux.HandleFunc("PATCH "+base+"/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request, user currentUser) {
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
			if writeValidationErrors(w, validateBody(body, cfg.UpdateColumns, validationUpdate)) {
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
		}))
	}

	if !cfg.DisableDelete {
		mux.HandleFunc("DELETE "+base+"/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request, user currentUser) {
			id, ok := parseID(r)
			if !ok {
				writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
				return
			}
			if err := service.DeleteOwned(r.Context(), id, user.ID); err != nil {
				message := cfg.DeleteFailedMessage
				if errorStatus(err) == http.StatusNotFound {
					message = cfg.NotFoundMessage
				}
				writeError(w, errorStatus(err), message)
				return
			}
			if cfg.AfterDelete != nil {
				if err := cfg.AfterDelete(r, user, id); err != nil {
					writeError(w, http.StatusInternalServerError, cfg.DeleteFailedMessage)
					return
				}
			}
			writeOK(w, map[string]any{"message": cfg.DeleteMessage})
		}))
	}

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
