package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/middleware"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

type QueueHandler struct {
	svc *service.QueueService
}

func NewQueueHandler(svc *service.QueueService) *QueueHandler {
	return &QueueHandler{svc: svc}
}

func (h *QueueHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repository.ListFilter{
		Status:      q.Get("status"),
		ServiceType: q.Get("service_type"),
		Active:      q.Get("active") == "true",
	}
	items, err := h.svc.List(r.Context(), f)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, items)
}

func (h *QueueHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	q, err := h.svc.Get(r.Context(), id)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, q)
}

type createQueueRequest struct {
	ServiceType string `json:"service_type"`
}

func (h *QueueHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	q, err := h.svc.Create(r.Context(), body.ServiceType)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	created(w, q)
}

type createGuestRequest struct {
	ServiceType string `json:"service_type"`
	Token       string `json:"token"`
	Name        string `json:"name"`
	Purpose     string `json:"purpose"`
}

// CreateGuest handles a Buku Tamu form submission from the visitor's phone.
func (h *QueueHandler) CreateGuest(w http.ResponseWriter, r *http.Request) {
	var body createGuestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	q, err := h.svc.CreateGuest(r.Context(), domain.GuestInput{
		ServiceType: body.ServiceType,
		Token:       body.Token,
		Name:        body.Name,
		Purpose:     body.Purpose,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	created(w, q)
}

// GetGuest lets the kiosk poll for the ticket assigned to a guest session
// token. Returns 404 until the visitor submits the form.
func (h *QueueHandler) GetGuest(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	q, err := h.svc.GetByGuestToken(r.Context(), token)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, q)
}

type callQueueRequest struct {
	CounterID   int    `json:"counter_id"`
	ServiceType string `json:"service_type"`
}

func (h *QueueHandler) Call(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		unauthorized(w, "Sesi tidak valid")
		return
	}

	var body callQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	q, err := h.svc.CallNext(r.Context(), body.CounterID, claims.UserID, body.ServiceType)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, q)
}

func (h *QueueHandler) Recall(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	q, err := h.svc.Recall(r.Context(), id)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, q)
}

func (h *QueueHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	q, err := h.svc.Complete(r.Context(), id)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, q)
}

func (h *QueueHandler) Skip(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	q, err := h.svc.Skip(r.Context(), id)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, q)
}

type rateRequest struct {
	Rating          int     `json:"rating"`
	Feedback        *string `json:"feedback"`
	Comment         *string `json:"comment"` // alias lama dari frontend rating
	RespondentName  *string `json:"respondent_name"`
	RespondentPhone *string `json:"respondent_phone"`
	IssueCategory   *string `json:"issue_category"`
}

func (h *QueueHandler) Rate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body rateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	feedback := body.Feedback
	if feedback == nil {
		feedback = body.Comment
	}
	q, err := h.svc.Rate(r.Context(), id, domain.RateInput{
		Rating:          body.Rating,
		Feedback:        feedback,
		RespondentName:  body.RespondentName,
		RespondentPhone: body.RespondentPhone,
		IssueCategory:   body.IssueCategory,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, q)
}
