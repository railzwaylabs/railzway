//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	invoicerepo "github.com/railzwaylabs/railzway/internal/invoice/repository"
	invoiceservice "github.com/railzwaylabs/railzway/internal/invoice/service"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	planrepo "github.com/railzwaylabs/railzway/internal/plan/repository"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	taxdomain "github.com/railzwaylabs/railzway/internal/tax/domain"
)

func TestInvoiceTaxes_ExclusiveRate(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	orgID, subID := seedFlatSubscription(t, tx, now, periodStart, periodEnd, periodStart, nil)

	rate := taxdomain.TaxRate{
		ID:         uuid.New(),
		OrgID:      orgID,
		Code:       "vat10",
		Name:       "VAT 10%",
		Percentage: 10,
		Inclusive:  false,
		Active:     true,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&rate).Error; err != nil {
		t.Fatalf("create tax rate: %v", err)
	}

	invoiceSvc := invoiceservice.NewService(invoiceservice.Params{
		DB:       tx,
		Repo:     invoicerepo.NewRepository(tx),
		PlanRepo: planrepo.NewRepository(tx),
		SubRepo:  subrepo.NewRepository(tx),
	})

	ctx := orgcontext.WithOrgID(context.Background(), orgID)
	resp, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IdempotencyKey: "e2e-tax-exclusive",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}

	if resp.Invoice.SubtotalCents != 10000 {
		t.Fatalf("expected subtotal 10000, got %d", resp.Invoice.SubtotalCents)
	}
	if resp.Invoice.TaxCents != 1000 {
		t.Fatalf("expected tax 1000, got %d", resp.Invoice.TaxCents)
	}
	if resp.Invoice.TotalCents != 11000 {
		t.Fatalf("expected total 11000, got %d", resp.Invoice.TotalCents)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 invoice items, got %d", len(resp.Items))
	}
	var taxLine *invoicedomain.InvoiceItem
	for i := range resp.Items {
		if resp.Items[i].LineType == invoicedomain.LineTypeTax {
			taxLine = &resp.Items[i]
			break
		}
	}
	if taxLine == nil {
		t.Fatalf("expected tax line item")
	}
	if taxLine.AmountCents != 1000 {
		t.Fatalf("expected tax line amount 1000, got %d", taxLine.AmountCents)
	}
}

func TestInvoiceTaxes_InclusiveAndExclusive(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	orgID, subID := seedFlatSubscription(t, tx, now, periodStart, periodEnd, periodStart, nil)

	inclusive := taxdomain.TaxRate{
		ID:         uuid.New(),
		OrgID:      orgID,
		Code:       "gst10",
		Name:       "GST 10%",
		Percentage: 10,
		Inclusive:  true,
		Active:     true,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	exclusive := taxdomain.TaxRate{
		ID:         uuid.New(),
		OrgID:      orgID,
		Code:       "svc5",
		Name:       "Service 5%",
		Percentage: 5,
		Inclusive:  false,
		Active:     true,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&inclusive).Error; err != nil {
		t.Fatalf("create inclusive rate: %v", err)
	}
	if err := tx.Create(&exclusive).Error; err != nil {
		t.Fatalf("create exclusive rate: %v", err)
	}

	invoiceSvc := invoiceservice.NewService(invoiceservice.Params{
		DB:       tx,
		Repo:     invoicerepo.NewRepository(tx),
		PlanRepo: planrepo.NewRepository(tx),
		SubRepo:  subrepo.NewRepository(tx),
	})

	ctx := orgcontext.WithOrgID(context.Background(), orgID)
	resp, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IdempotencyKey: "e2e-tax-mixed",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}

	// Subtotal is always the base amount for flat plan.
	if resp.Invoice.SubtotalCents != 10000 {
		t.Fatalf("expected subtotal 10000, got %d", resp.Invoice.SubtotalCents)
	}
	// Inclusive 10% -> base = 10000/1.1 = 9091 (rounded), inclusive tax = 909
	// Exclusive 5% on subtotal = 500
	if resp.Invoice.TaxCents != 1409 {
		t.Fatalf("expected tax 1409, got %d", resp.Invoice.TaxCents)
	}
	if resp.Invoice.TotalCents != 10500 {
		t.Fatalf("expected total 10500, got %d", resp.Invoice.TotalCents)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 invoice items, got %d", len(resp.Items))
	}
}
