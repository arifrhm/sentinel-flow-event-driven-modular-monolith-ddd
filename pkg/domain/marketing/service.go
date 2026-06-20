package marketing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"sentinel-flow/pkg/broker"
)

// FraudEventService defines the port for fetching and deleting screened events across domains.
type FraudEventService interface {
	GetEventsByUserID(userID string) ([]*broker.ScreenedEvent, error)
	DeleteUserEvents(userID string) (int, error)
}

// TelemetryTracker defines the port interface for updating workflow metrics.
type TelemetryTracker interface {
	IncrementMetric(name string, delta int64)
}

type MarketingService struct {
	broker       broker.Broker
	repo         MarketingRepository
	tracker      TelemetryTracker
	fraudService FraudEventService
	crmAdapter   CRMAdapter
	salt         string
}

func NewMarketingService(
	b broker.Broker,
	r MarketingRepository,
	t TelemetryTracker,
	f FraudEventService,
	crm CRMAdapter,
) *MarketingService {
	return &MarketingService{
		broker:       b,
		repo:         r,
		tracker:      t,
		fraudService: f,
		crmAdapter:   crm,
		salt:         "SentinelFlowSuperSecretSaltForGDPRCompliance",
	}
}

func (s *MarketingService) StartConsumer(ctx context.Context) error {
	ch, err := s.broker.SubscribeScreened(ctx)
	if err != nil {
		return err
	}

	slog.Info("[Marketing] Worker listening for screened events...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case screenedEvent, ok := <-ch:
			if !ok {
				return nil
			}
			if screenedEvent.IsFraudulent {
				continue
			}
			s.ProcessMarketingWorkflow(ctx, screenedEvent)
		}
	}
}

func (s *MarketingService) ProcessMarketingWorkflow(ctx context.Context, event *broker.ScreenedEvent) {
	var workflow string
	switch event.EventType {
	case "signup":
		workflow = "Welcome & Onboarding Campaign"
	case "checkout_completed":
		workflow = "Loyalty Rewards Activation"
	case "cart_abandoned":
		workflow = "Recovery Email Sequence"
	default:
		return
	}

	email := fmt.Sprintf("%s@example.com", event.UserID)
	if emailVal, ok := event.Payload["email"]; ok {
		if emailStr, okStr := emailVal.(string); okStr && emailStr != "" {
			email = emailStr
		}
	}

	contact := &CRMContact{
		UserID:            event.UserID,
		Email:             email,
		WorkflowTriggered: workflow,
		SyncStatus:        "pending",
		SyncedAt:          time.Now(),
	}

	slog.Info("[Marketing] Triggered workflow. Syncing with CRM...", "workflow", workflow, "user_id", event.UserID)

	s.tracker.IncrementMetric("crm_attempts", 1)

	err := s.crmAdapter.SyncContact(ctx, contact)
	if err != nil {
		contact.SyncStatus = "failed"
		s.tracker.IncrementMetric("crm_failures", 1)
		slog.Error("[Marketing] Failed to sync User to CRM", "user_id", event.UserID, "error", err)
	} else {
		contact.SyncStatus = "synced"
		s.tracker.IncrementMetric("crm_successes", 1)
		slog.Info("[Marketing] Successfully synced User to CRM", "user_id", event.UserID, "workflow", workflow)
	}

	if err := s.repo.SaveCRMContact(contact); err != nil {
		slog.Error("[Marketing] Error saving CRM contact status", "error", err)
	}
}
