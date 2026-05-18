package domain

import "time"

type AnalyticsOverview struct {
	TotalToday        int     `json:"total_today"`
	TotalDeltaPct     float64 `json:"total_delta_pct"`
	Waiting           int     `json:"waiting"`
	EstWaitMinutes    int     `json:"est_wait_minutes"`
	Completed         int     `json:"completed"`
	CompletedDeltaPct float64 `json:"completed_delta_pct"`
	AvgRating         float64 `json:"avg_rating"`
	RatingCount       int     `json:"rating_count"`
}

type ServiceDistribution struct {
	Service string `json:"service"`
	Count   int    `json:"count"`
}

type HourlyPoint struct {
	Hour           string `json:"hour"`
	Count          int    `json:"count"`
	AvgWaitMinutes int    `json:"avg_wait_minutes"`
}

type StaffPerf struct {
	UserID          string  `json:"user_id"`
	Name            string  `json:"name"`
	Counter         string  `json:"counter"`
	Served          int     `json:"served"`
	AvgServeSeconds int     `json:"avg_serve_seconds"`
	Rating          float64 `json:"rating"`
}

type IssueDistribution struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type RatingFeedback struct {
	QueueID     string    `json:"queue_id"`
	QueueNumber string    `json:"queue_number"`
	Service     string    `json:"service"`
	Rating      int       `json:"rating"`
	Comment     *string   `json:"comment"`
	RatedBy     *string   `json:"rated_by"`
	RatedAt     time.Time `json:"rated_at"`
}
