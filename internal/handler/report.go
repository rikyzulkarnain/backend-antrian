package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
)

type ReportHandler struct {
	svc *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// parseReportFilter reads period + dimension filters from the query string.
// Missing dates default to the last 30 days (inclusive of today).
func parseReportFilter(r *http.Request) domain.ReportFilter {
	q := r.URL.Query()
	now := time.Now()
	from := now.AddDate(0, 0, -29)
	to := now
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t
		}
	}
	f := domain.ReportFilter{DateFrom: from, DateTo: to}
	if v := q.Get("service"); v != "" {
		f.ServiceType = &v
	}
	if v := q.Get("counter"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Counter = &n
		}
	}
	if v := q.Get("staff"); v != "" {
		f.UserID = &v
	}
	if v := q.Get("issue"); v != "" {
		f.IssueCategory = &v
	}
	if v := q.Get("rating"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 5 {
			f.Rating = &n
		}
	}
	return f
}

func (h *ReportHandler) Report(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Report(r.Context(), parseReportFilter(r))
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}

func (h *ReportHandler) SKMDetail(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	out, err := h.svc.SKMDetail(r.Context(), parseReportFilter(r), page, size)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	ok(w, out)
}

func (h *ReportHandler) ExportExcel(w http.ResponseWriter, r *http.Request) {
	f := parseReportFilter(r)
	data, err := h.svc.ExportExcel(r.Context(), f)
	if err != nil {
		renderDomainError(w, err)
		return
	}
	filename := fmt.Sprintf("laporan-antrian-%s_%s.xlsx",
		f.DateFrom.Format("20060102"), f.DateTo.Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
