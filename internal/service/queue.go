package service

import (
	"context"
	"strings"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/sse"
)

type QueueService struct {
	repo   queueStore
	broker sse.Publisher
}

func NewQueueService(repo queueStore, broker sse.Publisher) *QueueService {
	return &QueueService{repo: repo, broker: broker}
}

func (s *QueueService) List(ctx context.Context, f repository.ListFilter) ([]domain.Queue, error) {
	return s.repo.List(ctx, f)
}

func (s *QueueService) Get(ctx context.Context, id string) (*domain.Queue, error) {
	return s.repo.Get(ctx, id)
}

func (s *QueueService) Create(ctx context.Context, serviceType string) (*domain.Queue, error) {
	serviceType = strings.ToUpper(strings.TrimSpace(serviceType))
	if serviceType == "" {
		return nil, domain.ErrInvalidInput
	}
	q, err := s.repo.Create(ctx, serviceType)
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.created", q)
	return q, nil
}

// CreateGuest issues a Buku Tamu ticket once the visitor submits the mobile
// form (name + purpose). The token correlates the kiosk session with the
// submission so the kiosk can pick up the assigned number.
func (s *QueueService) CreateGuest(ctx context.Context, in domain.GuestInput) (*domain.Queue, error) {
	in.ServiceType = strings.ToUpper(strings.TrimSpace(in.ServiceType))
	in.Token = strings.TrimSpace(in.Token)
	in.Name = strings.TrimSpace(in.Name)
	in.Purpose = strings.TrimSpace(in.Purpose)
	if in.ServiceType == "" || in.Token == "" || in.Name == "" || in.Purpose == "" {
		return nil, domain.ErrInvalidInput
	}
	if n := clipPtr(&in.Name, 120); n != nil {
		in.Name = *n
	}
	if p := clipPtr(&in.Purpose, 300); p != nil {
		in.Purpose = *p
	}

	q, err := s.repo.CreateGuest(ctx, in)
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.created", q)
	return q, nil
}

// GetByGuestToken returns the ticket linked to a guest session token, used by
// the kiosk to poll for the number after the visitor submits the form.
func (s *QueueService) GetByGuestToken(ctx context.Context, token string) (*domain.Queue, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.GetByGuestToken(ctx, token)
}

// CallNext picks the next waiting ticket for the given counter+staff.
// serviceType is optional; pass "" for "any service".
func (s *QueueService) CallNext(ctx context.Context, counterID int, userID, serviceType string) (*domain.Queue, error) {
	if counterID <= 0 {
		return nil, domain.ErrInvalidInput
	}
	if userID == "" {
		return nil, domain.ErrUnauthorized
	}
	if serviceType != "" {
		serviceType = strings.ToUpper(strings.TrimSpace(serviceType))
	}
	q, err := s.repo.CallNext(ctx, counterID, userID, serviceType)
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.called", q)
	return q, nil
}

// Recall re-emits the called event for the "Panggil Ulang" button. An
// already-calling/serving ticket is re-emitted without changing state; a
// skipped ticket is brought back into the "calling" state so it can be
// served again.
func (s *QueueService) Recall(ctx context.Context, id string) (*domain.Queue, error) {
	q, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	switch q.Status {
	case domain.StatusCalling, domain.StatusServing:
		// Already active: re-emit the called event without a state change.
	case domain.StatusSkipped:
		q, err = s.repo.UpdateStatus(ctx, id, domain.StatusCalling,
			[]domain.QueueStatus{domain.StatusSkipped})
		if err != nil {
			return nil, err
		}
	default:
		return nil, domain.ErrConflict
	}
	sse.PublishJSON(s.broker, "queue.called", q)
	return q, nil
}

// Complete marks the ticket completed. Allowed from "calling" or "serving"
// so frontends without an explicit start-serving step still work.
func (s *QueueService) Complete(ctx context.Context, id string) (*domain.Queue, error) {
	q, err := s.repo.UpdateStatus(ctx, id, domain.StatusCompleted,
		[]domain.QueueStatus{domain.StatusCalling, domain.StatusServing})
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.completed", q)
	return q, nil
}

// Skip marks a ticket skipped. Allowed from "waiting" or "calling".
func (s *QueueService) Skip(ctx context.Context, id string) (*domain.Queue, error) {
	q, err := s.repo.UpdateStatus(ctx, id, domain.StatusSkipped,
		[]domain.QueueStatus{domain.StatusWaiting, domain.StatusCalling})
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.skipped", q)
	return q, nil
}

// Rate stores SKM response (rating, optional feedback, optional identitas &
// issue_category) on a completed ticket.
func (s *QueueService) Rate(ctx context.Context, id string, in domain.RateInput) (*domain.Queue, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, domain.ErrInvalidInput
	}
	// Rating ≤ 3 must specify a valid issue category (selaras dengan UX).
	if in.Rating <= 3 {
		if in.IssueCategory == nil || !domain.ValidIssueCategories[*in.IssueCategory] {
			return nil, domain.ErrInvalidInput
		}
	} else if in.IssueCategory != nil && !domain.ValidIssueCategories[*in.IssueCategory] {
		return nil, domain.ErrInvalidInput
	}
	in.RespondentName = clipPtr(in.RespondentName, 80)
	in.RespondentPhone = clipPtr(in.RespondentPhone, 20)
	in.Feedback = clipPtr(in.Feedback, 500)

	q, err := s.repo.Rate(ctx, id, in)
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.rated", q)
	return q, nil
}

func clipPtr(s *string, max int) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	if len(v) > max {
		v = v[:max]
	}
	return &v
}
