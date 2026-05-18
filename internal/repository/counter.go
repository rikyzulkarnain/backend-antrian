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

type CounterRepository struct {
	db *pgxpool.Pool
}

func NewCounterRepository(db *pgxpool.Pool) *CounterRepository {
	return &CounterRepository{db: db}
}

const counterSelect = `
SELECT c.id, c.name, c.service_type, c.is_active, c.staff_id, u.name AS staff
FROM counters c
LEFT JOIN users u ON u.id = c.staff_id
`

func scanCounter(row pgx.Row) (*domain.Counter, error) {
	c := &domain.Counter{}
	err := row.Scan(&c.ID, &c.Name, &c.Service, &c.Active, &c.StaffID, &c.Staff)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *CounterRepository) List(ctx context.Context) ([]domain.Counter, error) {
	rows, err := r.db.Query(ctx, counterSelect+" ORDER BY c.id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Counter, 0)
	for rows.Next() {
		c, err := scanCounter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *CounterRepository) Get(ctx context.Context, id int) (*domain.Counter, error) {
	row := r.db.QueryRow(ctx, counterSelect+" WHERE c.id = $1", id)
	c, err := scanCounter(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return c, err
}

type CounterFields struct {
	Name    string
	Service *string
	Active  bool
	StaffID *string
}

func (r *CounterRepository) Create(ctx context.Context, f CounterFields) (*domain.Counter, error) {
	var id int
	err := r.db.QueryRow(ctx, `
		INSERT INTO counters (name, service_type, is_active, staff_id)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, f.Name, f.Service, f.Active, f.StaffID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

// CounterPatch holds partial updates; non-nil pointer = update that column.
type CounterPatch struct {
	Name    *string
	Service *string // value pointer; empty string clears, "UMUM" sets
	Active  *bool
	StaffID *string
	// nil-vs-set semantics for nullable columns:
	ClearService bool
	ClearStaff   bool
}

func (r *CounterRepository) Update(ctx context.Context, id int, p CounterPatch) (*domain.Counter, error) {
	if _, err := r.Get(ctx, id); err != nil {
		return nil, err
	}
	sets := []string{}
	args := []any{}
	idx := 1

	if p.Name != nil {
		sets = append(sets, "name = $"+strconv.Itoa(idx))
		args = append(args, *p.Name)
		idx++
	}
	if p.ClearService {
		sets = append(sets, "service_type = NULL")
	} else if p.Service != nil {
		sets = append(sets, "service_type = $"+strconv.Itoa(idx))
		args = append(args, *p.Service)
		idx++
	}
	if p.Active != nil {
		sets = append(sets, "is_active = $"+strconv.Itoa(idx))
		args = append(args, *p.Active)
		idx++
	}
	if p.ClearStaff {
		sets = append(sets, "staff_id = NULL")
	} else if p.StaffID != nil {
		sets = append(sets, "staff_id = $"+strconv.Itoa(idx))
		args = append(args, *p.StaffID)
		idx++
	}

	if len(sets) == 0 {
		return r.Get(ctx, id)
	}

	args = append(args, id)
	sql := "UPDATE counters SET " + strings.Join(sets, ", ") + " WHERE id = $" + strconv.Itoa(idx)
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *CounterRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM counters WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
