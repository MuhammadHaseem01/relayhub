package router

import (
	"errors"
	"net/http"

	"cargonex-backend/src/database/store"
)

func (s *Server) handleListLedgers(w http.ResponseWriter, r *http.Request, user currentUser) {
	query, ok := ledgerQuery(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid ledger query")
		return
	}
	result, err := s.ledgerService.ListLedgers(r.Context(), query, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Request failed")
		return
	}
	writeOK(w, result)
}

func (s *Server) handleGetLedger(w http.ResponseWriter, r *http.Request, user currentUser) {
	id := r.PathValue("id")
	if id == "undefined" || id == "null" {
		s.handleListLedgers(w, r, user)
		return
	}
	result, err := s.ledgerService.GetLedger(r.Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Ledger not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Request failed")
		return
	}
	writeOK(w, result)
}

func ledgerQuery(r *http.Request) (store.LedgerQuery, bool) {
	page, limit, ok := pagination(r)
	if !ok {
		return store.LedgerQuery{}, false
	}
	q := r.URL.Query()
	return store.LedgerQuery{
		Page:          page,
		Limit:         limit,
		SourceType:    q.Get("sourceType"),
		PaymentStatus: q.Get("paymentStatus"),
		ReferenceType: q.Get("referenceType"),
		DateFrom:      q.Get("dateFrom"),
		DateTo:        q.Get("dateTo"),
	}, true
}
