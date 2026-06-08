package service

import (
	"context"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

// authUsers is the subset of *repository.UserRepository that AuthService
// depends on. Defining it as an interface lets tests swap in a fake.
type authUsers interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
}

// queueStore captures the slice of *repository.QueueRepository QueueService
// touches. Same rationale as authUsers — concrete repo satisfies it
// structurally so main.go wiring needs no change.
type queueStore interface {
	List(ctx context.Context, f repository.ListFilter) ([]domain.Queue, error)
	Get(ctx context.Context, id string) (*domain.Queue, error)
	Create(ctx context.Context, serviceType string) (*domain.Queue, error)
	CreateGuest(ctx context.Context, in domain.GuestInput) (*domain.Queue, error)
	GetByGuestToken(ctx context.Context, token string) (*domain.Queue, error)
	CallNext(ctx context.Context, counterID int, userID, serviceType string) (*domain.Queue, error)
	CounterStaffID(ctx context.Context, counterID int) (*string, error)
	UpdateStatus(ctx context.Context, id string, newStatus domain.QueueStatus, allowed []domain.QueueStatus) (*domain.Queue, error)
	Rate(ctx context.Context, id string, in domain.RateInput) (*domain.Queue, error)
}
