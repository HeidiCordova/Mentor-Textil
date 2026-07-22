package ports

type SSEBroker interface {
	Subscribe(clientID string) <-chan []byte
	Unsubscribe(clientID string)
	Publish(eventType string, payload interface{})
	Start() error
	Shutdown()
}
