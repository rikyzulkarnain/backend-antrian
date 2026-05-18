package domain

import "time"

type VideoTarget string

const (
	TargetKiosk   VideoTarget = "kiosk"
	TargetDisplay VideoTarget = "display"
	TargetBoth    VideoTarget = "both"
)

type Video struct {
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	URL             string      `json:"url"`
	TargetScreen    VideoTarget `json:"target_screen"`
	DurationSeconds *int        `json:"duration_seconds,omitempty"`
	DisplayOrder    int         `json:"display_order"`
	IsActive        bool        `json:"is_active"`
	UploadedAt      time.Time   `json:"uploaded_at"`
	UploadedBy      *string     `json:"uploaded_by,omitempty"`
}
