// Package events implements the Andon Board - a central pub/sub event bus
// for reactive communication between factory components.
package events

import (
	"context"
	"sync"
	"time"
)

// EventType represents the type of event flowing through the Andon Board
type EventType string

const (
	// Job lifecycle events
	EventJobCreated   EventType = "job.created"
	EventJobQueued    EventType = "job.queued"
	EventJobStarted   EventType = "job.started"
	EventJobCompleted EventType = "job.completed"
	EventJobFailed    EventType = "job.failed"

	// Station lifecycle events
	EventStationReady   EventType = "station.ready"
	EventStationBusy    EventType = "station.busy"
	EventStationOffline EventType = "station.offline"

	// Step execution (DAG) events
	EventStepQueued      EventType = "step.queued"
	EventStepStarted     EventType = "step.started"
	EventStepCompleted   EventType = "step.completed"
	EventStepFailed      EventType = "step.failed"
	EventDependenciesMet EventType = "step.dependencies_met"

	// Merge operation events
	EventMergeReady    EventType = "merge.ready"
	EventMergeStarted  EventType = "merge.started"
	EventMergeCompleted EventType = "merge.completed"
	EventMergeConflict EventType = "merge.conflict"

	// Operator lifecycle events
	EventOperatorSpawned EventType = "operator.spawned"
	EventOperatorIdle    EventType = "operator.idle"
	EventOperatorStuck   EventType = "operator.stuck"
	EventOperatorHandoff EventType = "operator.handoff"

	// Quality & Rework events
	EventQualityFailed EventType = "quality.failed"
	EventReworkNeeded  EventType = "rework.needed"

	// System events
	EventHeartbeat       EventType = "system.heartbeat"
	EventFactoryShutdown EventType = "system.shutdown"
	EventHealthOK        EventType = "system.health_ok"
	EventCleanupDone     EventType = "system.cleanup_done"

	// Legacy event types for backward compatibility
	EvtStationStarted   EventType = "station.started"
	EvtStationInspecting EventType = "station.inspecting"
	EvtStationDone      EventType = "station.done"
	EvtStationFailed    EventType = "station.failed"
	EvtMerging          EventType = "merge.merging"
	EvtDone             EventType = "job.done"
)

// Event represents a single event on the Andon Board
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Source    string                 `json:"source"`  // Who emitted this
	Subject   string                 `json:"subject"` // What this is about
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// Handler is a function that processes an event
type Handler func(Event)

// EventBus is the Andon Board - an in-memory pub/sub system
type EventBus struct {
	mu         sync.RWMutex
	handlers   map[EventType][]Handler
	all        []Handler     // Handlers that want all events
	buffer     int           // Channel buffer size
	deadLetter []Event       // Dropped events
	eventLog   EventLogger   // Optional event logger
}

// EventLogger is an optional interface for logging events
type EventLogger interface {
	LogEvent(event Event) error
}

// NewEventBus creates a new Andon Board (event bus)
func NewEventBus(bufferSize int, logger EventLogger) *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]Handler),
		all:      make([]Handler, 0),
		buffer:   bufferSize,
		deadLetter: make([]Event, 0),
		eventLog: logger,
	}
}

// Subscribe registers a handler for specific event types
// Returns an unsubscribe function
func (b *EventBus) Subscribe(eventType EventType, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		handlers := b.handlers[eventType]
		for i, h := range handlers {
			if &h == &handler {
				b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// SubscribeAll registers a handler for all events (for logging/metrics)
// Returns an unsubscribe function
func (b *EventBus) SubscribeAll(handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.all = append(b.all, handler)

	index := len(b.all) - 1

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if index < len(b.all) {
			b.all = append(b.all[:index], b.all[index+1:]...)
		}
	}
}

// Publish emits an event to all subscribers (non-blocking)
func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Log event if logger is set
	if b.eventLog != nil {
		_ = b.eventLog.LogEvent(event)
	}

	// Call handlers that subscribe to all events
	for _, handler := range b.all {
		go handler(event)
	}

	// Call handlers for this specific event type
	if handlers, ok := b.handlers[event.Type]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}
}

// PublishAsync emits an event without blocking, logs to dead letter if fails
func (b *EventBus) PublishAsync(event Event) {
	// For v1, this is the same as Publish
	// In future, could use a buffered channel and handle overflow
	b.Publish(event)
}

// WaitFor blocks until an event of the given type is received
func (b *EventBus) WaitFor(ctx context.Context, eventType EventType, timeout time.Duration) (Event, error) {
	resultChan := make(chan Event, 1)
	errorChan := make(chan error, 1)

	unsubscribe := b.Subscribe(eventType, func(e Event) {
		select {
		case resultChan <- e:
		default:
		}
	})
	defer unsubscribe()

	select {
	case <-ctx.Done():
		return Event{}, ctx.Err()
	case event := <-resultChan:
		return event, nil
	case <-time.After(timeout):
		return Event{}, context.DeadlineExceeded
	case err := <-errorChan:
		return Event{}, err
	}
}

// GetDeadLetter returns events that were dropped
func (b *EventBus) GetDeadLetter() []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.deadLetter
}

// ClearDeadLetter clears the dead letter queue
func (b *EventBus) ClearDeadLetter() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.deadLetter = make([]Event, 0)
}

// Emit helper to create and publish an event
func (b *EventBus) Emit(eventType EventType, source, subject string, payload map[string]interface{}) Event {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Source:    source,
		Subject:   subject,
		Payload:   payload,
	}
	b.Publish(event)
	return event
}

// Legacy helper functions for backward compatibility

// StationStarted creates a station started event
func StationStarted(lineName, stationName string) Event {
	return Event{
		Type:        EvtStationStarted,
		Timestamp:   time.Now().Unix(),
		Source:      lineName,
		Subject:     stationName,
		Payload:     map[string]interface{}{"line_name": lineName, "station_name": stationName},
	}
}

// StationInspecting creates a station inspecting event
func StationInspecting(lineName, stationName string) Event {
	return Event{
		Type:        EvtStationInspecting,
		Timestamp:   time.Now().Unix(),
		Source:      lineName,
		Subject:     stationName,
		Payload:     map[string]interface{}{"line_name": lineName, "station_name": stationName},
	}
}

// StationDone creates a station done event
func StationDone(lineName, stationName string, duration time.Duration, output string, retries int) Event {
	return Event{
		Type:      EvtStationDone,
		Timestamp: time.Now().Unix(),
		Source:    lineName,
		Subject:   stationName,
		Payload: map[string]interface{}{
			"line_name":    lineName,
			"station_name": stationName,
			"duration":     duration.Milliseconds(),
			"output":       output,
			"retries":      retries,
		},
	}
}

// StationFailed creates a station failed event
func StationFailed(lineName, stationName string, duration time.Duration, err error, retries int) Event {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	return Event{
		Type:      EvtStationFailed,
		Timestamp: time.Now().Unix(),
		Source:    lineName,
		Subject:   stationName,
		Payload: map[string]interface{}{
			"line_name":    lineName,
			"station_name": stationName,
			"duration":     duration.Milliseconds(),
			"error":        errorMsg,
			"retries":      retries,
		},
	}
}

// Merging creates a merging event
func Merging() Event {
	return Event{
		Type:      EvtMerging,
		Timestamp: time.Now().Unix(),
	}
}

// Done creates a done event
func Done(output string) Event {
	return Event{
		Type:      EvtDone,
		Timestamp: time.Now().Unix(),
		Payload:   map[string]interface{}{"output": output},
	}
}
