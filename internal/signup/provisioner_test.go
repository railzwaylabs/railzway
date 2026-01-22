package signup

import (
	"context"
	"testing"

	"github.com/bwmarrin/snowflake"
	billingevent "github.com/smallbiznis/railzway/internal/billingevent/domain"
	"github.com/smallbiznis/railzway/internal/config"
	dbpkg "github.com/smallbiznis/railzway/pkg/db"
)

func TestNewProvisionerOSSModeUsesNoop(t *testing.T) {
	provisioner := newProvisioner(config.Config{Mode: config.ModeOSS}, nil, nil)
	if _, ok := provisioner.(*noopProvisioner); !ok {
		t.Fatalf("expected noop provisioner in OSS mode, got %T", provisioner)
	}
}

func TestEventProvisionerEmitsOrganizationCreated(t *testing.T) {
	db, err := dbpkg.NewTest()
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	if err := db.AutoMigrate(&billingevent.BillingEvent{}); err != nil {
		t.Fatalf("failed to migrate billing events: %v", err)
	}

	node, err := snowflake.NewNode(1)
	if err != nil {
		t.Fatalf("failed to create id generator: %v", err)
	}

	provisioner := NewEventProvisioner(db, node)
	organizationID := node.Generate().String()

	if err := provisioner.Provision(context.Background(), organizationID); err != nil {
		t.Fatalf("provision failed: %v", err)
	}

	var events []billingevent.BillingEvent
	if err := db.Find(&events).Error; err != nil {
		t.Fatalf("failed to load events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.EventType != OrganizationCreatedTopic {
		t.Fatalf("expected event type %q, got %q", OrganizationCreatedTopic, event.EventType)
	}

	value, ok := event.Payload["organization_id"].(string)
	if !ok {
		t.Fatalf("expected organization_id payload to be a string, got %T", event.Payload["organization_id"])
	}
	if value != organizationID {
		t.Fatalf("expected organization_id %q, got %q", organizationID, value)
	}
}

func TestNoopProvisionerDoesNothing(t *testing.T) {
	provisioner := NewNoopProvisioner()
	if err := provisioner.Provision(context.Background(), "org"); err != nil {
		t.Fatalf("expected noop provisioner to return nil, got %v", err)
	}
}
