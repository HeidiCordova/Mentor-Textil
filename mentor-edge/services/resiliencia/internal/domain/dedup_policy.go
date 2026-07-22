package domain

import (
	"container/list"
	"sync"
)

type lruEntry struct {
	key string
}

type InMemoryDedup struct {
	processed map[string]*list.Element
	order     *list.List
	mu        sync.Mutex
	maxSize   int
}

func NewInMemoryDedup(maxSize int) *InMemoryDedup {
	return &InMemoryDedup{
		processed: make(map[string]*list.Element, maxSize),
		order:     list.New(),
		maxSize:   maxSize,
	}
}

func (d *InMemoryDedup) IsDuplicate(eventID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if el, ok := d.processed[eventID]; ok {
		d.order.MoveToFront(el)
		return true
	}
	return false
}

func (d *InMemoryDedup) MarkProcessed(eventID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if el, ok := d.processed[eventID]; ok {
		d.order.MoveToFront(el)
		return
	}

	if d.order.Len() >= d.maxSize {
		oldest := d.order.Back()
		if oldest != nil {
			d.order.Remove(oldest)
			delete(d.processed, oldest.Value.(*lruEntry).key)
		}
	}

	el := d.order.PushFront(&lruEntry{key: eventID})
	d.processed[eventID] = el
}
