package service

import (
	"context"
	"strings"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

type CounterService struct {
	repo *repository.CounterRepository
}

func NewCounterService(repo *repository.CounterRepository) *CounterService {
	return &CounterService{repo: repo}
}

func (s *CounterService) List(ctx context.Context) ([]domain.Counter, error) {
	return s.repo.List(ctx)
}

func (s *CounterService) Get(ctx context.Context, id int) (*domain.Counter, error) {
	return s.repo.Get(ctx, id)
}

type CounterInput struct {
	Name    string
	Service *string
	Active  *bool
	StaffID *string
}

func (s *CounterService) Create(ctx context.Context, in CounterInput) (*domain.Counter, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	svc, err := normalizeService(in.Service)
	if err != nil {
		return nil, err
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	return s.repo.Create(ctx, repository.CounterFields{
		Name: name, Service: svc, Active: active, StaffID: in.StaffID,
	})
}

type CounterPatchInput struct {
	Name         *string
	Service      *string
	ClearService bool
	Active       *bool
	StaffID      *string
	ClearStaff   bool
}

func (s *CounterService) Update(ctx context.Context, id int, in CounterPatchInput) (*domain.Counter, error) {
	patch := repository.CounterPatch{
		Active:       in.Active,
		ClearService: in.ClearService,
		StaffID:      in.StaffID,
		ClearStaff:   in.ClearStaff,
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.Name = &name
	}
	if in.Service != nil && !in.ClearService {
		svc, err := normalizeService(in.Service)
		if err != nil {
			return nil, err
		}
		patch.Service = svc
	}
	return s.repo.Update(ctx, id, patch)
}

func (s *CounterService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func normalizeService(s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.ToUpper(strings.TrimSpace(*s))
	if v == "" {
		return nil, nil
	}
	return &v, nil
}
