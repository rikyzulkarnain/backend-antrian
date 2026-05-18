package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func ok(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{Data: data})
}

func created(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, envelope{Data: data})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, envelope{Error: &apiError{Code: code, Message: msg}})
}

func badRequest(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusBadRequest, "BAD_REQUEST", msg)
}

func notFound(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusNotFound, "NOT_FOUND", msg)
}

func unauthorized(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", msg)
}

func forbidden(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusForbidden, "FORBIDDEN", msg)
}

func conflict(w http.ResponseWriter, code, msg string) {
	writeError(w, http.StatusConflict, code, msg)
}

func internalError(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusInternalServerError, "INTERNAL", msg)
}

// renderDomainError maps a domain error to an appropriate HTTP response.
// Falls back to 500 for unknown errors and logs the cause.
func renderDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		notFound(w, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		conflict(w, "ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		unauthorized(w, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Password saat ini salah.")
	case errors.Is(err, domain.ErrForbidden):
		forbidden(w, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		badRequest(w, err.Error())
	case errors.Is(err, domain.ErrConflict):
		conflict(w, "CONFLICT", err.Error())
	default:
		slog.Error("unhandled error", "err", err)
		internalError(w, "Terjadi kesalahan tak terduga")
	}
}
