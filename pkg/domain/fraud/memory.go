package fraud

import (
	"sync"

	"sentinel-flow/pkg/broker"
)

type MemoryFraudRepository struct {
	mu         sync.RWMutex
	events     map[string][]*broker.ScreenedEvent
	crmDeleter func(userID string)
}

func NewMemoryFraudRepository(crmDeleter func(userID string)) *MemoryFraudRepository {
	return &MemoryFraudRepository{
		events:     make(map[string][]*broker.ScreenedEvent),
		crmDeleter: crmDeleter,
	}
}

func (m *MemoryFraudRepository) SaveEvent(event *broker.ScreenedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events[event.UserID] = append(m.events[event.UserID], event)
	return nil
}

func (m *MemoryFraudRepository) GetEventsByUserID(userID string) ([]*broker.ScreenedEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	evs, ok := m.events[userID]
	if !ok {
		return []*broker.ScreenedEvent{}, nil
	}
	copied := make([]*broker.ScreenedEvent, len(evs))
	copy(copied, evs)
	return copied, nil
}

func (m *MemoryFraudRepository) DeleteUserEvents(userID string) (int, error) {
	m.mu.Lock()
	count := len(m.events[userID])
	delete(m.events, userID)
	m.mu.Unlock()

	if m.crmDeleter != nil {
		m.crmDeleter(userID)
		count++
	}
	return count, nil
}
