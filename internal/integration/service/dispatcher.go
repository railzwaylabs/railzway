package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/events"
	"github.com/railzwaylabs/railzway/internal/integration/domain"
	"github.com/railzwaylabs/railzway/internal/security/vault"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	DispatcherConsumerID = "notification_dispatcher"
	BatchSize            = 50
)

type Dispatcher struct {
	db        *gorm.DB
	log       *zap.Logger
	vault     vault.Provider
	providers map[string]domain.NotificationProvider
}

func NewDispatcher(db *gorm.DB, log *zap.Logger, v vault.Provider, providers map[string]domain.NotificationProvider) *Dispatcher {
	return &Dispatcher{
		db:        db,
		log:       log.Named("integration.dispatcher"),
		vault:     v,
		providers: providers,
	}
}

func (d *Dispatcher) ProcessEvents(ctx context.Context) error {
	// 1. Get offset
	lastID, err := d.getLastEventID(ctx)
	if err != nil {
		return err
	}

	// 2. Fetch new events
	var rows []struct {
		ID        snowflake.ID
		OrgID     snowflake.ID
		EventType string
		Payload   json.RawMessage
	}

	err = d.db.WithContext(ctx).Raw(`
		SELECT id, org_id, event_type, payload
		FROM billing_events
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?
	`, lastID, BatchSize).Scan(&rows).Error
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	for _, row := range rows {
		if err := d.dispatch(ctx, row.ID, row.OrgID, row.EventType, row.Payload); err != nil {
			d.log.Error("failed to dispatch event", zap.Error(err), zap.String("event_id", row.ID.String()))
		}
		
		// Update offset one by one for safety or at the end of batch?
		// One by one is safer against crashes.
		if err := d.updateLastEventID(ctx, row.ID); err != nil {
			return err
		}
	}

	return nil
}

func (d *Dispatcher) dispatch(ctx context.Context, eventID, orgID snowflake.ID, eventType string, payloadRaw json.RawMessage) error {
	// For now, we only care about certain events
	if eventType != events.EventInvoiceFinalized && eventType != "integration.connected" {
		return nil
	}

	// Find active connections for notifications in this org
	var conns []domain.Connection
	if err := d.db.WithContext(ctx).
		Preload("Integration").
		Where("org_id = ? AND status = ?", orgID, "active").
		Find(&conns).Error; err != nil {
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return err
	}

	for _, conn := range conns {
		if conn.Integration == nil || conn.Integration.Type != domain.TypeNotification {
			continue
		}

		provider, ok := d.providers[conn.IntegrationID]
		if !ok {
			continue
		}

		// Decrypt credentials
		creds, err := d.vault.Decrypt(conn.EncryptedCreds)
		if err != nil {
			d.log.Error("failed to decrypt connection creds", zap.Error(err), zap.String("conn_id", conn.ID.String()))
			continue
		}

		var credsMap map[string]any
		if err := json.Unmarshal(creds, &credsMap); err != nil {
			continue
		}

		// Merge public config and private creds for the provider
		inputData := make(map[string]any)
		var configMap map[string]any
		if err := json.Unmarshal(conn.Config, &configMap); err == nil {
			for k, v := range configMap {
				inputData[k] = v
			}
		}
		for k, v := range credsMap {
			inputData[k] = v
		}

		// Add event data
		for k, v := range payload {
			inputData[k] = v
		}

		err = provider.Send(ctx, domain.NotificationInput{
			ConnectionID: conn.ID,
			TemplateID:   eventType,
			Data:         inputData,
		})
		if err != nil {
			d.log.Warn("notification provider failed", zap.Error(err), zap.String("provider", conn.IntegrationID))
		}
	}

	return nil
}

func (d *Dispatcher) getLastEventID(ctx context.Context) (snowflake.ID, error) {
	var offset struct {
		LastEventID snowflake.ID
	}
	err := d.db.WithContext(ctx).Raw("SELECT last_event_id FROM event_consumer_offsets WHERE consumer_id = ?", DispatcherConsumerID).Scan(&offset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return offset.LastEventID, nil
}

func (d *Dispatcher) updateLastEventID(ctx context.Context, id snowflake.ID) error {
	return d.db.WithContext(ctx).Exec(`
		INSERT INTO event_consumer_offsets (consumer_id, last_event_id, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (consumer_id) DO UPDATE SET last_event_id = EXCLUDED.last_event_id, updated_at = EXCLUDED.updated_at
	`, DispatcherConsumerID, id, time.Now().UTC()).Error
}
