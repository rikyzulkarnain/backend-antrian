package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

type QueueRepository struct {
	db *pgxpool.Pool
}

func NewQueueRepository(db *pgxpool.Pool) *QueueRepository {
	return &QueueRepository{db: db}
}

const queueSelect = `
SELECT q.id, q.queue_number, q.service_type, q.status,
       q.created_at, q.called_at, q.completed_at,
       q.counter_id, q.user_id, u.name AS staff,
       q.rating, q.feedback,
       q.respondent_name, q.respondent_phone, q.issue_category,
       q.guest_name, q.guest_purpose, q.guest_token
FROM queues q
LEFT JOIN users u ON u.id = q.user_id
`

func scanQueue(row pgx.Row) (*domain.Queue, error) {
	q := &domain.Queue{}
	err := row.Scan(
		&q.ID, &q.QueueNumber, &q.ServiceType, &q.Status,
		&q.CreatedAt, &q.CalledAt, &q.CompletedAt,
		&q.CounterID, &q.UserID, &q.Staff,
		&q.Rating, &q.Feedback,
		&q.RespondentName, &q.RespondentPhone, &q.IssueCategory,
		&q.GuestName, &q.GuestPurpose, &q.GuestToken,
	)
	if err != nil {
		return nil, err
	}
	return q, nil
}

type ListFilter struct {
	Status      string
	ServiceType string
	Active      bool // when true, only waiting/calling/serving
	Limit       int
}

func (r *QueueRepository) List(ctx context.Context, f ListFilter) ([]domain.Queue, error) {
	sql := queueSelect + " WHERE 1=1"
	args := []any{}
	idx := 1
	if f.Status != "" {
		sql += fmt.Sprintf(" AND q.status = $%d", idx)
		args = append(args, f.Status)
		idx++
	}
	if f.ServiceType != "" {
		sql += fmt.Sprintf(" AND q.service_type = $%d", idx)
		args = append(args, f.ServiceType)
		idx++
	}
	if f.Active {
		sql += " AND q.status IN ('waiting','calling','serving')"
	}
	sql += " ORDER BY q.created_at ASC"
	if f.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, f.Limit)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Queue, 0)
	for rows.Next() {
		q, err := scanQueue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *q)
	}
	return out, rows.Err()
}

func (r *QueueRepository) Get(ctx context.Context, id string) (*domain.Queue, error) {
	row := r.db.QueryRow(ctx, queueSelect+" WHERE q.id = $1", id)
	q, err := scanQueue(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return q, err
}

// Create inserts a new queue ticket with a per-(day,service) sequence number.
// Uses a transaction-scoped advisory lock keyed by service_type+date to
// serialize sequence allocation while letting other service types proceed.
func (r *QueueRepository) Create(ctx context.Context, serviceType string) (*domain.Queue, error) {
	var prefix string
	err := r.db.QueryRow(ctx,
		`SELECT code FROM services WHERE key = $1 AND is_active = true`,
		serviceType,
	).Scan(&prefix)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInvalidInput
	}
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	lockKey := fmt.Sprintf("qseq:%s:%s", serviceType, "today")
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		WITH next_seq AS (
		  SELECT COALESCE(
		    MAX(
		      CASE WHEN queue_number LIKE $1 || '-%' THEN
		        NULLIF(SPLIT_PART(queue_number, '-', 2), '')::INTEGER
		      END
		    ), 0
		  ) + 1 AS n
		  FROM queues
		  WHERE service_type = $2 AND created_at::date = CURRENT_DATE
		)
		INSERT INTO queues (queue_number, service_type)
		SELECT $1 || '-' || LPAD(n::TEXT, 2, '0'), $2 FROM next_seq
		RETURNING id
	`, prefix, serviceType)

	var id string
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.Get(ctx, id)
}

// GetByGuestToken returns the ticket linked to a Buku Tamu session token, or
// ErrNotFound when the visitor has not submitted the form yet.
func (r *QueueRepository) GetByGuestToken(ctx context.Context, token string) (*domain.Queue, error) {
	row := r.db.QueryRow(ctx, queueSelect+" WHERE q.guest_token = $1", token)
	q, err := scanQueue(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return q, err
}

// CreateGuest issues a ticket for the given service carrying the visitor's
// name + purpose. Idempotent per guest_token: re-submitting the same token (e.g.
// a double-tap or page refresh on the mobile form) returns the existing ticket
// instead of allocating a second number.
func (r *QueueRepository) CreateGuest(ctx context.Context, in domain.GuestInput) (*domain.Queue, error) {
	serviceType := in.ServiceType

	// Fast path: token already used → return the existing ticket.
	if existing, err := r.GetByGuestToken(ctx, in.Token); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	var prefix string
	err := r.db.QueryRow(ctx,
		`SELECT code FROM services WHERE key = $1 AND is_active = true`,
		serviceType,
	).Scan(&prefix)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrInvalidInput
	}
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	lockKey := fmt.Sprintf("qseq:%s:%s", serviceType, "today")
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		WITH next_seq AS (
		  SELECT COALESCE(
		    MAX(
		      CASE WHEN queue_number LIKE $1 || '-%' THEN
		        NULLIF(SPLIT_PART(queue_number, '-', 2), '')::INTEGER
		      END
		    ), 0
		  ) + 1 AS n
		  FROM queues
		  WHERE service_type = $2 AND created_at::date = CURRENT_DATE
		)
		INSERT INTO queues (queue_number, service_type, guest_name, guest_purpose, guest_token)
		SELECT $1 || '-' || LPAD(n::TEXT, 2, '0'), $2, $3, $4, $5 FROM next_seq
		ON CONFLICT (guest_token) DO NOTHING
		RETURNING id
	`, prefix, serviceType, in.Name, in.Purpose, in.Token)

	var id string
	if err := row.Scan(&id); err != nil {
		// Lost a race on the unique guest_token: another request inserted it
		// first. Roll back and return the winner's ticket.
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			return r.GetByGuestToken(ctx, in.Token)
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.Get(ctx, id)
}

