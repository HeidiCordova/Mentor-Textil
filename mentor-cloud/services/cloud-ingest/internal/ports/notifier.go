package ports

import "context"

// Notifier broadcasts configuration changes to connected edge devices.
type Notifier interface {
	BroadcastVariables(ctx context.Context, empresaID int64, vars interface{})
}
