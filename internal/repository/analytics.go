package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

// Overview returns today's counts vs yesterday for delta calculation.
func (r *AnalyticsRepository) Overview(ctx context.Context) (*domain.AnalyticsOverview, error) {
	const q = `
        WITH today AS (
            SELECT
              COUNT(*) FILTER (WHERE created_at::date = CURRENT_DATE) AS total_today,
              COUNT(*) FILTER (WHERE status = 'waiting') AS waiting,
              COUNT(*) FILTER (WHERE status = 'completed' AND completed_at::date = CURRENT_DATE) AS completed,
              COALESCE(AVG(rating) FILTER (WHERE rating IS NOT NULL AND completed_at::date = CURRENT_DATE), 0)::float AS avg_rating,
              COUNT(rating) FILTER (WHERE rating IS NOT NULL AND completed_at::date = CURRENT_DATE) AS rating_count,
              COALESCE(AVG(EXTRACT(EPOCH FROM (called_at - created_at))/60)
                  FILTER (WHERE called_at IS NOT NULL AND created_at::date = CURRENT_DATE), 0)::int AS est_wait_minutes
            FROM queues
        ),
        yest AS (
            SELECT
              COUNT(*) FILTER (WHERE created_at::date = CURRENT_DATE - INTERVAL '1 day') AS total_yest,
              COUNT(*) FILTER (WHERE status = 'completed' AND completed_at::date = CURRENT_DATE - INTERVAL '1 day') AS completed_yest
            FROM queues
        )
        SELECT
          today.total_today, today.waiting, today.completed, today.avg_rating,
          today.rating_count, today.est_wait_minutes,
          yest.total_yest, yest.completed_yest
        FROM today, yest;
    `
	var o domain.AnalyticsOverview
	var totalYest, completedYest int
	err := r.pool.QueryRow(ctx, q).Scan(
		&o.TotalToday, &o.Waiting, &o.Completed, &o.AvgRating,
		&o.RatingCount, &o.EstWaitMinutes,
		&totalYest, &completedYest,
	)
	if err != nil {
		return nil, err
	}
	o.TotalDeltaPct = deltaPct(o.TotalToday, totalYest)
	o.CompletedDeltaPct = deltaPct(o.Completed, completedYest)
	return &o, nil
}

func deltaPct(today, yest int) float64 {
	if yest == 0 {
		if today == 0 {
			return 0
		}
		return 100
	}
	return (float64(today-yest) / float64(yest)) * 100
}

func (r *AnalyticsRepository) ByService(ctx context.Context) ([]domain.ServiceDistribution, error) {
	const q = `
        SELECT service_type, COUNT(*)::int
        FROM queues
        WHERE created_at::date = CURRENT_DATE
        GROUP BY service_type
        ORDER BY service_type;
    `
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ServiceDistribution{}
	for rows.Next() {
		var s domain.ServiceDistribution
		if err := rows.Scan(&s.Service, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) ByHour(ctx context.Context) ([]domain.HourlyPoint, error) {
	const q = `
        SELECT
          to_char(date_trunc('hour', created_at), 'HH24:00') AS hour,
          COUNT(*)::int AS count,
          COALESCE(AVG(EXTRACT(EPOCH FROM (called_at - created_at))/60), 0)::int AS avg_wait_minutes
        FROM queues
        WHERE created_at::date = CURRENT_DATE
        GROUP BY date_trunc('hour', created_at)
        ORDER BY date_trunc('hour', created_at);
    `
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.HourlyPoint{}
	for rows.Next() {
		var h domain.HourlyPoint
		if err := rows.Scan(&h.Hour, &h.Count, &h.AvgWaitMinutes); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) Staff(ctx context.Context) ([]domain.StaffPerf, error) {
	const q = `
        SELECT
          u.id, u.name,
          COALESCE(c.name, '-') AS counter,
          COUNT(q.id) FILTER (WHERE q.status = 'completed' AND q.completed_at::date = CURRENT_DATE)::int AS served,
          COALESCE(AVG(EXTRACT(EPOCH FROM (q.completed_at - q.called_at)))
              FILTER (WHERE q.status = 'completed' AND q.completed_at::date = CURRENT_DATE), 0)::int AS avg_serve_seconds,
          COALESCE(AVG(q.rating)
              FILTER (WHERE q.rating IS NOT NULL AND q.completed_at::date = CURRENT_DATE), 0)::float AS rating
        FROM users u
        LEFT JOIN queues q ON q.user_id = u.id
        LEFT JOIN counters c ON c.id = q.counter_id
        WHERE u.role = 'staff' AND u.is_active = true
        GROUP BY u.id, u.name, c.name
        ORDER BY served DESC, u.name;
    `
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.StaffPerf{}
	for rows.Next() {
		var s domain.StaffPerf
		if err := rows.Scan(&s.UserID, &s.Name, &s.Counter, &s.Served, &s.AvgServeSeconds, &s.Rating); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Issues aggregates issue_category counts among today's rated SKM responses.
// NULL/'TIDAK_ADA' rows are omitted so the chart only shows real complaints.
func (r *AnalyticsRepository) Issues(ctx context.Context) ([]domain.IssueDistribution, error) {
	const q = `
        SELECT issue_category, COUNT(*)::int
        FROM queues
        WHERE rating IS NOT NULL
          AND completed_at::date = CURRENT_DATE
          AND issue_category IS NOT NULL
          AND issue_category <> 'TIDAK_ADA'
        GROUP BY issue_category
        ORDER BY COUNT(*) DESC, issue_category;
    `
	rows, err := r.pool.Query(ctx, q)
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

func (r *AnalyticsRepository) Ratings(ctx context.Context, limit int) ([]domain.RatingFeedback, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	const q = `
        SELECT q.id, q.queue_number, q.service_type, q.rating, q.feedback,
               u.name, q.completed_at
        FROM queues q
        LEFT JOIN users u ON u.id = q.user_id
        WHERE q.rating IS NOT NULL
        ORDER BY q.completed_at DESC NULLS LAST
        LIMIT $1;
    `
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.RatingFeedback{}
	for rows.Next() {
		var rf domain.RatingFeedback
		if err := rows.Scan(&rf.QueueID, &rf.QueueNumber, &rf.Service, &rf.Rating, &rf.Comment, &rf.RatedBy, &rf.RatedAt); err != nil {
			return nil, err
		}
		out = append(out, rf)
	}
	return out, rows.Err()
}
