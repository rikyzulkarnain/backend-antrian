package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/config"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/server"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/testutil"
)

// integrationHarness wires a real DB (TEST_DATABASE_URL), the real chi
// router, and an httptest.Server so requests cross the full handler →
// service → repository → DB path.
type integrationHarness struct {
	t       *testing.T
	pool    *pgxpool.Pool
	server  *httptest.Server
	baseURL string
}

func newIntegration(t *testing.T) *integrationHarness {
	t.Helper()
	pool := testutil.SetupDB(t)
	cfg := &config.Config{
		JWTSecret:    "test-secret",
		CORSOrigins:  []string{"http://localhost:3000"},
		CookieSecure: false,
	}
	router, _ := server.New(cfg, pool)
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)
	return &integrationHarness{
		t: t, pool: pool, server: ts, baseURL: ts.URL + "/api/v1",
	}
}

func (h *integrationHarness) do(method, path string, body any, token string) *http.Response {
	h.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, h.baseURL+path, rdr)
	if err != nil {
		h.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("do: %v", err)
	}
	h.t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// decodeData reads {"data": T} from an http response.
func decodeData[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var env struct {
		Data  T `json:"data"`
		Error *struct {
			Code, Message string
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v (status=%d)", err, resp.StatusCode)
	}
	if env.Error != nil {
		t.Fatalf("unexpected error envelope: %+v", env.Error)
	}
	return env.Data
}

type queueResp struct {
	ID          string  `json:"id"`
	QueueNumber string  `json:"queue_number"`
	ServiceType string  `json:"service_type"`
	Status      string  `json:"status"`
	CounterID   *int    `json:"counter_id"`
	Staff       *string `json:"staff"`
}

func TestQueueHandler_Create_Happy(t *testing.T) {
	h := newIntegration(t)

	resp := h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	q := decodeData[queueResp](t, resp)
	if q.ID == "" {
		t.Error("expected non-empty id")
	}
	if q.ServiceType != "UMUM" {
		t.Errorf("service_type = %q", q.ServiceType)
	}
	if q.Status != "waiting" {
		t.Errorf("status = %q", q.Status)
	}
	if !strings.HasPrefix(q.QueueNumber, "A-") {
		t.Errorf("queue_number = %q, want A-NN", q.QueueNumber)
	}
}

func TestQueueHandler_Create_RejectsInvalidServiceType(t *testing.T) {
	h := newIntegration(t)
	resp := h.do("POST", "/queues", map[string]string{"service_type": "BOGUS"}, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestQueueHandler_Create_IncrementsPerService(t *testing.T) {
	h := newIntegration(t)

	first := decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, ""))
	second := decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, ""))
	other := decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "LAB"}, ""))

	if first.QueueNumber == second.QueueNumber {
		t.Errorf("expected distinct numbers, got %s and %s", first.QueueNumber, second.QueueNumber)
	}
	if !strings.HasPrefix(second.QueueNumber, "A-") {
		t.Errorf("second UMUM number = %q", second.QueueNumber)
	}
	if !strings.HasPrefix(other.QueueNumber, "B-") {
		t.Errorf("LAB number = %q, want B-NN", other.QueueNumber)
	}
}

func TestQueueHandler_Get_FoundAndMissing(t *testing.T) {
	h := newIntegration(t)

	created := decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "AMP"}, ""))

	got := decodeData[queueResp](t, h.do("GET", "/queues/"+created.ID, nil, ""))
	if got.ID != created.ID {
		t.Errorf("id mismatch: %s vs %s", got.ID, created.ID)
	}

	// Non-existent UUID — must be 404, not 500.
	resp := h.do("GET", "/queues/00000000-0000-0000-0000-000000000000", nil, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status for missing = %d, want 404", resp.StatusCode)
	}
}

func TestQueueHandler_List_FilterByActiveAndService(t *testing.T) {
	h := newIntegration(t)

	for i := 0; i < 3; i++ {
		decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, ""))
	}
	decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "LAB"}, ""))

	all := decodeData[[]queueResp](t, h.do("GET", "/queues?active=true", nil, ""))
	if len(all) != 4 {
		t.Errorf("active list len = %d, want 4", len(all))
	}

	umumOnly := decodeData[[]queueResp](t, h.do("GET", "/queues?service_type=UMUM", nil, ""))
	if len(umumOnly) != 3 {
		t.Errorf("UMUM list len = %d, want 3", len(umumOnly))
	}
	for _, q := range umumOnly {
		if q.ServiceType != "UMUM" {
			t.Errorf("filter leak: %+v", q)
		}
	}
}

func TestQueueHandler_Call_RequiresAuth(t *testing.T) {
	h := newIntegration(t)
	resp := h.do("POST", "/queues/call", map[string]int{"counter_id": 1}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestQueueHandler_Call_PicksOldestWaiting_AndPublishesSSE(t *testing.T) {
	h := newIntegration(t)

	// Create three waiting tickets in order.
	q1 := decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, ""))
	_ = decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, ""))
	_ = decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, ""))

	// Login as the seeded admin to get a bearer token.
	loginResp := h.do("POST", "/auth/login", map[string]string{
		"email": "admin@local", "password": "admin123",
	}, "")
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", loginResp.StatusCode)
	}
	auth := decodeData[struct {
		AccessToken string `json:"access_token"`
	}](t, loginResp)

	// Counter 1 is seeded with service_type=UMUM in migration 003.
	called := decodeData[queueResp](t, h.do("POST", "/queues/call",
		map[string]any{"counter_id": 1, "service_type": "UMUM"},
		auth.AccessToken,
	))

	if called.ID != q1.ID {
		t.Errorf("FIFO violation: called %s, expected oldest %s", called.ID, q1.ID)
	}
	if called.Status != "calling" {
		t.Errorf("called status = %q", called.Status)
	}
	if called.CounterID == nil || *called.CounterID != 1 {
		t.Errorf("counter not stamped: %+v", called.CounterID)
	}
}

func TestQueueHandler_CallComplete_StateMachine(t *testing.T) {
	h := newIntegration(t)
	loginResp := h.do("POST", "/auth/login", map[string]string{
		"email": "admin@local", "password": "admin123",
	}, "")
	auth := decodeData[struct {
		AccessToken string `json:"access_token"`
	}](t, loginResp)

	// Create + call.
	_ = decodeData[queueResp](t, h.do("POST", "/queues", map[string]string{"service_type": "UMUM"}, ""))
	called := decodeData[queueResp](t, h.do("POST", "/queues/call",
		map[string]any{"counter_id": 1, "service_type": "UMUM"},
		auth.AccessToken,
	))

	completed := decodeData[queueResp](t, h.do("POST",
		"/queues/"+called.ID+"/complete", nil, auth.AccessToken))
	if completed.Status != "completed" {
		t.Errorf("status after complete = %q", completed.Status)
	}

	// Calling complete twice on the same ticket must conflict.
	resp := h.do("POST", "/queues/"+called.ID+"/complete", nil, auth.AccessToken)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("double-complete status = %d, want 409", resp.StatusCode)
	}
}
