package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/middleware"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

const refreshCookieName = "refresh_token"

type AuthHandler struct {
	svc        *service.AuthService
	userSvc    *service.UserService
	cookiePath string
	secure     bool
}

func NewAuthHandler(svc *service.AuthService, userSvc *service.UserService, secure bool) *AuthHandler {
	return &AuthHandler{svc: svc, userSvc: userSvc, cookiePath: "/api/v1/auth", secure: secure}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User        *domain.User `json:"user"`
	AccessToken string       `json:"access_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	sess, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	h.setRefreshCookie(w, sess.RefreshToken)
	ok(w, authResponse{User: sess.User, AccessToken: sess.AccessToken})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		unauthorized(w, "Sesi tidak ditemukan")
		return
	}
	sess, err := h.svc.Refresh(r.Context(), c.Value)
	if err != nil {
		h.clearRefreshCookie(w)
		renderDomainError(w, err)
		return
	}
	h.setRefreshCookie(w, sess.RefreshToken)
	ok(w, authResponse{User: sess.User, AccessToken: sess.AccessToken})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	h.clearRefreshCookie(w)
	ok(w, map[string]bool{"ok": true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok2 := middleware.ClaimsFromContext(r.Context())
	if !ok2 || claims == nil {
		unauthorized(w, "Sesi tidak valid")
		return
	}
	u, err := h.svc.Me(r.Context(), claims.UserID)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, u)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok2 := middleware.ClaimsFromContext(r.Context())
	if !ok2 || claims == nil || claims.UserID == "" {
		unauthorized(w, "Sesi tidak valid")
		return
	}
	var body changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	if err := h.userSvc.ChangePassword(r.Context(), claims.UserID, body.CurrentPassword, body.NewPassword); err != nil {
		renderDomainError(w, err)
		return
	}
	h.clearRefreshCookie(w)
	ok(w, map[string]bool{"ok": true})
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     h.cookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: h.sameSite(),
		MaxAge:   int(h.svc.RefreshTokenTTL().Seconds()),
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     h.cookiePath,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: h.sameSite(),
		MaxAge:   -1,
	})
}

// sameSite picks the cookie SameSite policy. In production the frontend
// (Vercel) and backend (Railway) live on different domains, so the refresh
// cookie is sent on cross-site requests only with SameSite=None — which the
// browser additionally requires to be Secure. Locally everything is same-site
// over http://localhost, where Lax works and None would be dropped (no HTTPS).
func (h *AuthHandler) sameSite() http.SameSite {
	if h.secure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}
