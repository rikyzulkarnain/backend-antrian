package service

import (
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
)

type ReportService struct {
	repo *repository.ReportRepository
}

// reportRepo is the subset of repository methods the service needs. Declared
// inline via the concrete type below; kept simple to mirror AnalyticsService.

func NewReportService(repo *repository.ReportRepository) *ReportService {
	return &ReportService{repo: repo}
}

// unsurLabels maps an SKM issue_category to its PermenPAN "unsur" name. The
// current survey only records one complaint category, so the IKM element
// breakdown reflects complaint frequency per unsur rather than per-unsur scores.
var unsurLabels = []struct{ Key, Label string }{
	{"PROSEDUR", "Sistem, Mekanisme & Prosedur"},
	{"WAKTU", "Waktu Penyelesaian"},
	{"PETUGAS", "Perilaku & Kompetensi Pelaksana"},
	{"FASILITAS", "Sarana & Prasarana"},
	{"LAINNYA", "Lainnya"},
}

// ikmGrade converts an avg rating (1–5) into the PermenPAN RB index (0–100)
// and service-quality grade. Returns ("-","") when there are no ratings.
func ikmFromRating(avgRating float64, ratingCount int) (index float64, grade, label string) {
	if ratingCount == 0 {
		return 0, "-", "Belum ada data"
	}
	index = (avgRating / 5.0) * 100.0
	switch {
	case index >= 88.31:
		return index, "A", "Sangat Baik"
	case index >= 76.61:
		return index, "B", "Baik"
	case index >= 65.00:
		return index, "C", "Kurang Baik"
	default:
		return index, "D", "Tidak Baik"
	}
}

func (s *ReportService) buildIKM(summary domain.ReportSummary, issues []domain.IssueDistribution) domain.IKMReport {
	idx, grade, label := ikmFromRating(summary.AvgRating, summary.RatingCount)
	counts := map[string]int{}
	for _, i := range issues {
		counts[i.Category] += i.Count
	}
	elements := make([]domain.IKMElement, 0, len(unsurLabels))
	for _, u := range unsurLabels {
		elements = append(elements, domain.IKMElement{Key: u.Key, Label: u.Label, Issues: counts[u.Key]})
	}
	return domain.IKMReport{
		Index:       idx,
		Grade:       grade,
		GradeLabel:  label,
		RatingCount: summary.RatingCount,
		Elements:    elements,
	}
}

// Report assembles the on-screen dashboard payload for the filtered period.
func (s *ReportService) Report(ctx context.Context, f domain.ReportFilter) (*domain.ReportBundle, error) {
	summary, err := s.repo.Summary(ctx, f)
	if err != nil {
		return nil, err
	}
	byService, err := s.repo.ByService(ctx, f)
	if err != nil {
		return nil, err
	}
	trend, err := s.repo.Trend(ctx, f)
	if err != nil {
		return nil, err
	}
	staff, err := s.repo.Staff(ctx, f)
	if err != nil {
		return nil, err
	}
	issues, err := s.repo.Issues(ctx, f)
	if err != nil {
		return nil, err
	}
	complaints, err := s.repo.Complaints(ctx, f)
	if err != nil {
		return nil, err
	}
	guests, err := s.repo.GuestBook(ctx, f)
	if err != nil {
		return nil, err
	}
	skipped, err := s.repo.Skipped(ctx, f)
	if err != nil {
		return nil, err
	}

	ikm := s.buildIKM(summary, issues)
	summary.IKMIndex = ikm.Index
	summary.IKMGrade = ikm.Grade
	summary.IKMGradeLabel = ikm.GradeLabel

	return &domain.ReportBundle{
		Summary:    summary,
		ByService:  byService,
		Trend:      trend,
		Staff:      staff,
		IKM:        ikm,
		Issues:     issues,
		Complaints: complaints,
		GuestBook:  guests,
		Skipped:    skipped,
	}, nil
}

