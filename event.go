package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/ksuid"
)

// Event is a server sent event
type Event struct {
	ID    string
	Type  string
	Data  string
	Retry time.Duration
}

// NewEvent returns a new event
func NewEvent(eventType, data string) *Event {
	return &Event{
		ID:   ksuid.New().String(),
		Type: eventType,
		Data: data,
	}
}

func (e *Event) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "id:%s\n", e.ID)

	if e.Type != "" {
		fmt.Fprintf(&b, "event: %s\n", e.Type)
	}

	if e.Retry != 0 {
		fmt.Fprintf(&b, "retry: %d\n", e.Retry)
	}

	fmt.Fprintf(&b, "data:%s\n\n", e.Data)
	return b.String()
}
