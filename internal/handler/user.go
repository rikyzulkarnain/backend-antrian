package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, items)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, u)
}

type userBody struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body userBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	if body.Name == nil || body.Email == nil || body.Password == nil || body.Role == nil {
		badRequest(w, "Field name, email, password, role wajib diisi")
		return
	}
	u, err := h.svc.Create(r.Context(), service.UserInput{
		Name: *body.Name, Email: *body.Email, Password: *body.Password,
		Role: *body.Role, IsActive: body.IsActive,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	created(w, u)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body userBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	u, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), service.UserPatchInput{
		Name: body.Name, Email: body.Email, Password: body.Password,
		Role: body.Role, IsActive: body.IsActive,
	})
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, u)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, map[string]bool{"ok": true})
}