// SKMDetail returns one paginated page of survey responses.
func (s *ReportService) SKMDetail(ctx context.Context, f domain.ReportFilter, page, size int) (*domain.SKMDetailPage, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 200 {
		size = 25
	}
	total, err := s.repo.SKMDetailCount(ctx, f)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.SKMDetail(ctx, f, size, (page-1)*size)
	if err != nil {
		return nil, err
	}
	return &domain.SKMDetailPage{Items: items, Total: total, Page: page, Size: size}, nil
}

// ExportExcel renders the full report as a multi-sheet .xlsx workbook.
func (s *ReportService) ExportExcel(ctx context.Context, f domain.ReportFilter) ([]byte, error) {
	bundle, err := s.Report(ctx, f)
	if err != nil {
		return nil, err
	}
	// SKM detail can be large; pull a generous first page for the export.
	detail, err := s.SKMDetail(ctx, f, 1, 200)
	if err != nil {
		return nil, err
	}

	xl := excelize.NewFile()
	defer func() { _ = xl.Close() }()

	bold, _ := xl.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	periode := fmt.Sprintf("Periode: %s s/d %s",
		f.DateFrom.Format("02-01-2006"), f.DateTo.Format("02-01-2006"))

	// --- Ringkasan (rename default sheet) ---
	const s1 = "Ringkasan"
	xl.SetSheetName("Sheet1", s1)
	sum := bundle.Summary
	rows := [][]any{
		{"Laporan Sistem Antrian — BBPJN Sumsel"},
		{periode},
		{},
		{"Metrik", "Nilai"},
		{"Total antrian", sum.Total},
		{"Selesai dilayani", sum.Completed},
		{"Terlewat (skipped)", sum.Skipped},
		{"Masih menunggu", sum.Waiting},
		{"Rata-rata waktu tunggu (menit)", sum.AvgWaitMin},
		{"Rata-rata waktu layanan (detik)", sum.AvgServeSec},
		{"Rating rata-rata", fmt.Sprintf("%.2f", sum.AvgRating)},
		{"Jumlah penilaian", sum.RatingCount},
		{"Indeks IKM", fmt.Sprintf("%.2f", sum.IKMIndex)},
		{"Mutu pelayanan", fmt.Sprintf("%s — %s", sum.IKMGrade, sum.IKMGradeLabel)},
	}
	writeSheet(xl, s1, rows)
	_ = xl.SetCellStyle(s1, "A1", "A1", bold)
	_ = xl.SetCellStyle(s1, "A4", "B4", bold)

	// --- Volume per Layanan ---
	addTable(xl, "Volume per Layanan", periode, bold,
		[]string{"Layanan", "Total", "Selesai", "Terlewat", "Rating rata-rata"},
		len(bundle.ByService), func(i int) []any {
			v := bundle.ByService[i]
			return []any{v.Service, v.Count, v.Completed, v.Skipped, fmt.Sprintf("%.2f", v.AvgRating)}
		})

	// --- Tren Harian ---
	addTable(xl, "Tren Harian", periode, bold,
		[]string{"Tanggal", "Total", "Selesai"},
		len(bundle.Trend), func(i int) []any {
			t := bundle.Trend[i]
			return []any{t.Date, t.Count, t.Completed}
		})

	// --- Kinerja Petugas ---
	addTable(xl, "Kinerja Petugas", periode, bold,
		[]string{"Petugas", "Loket", "Dilayani", "Rata-rata layanan (detik)", "Rating"},
		len(bundle.Staff), func(i int) []any {
			st := bundle.Staff[i]
			return []any{st.Name, st.Counter, st.Served, st.AvgServeSeconds, fmt.Sprintf("%.2f", st.Rating)}
		})

	// --- IKM ---
	ikmRows := [][]any{
		{"Laporan Indeks Kepuasan Masyarakat (IKM)"},
		{periode},
		{"Catatan: pendekatan dari 1 rating keseluruhan (skala 1–5), bukan 9 unsur penuh."},
		{},
		{"Indeks", fmt.Sprintf("%.2f", bundle.IKM.Index)},
		{"Mutu", fmt.Sprintf("%s — %s", bundle.IKM.Grade, bundle.IKM.GradeLabel)},
		{"Jumlah penilaian", bundle.IKM.RatingCount},
		{},
		{"Unsur Pelayanan", "Jumlah Keluhan"},
	}
	for _, e := range bundle.IKM.Elements {
		ikmRows = append(ikmRows, []any{e.Label, e.Issues})
	}
	xl.NewSheet("IKM")
	writeSheet(xl, "IKM", ikmRows)
	_ = xl.SetCellStyle("IKM", "A1", "A1", bold)
	_ = xl.SetCellStyle("IKM", "A9", "B9", bold)

	// --- Detail SKM (identitas pemohon ditampilkan penuh) ---
	addTable(xl, "Detail SKM", periode, bold,
		[]string{"No. Antrian", "Layanan", "Rating", "Komentar", "Kategori Masalah", "Nama Pemohon", "No. HP", "Petugas", "Loket", "Waktu Selesai"},
		len(detail.Items), func(i int) []any {
			d := detail.Items[i]
			return []any{d.QueueNumber, d.Service, d.Rating, deref(d.Comment), deref(d.IssueCategory),
				deref(d.RespondentName), deref(d.RespondentPhone), deref(d.Staff), deref(d.Counter),
				fmtTime(d.CompletedAt)}
		})

	// --- Keluhan ---
	addTable(xl, "Keluhan", periode, bold,
		[]string{"No. Antrian", "Layanan", "Rating", "Kategori", "Komentar", "Nama Pemohon", "No. HP", "Waktu"},
		len(bundle.Complaints), func(i int) []any {
			c := bundle.Complaints[i]
			return []any{c.QueueNumber, c.Service, c.Rating, deref(c.IssueCategory), deref(c.Comment),
				deref(c.RespondentName), deref(c.RespondentPhone), fmtTime(c.CompletedAt)}
		})

	// --- Buku Tamu ---
	addTable(xl, "Buku Tamu", periode, bold,
		[]string{"No. Antrian", "Nama Tamu", "Keperluan", "Layanan", "Status", "Waktu"},
		len(bundle.GuestBook), func(i int) []any {
			g := bundle.GuestBook[i]
			return []any{g.QueueNumber, deref(g.GuestName), deref(g.GuestPurpose), g.Service, g.Status, fmtTime(g.CreatedAt)}
		})

	// --- Antrian Terlewat ---
	addTable(xl, "Antrian Terlewat", periode, bold,
		[]string{"No. Antrian", "Layanan", "Loket", "Waktu"},
		len(bundle.Skipped), func(i int) []any {
			sk := bundle.Skipped[i]
			return []any{sk.QueueNumber, sk.Service, deref(sk.Counter), fmtTime(sk.CreatedAt)}
		})

	xl.SetActiveSheet(0)
	buf, err := xl.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// addTable creates a sheet with a title + period header then a bold header row
// and `n` data rows produced by `row`.
func addTable(xl *excelize.File, sheet, periode string, boldStyle int, headers []string, n int, row func(i int) []any) {
	xl.NewSheet(sheet)
	data := [][]any{
		{sheet},
		{periode},
		{},
	}
	hdr := make([]any, len(headers))
	for i, h := range headers {
		hdr[i] = h
	}
	data = append(data, hdr)
	for i := range n {
		data = append(data, row(i))
	}
	if n == 0 {
		data = append(data, []any{"(tidak ada data)"})
	}
	writeSheet(xl, sheet, data)
	_ = xl.SetCellStyle(sheet, "A1", "A1", boldStyle)
	lastCol, _ := excelize.ColumnNumberToName(len(headers))
	_ = xl.SetCellStyle(sheet, "A4", lastCol+"4", boldStyle)
}

// writeSheet writes rows starting at A1, each inner slice a row of cell values.
func writeSheet(xl *excelize.File, sheet string, rows [][]any) {
	for r, cols := range rows {
		for c, val := range cols {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				continue
			}
			_ = xl.SetCellValue(sheet, cell, val)
		}
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02-01-2006 15:04")
}
