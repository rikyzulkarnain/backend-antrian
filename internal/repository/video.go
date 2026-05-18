package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

type VideoRepository struct {
	db *pgxpool.Pool
}

func NewVideoRepository(db *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{db: db}
}

const videoSelect = `
SELECT id, title, url, target_screen, duration_seconds, display_order,
       is_active, uploaded_at, uploaded_by
FROM videos
`

func scanVideo(row pgx.Row) (*domain.Video, error) {
	v := &domain.Video{}
	err := row.Scan(
		&v.ID, &v.Title, &v.URL, &v.TargetScreen, &v.DurationSeconds,
		&v.DisplayOrder, &v.IsActive, &v.UploadedAt, &v.UploadedBy,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// List returns videos ordered by display_order, optionally filtered by target.
// target=="" returns all rows; target="kiosk" returns rows where target_screen
// is "kiosk" or "both" (and analogously for "display").
func (r *VideoRepository) List(ctx context.Context, target string) ([]domain.Video, error) {
	sql := videoSelect
	args := []any{}
	switch target {
	case "kiosk":
		sql += " WHERE target_screen IN ('kiosk', 'both')"
	case "display":
		sql += " WHERE target_screen IN ('display', 'both')"
	case "":
		// no filter
	default:
		sql += " WHERE target_screen = $1"
		args = append(args, target)
	}
	sql += " ORDER BY display_order ASC, uploaded_at DESC"

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Video, 0)
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *VideoRepository) Get(ctx context.Context, id string) (*domain.Video, error) {
	row := r.db.QueryRow(ctx, videoSelect+" WHERE id = $1", id)
	v, err := scanVideo(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return v, err
}

type VideoFields struct {
	Title           string
	URL             string
	TargetScreen    string
	DurationSeconds *int
	DisplayOrder    int
	IsActive        bool
	UploadedBy      *string
}

func (r *VideoRepository) Create(ctx context.Context, f VideoFields) (*domain.Video, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO videos (title, url, target_screen, duration_seconds, display_order, is_active, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, f.Title, f.URL, f.TargetScreen, f.DurationSeconds, f.DisplayOrder, f.IsActive, f.UploadedBy).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

type VideoPatch struct {
	Title           *string
	URL             *string
	TargetScreen    *string
	DurationSeconds *int
	DisplayOrder    *int
	IsActive        *bool
	ClearDuration   bool
}

func (r *VideoRepository) Update(ctx context.Context, id string, p VideoPatch) (*domain.Video, error) {
	if _, err := r.Get(ctx, id); err != nil {
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
	if p.Title != nil {
		add("title", *p.Title)
	}
	if p.URL != nil {
		add("url", *p.URL)
	}
	if p.TargetScreen != nil {
		add("target_screen", *p.TargetScreen)
	}
	if p.ClearDuration {
		sets = append(sets, "duration_seconds = NULL")
	} else if p.DurationSeconds != nil {
		add("duration_seconds", *p.DurationSeconds)
	}
	if p.DisplayOrder != nil {
		add("display_order", *p.DisplayOrder)
	}
	if p.IsActive != nil {
		add("is_active", *p.IsActive)
	}
	if len(sets) == 0 {
		return r.Get(ctx, id)
	}
	args = append(args, id)
	sql := "UPDATE videos SET " + strings.Join(sets, ", ") + " WHERE id = $" + strconv.Itoa(idx)
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *VideoRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM videos WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
