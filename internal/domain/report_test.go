package domain

import (
	"strings"
	"testing"
	"time"
)

func TestReportFilterWhere_DateOnly(t *testing.T) {
	f := ReportFilter{DateFrom: time.Now().AddDate(0, 0, -7), DateTo: time.Now()}
	clause, args := f.Where("q", 1)
	if !strings.Contains(clause, "q.created_at::date BETWEEN $1 AND $2") {
		t.Fatalf("unexpected clause: %s", clause)
	}
	if len(args) != 2 {
		t.Fatalf("want 2 args, got %d", len(args))
	}
}

func TestReportFilterWhere_AllDimensions(t *testing.T) {
	svc := "UMUM"
	counter := 2
	uid := "u-1"
	issue := "PETUGAS"
	rating := 3
	f := ReportFilter{
		DateFrom:      time.Now().AddDate(0, 0, -7),
		DateTo:        time.Now(),
		ServiceType:   &svc,
		Counter:       &counter,
		UserID:        &uid,
		IssueCategory: &issue,
		Rating:        &rating,
	}
	clause, args := f.Where("q", 1)

	// Placeholders must be sequential and contiguous from $1..$7.
	for _, want := range []string{"$1", "$2", "$3", "$4", "$5", "$6", "$7"} {
		if !strings.Contains(clause, want) {
			t.Errorf("clause missing placeholder %s: %s", want, clause)
		}
	}
	if len(args) != 7 {
		t.Fatalf("want 7 args, got %d", len(args))
	}
	for _, frag := range []string{"service_type", "counter_id", "user_id", "issue_category", "rating"} {
		if !strings.Contains(clause, "q."+frag) {
			t.Errorf("clause missing %s: %s", frag, clause)
		}
	}
}

func TestReportFilterWhere_StartIdxOffsetAndNoPrefix(t *testing.T) {
	svc := "LAB"
	f := ReportFilter{DateFrom: time.Now(), DateTo: time.Now(), ServiceType: &svc}
	clause, args := f.Where("", 5)
	if !strings.Contains(clause, "created_at::date BETWEEN $5 AND $6") {
		t.Fatalf("offset not honored: %s", clause)
	}
	if !strings.Contains(clause, "service_type = $7") {
		t.Fatalf("dimension placeholder not offset: %s", clause)
	}
	if strings.Contains(clause, "q.") {
		t.Fatalf("empty prefix should not produce alias: %s", clause)
	}
	if len(args) != 3 {
		t.Fatalf("want 3 args, got %d", len(args))
	}
}
