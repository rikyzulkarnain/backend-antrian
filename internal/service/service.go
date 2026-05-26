package service

import (
	"context"
	"strings"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

type ServiceService struct {
	repo *repository.ServiceRepository
}

func NewServiceService(r *repository.ServiceRepository) *ServiceService {
	return &ServiceService{repo: r}
}

func (s *ServiceService) List(ctx context.Context, activeOnly bool) ([]domain.Service, error) {
	return s.repo.List(ctx, activeOnly)
}

func (s *ServiceService) Get(ctx context.Context, key string) (*domain.Service, error) {
	return s.repo.Get(ctx, key)
}

type ServicePatchInput struct {
	Name         *string
	Description  *string
	Glyph        *string
	ColorBg      *string
	ColorFg      *string
	ColorBorder  *string
	SOPSteps     *[]string
	SOPPDFURL    *string
	ClearSOPPDF  bool
	QRURL        *string
	ClearQR      bool
	AvgWaitMin   *int
	IsActive     *bool
	DisplayOrder *int
}

func validateHTTPURL(u string) bool {
	low := strings.ToLower(u)
	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://")
}

func (s *ServiceService) Update(ctx context.Context, key string, in ServicePatchInput) (*domain.Service, error) {
	patch := repository.ServicePatch{
		AvgWaitMin:   in.AvgWaitMin,
		IsActive:     in.IsActive,
		DisplayOrder: in.DisplayOrder,
		ClearSOPPDF:  in.ClearSOPPDF,
		ClearQR:      in.ClearQR,
	}

	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" || len(v) > 80 {
			return nil, domain.ErrInvalidInput
		}
		patch.Name = &v
	}
	if in.Description != nil {
		v := strings.TrimSpace(*in.Description)
		if len(v) > 500 {
			return nil, domain.ErrInvalidInput
		}
		patch.Description = &v
	}
	if in.Glyph != nil {
		v := strings.TrimSpace(*in.Glyph)
		if v == "" || len([]rune(v)) > 2 {
			return nil, domain.ErrInvalidInput
		}
		patch.Glyph = &v
	}
	if in.ColorBg != nil {
		v := strings.TrimSpace(*in.ColorBg)
		if v == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.ColorBg = &v
	}
	if in.ColorFg != nil {
		v := strings.TrimSpace(*in.ColorFg)
		if v == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.ColorFg = &v
	}
	if in.ColorBorder != nil {
		v := strings.TrimSpace(*in.ColorBorder)
		if v == "" {
			return nil, domain.ErrInvalidInput
		}
		patch.ColorBorder = &v
	}
	if in.SOPSteps != nil {
		if len(*in.SOPSteps) > 20 {
			return nil, domain.ErrInvalidInput
		}
		cleaned := make([]string, 0, len(*in.SOPSteps))
		for _, step := range *in.SOPSteps {
			t := strings.TrimSpace(step)
			if t == "" {
				return nil, domain.ErrInvalidInput
			}
			if len([]rune(t)) > 300 {
				return nil, domain.ErrInvalidInput
			}
			cleaned = append(cleaned, t)
		}
		patch.SOPSteps = &cleaned
	}
	if !in.ClearSOPPDF && in.SOPPDFURL != nil {
		v := strings.TrimSpace(*in.SOPPDFURL)
		if v == "" {
			patch.ClearSOPPDF = true
		} else {
			if !validateHTTPURL(v) {
				return nil, domain.ErrInvalidInput
			}
			patch.SOPPDFURL = &v
		}
	}
	if !in.ClearQR && in.QRURL != nil {
		v := strings.TrimSpace(*in.QRURL)
		if v == "" {
			patch.ClearQR = true
		} else {
			if !validateHTTPURL(v) {
				return nil, domain.ErrInvalidInput
			}
			patch.QRURL = &v
		}
	}
	if in.AvgWaitMin != nil {
		if *in.AvgWaitMin < 0 || *in.AvgWaitMin > 999 {
			return nil, domain.ErrInvalidInput
		}
	}
	if in.DisplayOrder != nil {
		if *in.DisplayOrder < 0 || *in.DisplayOrder > 9999 {
			return nil, domain.ErrInvalidInput
		}
	}

	return s.repo.Update(ctx, key, patch)
}
