package marketing

import (
	"errors"
	"sync"
)

type MemoryMarketingRepository struct {
	crmMu     sync.RWMutex
	crm       map[string]*CRMContact
	privacyMu sync.RWMutex
	privacy   map[string][]*PrivacyLog
}

func NewMemoryMarketingRepository() *MemoryMarketingRepository {
	return &MemoryMarketingRepository{
		crm:     make(map[string]*CRMContact),
		privacy: make(map[string][]*PrivacyLog),
	}
}

func (m *MemoryMarketingRepository) SaveCRMContact(contact *CRMContact) error {
	m.crmMu.Lock()
	defer m.crmMu.Unlock()
	m.crm[contact.UserID] = contact
	return nil
}

func (m *MemoryMarketingRepository) GetCRMContact(userID string) (*CRMContact, error) {
	m.crmMu.RLock()
	defer m.crmMu.RUnlock()
	c, ok := m.crm[userID]
	if !ok {
		return nil, errors.New("crm contact not found")
	}
	return c, nil
}

func (m *MemoryMarketingRepository) DeleteCRMContact(userID string) error {
	m.crmMu.Lock()
	defer m.crmMu.Unlock()
	delete(m.crm, userID)
	return nil
}

func (m *MemoryMarketingRepository) SavePrivacyLog(log *PrivacyLog) error {
	m.privacyMu.Lock()
	defer m.privacyMu.Unlock()
	m.privacy[log.UserID] = append(m.privacy[log.UserID], log)
	return nil
}

func (m *MemoryMarketingRepository) GetPrivacyLogsByUserID(userID string) ([]*PrivacyLog, error) {
	m.privacyMu.RLock()
	defer m.privacyMu.RUnlock()
	logs, ok := m.privacy[userID]
	if !ok {
		return []*PrivacyLog{}, nil
	}
	copied := make([]*PrivacyLog, len(logs))
	copy(copied, logs)
	return copied, nil
}
