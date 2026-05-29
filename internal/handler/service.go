package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

type ServiceHandler struct {
	svc *service.ServiceService
}

func NewServiceHandler(svc *service.ServiceService) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := strings.EqualFold(r.URL.Query().Get("active"), "true")
	items, err := h.svc.List(r.Context(), activeOnly)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, items)
}

func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Get(r.Context(), chi.URLParam(r, "key"))
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, s)
}

type serviceBody struct {
	Name         *string   `json:"name"`
	Description  *string   `json:"description"`
	Glyph        *string   `json:"glyph"`
	ColorBg      *string   `json:"color_bg"`
	ColorFg      *string   `json:"color_fg"`
	ColorBorder  *string   `json:"color_border"`
	SOPSteps     *[]string `json:"sop_steps"`
	SOPPDFURL    *string   `json:"sop_pdf_url"`
	QRURL        *string   `json:"qr_url"`
	AvgWaitMin   *int      `json:"avg_wait_min"`
	IsActive     *bool     `json:"is_active"`
	DisplayOrder *int      `json:"display_order"`

	rawSOPPDFURL json.RawMessage
	rawQRURL     json.RawMessage
}

func (b *serviceBody) UnmarshalJSON(data []byte) error {
	type rawT struct {
		Name         *string         `json:"name"`
		Description  *string         `json:"description"`
		Glyph        *string         `json:"glyph"`
		ColorBg      *string         `json:"color_bg"`
		ColorFg      *string         `json:"color_fg"`
		ColorBorder  *string         `json:"color_border"`
		SOPSteps     *[]string       `json:"sop_steps"`
		SOPPDFURL    json.RawMessage `json:"sop_pdf_url"`
		QRURL        json.RawMessage `json:"qr_url"`
		AvgWaitMin   *int            `json:"avg_wait_min"`
		IsActive     *bool           `json:"is_active"`
		DisplayOrder *int            `json:"display_order"`
	}
	var raw rawT
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.Name = raw.Name
	b.Description = raw.Description
	b.Glyph = raw.Glyph
	b.ColorBg = raw.ColorBg
	b.ColorFg = raw.ColorFg
	b.ColorBorder = raw.ColorBorder
	b.SOPSteps = raw.SOPSteps
	b.AvgWaitMin = raw.AvgWaitMin
	b.IsActive = raw.IsActive
	b.DisplayOrder = raw.DisplayOrder

	if len(raw.SOPPDFURL) > 0 {
		b.rawSOPPDFURL = raw.SOPPDFURL
		if string(raw.SOPPDFURL) != "null" {
			var v string
			if err := json.Unmarshal(raw.SOPPDFURL, &v); err != nil {
				return err
			}
			b.SOPPDFURL = &v
		}
	}
	if len(raw.QRURL) > 0 {
		b.rawQRURL = raw.QRURL
		if string(raw.QRURL) != "null" {
			var v string
			if err := json.Unmarshal(raw.QRURL, &v); err != nil {
				return err
			}
			b.QRURL = &v
		}
	}
	return nil
}

type serviceCreateBody struct {
	Key          string   `json:"key"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Glyph        string   `json:"glyph"`
	ColorBg      string   `json:"color_bg"`
	ColorFg      string   `json:"color_fg"`
	ColorBorder  string   `json:"color_border"`
	SOPSteps     []string `json:"sop_steps"`
	SOPPDFURL    *string  `json:"sop_pdf_url"`
	QRURL        *string  `json:"qr_url"`
	AvgWaitMin   int      `json:"avg_wait_min"`
	IsActive     *bool    `json:"is_active"`
	DisplayOrder int      `json:"display_order"`
}

func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body serviceCreateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	active := true
	if body.IsActive != nil {
		active = *body.IsActive
	}
	in := service.ServiceCreateInput{
		Key:          body.Key,
		Code:         body.Code,
		Name:         body.Name,
		Description:  body.Description,
		Glyph:        body.Glyph,
		ColorBg:      body.ColorBg,
		ColorFg:      body.ColorFg,
		ColorBorder:  body.ColorBorder,
		SOPSteps:     body.SOPSteps,
		SOPPDFURL:    body.SOPPDFURL,
		QRURL:        body.QRURL,
		AvgWaitMin:   body.AvgWaitMin,
		IsActive:     active,
		DisplayOrder: body.DisplayOrder,
	}
	s, err := h.svc.Create(r.Context(), in)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	created(w, s)
}

func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "key")); err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, map[string]bool{"ok": true})
}

func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body serviceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "Body JSON tidak valid")
		return
	}
	in := service.ServicePatchInput{
		Name:         body.Name,
		Description:  body.Description,
		Glyph:        body.Glyph,
		ColorBg:      body.ColorBg,
		ColorFg:      body.ColorFg,
		ColorBorder:  body.ColorBorder,
		SOPSteps:     body.SOPSteps,
		SOPPDFURL:    body.SOPPDFURL,
		ClearSOPPDF:  len(body.rawSOPPDFURL) > 0 && body.SOPPDFURL == nil,
		QRURL:        body.QRURL,
		ClearQR:      len(body.rawQRURL) > 0 && body.QRURL == nil,
		AvgWaitMin:   body.AvgWaitMin,
		IsActive:     body.IsActive,
		DisplayOrder: body.DisplayOrder,
	}
	s, err := h.svc.Update(r.Context(), chi.URLParam(r, "key"), in)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, s)
}
