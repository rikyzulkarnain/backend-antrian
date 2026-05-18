package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/middleware"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

type VideoHandler struct {
	svc *service.VideoService
}

func NewVideoHandler(svc *service.VideoService) *VideoHandler {
	return &VideoHandler{svc: svc}
}

func (h *VideoHandler) List(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	items, err := h.svc.List(r.Context(), target)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, items)
}

func (h *VideoHandler) Get(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, v)
}

type videoBody struct {
	Title           *string `json:"title"`
	URL             *string `json:"url"`
	TargetScreen    *string `json:"target_screen"`
	DurationSeconds *int    `json:"duration_seconds"`
	DisplayOrder    *int    `json:"display_order"`
	IsActive        *bool   `json:"is_active"`

	rawDuration json.RawMessage
}

func (b *videoBody) UnmarshalJSON(data []byte) error {
	type rawT struct {
		Title           *string         `json:"title"`
		URL             *string         `json:"url"`
		TargetScreen    *string         `json:"target_screen"`
		DurationSeconds json.RawMessage `json:"duration_seconds"`
		DisplayOrder    *int            `json:"display_order"`
		IsActive        *bool           `json:"is_active"`
	}
	var raw rawT
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.Title = raw.Title
	b.URL = raw.URL
	b.TargetScreen = raw.TargetScreen
	b.DisplayOrder = raw.DisplayOrder
	b.IsActive = raw.IsActive
	if len(raw.DurationSeconds) > 0 {
		b.rawDuration = raw.DurationSeconds
		if string(raw.DurationSeconds) != "null" {
			var d int
			if err := json.Unmarshal(raw.DurationSeconds, &d); err != nil {
				return err
			}
			b.DurationSeconds = &d
		}
	}
	return nil
}

func (h *VideoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body videoBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	if body.Title == nil || body.URL == nil || body.TargetScreen == nil {
		badRequest(w, "Field title, url, target_screen wajib diisi")
		return
	}
	var uploadedBy *string
	if c, ok := middleware.ClaimsFromContext(r.Context()); ok && c.UserID != "" {
		id := c.UserID
		uploadedBy = &id
	}
	v, err := h.svc.Create(r.Context(), service.VideoInput{
		Title: *body.Title, URL: *body.URL, TargetScreen: *body.TargetScreen,
		DurationSeconds: body.DurationSeconds, DisplayOrder: body.DisplayOrder,
		IsActive: body.IsActive, UploadedBy: uploadedBy,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	created(w, v)
}

func (h *VideoHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body videoBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	v, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), service.VideoPatchInput{
		Title:           body.Title,
		URL:             body.URL,
		TargetScreen:    body.TargetScreen,
		DurationSeconds: body.DurationSeconds,
		ClearDuration:   len(body.rawDuration) > 0 && body.DurationSeconds == nil,
		DisplayOrder:    body.DisplayOrder,
		IsActive:        body.IsActive,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, v)
}

func (h *VideoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, map[string]bool{"ok": true})
}

func (h *VideoHandler) UploadSignature(w http.ResponseWriter, _ *http.Request) {
	sig, err := h.svc.UploadSignature()
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, sig)
}
