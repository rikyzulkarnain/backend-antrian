package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/domain"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/sse"
)

// fakeQueueStore captures call args and returns scripted results.
type fakeQueueStore struct {
	created      *domain.Queue
	calledNext   *domain.Queue
	updated      *domain.Queue
	rated        *domain.Queue
	gotByID      *domain.Queue
	createErr    error
	callNextErr  error
	updateErr    error
	rateErr      error
	getErr       error
	listErr      error

	lastCreateService string
	lastCallCounter   int
	lastCallUser      string
	lastCallService   string
	lastUpdateID      string
	lastUpdateStatus  domain.QueueStatus
	lastUpdateAllowed []domain.QueueStatus
	lastRateID        string
	lastRateScore     int
}

func (f *fakeQueueStore) List(_ context.Context, _ repository.ListFilter) ([]domain.Queue, error) {
	return nil, f.listErr
}
func (f *fakeQueueStore) Get(_ context.Context, _ string) (*domain.Queue, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.gotByID, nil
}
func (f *fakeQueueStore) Create(_ context.Context, serviceType string) (*domain.Queue, error) {
	f.lastCreateService = serviceType
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.created, nil
}
func (f *fakeQueueStore) CallNext(_ context.Context, counterID int, userID, serviceType string) (*domain.Queue, error) {
	f.lastCallCounter = counterID
	f.lastCallUser = userID
	f.lastCallService = serviceType
	if f.callNextErr != nil {
		return nil, f.callNextErr
	}
	return f.calledNext, nil
}
func (f *fakeQueueStore) UpdateStatus(_ context.Context, id string, newStatus domain.QueueStatus, allowed []domain.QueueStatus) (*domain.Queue, error) {
	f.lastUpdateID = id
	f.lastUpdateStatus = newStatus
	f.lastUpdateAllowed = allowed
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updated, nil
}
func (f *fakeQueueStore) Rate(_ context.Context, id string, in domain.RateInput) (*domain.Queue, error) {
	f.lastRateID = id
	f.lastRateScore = in.Rating
	if f.rateErr != nil {
		return nil, f.rateErr
	}
	return f.rated, nil
}

// capturingBroker records all events published through it.
type capturingBroker struct {
	mu     sync.Mutex
	events []sse.Event
}

func (c *capturingBroker) Publish(ev sse.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}
func (c *capturingBroker) names() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.events))
	for i, ev := range c.events {
		out[i] = ev.Name
	}
	return out
}

func newQueueSvc(store *fakeQueueStore) (*QueueService, *capturingBroker) {
	b := &capturingBroker{}
	return NewQueueService(store, b), b
}

func TestQueueService_Create_RejectsInvalidServiceType(t *testing.T) {
	svc, broker := newQueueSvc(&fakeQueueStore{})
	_, err := svc.Create(context.Background(), "INVALID")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("want ErrInvalidInput, got %v", err)
	}
	if len(broker.events) != 0 {
		t.Errorf("expected no events on validation failure")
	}
}

func TestQueueService_Create_NormalizesAndPublishesEvent(t *testing.T) {
	store := &fakeQueueStore{created: &domain.Queue{ID: "q1", QueueNumber: "A-01"}}
	svc, broker := newQueueSvc(store)

	q, err := svc.Create(context.Background(), "  umum ")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if q.ID != "q1" {
		t.Errorf("returned queue id = %q", q.ID)
	}
	if store.lastCreateService != "UMUM" {
		t.Errorf("service_type not normalized: %q", store.lastCreateService)
	}
	if got := broker.names(); len(got) != 1 || got[0] != "queue.created" {
		t.Errorf("events = %v, want [queue.created]", got)
	}
}

func TestQueueService_CallNext_ValidatesCounterAndUser(t *testing.T) {
	svc, broker := newQueueSvc(&fakeQueueStore{})

	if _, err := svc.CallNext(context.Background(), 0, "u1", ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("counter=0 should be invalid, got %v", err)
	}
	if _, err := svc.CallNext(context.Background(), 1, "", ""); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("missing user should be unauthorized, got %v", err)
	}
	if _, err := svc.CallNext(context.Background(), 1, "u1", "BOGUS"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("bogus service should be invalid, got %v", err)
	}
	if len(broker.events) != 0 {
		t.Errorf("validation failures must not publish events")
	}
}

