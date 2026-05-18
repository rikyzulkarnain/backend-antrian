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
	if _, ok := domain.ServicePrefix(serviceType); !ok {
		return nil, domain.ErrInvalidInput
	}
	q, err := s.repo.Create(ctx, serviceType)
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.created", q)
	return q, nil
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
		if _, ok := domain.ServicePrefix(serviceType); !ok {
			return nil, domain.ErrInvalidInput
		}
	}
	q, err := s.repo.CallNext(ctx, counterID, userID, serviceType)
	if err != nil {
		return nil, err
	}
	sse.PublishJSON(s.broker, "queue.called", q)
	return q, nil
}

// Recall re-emits the called event for an already-calling/serving ticket
// without changing state. Used by the "Panggil Ulang" button.
func (s *QueueService) Recall(ctx context.Context, id string) (*domain.Queue, error) {
	q, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if q.Status != domain.StatusCalling && q.Status != domain.StatusServing {
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
