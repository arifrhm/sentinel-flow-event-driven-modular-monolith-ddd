package fraud

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"sentinel-flow/pkg/broker"
)

type PostgresFraudRepository struct {
	db *sql.DB
}

func NewPostgresFraudRepository(db *sql.DB) *PostgresFraudRepository {
	return &PostgresFraudRepository{db: db}
}

func (p *PostgresFraudRepository) SaveEvent(event *broker.ScreenedEvent) error {
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	query := `
	INSERT INTO events (
		event_id, user_id, event_type, ip_address, user_agent, payload, 
		timestamp, gdpr_consent, is_fraudulent, fraud_score, fraud_reason, screened_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (event_id) DO NOTHING;
	`
	_, err = p.db.Exec(
		query,
		event.EventID,
		event.UserID,
		event.EventType,
		event.IPAddress,
		event.UserAgent,
		payloadJSON,
		event.Timestamp,
		event.GDPRConsent,
		event.IsFraudulent,
		event.FraudScore,
		event.FraudReason,
		event.ScreenedAt,
	)
	return err
}

func (p *PostgresFraudRepository) GetEventsByUserID(userID string) ([]*broker.ScreenedEvent, error) {
	query := `
	SELECT 
		event_id, user_id, event_type, ip_address, user_agent, payload, 
		timestamp, gdpr_consent, is_fraudulent, fraud_score, fraud_reason, screened_at
	FROM events 
	WHERE user_id = $1
	ORDER BY timestamp DESC;
	`
	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*broker.ScreenedEvent
	for rows.Next() {
		var ev broker.ScreenedEvent
		var payloadBytes []byte

		err := rows.Scan(
			&ev.EventID,
			&ev.UserID,
			&ev.EventType,
			&ev.IPAddress,
			&ev.UserAgent,
			&payloadBytes,
			&ev.Timestamp,
			&ev.GDPRConsent,
			&ev.IsFraudulent,
			&ev.FraudScore,
			&ev.FraudReason,
			&ev.ScreenedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		if len(payloadBytes) > 0 {
			if err := json.Unmarshal(payloadBytes, &ev.Payload); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event payload: %w", err)
			}
		}

		events = append(events, &ev)
	}

	return events, rows.Err()
}

func (p *PostgresFraudRepository) DeleteUserEvents(userID string) (int, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	resEvents, err := tx.Exec("DELETE FROM events WHERE user_id = $1", userID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete events: %w", err)
	}
	eventsDeleted, _ := resEvents.RowsAffected()

	resCRM, err := tx.Exec("DELETE FROM crm_contacts WHERE user_id = $1", userID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete CRM contacts: %w", err)
	}
	crmDeleted, _ := resCRM.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(eventsDeleted + crmDeleted), nil
}
