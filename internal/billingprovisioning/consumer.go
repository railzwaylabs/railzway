package billingprovisioning

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/organization"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	batchSize = 50
)

type Consumer struct {
	db    *gorm.DB
	log   *zap.Logger
	genID *snowflake.Node
}

func NewConsumer(db *gorm.DB, log *zap.Logger, genID *snowflake.Node) *Consumer {
	return &Consumer{
		db:    db,
		log:   log.Named("billing.provisioning"),
		genID: genID,
	}
}

type eventRow struct {
	ID      snowflake.ID   `gorm:"column:id"`
	OrgID   snowflake.ID   `gorm:"column:org_id"`
	Payload datatypes.JSON `gorm:"column:payload"`
}

type organizationCreatedPayload struct {
	OrganizationID  string `json:"organization_id"`
	OwnerUserID     string `json:"owner_user_id"`
	CountryCode     string `json:"country_code"`
	TimezoneName    string `json:"timezone_name"`
	DefaultCurrency string `json:"default_currency"`
	CreatedAt       string `json:"created_at"`
}

func (c *Consumer) ProcessPending(ctx context.Context) error {
	var events []eventRow
	err := c.db.WithContext(ctx).Raw(
		`SELECT id, org_id, payload FROM billing_events
		 WHERE event_type = ? AND published = false
		 ORDER BY created_at ASC
		 LIMIT ?`,
		organization.OrganizationCreatedTopic,
		batchSize,
	).Scan(&events).Error
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := c.processEvent(ctx, event); err != nil {
			c.log.Error("failed to provision organization", zap.Error(err), zap.String("organization_id", event.OrgID.String()))
		}
	}

	return nil
}

func (c *Consumer) processEvent(ctx context.Context, event eventRow) error {
	var payload organizationCreatedPayload
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		return err
	}

	if payload.OrganizationID == "" {
		return errors.New("missing organization_id")
	}

	orgID, err := snowflake.ParseString(payload.OrganizationID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspaceExists, err := c.billingWorkspaceExists(ctx, tx, orgID)
		if err != nil {
			return err
		}
		if !workspaceExists {
			if err := c.createBillingWorkspace(ctx, tx, orgID, now); err != nil {
				return err
			}
		}

		return c.markPublished(ctx, tx, event.ID, now)
	})
}

func (c *Consumer) billingWorkspaceExists(ctx context.Context, tx *gorm.DB, orgID snowflake.ID) (bool, error) {
	var count int64
	if err := tx.WithContext(ctx).Raw(
		`SELECT COUNT(1) FROM billing_workspaces WHERE organization_id = ?`,
		orgID,
	).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *Consumer) createBillingWorkspace(ctx context.Context, tx *gorm.DB, orgID snowflake.ID, now time.Time) error {
	return tx.WithContext(ctx).Exec(
		`INSERT INTO billing_workspaces (id, organization_id, created_at)
		 VALUES (?, ?, ?)`,
		c.genID.Generate(),
		orgID,
		now,
	).Error
}

func (c *Consumer) markPublished(ctx context.Context, tx *gorm.DB, eventID snowflake.ID, now time.Time) error {
	return tx.WithContext(ctx).Exec(
		`UPDATE billing_events SET published = true, published_at = ? WHERE id = ?`,
		now,
		eventID,
	).Error
}
