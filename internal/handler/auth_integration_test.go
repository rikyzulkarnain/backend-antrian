package handler_test

import (
	"net/http"
	"testing"
)

type loginResp struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
	AccessToken string `json:"access_token"`
}

func TestAuthHandler_Login_Success_SetsCookie(t *testing.T) {
	h := newIntegration(t)

	resp := h.do("POST", "/auth/login", map[string]string{
		"email": "admin@local", "password": "admin123",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	body := decodeData[loginResp](t, resp)
	if body.User.Email != "admin@local" {
		t.Errorf("user.email = %q", body.User.Email)
	}
	if body.User.Role != "admin" {
		t.Errorf("user.role = %q", body.User.Role)
	}
	if body.AccessToken == "" {
		t.Error("access_token missing")
	}

	// Refresh token must come back as an HttpOnly cookie scoped to /api/v1/auth.
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "refresh_token" {
			found = true
			if !c.HttpOnly {
				t.Error("refresh cookie must be HttpOnly")
			}
			if c.Path != "/api/v1/auth" {
				t.Errorf("refresh cookie path = %q", c.Path)
			}
			if c.Value == "" {
				t.Error("refresh cookie value empty")
			}
		}
	}
	if !found {
		t.Error("refresh_token cookie not set")
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	h := newIntegration(t)
	resp := h.do("POST", "/auth/login", map[string]string{
		"email": "admin@local", "password": "WRONG",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthHandler_Login_UnknownEmail(t *testing.T) {
	h := newIntegration(t)
	resp := h.do("POST", "/auth/login", map[string]string{
		"email": "nobody@nowhere", "password": "x",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthHandler_Me_RequiresBearer(t *testing.T) {
	h := newIntegration(t)
	resp := h.do("GET", "/auth/me", nil, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuthHandler_Me_WithBearerReturnsUser(t *testing.T) {
	h := newIntegration(t)
	login := decodeData[loginResp](t, h.do("POST", "/auth/login", map[string]string{
		"email": "admin@local", "password": "admin123",
	}, ""))

	resp := h.do("GET", "/auth/me", nil, login.AccessToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	type meBody struct {
		ID, Email, Role string
	}
	me := decodeData[meBody](t, resp)
	if me.Email != "admin@local" {
		t.Errorf("me.email = %q", me.Email)
	}
}

func TestAuthHandler_AdminEndpointRejectsStaffRole(t *testing.T) {
	h := newIntegration(t)
	staff := decodeData[loginResp](t, h.do("POST", "/auth/login", map[string]string{
		"email": "staff@local", "password": "staff123",
	}, ""))

	// /users is admin-only — staff bearer should get 403, not 401.
	resp := h.do("GET", "/users", nil, staff.AccessToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
}
