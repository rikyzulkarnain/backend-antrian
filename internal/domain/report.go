package domain

import (
	"fmt"
	"time"
)

// ReportFilter holds the period + dimension filters that scope every report
// query. Pointer fields are optional ("semua" when nil). DateFrom/DateTo are
// inclusive day bounds; the repository compares against created_at::date.
type ReportFilter struct {
	DateFrom      time.Time
	DateTo        time.Time
	ServiceType   *string
	Counter       *int
	UserID        *string
	IssueCategory *string
	Rating        *int
}

// Where builds a parametrised SQL predicate (without the leading WHERE) plus
// the matching args, starting numbering at startIdx so callers can prepend
// their own placeholders. Always constrains the date range; other clauses are
// added only when the corresponding filter is set. The returned alias-less
// column names assume the queues table is the primary/aliased `q` — callers
// pass a column prefix to disambiguate joins.
func (f ReportFilter) Where(prefix string, startIdx int) (string, []any) {
	if prefix != "" {
		prefix += "."
	}
	clause := fmt.Sprintf("%screated_at::date BETWEEN $%d AND $%d", prefix, startIdx, startIdx+1)
	args := []any{f.DateFrom, f.DateTo}
	idx := startIdx + 2

	if f.ServiceType != nil {
		clause += fmt.Sprintf(" AND %sservice_type = $%d", prefix, idx)
		args = append(args, *f.ServiceType)
		idx++
	}
	if f.Counter != nil {
		clause += fmt.Sprintf(" AND %scounter_id = $%d", prefix, idx)
		args = append(args, *f.Counter)
		idx++
	}
	if f.UserID != nil {
		clause += fmt.Sprintf(" AND %suser_id = $%d", prefix, idx)
		args = append(args, *f.UserID)
		idx++
	}
	if f.IssueCategory != nil {
		clause += fmt.Sprintf(" AND %sissue_category = $%d", prefix, idx)
		args = append(args, *f.IssueCategory)
		idx++
	}
	if f.Rating != nil {
		clause += fmt.Sprintf(" AND %srating = $%d", prefix, idx)
		args = append(args, *f.Rating)
		idx++
	}
	return clause, args
}

// ReportSummary is the period headline shown as KPI cards.
type ReportSummary struct {
	Total         int     `json:"total"`
	Completed     int     `json:"completed"`
	Skipped       int     `json:"skipped"`
	Waiting       int     `json:"waiting"`
	AvgWaitMin    int     `json:"avg_wait_min"`
	AvgServeSec   int     `json:"avg_serve_sec"`
	AvgRating     float64 `json:"avg_rating"`
	RatingCount   int     `json:"rating_count"`
	IKMIndex      float64 `json:"ikm_index"`
	IKMGrade      string  `json:"ikm_grade"`
	IKMGradeLabel string  `json:"ikm_grade_label"`
}

// ServiceVolume is one row of the per-service volume table/donut.
type ServiceVolume struct {
	Service   string  `json:"service"`
	Count     int     `json:"count"`
	Completed int     `json:"completed"`
	Skipped   int     `json:"skipped"`
	AvgRating float64 `json:"avg_rating"`
}

// TrendPoint is one day in the volume trend chart.
type TrendPoint struct {
	Date      string `json:"date"`
	Count     int    `json:"count"`
	Completed int    `json:"completed"`
}

// ReportStaffPerf is staff/counter performance over the filtered period.
type ReportStaffPerf struct {
	UserID          string  `json:"user_id"`
	Name            string  `json:"name"`
	Counter         string  `json:"counter"`
	Served          int     `json:"served"`
	AvgServeSeconds int     `json:"avg_serve_seconds"`
	Rating          float64 `json:"rating"`
}

// IKMElement is one "unsur" of the public satisfaction index, derived from the
// complaint category counts (see service.go for the mapping).
type IKMElement struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Issues int    `json:"issues"`
}

// IKMReport approximates the PermenPAN public-satisfaction index from the
// single overall rating (the SKM form captures one rating, not 9 unsur).
type IKMReport struct {
	Index       float64      `json:"index"`
	Grade       string       `json:"grade"`
	GradeLabel  string       `json:"grade_label"`
	RatingCount int          `json:"rating_count"`
	Elements    []IKMElement `json:"elements"`
}

// SKMDetail is one applicant survey response (laporan detail penilaian).
type SKMDetail struct {
	QueueNumber     string    `json:"queue_number"`
	Service         string    `json:"service"`
	Rating          int       `json:"rating"`
	Comment         *string   `json:"comment"`
	IssueCategory   *string   `json:"issue_category"`
	RespondentName  *string   `json:"respondent_name"`
	RespondentPhone *string   `json:"respondent_phone"`
	Staff           *string   `json:"staff"`
	Counter         *string   `json:"counter"`
	CompletedAt     time.Time `json:"completed_at"`
}

// Complaint is a low-rating (≤3) response surfaced for follow-up.
type Complaint struct {
	QueueNumber     string    `json:"queue_number"`
	Service         string    `json:"service"`
	Rating          int       `json:"rating"`
	IssueCategory   *string   `json:"issue_category"`
	Comment         *string   `json:"comment"`
	RespondentName  *string   `json:"respondent_name"`
	RespondentPhone *string   `json:"respondent_phone"`
	CompletedAt     time.Time `json:"completed_at"`
}

// GuestEntry is one buku-tamu visit.
type GuestEntry struct {
	QueueNumber  string    `json:"queue_number"`
	GuestName    *string   `json:"guest_name"`
	GuestPurpose *string   `json:"guest_purpose"`
	Service      string    `json:"service"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// SkippedEntry is one skipped (no-show) ticket.
type SkippedEntry struct {
	QueueNumber string    `json:"queue_number"`
	Service     string    `json:"service"`
	Counter     *string   `json:"counter"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReportBundle is the single payload powering the on-screen report dashboard.
type ReportBundle struct {
	Summary    ReportSummary       `json:"summary"`
	ByService  []ServiceVolume     `json:"by_service"`
	Trend      []TrendPoint        `json:"trend"`
	Staff      []ReportStaffPerf   `json:"staff"`
	IKM        IKMReport           `json:"ikm"`
	Issues     []IssueDistribution `json:"issues"`
	Complaints []Complaint         `json:"complaints"`
	GuestBook  []GuestEntry        `json:"guest_book"`
	Skipped    []SkippedEntry      `json:"skipped"`
}

// SKMDetailPage is a paginated slice of survey responses.
type SKMDetailPage struct {
	Items []SKMDetail `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}
