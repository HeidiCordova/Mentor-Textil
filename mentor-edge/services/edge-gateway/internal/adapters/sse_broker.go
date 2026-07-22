package adapters

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"edge-gateway/internal/ports"

	"github.com/jackc/pgx/v5"
)

type SSEBrokerImpl struct {
	connStr  string
	ctx      context.Context
	cancel   context.CancelFunc
	clients  map[string]chan []byte
	mu       sync.RWMutex
	channels []string
}

func NewSSEBroker(connStr string) ports.SSEBroker {
	ctx, cancel := context.WithCancel(context.Background())
	return &SSEBrokerImpl{
		connStr:  connStr,
		ctx:      ctx,
		cancel:   cancel,
		clients:  make(map[string]chan []byte),
		channels: []string{"event_created", "stop_changed", "config_updated", "command_changed"},
	}
}

func (b *SSEBrokerImpl) Start() error {
	go b.dispatchLoop()
	return nil
}

func (b *SSEBrokerImpl) Shutdown() {
	b.cancel()
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, ch := range b.clients {
		close(ch)
		delete(b.clients, id)
	}
}

func (b *SSEBrokerImpl) Subscribe(clientID string) <-chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan []byte, 64)
	b.clients[clientID] = ch
	log.Printf("[SSE] client subscribed: %s (total: %d)", clientID, len(b.clients))
	return ch
}

func (b *SSEBrokerImpl) Unsubscribe(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.clients[clientID]; ok {
		close(ch)
		delete(b.clients, clientID)
		log.Printf("[SSE] client unsubscribed: %s (total: %d)", clientID, len(b.clients))
	}
}

func (b *SSEBrokerImpl) Publish(eventType string, payload interface{}) {
	data, err := json.Marshal(map[string]interface{}{
		"type":    eventType,
		"payload": payload,
		"ts":      time.Now().UTC().UnixMilli(),
	})
	if err != nil {
		log.Printf("[SSE] marshal error: %v", err)
		return
	}

	b.broadcast(data)
}

// dispatchLoop mantiene la conexión LISTEN activa y reconecta en caso de error.
func (b *SSEBrokerImpl) dispatchLoop() {
	for {
		if err := b.listenLoop(); err != nil {
			select {
			case <-b.ctx.Done():
				return
			case <-time.After(10 * time.Second):
				log.Printf("[SSE] reconectando tras error: %v", err)
			}
		} else {
			return // shutdown limpio via cancelación de contexto
		}
	}
}

// listenLoop abre una conexión pgx dedicada para LISTEN/NOTIFY.
// Retorna nil en shutdown limpio, error en caso de fallo de conexión.
func (b *SSEBrokerImpl) listenLoop() error {
	conn, err := pgx.Connect(b.ctx, b.connStr)
	if err != nil {
		if b.ctx.Err() != nil {
			return nil
		}
		return err
	}
	defer conn.Close(context.Background())

	for _, ch := range b.channels {
		if _, err := conn.Exec(b.ctx, "LISTEN "+ch); err != nil {
			return err
		}
		log.Printf("[SSE] listening on channel: %s", ch)
	}

	for {
		notification, err := conn.WaitForNotification(b.ctx)
		if err != nil {
			if b.ctx.Err() != nil {
				return nil // shutdown solicitado via ctx.cancel()
			}
			return err // error de red/conexión — reconectar
		}
		b.handleNotification(notification.Channel, notification.Payload)
	}
}

func (b *SSEBrokerImpl) handleNotification(channel, payload string) {
	sseType := channelToSSEType(channel)

	var payloadData interface{}
	if err := json.Unmarshal([]byte(payload), &payloadData); err != nil {
		payloadData = payload
	}

	data, err := json.Marshal(map[string]interface{}{
		"type":    sseType,
		"payload": payloadData,
		"ts":      time.Now().UTC().UnixMilli(),
	})
	if err != nil {
		log.Printf("[SSE] marshal error for channel %s: %v", channel, err)
		return
	}

	b.broadcast(data)
}

func (b *SSEBrokerImpl) broadcast(data []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for clientID, ch := range b.clients {
		select {
		case ch <- data:
		default:
			log.Printf("[SSE] client %s buffer full, dropping message", clientID)
		}
	}
}

func channelToSSEType(channel string) string {
	switch channel {
	case "event_created":
		return "event.created"
	case "stop_changed":
		return "stop.changed"
	case "config_updated":
		return "config.updated"
	case "command_changed":
		return "command.applied"
	default:
		return channel
	}
}