func TestQueueService_CallNext_PublishesQueueCalled(t *testing.T) {
	store := &fakeQueueStore{calledNext: &domain.Queue{ID: "q1", Status: domain.StatusCalling}}
	svc, broker := newQueueSvc(store)

	q, err := svc.CallNext(context.Background(), 3, "u1", "lab")
	if err != nil {
		t.Fatalf("CallNext: %v", err)
	}
	if q.ID != "q1" {
		t.Errorf("returned id = %q", q.ID)
	}
	if store.lastCallCounter != 3 || store.lastCallUser != "u1" || store.lastCallService != "LAB" {
		t.Errorf("repo args wrong: counter=%d user=%q svc=%q",
			store.lastCallCounter, store.lastCallUser, store.lastCallService)
	}
	if got := broker.names(); len(got) != 1 || got[0] != "queue.called" {
		t.Errorf("events = %v", got)
	}
}

func TestQueueService_CallNext_EmptyReturnsNotFound(t *testing.T) {
	store := &fakeQueueStore{callNextErr: domain.ErrNotFound}
	svc, broker := newQueueSvc(store)

	_, err := svc.CallNext(context.Background(), 1, "u1", "")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	if len(broker.events) != 0 {
		t.Errorf("no event should fire when queue empty")
	}
}

func TestQueueService_Complete_AllowsCallingOrServing(t *testing.T) {
	store := &fakeQueueStore{updated: &domain.Queue{ID: "q1", Status: domain.StatusCompleted}}
	svc, broker := newQueueSvc(store)

	_, err := svc.Complete(context.Background(), "q1")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if store.lastUpdateStatus != domain.StatusCompleted {
		t.Errorf("status arg = %q", store.lastUpdateStatus)
	}
	allowed := map[domain.QueueStatus]bool{}
	for _, s := range store.lastUpdateAllowed {
		allowed[s] = true
	}
	if !allowed[domain.StatusCalling] || !allowed[domain.StatusServing] {
		t.Errorf("Complete must allow calling+serving, got %v", store.lastUpdateAllowed)
	}
	if got := broker.names(); len(got) != 1 || got[0] != "queue.completed" {
		t.Errorf("events = %v", got)
	}
}

func TestQueueService_Skip_AllowedFromWaitingOrCalling(t *testing.T) {
	store := &fakeQueueStore{updated: &domain.Queue{ID: "q1", Status: domain.StatusSkipped}}
	svc, broker := newQueueSvc(store)

	if _, err := svc.Skip(context.Background(), "q1"); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	allowed := map[domain.QueueStatus]bool{}
	for _, s := range store.lastUpdateAllowed {
		allowed[s] = true
	}
	if !allowed[domain.StatusWaiting] || !allowed[domain.StatusCalling] {
		t.Errorf("Skip must allow waiting+calling, got %v", store.lastUpdateAllowed)
	}
	if got := broker.names(); len(got) != 1 || got[0] != "queue.skipped" {
		t.Errorf("events = %v", got)
	}
}

func TestQueueService_Recall_NoStateChange_RepublishesCalled(t *testing.T) {
	store := &fakeQueueStore{gotByID: &domain.Queue{ID: "q1", Status: domain.StatusServing}}
	svc, broker := newQueueSvc(store)

	if _, err := svc.Recall(context.Background(), "q1"); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if got := broker.names(); len(got) != 1 || got[0] != "queue.called" {
		t.Errorf("events = %v", got)
	}
}

func TestQueueService_Recall_RejectsWaitingStatus(t *testing.T) {
	store := &fakeQueueStore{gotByID: &domain.Queue{ID: "q1", Status: domain.StatusWaiting}}
	svc, broker := newQueueSvc(store)

	_, err := svc.Recall(context.Background(), "q1")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("want ErrConflict, got %v", err)
	}
	if len(broker.events) != 0 {
		t.Errorf("no event on conflict")
	}
}

func TestQueueService_Rate_RejectsOutOfRange(t *testing.T) {
	svc, _ := newQueueSvc(&fakeQueueStore{})

	for _, bad := range []int{0, -1, 6, 100} {
		_, err := svc.Rate(context.Background(), "q1", domain.RateInput{Rating: bad})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("Rate(%d): want ErrInvalidInput, got %v", bad, err)
		}
	}
}

func TestQueueService_Rate_HappyPath(t *testing.T) {
	store := &fakeQueueStore{rated: &domain.Queue{ID: "q1", Status: domain.StatusCompleted}}
	svc, broker := newQueueSvc(store)

	if _, err := svc.Rate(context.Background(), "q1", domain.RateInput{Rating: 5}); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if store.lastRateScore != 5 {
		t.Errorf("rating arg = %d", store.lastRateScore)
	}
	if got := broker.names(); len(got) != 1 || got[0] != "queue.rated" {
		t.Errorf("events = %v", got)
	}
}
