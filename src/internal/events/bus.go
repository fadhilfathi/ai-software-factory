// Package events provides an in-process pub/sub bus for execution
// lifecycle events. Sprint 5 (TASK-503) is the producer: every
// successful state machine transition emits an Event. TASK-505
// (Deliverable Capture) and TASK-506 (Monitoring Dashboard) are
// the consumers.
//
// Delivery semantics (per Lead, 2026-06-13):
//
//   - At-most-once. Publish is non-blocking; a slow consumer whose
//     channel is full has the event dropped. SSE consumers reconnect
//     with last-event-id and replay from Last().
//   - Per-project ring buffer (200 events) on Last() for replay.
//   - ProjectID is the partition key: Subscribe and Last filter on
//     it. The bus does not enforce tenant isolation beyond that —
//     cross-tenant access control is the consumer's responsibility
//     (it must verify event.ProjectID == caller's project before
//     exposing the event to the user).
//
// Sprint 6+ may add: persistent publish, ack/consumer groups, and
// per-event TTL. The interface is deliberately small to make those
// additions a non-breaking extension.
package events

import (
	"sync"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// Event is the payload published on every successful execution
// status transition. It is the single contract between the state
// manager (TASK-503) and its consumers (TASK-505, TASK-506).
//
// ID is monotonic per-process; consumers can use it as a
// last-event-id to resume after a disconnect (see Bus.Last).
//
// ErrorMessage is non-nil iff To == model.ExecutionStatusFailed.
// It carries the failure detail (mock error in tests, real runtime
// error in production). It is nil for all other transitions.
type Event struct {
	ID           uint64 // monotonic per-process; usable as last-event-id
	ExecutionID  uuid.UUID
	ProjectID    uuid.UUID
	TaskID       uuid.UUID
	AgentID      uuid.UUID
	From         model.ExecutionStatus
	To           model.ExecutionStatus
	ErrorMessage *string
	At           time.Time
}

// Bus is the Sprint 5 in-process pub/sub for execution events.
// See the package comment for delivery semantics.
//
// The interface is small on purpose: one way in (Publish), one way
// to listen (Subscribe → chan + unsubscribe), one way to replay
// (Last). Sprint 6+ may extend with persistence, ack, and TTL.
type Bus interface {
	// Publish broadcasts event to all current subscribers of
	// event.ProjectID. Non-blocking: a subscriber whose channel
	// is full has the event dropped (at-most-once). The returned
	// bool is true iff every subscriber's channel accepted the
	// event; false indicates at least one subscriber dropped it.
	// Publish never blocks the caller. If event.ID == 0 the bus
	// assigns a monotonic ID; if event.At is zero the bus stamps
	// the publish time.
	Publish(event Event) bool

	// Subscribe returns a receive-only channel of events for
	// projectID and an unsubscribe function. The caller MUST call
	// unsubscribe when done to release the slot. The channel is
	// buffered (size 64 — see subscriberBufferSize) and is closed
	// by unsubscribe. Multiple subscribers on the same projectID
	// are supported; each gets its own channel.
	Subscribe(projectID uuid.UUID) (<-chan Event, func())

	// Last returns the most recent n events for projectID, in
	// chronological order (oldest first). n is clamped to the
	// ring buffer size (200). Used by SSE consumers on reconnect
	// to resume from last-event-id; the consumer filters the
	// returned window to skip events <= its last seen ID.
	// Returns nil if projectID has never had a published event.
	Last(projectID uuid.UUID, n int) []Event
}

// Compile-time assertion that *MemoryBus implements Bus. If the
// interface and implementation drift, this fails at build time
// rather than at the first Publish call.
var _ Bus = (*MemoryBus)(nil)

// NewMemoryBus constructs the in-process Bus. The returned *MemoryBus
// is safe for concurrent use. The ring buffer is 200 events per
// project; the subscriber channel is 64 events deep.
//
// main.go creates one *MemoryBus at process start and passes it to
// every producer (currently ExecutionService) and to any consumer
// (TASK-505 DeliverableService, TASK-506 SSE handler).
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		bufferSize:  200,
		subs:        make(map[uuid.UUID][]*memorySub),
		perProject:  make(map[uuid.UUID]*ringBuffer),
		nextEventID: 1,
	}
}

