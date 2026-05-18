package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

// fakeUsers is an in-memory authUsers used by the AuthService tests.
type fakeUsers struct {
	byEmail map[string]*domain.User
	byID    map[string]*domain.User
}

func newFakeUsers(users ...*domain.User) *fakeUsers {
	f := &fakeUsers{
		byEmail: make(map[string]*domain.User, len(users)),
		byID:    make(map[string]*domain.User, len(users)),
	}
	for _, u := range users {
		f.byEmail[u.Email] = u
		f.byID[u.ID] = u
	}
	return f
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) FindByID(_ context.Context, id string) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func mustHash(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return string(h)
}

func newAuthSvc(t *testing.T, users ...*domain.User) *AuthService {
	t.Helper()
	return NewAuthService(newFakeUsers(users...), []byte("test-secret"))
}

func TestAuthService_Login_Success(t *testing.T) {
	u := &domain.User{
		ID: "u1", Email: "admin@local", Name: "Admin",
		PasswordHash: mustHash(t, "secret123"),
		Role:         domain.RoleAdmin, IsActive: true,
	}
	svc := newAuthSvc(t, u)

	sess, err := svc.Login(context.Background(), "admin@local", "secret123")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if sess.User.ID != u.ID {
		t.Errorf("user mismatch: %s", sess.User.ID)
	}
	if sess.AccessToken == "" || sess.RefreshToken == "" {
		t.Error("expected both access and refresh tokens")
	}
}

func TestAuthService_Login_EmailTrimmedAndLowercased(t *testing.T) {
	u := &domain.User{
		ID: "u1", Email: "admin@local",
		PasswordHash: mustHash(t, "x"),
		Role:         domain.RoleAdmin, IsActive: true,
	}
	svc := newAuthSvc(t, u)

	if _, err := svc.Login(context.Background(), "  Admin@Local  ", "x"); err != nil {
		t.Errorf("expected normalization to succeed, got %v", err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	u := &domain.User{
		ID: "u1", Email: "a@b", PasswordHash: mustHash(t, "good"),
		Role: domain.RoleStaff, IsActive: true,
	}
	svc := newAuthSvc(t, u)

	_, err := svc.Login(context.Background(), "a@b", "bad")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	svc := newAuthSvc(t)
	_, err := svc.Login(context.Background(), "missing@x", "x")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestAuthService_Login_InactiveUserRejected(t *testing.T) {
	u := &domain.User{
		ID: "u1", Email: "a@b", PasswordHash: mustHash(t, "x"),
		Role: domain.RoleStaff, IsActive: false,
	}
	svc := newAuthSvc(t, u)
	_, err := svc.Login(context.Background(), "a@b", "x")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("want ErrForbidden for inactive user, got %v", err)
	}
}

func TestAuthService_Login_BlankInputsRejected(t *testing.T) {
	svc := newAuthSvc(t)
	_, err := svc.Login(context.Background(), "", "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
}

func TestAuthService_Refresh_Success(t *testing.T) {
	u := &domain.User{
		ID: "u1", Email: "a@b", PasswordHash: mustHash(t, "x"),
		Role: domain.RoleStaff, IsActive: true,
	}
	svc := newAuthSvc(t, u)

	first, err := svc.Login(context.Background(), "a@b", "x")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	sess, err := svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if sess.User.ID != u.ID {
		t.Errorf("user mismatch")
	}
	if sess.AccessToken == "" || sess.RefreshToken == "" {
		t.Error("expected new tokens")
	}
}

func TestAuthService_Refresh_RejectsAccessToken(t *testing.T) {
	u := &domain.User{
		ID: "u1", Email: "a@b", PasswordHash: mustHash(t, "x"),
		Role: domain.RoleStaff, IsActive: true,
	}
	svc := newAuthSvc(t, u)
	first, err := svc.Login(context.Background(), "a@b", "x")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	// Access token should NOT be usable as refresh token.
	_, err = svc.Refresh(context.Background(), first.AccessToken)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized for access-as-refresh, got %v", err)
	}
}

func TestAuthService_Refresh_RejectsExpiredToken(t *testing.T) {
	svc := newAuthSvc(t, &domain.User{
		ID: "u1", Email: "a@b", PasswordHash: mustHash(t, "x"),
		Role: domain.RoleStaff, IsActive: true,
	})
	// Hand-craft an expired refresh token.
	claims := jwt.MapClaims{
		"sub":  "u1",
		"type": "refresh",
		"iat":  time.Now().Add(-2 * time.Hour).Unix(),
		"exp":  time.Now().Add(-1 * time.Hour).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	_, err = svc.Refresh(context.Background(), tok)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized for expired, got %v", err)
	}
}

func TestAuthService_Refresh_RejectsBadSignature(t *testing.T) {
	svc := newAuthSvc(t, &domain.User{
		ID: "u1", Email: "a@b", PasswordHash: mustHash(t, "x"),
		Role: domain.RoleStaff, IsActive: true,
	})
	claims := jwt.MapClaims{
		"sub": "u1", "type": "refresh",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("WRONG-secret"))
	_, err := svc.Refresh(context.Background(), tok)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized for bad signature, got %v", err)
	}
}

func TestAuthService_Refresh_RejectsBlank(t *testing.T) {
	svc := newAuthSvc(t)
	_, err := svc.Refresh(context.Background(), "")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized for blank, got %v", err)
	}
}

func TestAuthService_Me_ReturnsUser(t *testing.T) {
	u := &domain.User{
		ID: "u1", Email: "a@b", PasswordHash: "x",
		Role: domain.RoleAdmin, IsActive: true,
	}
	svc := newAuthSvc(t, u)
	got, err := svc.Me(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("got id %q", got.ID)
	}
}

func TestAuthService_Me_BlankIDRejected(t *testing.T) {
	svc := newAuthSvc(t)
	_, err := svc.Me(context.Background(), "")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}
