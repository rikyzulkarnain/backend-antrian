package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
)

type AuthService struct {
	users     authUsers
	jwtSecret []byte
}

func NewAuthService(users authUsers, jwtSecret []byte) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

// Session is the result of a successful login or refresh.
type Session struct {
	User         *domain.User
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil, domain.ErrInvalidInput
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if !u.IsActive {
		return nil, domain.ErrForbidden
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	return s.issueSession(u)
}

// Refresh validates a refresh token and issues a fresh access+refresh pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	if refreshToken == "" {
		return nil, domain.ErrUnauthorized
	}
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(refreshToken, claims, func(_ *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil || !tok.Valid {
		return nil, domain.ErrUnauthorized
	}
	if t, _ := claims["type"].(string); t != "refresh" {
		return nil, domain.ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, domain.ErrUnauthorized
	}

	u, err := s.users.FindByID(ctx, sub)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if !u.IsActive {
		return nil, domain.ErrForbidden
	}
	return s.issueSession(u)
}

// Me returns the user for the given id; used by /auth/me.
func (s *AuthService) Me(ctx context.Context, userID string) (*domain.User, error) {
	if userID == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.users.FindByID(ctx, userID)
}

func (s *AuthService) issueSession(u *domain.User) (*Session, error) {
	access, err := s.signToken(u, "access", accessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.signToken(u, "refresh", refreshTokenTTL)
	if err != nil {
		return nil, err
	}
	return &Session{User: u, AccessToken: access, RefreshToken: refresh}, nil
}

func (s *AuthService) signToken(u *domain.User, kind string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   u.ID,
		"email": u.Email,
		"role":  string(u.Role),
		"type":  kind,
		"iat":   now.Unix(),
		"exp":   now.Add(ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// RefreshTokenTTL exposes the TTL so handlers can match cookie max-age.
func (s *AuthService) RefreshTokenTTL() time.Duration { return refreshTokenTTL }