// MemoryBus is the in-process implementation of Bus. See the Bus
// interface comment for the contract.
type MemoryBus struct {
	mu          sync.RWMutex
	bufferSize  int
	subs        map[uuid.UUID][]*memorySub
	perProject  map[uuid.UUID]*ringBuffer
	nextEventID uint64
}

type memorySub struct {
	projectID uuid.UUID
	ch        chan Event
}

// subscriberBufferSize is the per-subscriber channel depth. 64 is
// large enough for typical SSE bursts (a few state transitions in
// quick succession) but small enough that a stuck consumer drops
// events quickly rather than holding them in memory forever.
const subscriberBufferSize = 64

// Publish implements Bus. See Bus.Publish for the full contract.
func (b *MemoryBus) Publish(event Event) bool {
	b.mu.Lock()
	if event.ID == 0 {
		event.ID = b.nextEventID
		b.nextEventID++
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}

	// Snapshot subscribers under the lock so the channel sends
	// below happen lock-free.
	subs := append([]*memorySub(nil), b.subs[event.ProjectID]...)

	// Append to per-project ring buffer.
	rb, ok := b.perProject[event.ProjectID]
	if !ok {
		rb = newRingBuffer(b.bufferSize)
		b.perProject[event.ProjectID] = rb
	}
	rb.push(event)
	b.mu.Unlock()

	allDelivered := true
	for _, sub := range subs {
		select {
		case sub.ch <- event:
		default:
			// Slow consumer — drop the event. We log nothing
			// here because the bus has no logger dependency;
			// consumers that care about delivery failures
			// should monitor the bool return and surface it.
			allDelivered = false
		}
	}
	return allDelivered
}

// Subscribe implements Bus. See Bus.Subscribe for the full contract.
func (b *MemoryBus) Subscribe(projectID uuid.UUID) (<-chan Event, func()) {
	ch := make(chan Event, subscriberBufferSize)
	sub := &memorySub{projectID: projectID, ch: ch}

	b.mu.Lock()
	b.subs[projectID] = append(b.subs[projectID], sub)
	b.mu.Unlock()

	// sync.Once guards against the caller calling unsubscribe
	// twice (which would panic on close). The second call is a
	// silent no-op.
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			subs := b.subs[projectID]
			for i, s := range subs {
				if s == sub {
					// Order-preserving removal via append+slice.
					b.subs[projectID] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
			close(ch)
		})
	}
	return ch, unsubscribe
}

// Last implements Bus. See Bus.Last for the full contract.
func (b *MemoryBus) Last(projectID uuid.UUID, n int) []Event {
	if n <= 0 {
		return []Event{}
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	rb, ok := b.perProject[projectID]
	if !ok {
		return []Event{}
	}
	return rb.last(n, b.bufferSize)
}

// ringBuffer is a fixed-size FIFO of events. When full, the oldest
// event is evicted on push. It is NOT thread-safe on its own — the
// parent MemoryBus serializes access via its mutex.
type ringBuffer struct {
	buf  []Event
	head int  // index of next write slot
	full bool // true once we've wrapped at least once
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{buf: make([]Event, size)}
}

func (r *ringBuffer) push(e Event) {
	r.buf[r.head] = e
	r.head = (r.head + 1) % len(r.buf)
	if r.head == 0 {
		r.full = true
	}
}

// last returns up to n events in chronological order (oldest first).
// It clamps n to the number of events currently in the buffer.
// bufferSize is passed in to avoid exposing it as a struct field.
func (r *ringBuffer) last(n, bufferSize int) []Event {
	available := bufferSize
	if !r.full {
		available = r.head
	}
	if n > available {
		n = available
	}
	if n == 0 {
		return []Event{}
	}

	out := make([]Event, n)
	size := len(r.buf)
	for i := 0; i < n; i++ {
		// Walk back from r.head (the next-write slot) by i+1
		// steps to reach the (i+1)-th most recent event. Wrap
		// around the end of the slice.
		idx := r.head - 1 - i
		if idx < 0 {
			idx += size
		}
		// Reverse the assignment so the output is oldest-first.
		out[n-1-i] = r.buf[idx]
	}
	return out
}
