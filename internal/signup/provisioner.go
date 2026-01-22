package signup

import (
	"context"
	"strings"

	"github.com/bwmarrin/snowflake"
	billingeventdomain "github.com/smallbiznis/railzway/internal/billingevent/domain"
	"github.com/smallbiznis/railzway/internal/signup/domain"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const OrganizationCreatedTopic = "organization.created"

type noopProvisioner struct{}

func NewNoopProvisioner() domain.Provisioner {
	return &noopProvisioner{}
}

func (p *noopProvisioner) Provision(ctx context.Context, organizationID string) error {
	_ = ctx
	_ = organizationID
	return nil
}

type EventProvisioner struct {
	db    *gorm.DB
	genID *snowflake.Node
}

func NewEventProvisioner(db *gorm.DB, genID *snowflake.Node) domain.Provisioner {
	return &EventProvisioner{
		db:    db,
		genID: genID,
	}
}

func (p *EventProvisioner) Provision(ctx context.Context, organizationID string) error {
	orgID := strings.TrimSpace(organizationID)
	parsedID, err := snowflake.ParseString(orgID)
	if err != nil {
		return err
	}

	event := &billingeventdomain.BillingEvent{
		ID:        p.genID.Generate(),
		OrgID:     parsedID,
		EventType: OrganizationCreatedTopic,
		Payload: datatypes.JSONMap{
			"organization_id": orgID,
		},
	}

	return p.db.WithContext(ctx).Create(event).Error
}
