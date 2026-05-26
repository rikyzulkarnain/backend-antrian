package domain

import "time"

type Service struct {
	Key          string    `json:"key"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Glyph        string    `json:"glyph"`
	ColorBg      string    `json:"color_bg"`
	ColorFg      string    `json:"color_fg"`
	ColorBorder  string    `json:"color_border"`
	SOPSteps     []string  `json:"sop_steps"`
	SOPPDFURL    *string   `json:"sop_pdf_url,omitempty"`
	QRURL        *string   `json:"qr_url,omitempty"`
	AvgWaitMin   int       `json:"avg_wait_min"`
	IsActive     bool      `json:"is_active"`
	DisplayOrder int       `json:"display_order"`
	UpdatedAt    time.Time `json:"updated_at"`
}
