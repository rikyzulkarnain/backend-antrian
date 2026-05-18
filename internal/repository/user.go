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

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

const userSelect = `
SELECT id, name, email, password_hash, role, is_active, created_at
FROM users
`

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRow(ctx, userSelect+" WHERE email = $1", email)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.db.QueryRow(ctx, userSelect+" WHERE id = $1", id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.Query(ctx, userSelect+" ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

type UserFields struct {
	Name         string
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
}

func (r *UserRepository) Create(ctx context.Context, f UserFields) (*domain.User, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, f.Name, f.Email, f.PasswordHash, f.Role, f.IsActive).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, err
	}
	return r.FindByID(ctx, id)
}

type UserPatch struct {
	Name         *string
	Email        *string
	PasswordHash *string
	Role         *string
	IsActive     *bool
}

func (r *UserRepository) Update(ctx context.Context, id string, p UserPatch) (*domain.User, error) {
	if _, err := r.FindByID(ctx, id); err != nil {
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
	if p.Email != nil {
		sets = append(sets, "email = $"+strconv.Itoa(idx))
		args = append(args, *p.Email)
		idx++
	}
	if p.PasswordHash != nil {
		sets = append(sets, "password_hash = $"+strconv.Itoa(idx))
		args = append(args, *p.PasswordHash)
		idx++
	}
	if p.Role != nil {
		sets = append(sets, "role = $"+strconv.Itoa(idx))
		args = append(args, *p.Role)
		idx++
	}
	if p.IsActive != nil {
		sets = append(sets, "is_active = $"+strconv.Itoa(idx))
		args = append(args, *p.IsActive)
		idx++
	}
	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}
	args = append(args, id)
	sql := "UPDATE users SET " + strings.Join(sets, ", ") + " WHERE id = $" + strconv.Itoa(idx)
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
