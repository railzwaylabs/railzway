package event

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const OrganizationCreatedTopic = "organization.created"

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

type outboxPublisher struct {
	db    *gorm.DB
	genID *snowflake.Node
}

func NewOutboxPublisher(db *gorm.DB, genID *snowflake.Node) EventPublisher {
	return &outboxPublisher{
		db:    db,
		genID: genID,
	}
}

type organizationCreatedPayload struct {
	OrganizationID string `json:"organization_id"`
}

func (p *outboxPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	var parsed organizationCreatedPayload
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return err
	}

	orgID := strings.TrimSpace(parsed.OrganizationID)
	if orgID == "" {
		return errors.New("missing organization_id")
	}

	parsedID, err := snowflake.ParseString(orgID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	return p.db.WithContext(ctx).Exec(
		`INSERT INTO billing_events (id, org_id, event_type, payload, published, created_at)
		 VALUES (?, ?, ?, ?, false, ?)`,
		p.genID.Generate(),
		parsedID,
		topic,
		datatypes.JSON(payload),
		now,
	).Error
}
