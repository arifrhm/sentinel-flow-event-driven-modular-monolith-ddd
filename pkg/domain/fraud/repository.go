package fraud

import (
	"sentinel-flow/pkg/broker"
)

// FraudRepository defines the port interface for persisting and retrieving screened events.
type FraudRepository interface {
	SaveEvent(event *broker.ScreenedEvent) error
	GetEventsByUserID(userID string) ([]*broker.ScreenedEvent, error)
	DeleteUserEvents(userID string) (int, error)
}
