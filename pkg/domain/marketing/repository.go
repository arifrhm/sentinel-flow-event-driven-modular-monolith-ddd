package marketing

// MarketingRepository defines the port interface for persisting CRM configurations and privacy logs.
type MarketingRepository interface {
	SaveCRMContact(contact *CRMContact) error
	GetCRMContact(userID string) (*CRMContact, error)
	DeleteCRMContact(userID string) error
	SavePrivacyLog(log *PrivacyLog) error
	GetPrivacyLogsByUserID(userID string) ([]*PrivacyLog, error)
}