// CallNext picks the oldest waiting ticket of any service type (or filtered
// by serviceType when provided), transitions it to "calling", stamps the
// counter/user/called_at, and returns it. Returns ErrNotFound when no
// ticket is available.
func (r *QueueRepository) CallNext(ctx context.Context, counterID int, userID, serviceType string) (*domain.Queue, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	pickSQL := `
		SELECT id FROM queues
		WHERE status = 'waiting'
	`
	args := []any{}
	idx := 1
	if serviceType != "" {
		pickSQL += fmt.Sprintf(" AND service_type = $%d", idx)
		args = append(args, serviceType)
		idx++
	}
	pickSQL += " ORDER BY created_at ASC FOR UPDATE SKIP LOCKED LIMIT 1"

	var pickID string
	if err := tx.QueryRow(ctx, pickSQL, args...).Scan(&pickID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE queues
		SET status = 'calling', called_at = NOW(), counter_id = $1, user_id = $2
		WHERE id = $3
	`, counterID, userID, pickID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, pickID)
}

// UpdateStatus transitions a ticket to newStatus when its current status is
// one of allowed. Returns ErrConflict when the current status disallows the
// transition, or ErrNotFound when the ticket does not exist.
func (r *QueueRepository) UpdateStatus(ctx context.Context, id string, newStatus domain.QueueStatus, allowed []domain.QueueStatus) (*domain.Queue, error) {
	if len(allowed) == 0 {
		return nil, domain.ErrInvalidInput
	}

	sql := `UPDATE queues SET status = $1`
	args := []any{string(newStatus)}
	idx := 2
	if newStatus == domain.StatusCompleted {
		sql += ", completed_at = NOW()"
	}
	sql += fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, id)
	idx++

	placeholders := ""
	for i, s := range allowed {
		if i > 0 {
			placeholders += ","
		}
		placeholders += fmt.Sprintf("$%d", idx)
		args = append(args, string(s))
		idx++
	}
	sql += fmt.Sprintf(" AND status IN (%s)", placeholders)

	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		// Distinguish missing vs invalid-transition by re-reading.
		if _, getErr := r.Get(ctx, id); errors.Is(getErr, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrConflict
	}
	return r.Get(ctx, id)
}

// Rate stores SKM response on a completed ticket.
func (r *QueueRepository) Rate(ctx context.Context, id string, in domain.RateInput) (*domain.Queue, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, domain.ErrInvalidInput
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE queues
		SET rating = $1,
		    feedback = $2,
		    respondent_name = $3,
		    respondent_phone = $4,
		    issue_category = $5
		WHERE id = $6 AND status = 'completed'
	`, in.Rating, in.Feedback, in.RespondentName, in.RespondentPhone, in.IssueCategory, id)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if _, getErr := r.Get(ctx, id); errors.Is(getErr, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrConflict
	}
	return r.Get(ctx, id)
}
