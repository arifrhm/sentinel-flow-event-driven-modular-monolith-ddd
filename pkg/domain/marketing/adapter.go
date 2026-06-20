package marketing

import "context"

// CRMAdapter defines the port interface for syncing contacts with external CRM platforms.
type CRMAdapter interface {
	SyncContact(ctx context.Context, contact *CRMContact) error
}
