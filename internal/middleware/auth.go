package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const claimsContextKey ctxKey = "claims"

// Claims is the subset of JWT claims that the rest of the app consumes.
type Claims struct {
	UserID string
	Role   string
	Email  string
	Type   string // "access" or "refresh"
}

func JWT(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeUnauthorized(w)
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims := jwt.MapClaims{}
			tok, err := jwt.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (any, error) {
				return secret, nil
			})
			if err != nil || !tok.Valid {
				writeUnauthorized(w)
				return
			}

			c := mapToClaims(claims)
			if c.Type != "" && c.Type != "access" {
				writeUnauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), claimsContextKey, c)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole wraps a handler so only the listed roles may access it.
// Must run after JWT.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := ClaimsFromContext(r.Context())
			if !ok {
				writeUnauthorized(w)
				return
			}
			if _, allow := allowed[c.Role]; !allow {
				writeForbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext returns the Claims attached by JWT middleware, if any.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	v, ok := ctx.Value(claimsContextKey).(Claims)
	if !ok {
		return nil, false
	}
	return &v, true
}

func mapToClaims(m jwt.MapClaims) Claims {
	get := func(k string) string {
		if v, ok := m[k].(string); ok {
			return v
		}
		return ""
	}
	return Claims{
		UserID: firstNonEmpty(get("sub"), get("user_id")),
		Role:   get("role"),
		Email:  get("email"),
		Type:   get("type"),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Token tidak valid"}}`))
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":{"code":"FORBIDDEN","message":"Akses ditolak"}}`))
}
