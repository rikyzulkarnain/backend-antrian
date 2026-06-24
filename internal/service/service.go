package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/cache"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

type ServiceService struct {
	repo *repository.ServiceRepository
	// listCache serves kiosk/display polls of the (near-static) service list
	// from memory; invalidated on any admin CRUD so Neon can stay suspended
	// between changes. See package cache.
	listCache *cache.Keyed[[]domain.Service]
}

func NewServiceService(r *repository.ServiceRepository) *ServiceService {
	return &ServiceService{repo: r, listCache: cache.NewKeyed[[]domain.Service]()}
}

func (s *ServiceService) List(ctx context.Context, activeOnly bool) ([]domain.Service, error) {
	key := fmt.Sprintf("%t", activeOnly)
	if v, ok := s.listCache.Get(key); ok {
		return v, nil
	}
	v, err := s.repo.List(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	s.listCache.Set(key, v)
	return v, nil
}

func (s *ServiceService) Get(ctx context.Context, key string) (*domain.Service, error) {
	return s.repo.Get(ctx, key)
}

type ServiceCreateInput struct {
	Key          string
	Code         string
	Name         string
	Description  string
	Glyph        string
	ColorBg      string
	ColorFg      string
	ColorBorder  string
	SOPSteps     []string
	SOPPDFURL    *string
	QRURL        *string
	AvgWaitMin   int
	IsActive     bool
	DisplayOrder int
}

var serviceKeyRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,15}$`)

func (s *ServiceService) Create(ctx context.Context, in ServiceCreateInput) (*domain.Service, error) {
	key := strings.ToUpper(strings.TrimSpace(in.Key))
	if !serviceKeyRe.MatchString(key) {
		return nil, domain.ErrInvalidInput
	}
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if code == "" || len([]rune(code)) > 4 {
		return nil, domain.ErrInvalidInput
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 80 {
		return nil, domain.ErrInvalidInput
	}
	description := strings.TrimSpace(in.Description)
	if len(description) > 500 {
		return nil, domain.ErrInvalidInput
	}
	glyph := strings.TrimSpace(in.Glyph)
	if glyph == "" || len([]rune(glyph)) > 2 {
		return nil, domain.ErrInvalidInput
	}
	if strings.TrimSpace(in.ColorBg) == "" ||
		strings.TrimSpace(in.ColorFg) == "" ||
		strings.TrimSpace(in.ColorBorder) == "" {
		return nil, domain.ErrInvalidInput
	}
	if in.AvgWaitMin < 0 || in.AvgWaitMin > 999 {
		return nil, domain.ErrInvalidInput
	}
	if in.DisplayOrder < 0 || in.DisplayOrder > 9999 {
		return nil, domain.ErrInvalidInput
	}
	if len(in.SOPSteps) > 20 {
		return nil, domain.ErrInvalidInput
	}
	cleanedSteps := make([]string, 0, len(in.SOPSteps))
	for _, step := range in.SOPSteps {
		t := strings.TrimSpace(step)
		if t == "" {
			return nil, domain.ErrInvalidInput
		}
		if len([]rune(t)) > 300 {
			return nil, domain.ErrInvalidInput
		}
		cleanedSteps = append(cleanedSteps, t)
	}
	var pdf, qr *string
	if in.SOPPDFURL != nil {
		v := strings.TrimSpace(*in.SOPPDFURL)
		if v != "" {
			if !validateHTTPURL(v) {
				return nil, domain.ErrInvalidInput
			}
			pdf = &v
		}
	}
	if in.QRURL != nil {
		v := strings.TrimSpace(*in.QRURL)
		if v != "" {
			if !validateHTTPURL(v) {
				return nil, domain.ErrInvalidInput
			}
			qr = &v
		}
	}
	created, err := s.repo.Create(ctx, repository.ServiceCreate{
		Key:          key,
		Code:         code,
		Name:         name,
		Description:  description,
		Glyph:        glyph,
		ColorBg:      strings.TrimSpace(in.ColorBg),
		ColorFg:      strings.TrimSpace(in.ColorFg),
		ColorBorder:  strings.TrimSpace(in.ColorBorder),
		SOPSteps:     cleanedSteps,
		SOPPDFURL:    pdf,
		QRURL:        qr,
		AvgWaitMin:   in.AvgWaitMin,
		IsActive:     in.IsActive,
		DisplayOrder: in.DisplayOrder,
	})
	if err != nil {
		return nil, err
	}
	s.listCache.Invalidate()
	return created, nil
}

func (s *ServiceService) Delete(ctx context.Context, key string) error {
	if err := s.repo.Delete(ctx, key); err != nil {
		return err
	}
	s.listCache.Invalidate()
	return nil
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

	updated, err := s.repo.Update(ctx, key, patch)
	if err != nil {
		return nil, err
	}
	s.listCache.Invalidate()
	return updated, nil
}
