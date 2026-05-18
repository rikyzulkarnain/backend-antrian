package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

type CounterHandler struct {
	svc *service.CounterService
}

func NewCounterHandler(svc *service.CounterService) *CounterHandler {
	return &CounterHandler{svc: svc}
}

func (h *CounterHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, items)
}

func (h *CounterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		badRequest(w, "ID loket tidak valid")
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, c)
}

// counterBody intentionally mirrors the frontend payload (active vs is_active,
// service vs service_type). Pointers preserve missing-vs-null semantics for PATCH.
type counterBody struct {
	Name    *string  `json:"name"`
	Service *string  `json:"service"`
	Active  *bool    `json:"active"`
	StaffID *string  `json:"staff_id"`
	hasSet  struct { // populated by UnmarshalJSON
		service bool
		staff   bool
	} `json:"-"`
}

func (b *counterBody) UnmarshalJSON(data []byte) error {
	type rawT struct {
		Name    *string         `json:"name"`
		Service json.RawMessage `json:"service"`
		Active  *bool           `json:"active"`
		StaffID json.RawMessage `json:"staff_id"`
	}
	var raw rawT
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.Name = raw.Name
	b.Active = raw.Active
	if len(raw.Service) > 0 {
		b.hasSet.service = true
		if string(raw.Service) != "null" {
			var s string
			if err := json.Unmarshal(raw.Service, &s); err != nil {
				return err
			}
			b.Service = &s
		}
	}
	if len(raw.StaffID) > 0 {
		b.hasSet.staff = true
		if string(raw.StaffID) != "null" {
			var s string
			if err := json.Unmarshal(raw.StaffID, &s); err != nil {
				return err
			}
			b.StaffID = &s
		}
	}
	return nil
}

func (h *CounterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body counterBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	if body.Name == nil {
		badRequest(w, "Field name wajib diisi")
		return
	}
	c, err := h.svc.Create(r.Context(), service.CounterInput{
		Name: *body.Name, Service: body.Service, Active: body.Active, StaffID: body.StaffID,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	created(w, c)
}

func (h *CounterHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		badRequest(w, "ID loket tidak valid")
		return
	}
	var body counterBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	c, err := h.svc.Update(r.Context(), id, service.CounterPatchInput{
		Name:         body.Name,
		Service:      body.Service,
		ClearService: body.hasSet.service && body.Service == nil,
		Active:       body.Active,
		StaffID:      body.StaffID,
		ClearStaff:   body.hasSet.staff && body.StaffID == nil,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, c)
}

func (h *CounterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		badRequest(w, "ID loket tidak valid")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, map[string]bool{"ok": true})
}
