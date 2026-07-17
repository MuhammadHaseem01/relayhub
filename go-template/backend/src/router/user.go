package router

import (
	"errors"
	"net/http"

	"cargonex-backend/src/database/store"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	body, err := decodeJSON(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if writeValidationErrors(w, validateLoginBody(body)) {
		return
	}
	email, _ := body["email"].(string)
	password, _ := body["password"].(string)
	if email == "" || password == "" {
		writeError(w, http.StatusBadRequest, "Email and password are required")
		return
	}
	result, err := s.userService.Login(r.Context(), email, password)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "User Not Found")
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "Your account is inactive. Please contact your administrator")
		case errors.Is(err, store.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "Invalid Credentials")
		default:
			writeError(w, http.StatusInternalServerError, "Login failed")
		}
		return
	}
	writeOK(w, result)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request, user currentUser) {
	body, err := decodeJSON(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if writeValidationErrors(w, validateBody(body, userCreateColumns(), validationCreate)) {
		return
	}
	result, err := s.userService.Register(r.Context(), body, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrConflict):
			writeError(w, http.StatusConflict, "Email already exists")
		case errors.Is(err, store.ErrBadRequest):
			writeError(w, http.StatusBadRequest, "organizationName is required")
		case errors.Is(err, store.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "Unauthorized")
		default:
			writeError(w, http.StatusInternalServerError, "User registration failed")
		}
		return
	}
	writeCreated(w, result)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request, user currentUser) {
	page, limit, ok := pagination(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid pagination query")
		return
	}
	result, err := s.userService.ListUsers(r.Context(), user.ID, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Request failed")
		return
	}
	writeOK(w, result)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	result, err := s.userService.GetUser(r.Context(), id, user.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	writeOK(w, result)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, user currentUser) {
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
	if writeValidationErrors(w, validateBody(body, userUpdateColumns(), validationUpdate)) {
		return
	}
	result, err := s.userService.UpdateUser(r.Context(), id, user.ID, body)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrEmptyUpdate):
			writeError(w, http.StatusBadRequest, "At least one field is required")
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "User not found")
		case errors.Is(err, store.ErrConflict):
			writeError(w, http.StatusConflict, "Email already exists")
		default:
			writeError(w, http.StatusInternalServerError, "User update failed")
		}
		return
	}
	writeOK(w, result)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, user currentUser) {
	id, ok := parseID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Validation failed (numeric string is expected)")
		return
	}
	result, err := s.userService.DeleteUser(r.Context(), id, user.ID)
	if errors.Is(err, store.ErrNotFound) {
		result = map[string]any{"message": "User deleted successfully"}
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "User deletion failed")
		return
	}
	writeOK(w, result)
}
