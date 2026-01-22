package liveevents

import (
	"errors"
	"strings"
	"sync"
)

const (
	StatusAccepted     = "accepted"
	StatusDeduplicated = "deduplicated"

	SourceAPI      = "api"
	SourceReplay   = "replay"
	SourceBackfill = "backfill"
)

const (
	DefaultBufferSize       = 50
	DefaultSubscriberBuffer = 16
)

type LiveEvent struct {
	MeterID        string  `json:"meter_id"`
	CustomerID     string  `json:"customer_id"`
	Value          float64 `json:"value"`
	RecordedAt     string  `json:"recorded_at"`
	IdempotencyKey string  `json:"idempotency_key"`
	Status         string  `json:"status"`
	Source         string  `json:"source"`
}

type Hub struct {
	mu               sync.RWMutex
	streams          map[string]*stream
	bufferSize       int
	subscriberBuffer int
}

type stream struct {
	mu     sync.Mutex
	buffer []LiveEvent
	subs   map[uint64]chan LiveEvent
	nextID uint64
}

type Subscription struct {
	hub       *Hub
	meterCode string
	id        uint64
	ch        chan LiveEvent
	once      sync.Once
}

func NewHub() *Hub {
	return &Hub{
		streams:          make(map[string]*stream),
		bufferSize:       DefaultBufferSize,
		subscriberBuffer: DefaultSubscriberBuffer,
	}
}

func (h *Hub) Publish(meterCode string, event LiveEvent) {
	if h == nil {
		return
	}
	code := strings.TrimSpace(meterCode)
	if code == "" {
		return
	}
	h.mu.RLock()
	stream := h.streams[code]
	h.mu.RUnlock()
	if stream == nil {
		return
	}

	stream.mu.Lock()
	stream.buffer = append(stream.buffer, event)
	if len(stream.buffer) > h.bufferSize {
		stream.buffer = stream.buffer[len(stream.buffer)-h.bufferSize:]
	}
	subs := make([]chan LiveEvent, 0, len(stream.subs))
	for _, ch := range stream.subs {
		subs = append(subs, ch)
	}
	stream.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *Hub) Subscribe(meterCode string) (*Subscription, []LiveEvent, error) {
	if h == nil {
		return nil, nil, errors.New("hub_unavailable")
	}
	code := strings.TrimSpace(meterCode)
	if code == "" {
		return nil, nil, errors.New("invalid_meter_code")
	}

	stream := h.ensureStream(code)
	stream.mu.Lock()
	if stream.subs == nil {
		stream.subs = make(map[uint64]chan LiveEvent)
	}
	id := stream.nextID
	stream.nextID++
	ch := make(chan LiveEvent, h.subscriberBuffer)
	stream.subs[id] = ch
	buffer := append([]LiveEvent(nil), stream.buffer...)
	stream.mu.Unlock()

	return &Subscription{
		hub:       h,
		meterCode: code,
		id:        id,
		ch:        ch,
	}, buffer, nil
}

func (h *Hub) ensureStream(meterCode string) *stream {
	h.mu.RLock()
	current := h.streams[meterCode]
	h.mu.RUnlock()
	if current != nil {
		return current
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	current = h.streams[meterCode]
	if current == nil {
		current = &stream{subs: make(map[uint64]chan LiveEvent)}
		h.streams[meterCode] = current
	}
	return current
}

func (h *Hub) unsubscribe(meterCode string, id uint64) {
	if h == nil {
		return
	}
	code := strings.TrimSpace(meterCode)
	if code == "" {
		return
	}

	h.mu.RLock()
	stream := h.streams[code]
	h.mu.RUnlock()
	if stream == nil {
		return
	}

	stream.mu.Lock()
	delete(stream.subs, id)
	remaining := len(stream.subs)
	stream.mu.Unlock()
	if remaining != 0 {
		return
	}

	h.mu.Lock()
	current := h.streams[code]
	if current != stream {
		h.mu.Unlock()
		return
	}
	stream.mu.Lock()
	empty := len(stream.subs) == 0
	stream.mu.Unlock()
	if empty {
		delete(h.streams, code)
	}
	h.mu.Unlock()
}

func (s *Subscription) Events() <-chan LiveEvent {
	if s == nil {
		return nil
	}
	return s.ch
}

func (s *Subscription) Close() {
	if s == nil || s.hub == nil {
		return
	}
	s.once.Do(func() {
		s.hub.unsubscribe(s.meterCode, s.id)
	})
}
