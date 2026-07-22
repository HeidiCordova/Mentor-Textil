package adapters

import (
	"encoding/json"
	"sync"
)

// BrowserSSEHub distributes real-time events to connected browser clients,
// scoped by empresa_id. Each empresa can have multiple concurrent browser tabs.
type BrowserSSEHub struct {
	mu   sync.RWMutex
	subs map[int64][]chan BrowserEvent
}

// BrowserEvent is a push notification sent from server to browser via SSE.
type BrowserEvent struct {
	Type      string          `json:"type"`
	LineaID   int64           `json:"linea_id,omitempty"`
	EmpresaID int64           `json:"empresa_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

func NewBrowserSSEHub() *BrowserSSEHub {
	return &BrowserSSEHub{subs: make(map[int64][]chan BrowserEvent)}
}

func (h *BrowserSSEHub) Subscribe(empresaID int64) (<-chan BrowserEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan BrowserEvent, 16)
	h.subs[empresaID] = append(h.subs[empresaID], ch)

	unsub := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		list := h.subs[empresaID]
		for i, c := range list {
			if c == ch {
				h.subs[empresaID] = append(list[:i], list[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, unsub
}

// Notify pushes an event to all browsers subscribed to the same empresa_id.
func (h *BrowserSSEHub) Notify(event BrowserEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[event.EmpresaID] {
		select {
		case ch <- event:
		default:
		}
	}
}
