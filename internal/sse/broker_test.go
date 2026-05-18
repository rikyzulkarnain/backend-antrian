package sse

import (
	"sync"
	"testing"
	"time"
)

func recvWithTimeout(t *testing.T, ch chan Event, d time.Duration) (Event, bool) {
	t.Helper()
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return Event{}, false
	}
}

func TestBroker_PublishDeliversToSubscriber(t *testing.T) {
	b := NewBroker()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Publish(Event{Name: "queue.created", Data: []byte(`{"id":"x"}`)})

	ev, ok := recvWithTimeout(t, ch, 100*time.Millisecond)
	if !ok {
		t.Fatal("expected event, got timeout")
	}
	if ev.Name != "queue.created" || string(ev.Data) != `{"id":"x"}` {
		t.Errorf("unexpected event: %+v", ev)
	}
}

func TestBroker_PublishFansOutToMultipleSubscribers(t *testing.T) {
	b := NewBroker()
	a := b.Subscribe()
	c := b.Subscribe()
	defer b.Unsubscribe(a)
	defer b.Unsubscribe(c)

	b.Publish(Event{Name: "queue.called", Data: []byte("payload")})

	for i, ch := range []chan Event{a, c} {
		if _, ok := recvWithTimeout(t, ch, 100*time.Millisecond); !ok {
			t.Errorf("subscriber %d did not receive event", i)
		}
	}
}

func TestBroker_UnsubscribeStopsDelivery(t *testing.T) {
	b := NewBroker()
	ch := b.Subscribe()
	b.Unsubscribe(ch)

	// Channel must already be closed by Unsubscribe.
	if _, open := <-ch; open {
		t.Errorf("expected channel closed after unsubscribe")
	}

	// Publishing afterwards must not panic.
	b.Publish(Event{Name: "queue.skipped", Data: []byte("x")})
}

func TestBroker_SlowSubscriberDoesNotBlockPublish(t *testing.T) {
	b := NewBroker()
	slow := b.Subscribe()
	defer b.Unsubscribe(slow)

	// Buffer is 16; fire 100 events with no reader on `slow` and ensure
	// Publish returns within a tight bound.
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			b.Publish(Event{Name: "spam", Data: []byte("x")})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Publish blocked on slow subscriber")
	}
	wg.Wait()
}

func TestPublishJSON_NilPublisherIsNoop(t *testing.T) {
	// Must not panic.
	PublishJSON(nil, "queue.created", map[string]string{"id": "x"})
}

type collectingPublisher struct {
	mu     sync.Mutex
	events []Event
}

func (c *collectingPublisher) Publish(ev Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func TestPublishJSON_MarshalsPayload(t *testing.T) {
	c := &collectingPublisher{}
	PublishJSON(c, "queue.called", map[string]string{"id": "q1"})

	if len(c.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(c.events))
	}
	if c.events[0].Name != "queue.called" {
		t.Errorf("event name = %q", c.events[0].Name)
	}
	if string(c.events[0].Data) != `{"id":"q1"}` {
		t.Errorf("event data = %s", c.events[0].Data)
	}
}
