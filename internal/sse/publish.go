package sse

import (
	"encoding/json"
	"log/slog"
)

// Publisher is satisfied by *Broker; the indirection lets services depend on
// an interface and keeps testing straightforward.
type Publisher interface {
	Publish(ev Event)
}

// PublishJSON marshals payload and publishes it under the given event name.
// Marshal failures are logged but never returned — SSE delivery must never
// block or fail a domain operation.
func PublishJSON(p Publisher, name string, payload any) {
	if p == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("sse marshal", "event", name, "err", err)
		return
	}
	p.Publish(Event{Name: name, Data: data})
}
