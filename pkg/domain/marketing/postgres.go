package marketing

import (
	"database/sql"
	"errors"
	"fmt"
)

type PostgresMarketingRepository struct {
	db *sql.DB
}

func NewPostgresMarketingRepository(db *sql.DB) *PostgresMarketingRepository {
	return &PostgresMarketingRepository{db: db}
}

func (p *PostgresMarketingRepository) SaveCRMContact(contact *CRMContact) error {
	query := `
	INSERT INTO crm_contacts (user_id, email, workflow_triggered, sync_status, synced_at, retry_count)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (user_id) DO UPDATE SET
		email = EXCLUDED.email,
		workflow_triggered = EXCLUDED.workflow_triggered,
		sync_status = EXCLUDED.sync_status,
		synced_at = EXCLUDED.synced_at,
		retry_count = EXCLUDED.retry_count;
	`
	_, err := p.db.Exec(
		query,
		contact.UserID,
		contact.Email,
		contact.WorkflowTriggered,
		contact.SyncStatus,
		contact.SyncedAt,
		contact.RetryCount,
	)
	return err
}

func (p *PostgresMarketingRepository) GetCRMContact(userID string) (*CRMContact, error) {
	query := `
	SELECT user_id, email, workflow_triggered, sync_status, synced_at, retry_count
	FROM crm_contacts
	WHERE user_id = $1;
	`
	var c CRMContact
	err := p.db.QueryRow(query, userID).Scan(
		&c.UserID,
		&c.Email,
		&c.WorkflowTriggered,
		&c.SyncStatus,
		&c.SyncedAt,
		&c.RetryCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("crm contact not found")
		}
		return nil, fmt.Errorf("failed to query crm contact: %w", err)
	}
	return &c, nil
}

func (p *PostgresMarketingRepository) DeleteCRMContact(userID string) error {
	_, err := p.db.Exec("DELETE FROM crm_contacts WHERE user_id = $1", userID)
	return err
}

func (p *PostgresMarketingRepository) SavePrivacyLog(log *PrivacyLog) error {
	query := `
	INSERT INTO privacy_logs (log_id, user_id, action, timestamp)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (log_id) DO NOTHING;
	`
	_, err := p.db.Exec(query, log.LogID, log.UserID, log.Action, log.Timestamp)
	return err
}

func (p *PostgresMarketingRepository) GetPrivacyLogsByUserID(userID string) ([]*PrivacyLog, error) {
	query := `
	SELECT log_id, user_id, action, timestamp
	FROM privacy_logs
	WHERE user_id = $1
	ORDER BY timestamp DESC;
	`
	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query privacy logs: %w", err)
	}
	defer rows.Close()

	var logs []*PrivacyLog
	for rows.Next() {
		var l PrivacyLog
		err := rows.Scan(&l.LogID, &l.UserID, &l.Action, &l.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan privacy log row: %w", err)
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}
