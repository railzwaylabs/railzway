package service

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestBuildXenditExternalID(t *testing.T) {
	node, _ := snowflake.NewNode(1)
	customerID := node.Generate()
	invoiceID := node.Generate()

	require.Equal(t, "customer_"+customerID.String()+"_invoice_"+invoiceID.String(), buildXenditExternalID(customerID, invoiceID))
	require.Equal(t, "customer_"+customerID.String(), buildXenditExternalID(customerID, 0))
	require.Equal(t, "", buildXenditExternalID(0, invoiceID))
}

func TestAmountToMajor(t *testing.T) {
	require.Equal(t, 123.45, amountToMajor(12345, "USD"))
	require.Equal(t, 12345.0, amountToMajor(12345, "IDR"))
}

func TestMergeInvoiceMetadataPreservesExisting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&invoicedomain.Invoice{}))

	node, _ := snowflake.NewNode(1)
	now := time.Now().UTC()
	inv := invoicedomain.Invoice{
		ID:             node.Generate(),
		OrgID:          node.Generate(),
		BillingCycleID: node.Generate(),
		SubscriptionID: node.Generate(),
		CustomerID:     node.Generate(),
		InvoiceNumber:  "INV-123",
		Currency:       "USD",
		Status:         invoicedomain.InvoiceStatusFinalized,
		SubtotalAmount: 1000,
		TotalAmount:    1000,
		Metadata:       datatypes.JSONMap{"existing": "keep"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&inv).Error)

	svc := &Service{db: db, log: zap.NewNop()}
	require.NoError(t, svc.mergeInvoiceMetadata(context.Background(), inv.OrgID, inv.ID, map[string]any{
		"new_key": "new_value",
	}))

	var metadata datatypes.JSONMap
	require.NoError(t, db.Raw("SELECT metadata FROM invoices WHERE id = ?", inv.ID).Scan(&metadata).Error)
	require.Equal(t, "keep", metadata["existing"])
	require.Equal(t, "new_value", metadata["new_key"])
}
