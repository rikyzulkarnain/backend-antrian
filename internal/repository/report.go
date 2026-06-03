package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

type ReportRepository struct {
	pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

// Summary aggregates the period headline in a single row. IKM fields are left
// zero here and filled by the service layer from AvgRating.
func (r *ReportRepository) Summary(ctx context.Context, f domain.ReportFilter) (domain.ReportSummary, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT
          COUNT(*)::int AS total,
          COUNT(*) FILTER (WHERE q.status = 'completed')::int AS completed,
          COUNT(*) FILTER (WHERE q.status = 'skipped')::int AS skipped,
          COUNT(*) FILTER (WHERE q.status = 'waiting')::int AS waiting,
          COALESCE(AVG(EXTRACT(EPOCH FROM (q.called_at - q.created_at))/60)
              FILTER (WHERE q.called_at IS NOT NULL), 0)::int AS avg_wait_min,
          COALESCE(AVG(EXTRACT(EPOCH FROM (q.completed_at - q.called_at)))
              FILTER (WHERE q.status = 'completed' AND q.completed_at IS NOT NULL), 0)::int AS avg_serve_sec,
          COALESCE(AVG(q.rating) FILTER (WHERE q.rating IS NOT NULL), 0)::float AS avg_rating,
          COUNT(q.rating) FILTER (WHERE q.rating IS NOT NULL)::int AS rating_count
        FROM queues q
        WHERE %s;`, where)

	var s domain.ReportSummary
	err := r.pool.QueryRow(ctx, q, args...).Scan(
		&s.Total, &s.Completed, &s.Skipped, &s.Waiting,
		&s.AvgWaitMin, &s.AvgServeSec, &s.AvgRating, &s.RatingCount,
	)
	return s, err
}

func (r *ReportRepository) ByService(ctx context.Context, f domain.ReportFilter) ([]domain.ServiceVolume, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT q.service_type,
          COUNT(*)::int,
          COUNT(*) FILTER (WHERE q.status = 'completed')::int,
          COUNT(*) FILTER (WHERE q.status = 'skipped')::int,
          COALESCE(AVG(q.rating) FILTER (WHERE q.rating IS NOT NULL), 0)::float
        FROM queues q
        WHERE %s
        GROUP BY q.service_type
        ORDER BY COUNT(*) DESC, q.service_type;`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ServiceVolume{}
	for rows.Next() {
		var v domain.ServiceVolume
		if err := rows.Scan(&v.Service, &v.Count, &v.Completed, &v.Skipped, &v.AvgRating); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *ReportRepository) Trend(ctx context.Context, f domain.ReportFilter) ([]domain.TrendPoint, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT to_char(q.created_at::date, 'YYYY-MM-DD') AS date,
          COUNT(*)::int,
          COUNT(*) FILTER (WHERE q.status = 'completed')::int
        FROM queues q
        WHERE %s
        GROUP BY q.created_at::date
        ORDER BY q.created_at::date;`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TrendPoint{}
	for rows.Next() {
		var p domain.TrendPoint
		if err := rows.Scan(&p.Date, &p.Count, &p.Completed); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ReportRepository) Staff(ctx context.Context, f domain.ReportFilter) ([]domain.ReportStaffPerf, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT u.id, u.name, COALESCE(c.name, '-') AS counter,
          COUNT(q.id) FILTER (WHERE q.status = 'completed')::int AS served,
          COALESCE(AVG(EXTRACT(EPOCH FROM (q.completed_at - q.called_at)))
              FILTER (WHERE q.status = 'completed' AND q.completed_at IS NOT NULL), 0)::int AS avg_serve_seconds,
          COALESCE(AVG(q.rating) FILTER (WHERE q.rating IS NOT NULL), 0)::float AS rating
        FROM queues q
        JOIN users u ON u.id = q.user_id
        LEFT JOIN counters c ON c.id = q.counter_id
        WHERE %s
        GROUP BY u.id, u.name, c.name
        ORDER BY served DESC, u.name;`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ReportStaffPerf{}
	for rows.Next() {
		var s domain.ReportStaffPerf
		if err := rows.Scan(&s.UserID, &s.Name, &s.Counter, &s.Served, &s.AvgServeSeconds, &s.Rating); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Issues returns complaint-category counts (excludes NULL/TIDAK_ADA) for the
// keluhan distribution chart.
func (r *ReportRepository) Issues(ctx context.Context, f domain.ReportFilter) ([]domain.IssueDistribution, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT q.issue_category, COUNT(*)::int
        FROM queues q
        WHERE %s
          AND q.issue_category IS NOT NULL
          AND q.issue_category <> 'TIDAK_ADA'
        GROUP BY q.issue_category
        ORDER BY COUNT(*) DESC, q.issue_category;`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.IssueDistribution{}
	for rows.Next() {
		var d domain.IssueDistribution
		if err := rows.Scan(&d.Category, &d.Count); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *ReportRepository) SKMDetailCount(ctx context.Context, f domain.ReportFilter) (int, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`SELECT COUNT(*)::int FROM queues q WHERE %s AND q.rating IS NOT NULL;`, where)
	var n int
	err := r.pool.QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

func (r *ReportRepository) SKMDetail(ctx context.Context, f domain.ReportFilter, limit, offset int) ([]domain.SKMDetail, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT q.queue_number, q.service_type, q.rating, q.feedback, q.issue_category,
               q.respondent_name, q.respondent_phone, u.name, c.name, q.completed_at
        FROM queues q
        LEFT JOIN users u ON u.id = q.user_id
        LEFT JOIN counters c ON c.id = q.counter_id
        WHERE %s AND q.rating IS NOT NULL
        ORDER BY q.completed_at DESC NULLS LAST
        LIMIT $%d OFFSET $%d;`, where, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SKMDetail{}
	for rows.Next() {
		var d domain.SKMDetail
		if err := rows.Scan(&d.QueueNumber, &d.Service, &d.Rating, &d.Comment, &d.IssueCategory,
			&d.RespondentName, &d.RespondentPhone, &d.Staff, &d.Counter, &d.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *ReportRepository) Complaints(ctx context.Context, f domain.ReportFilter) ([]domain.Complaint, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT q.queue_number, q.service_type, q.rating, q.issue_category, q.feedback,
               q.respondent_name, q.respondent_phone, q.completed_at
        FROM queues q
        WHERE %s AND q.rating IS NOT NULL AND q.rating <= 3
        ORDER BY q.completed_at DESC NULLS LAST
        LIMIT 200;`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Complaint{}
	for rows.Next() {
		var c domain.Complaint
		if err := rows.Scan(&c.QueueNumber, &c.Service, &c.Rating, &c.IssueCategory, &c.Comment,
			&c.RespondentName, &c.RespondentPhone, &c.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ReportRepository) GuestBook(ctx context.Context, f domain.ReportFilter) ([]domain.GuestEntry, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT q.queue_number, q.guest_name, q.guest_purpose, q.service_type, q.status, q.created_at
        FROM queues q
        WHERE %s AND q.guest_name IS NOT NULL
        ORDER BY q.created_at DESC
        LIMIT 500;`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GuestEntry{}
	for rows.Next() {
		var g domain.GuestEntry
		if err := rows.Scan(&g.QueueNumber, &g.GuestName, &g.GuestPurpose, &g.Service, &g.Status, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *ReportRepository) Skipped(ctx context.Context, f domain.ReportFilter) ([]domain.SkippedEntry, error) {
	where, args := f.Where("q", 1)
	q := fmt.Sprintf(`
        SELECT q.queue_number, q.service_type, c.name, q.created_at
        FROM queues q
        LEFT JOIN counters c ON c.id = q.counter_id
        WHERE %s AND q.status = 'skipped'
        ORDER BY q.created_at DESC
        LIMIT 500;`, where)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SkippedEntry{}
	for rows.Next() {
		var s domain.SkippedEntry
		if err := rows.Scan(&s.QueueNumber, &s.Service, &s.Counter, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
