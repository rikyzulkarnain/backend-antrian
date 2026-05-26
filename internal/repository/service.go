package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

type ServiceRepository struct {
	db *pgxpool.Pool
}

func NewServiceRepository(db *pgxpool.Pool) *ServiceRepository {
	return &ServiceRepository{db: db}
}

const serviceSelect = `
SELECT key, code, name, description, glyph, color_bg, color_fg, color_border,
       sop_steps, sop_pdf_url, qr_url, avg_wait_min, is_active, display_order, updated_at
FROM services
`

func scanService(row pgx.Row) (*domain.Service, error) {
	s := &domain.Service{}
	var stepsRaw []byte
	if err := row.Scan(
		&s.Key, &s.Code, &s.Name, &s.Description, &s.Glyph,
		&s.ColorBg, &s.ColorFg, &s.ColorBorder,
		&stepsRaw, &s.SOPPDFURL, &s.QRURL,
		&s.AvgWaitMin, &s.IsActive, &s.DisplayOrder, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(stepsRaw) > 0 {
		if err := json.Unmarshal(stepsRaw, &s.SOPSteps); err != nil {
			return nil, err
		}
	}
	if s.SOPSteps == nil {
		s.SOPSteps = []string{}
	}
	return s, nil
}

func (r *ServiceRepository) List(ctx context.Context, activeOnly bool) ([]domain.Service, error) {
	sql := serviceSelect
	if activeOnly {
		sql += " WHERE is_active = true"
	}
	sql += " ORDER BY display_order ASC, key ASC"

	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Service, 0)
	for rows.Next() {
		s, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *ServiceRepository) Get(ctx context.Context, key string) (*domain.Service, error) {
	row := r.db.QueryRow(ctx, serviceSelect+" WHERE key = $1", key)
	s, err := scanService(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return s, err
}

type ServicePatch struct {
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

func (r *ServiceRepository) Update(ctx context.Context, key string, p ServicePatch) (*domain.Service, error) {
	if _, err := r.Get(ctx, key); err != nil {
		return nil, err
	}
	sets := []string{}
	args := []any{}
	idx := 1
	add := func(col string, v any) {
		sets = append(sets, col+" = $"+strconv.Itoa(idx))
		args = append(args, v)
		idx++
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.Description != nil {
		add("description", *p.Description)
	}
	if p.Glyph != nil {
		add("glyph", *p.Glyph)
	}
	if p.ColorBg != nil {
		add("color_bg", *p.ColorBg)
	}
	if p.ColorFg != nil {
		add("color_fg", *p.ColorFg)
	}
	if p.ColorBorder != nil {
		add("color_border", *p.ColorBorder)
	}
	if p.SOPSteps != nil {
		raw, err := json.Marshal(*p.SOPSteps)
		if err != nil {
			return nil, err
		}
		add("sop_steps", raw)
	}
	if p.ClearSOPPDF {
		sets = append(sets, "sop_pdf_url = NULL")
	} else if p.SOPPDFURL != nil {
		add("sop_pdf_url", *p.SOPPDFURL)
	}
	if p.ClearQR {
		sets = append(sets, "qr_url = NULL")
	} else if p.QRURL != nil {
		add("qr_url", *p.QRURL)
	}
	if p.AvgWaitMin != nil {
		add("avg_wait_min", *p.AvgWaitMin)
	}
	if p.IsActive != nil {
		add("is_active", *p.IsActive)
	}
	if p.DisplayOrder != nil {
		add("display_order", *p.DisplayOrder)
	}
	if len(sets) == 0 {
		return r.Get(ctx, key)
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, key)
	sql := "UPDATE services SET " + strings.Join(sets, ", ") + " WHERE key = $" + strconv.Itoa(idx)
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return nil, err
	}
	return r.Get(ctx, key)
}
