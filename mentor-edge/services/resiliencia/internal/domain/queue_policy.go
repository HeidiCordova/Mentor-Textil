package domain

type DefaultQueuePolicy struct{}

func NewDefaultQueuePolicy() *DefaultQueuePolicy {
	return &DefaultQueuePolicy{}
}

func (q *DefaultQueuePolicy) ShouldAccept(event *EventBuffer) bool {
	switch event.EventType {
	case "OEE_SNAPSHOT", "CORTE", "ANOMALIA", "CALIBRACION", "ENERGIA_SNAPSHOT":
		return true
	default:
		return false
	}
}

func (q *DefaultQueuePolicy) GetPriority(event *EventBuffer) int {
	switch event.EventType {
	case "OEE_SNAPSHOT":
		return 0
	case "CORTE":
		return 1
	case "ANOMALIA":
		return 2
	case "CALIBRACION":
		return 3
	case "ENERGIA_SNAPSHOT":
		return 4
	default:
		return 99
	}
}
