package handler

import (
	"net/http"
	"strconv"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

type AnalyticsHandler struct {
	svc *service.AnalyticsService
}

func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

func (h *AnalyticsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Overview(r.Context())
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}

func (h *AnalyticsHandler) ByService(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ByService(r.Context())
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}

func (h *AnalyticsHandler) ByHour(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.ByHour(r.Context())
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}

func (h *AnalyticsHandler) Staff(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Staff(r.Context())
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}

func (h *AnalyticsHandler) Issues(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Issues(r.Context())
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}

func (h *AnalyticsHandler) Ratings(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := h.svc.Ratings(r.Context(), limit)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}
