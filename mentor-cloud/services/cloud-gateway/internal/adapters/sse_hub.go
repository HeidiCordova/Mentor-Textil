package adapters

import (
	"sync"

	"cloud-gateway/internal/domain"
)

// SSEHub distribuye comandos en tiempo real a los dispositivos conectados por SSE.
// Cada dispositivo puede tener múltiples conexiones simultáneas (reconexiones).
type SSEHub struct {
	mu   sync.RWMutex
	subs map[string][]chan domain.Command
}

func NewSSEHub() *SSEHub {
	return &SSEHub{subs: make(map[string][]chan domain.Command)}
}

// Subscribe registra un canal para deviceID.
// Devuelve el canal de lectura y una función unsub para limpiar al cerrar la conexión.
func (h *SSEHub) Subscribe(deviceID string) (<-chan domain.Command, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan domain.Command, 16)
	h.subs[deviceID] = append(h.subs[deviceID], ch)

	unsub := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		list := h.subs[deviceID]
		for i, c := range list {
			if c == ch {
				h.subs[deviceID] = append(list[:i], list[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, unsub
}

// Notify empuja un comando a todos los canales suscritos del dispositivo.
// Si el canal está lleno (consumidor lento) el evento se descarta:
// el comando ya está persistido en la DB y se entregará en el próximo poll.
func (h *SSEHub) Notify(cmd domain.Command) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[cmd.DeviceID] {
		select {
		case ch <- cmd:
		default:
		}
	}
}
