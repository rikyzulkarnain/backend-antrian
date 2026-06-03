package domain

import "time"

type QueueStatus string

const (
	StatusWaiting   QueueStatus = "waiting"
	StatusCalling   QueueStatus = "calling"
	StatusServing   QueueStatus = "serving"
	StatusCompleted QueueStatus = "completed"
	StatusSkipped   QueueStatus = "skipped"
)

type Queue struct {
	ID          string      `json:"id"`
	QueueNumber string      `json:"queue_number"`
	ServiceType string      `json:"service_type"`
	Status      QueueStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	CalledAt    *time.Time  `json:"called_at"`
	CompletedAt *time.Time  `json:"completed_at"`
	CounterID   *int        `json:"counter_id"`
	UserID      *string     `json:"user_id,omitempty"`
	Staff       *string     `json:"staff"`
	Rating      *int        `json:"rating"`
	Feedback    *string     `json:"feedback,omitempty"`

	RespondentName  *string `json:"respondent_name,omitempty"`
	RespondentPhone *string `json:"respondent_phone,omitempty"`
	IssueCategory   *string `json:"issue_category,omitempty"`

	// Buku Tamu (guest book) fields, populated only for service_type "TAMU".
	GuestName    *string `json:"guest_name,omitempty"`
	GuestPurpose *string `json:"guest_purpose,omitempty"`
	GuestToken   *string `json:"guest_token,omitempty"`
}

// GuestInput is the validated payload accepted by QueueService.CreateGuest.
// ServiceType is the queue service the visitor is registering for (any active
// service, not only Buku Tamu).
type GuestInput struct {
	ServiceType string
	Token       string
	Name        string
	Purpose     string
}

// RateInput is the validated payload accepted by QueueService.Rate. Lives in
// domain so both service and repository can reference it without a cycle.
type RateInput struct {
	Rating          int
	Feedback        *string
	RespondentName  *string
	RespondentPhone *string
	IssueCategory   *string
}

var ValidIssueCategories = map[string]bool{
	"TIDAK_ADA": true,
	"PETUGAS":   true,
	"FASILITAS": true,
	"PROSEDUR":  true,
	"WAKTU":     true,
	"LAINNYA":   true,
}

// ValidServiceTypes lists accepted service_type codes (matches frontend).
var ValidServiceTypes = map[string]string{
	"UMUM": "A",
	"LAB":  "B",
	"AMP":  "C",
	"UTIL": "D",
	"SEWA": "E",
	"TAMU": "F",
}

func ServicePrefix(serviceType string) (string, bool) {
	p, ok := ValidServiceTypes[serviceType]
	return p, ok
}
