package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/clock"
	coupondomain "github.com/railzwaylabs/railzway/internal/coupon/domain"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/invoice/domain"
	ledgerdomain "github.com/railzwaylabs/railzway/internal/ledger/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	"github.com/railzwaylabs/railzway/internal/proration"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type service struct {
	db            *gorm.DB
	repo          domain.Repository
	planRepo      plandomain.Repository
	subRepo       subscriptiondomain.Repository
	customerRepo  customerdomain.Repository
	testClockRepo testclockdomain.Repository
	audit         *auditlog.Service
	clock         clock.Clock
	ledger        ledgerdomain.Service
	ledgerRepo    ledgerdomain.Repository
	coupons       coupondomain.Service
}

type Params struct {
	fx.In

	DB            *gorm.DB
	Repo          domain.Repository
	PlanRepo      plandomain.Repository
	SubRepo       subscriptiondomain.Repository
	CustomerRepo  customerdomain.Repository  `optional:"true"`
	TestClockRepo testclockdomain.Repository `optional:"true"`
	Audit         *auditlog.Service          `optional:"true"`
	Clock         clock.Clock                `optional:"true"`
	Ledger        ledgerdomain.Service       `optional:"true"`
	LedgerRepo    ledgerdomain.Repository    `optional:"true"`
	Coupons       coupondomain.Service       `optional:"true"`
}

func NewService(p Params) domain.Service {
	clk := p.Clock
	if clk == nil {
		clk = clock.SystemClock{}
	}
	return &service{
		db:            p.DB,
		repo:          p.Repo,
		planRepo:      p.PlanRepo,
		subRepo:       p.SubRepo,
		customerRepo:  p.CustomerRepo,
		testClockRepo: p.TestClockRepo,
		audit:         p.Audit,
		clock:         clk,
		ledger:        p.Ledger,
		ledgerRepo:    p.LedgerRepo,
		coupons:       p.Coupons,
	}
}

func (s *service) GenerateInvoice(ctx context.Context, req domain.GenerateInvoiceRequest) (resp domain.GenerateInvoiceResponse, err error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.GenerateInvoiceResponse{}, domain.ErrInvalidOrganization
	}

	ctx, span := telemetry.StartSpan(
		ctx,
		"invoice.generate",
		telemetry.UUIDAttr("billing.org_id", orgID),
		telemetry.StringAttr("billing.subscription_id", strings.TrimSpace(req.SubscriptionID)),
		telemetry.StringAttr("billing.idempotency_key", strings.TrimSpace(req.IdempotencyKey)),
	)
	defer func() { telemetry.EndSpan(span, err) }()

	startedAt := time.Now()
	defer func() { telemetry.ObserveOperation("invoice.generate", time.Since(startedAt), err) }()

	subID, err := parseID(req.SubscriptionID)
	if err != nil {
		return domain.GenerateInvoiceResponse{}, domain.ErrInvalidSubscription
	}

	if req.PeriodEnd.Before(req.PeriodStart) || req.PeriodStart.IsZero() || req.PeriodEnd.IsZero() {
		return domain.GenerateInvoiceResponse{}, domain.ErrInvalidPeriod
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindInvoiceByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.GenerateInvoiceResponse{}, err
		}
		if existing != nil {
			items, _ := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, existing.ID)
			resp = domain.GenerateInvoiceResponse{Invoice: *existing, Items: derefItems(items)}
			span.SetAttributes(
				telemetry.UUIDAttr("billing.invoice_id", existing.ID),
				telemetry.BoolAttr("billing.idempotent_hit", true),
			)
			return resp, nil
		}
	}

	sub, err := s.subRepo.FindSubscriptionByID(ctx, orgID, subID)
	if err != nil {
		return domain.GenerateInvoiceResponse{}, err
	}
	if sub == nil {
		return domain.GenerateInvoiceResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, sub.CustomerID)
	if err != nil {
		return domain.GenerateInvoiceResponse{}, err
	}

	periodStart := req.PeriodStart.UTC()
	periodEnd := req.PeriodEnd.UTC()

	pending, err := s.hasPendingUsage(ctx, orgID, sub, periodStart, periodEnd)
	if err != nil {
		return domain.GenerateInvoiceResponse{}, err
	}
	if pending {
		return domain.GenerateInvoiceResponse{}, domain.ErrUsageNotReady
	}

	now := s.clock.Now(ctx)
	issueAt := now
	if req.IssueAt != nil {
		issueAt = req.IssueAt.UTC()
	}

	config, err := s.getInvoiceNumberConfig(ctx, orgID, issueAt)
	if err != nil {
		return domain.GenerateInvoiceResponse{}, err
	}

	issueAtLocal := issueAt
	if config.Timezone != "" {
		if loc, err := time.LoadLocation(config.Timezone); err == nil {
			issueAtLocal = issueAt.In(loc)
		}
	}

	periodKey := buildSequencePeriodKey(config.SequenceScope, issueAtLocal)
	inv := domain.Invoice{
		ID:             uuid.New(),
		OrgID:          orgID,
		CustomerID:     sub.CustomerID,
		SubscriptionID: &sub.ID,
		Number:         "",
		Status:         domain.StatusDraft,
		Currency:       sub.Currency,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if req.IssueAt != nil {
		t := issueAt
		inv.IssuedAt = &t
	}
	if req.DueAt != nil {
		t := req.DueAt.UTC()
		inv.DueAt = &t
	}
	if idempotencyKey != "" {
		inv.IdempotencyKey = &idempotencyKey
	}

	items, subtotal, taxTotals, err := s.buildDraftInvoiceItems(ctx, orgID, sub, inv.PeriodStart, inv.PeriodEnd)
	if err != nil {
		return domain.GenerateInvoiceResponse{}, err
	}
	if len(items) == 0 {
		return domain.GenerateInvoiceResponse{}, domain.ErrNoBillableItems
	}

	inv.SubtotalCents = subtotal
	inv.TaxCents = taxTotals.Total
	inv.TotalCents = subtotal + taxTotals.Exclusive
	inv.AmountDueCents = inv.TotalCents
	inv.Checksum = buildInvoiceChecksum(inv, items)

	var existingInvoice *domain.Invoice
	var existingItems []*domain.InvoiceItem

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockInvoiceKey(ctx, tx, buildInvoiceLockKey(orgID, sub.ID, periodStart, periodEnd)); err != nil {
			return err
		}

		repo := s.repo.WithTx(tx)
		existing, err := repo.FindInvoiceBySubscriptionPeriod(ctx, orgID, sub.ID, periodStart, periodEnd)
		if err != nil {
			return err
		}

		if existing != nil {
			existingInvoice = existing
			existingItems, _ = repo.ListInvoiceItemsByInvoice(ctx, orgID, existing.ID)
			return nil
		}

		sequence, err := nextInvoiceSequenceWithDB(ctx, tx, orgID, periodKey)
		if err != nil {
			return err
		}

		inv.Number = renderInvoiceNumber(config.Format, config.Prefix, issueAtLocal, sequence, orgID)
		if err := repo.CreateInvoice(ctx, inv); err != nil {
			return err
		}

		for i := range items {
			items[i].InvoiceID = inv.ID
		}

		return repo.CreateInvoiceItems(ctx, items)

	}); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindInvoiceByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.GenerateInvoiceResponse{}, findErr
			}
			if existing != nil {
				itemList, _ := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, existing.ID)
				resp = domain.GenerateInvoiceResponse{Invoice: *existing, Items: derefItems(itemList)}
				span.SetAttributes(
					telemetry.UUIDAttr("billing.invoice_id", existing.ID),
					telemetry.BoolAttr("billing.idempotent_hit", true),
				)
				return resp, nil
			}
		}
		return domain.GenerateInvoiceResponse{}, err
	}

	if existingInvoice != nil {
		resp = domain.GenerateInvoiceResponse{Invoice: *existingInvoice, Items: derefItems(existingItems)}
		span.SetAttributes(
			telemetry.UUIDAttr("billing.invoice_id", existingInvoice.ID),
			telemetry.BoolAttr("billing.existing_invoice", true),
		)
		return resp, nil
	}

	resp = domain.GenerateInvoiceResponse{Invoice: inv, Items: items}
	span.SetAttributes(
		telemetry.UUIDAttr("billing.invoice_id", inv.ID),
		telemetry.UUIDAttr("billing.customer_id", inv.CustomerID),
		telemetry.UUIDAttr("billing.subscription_id", sub.ID),
		telemetry.Int64Attr("billing.invoice_total_cents", inv.TotalCents),
		telemetry.Int64Attr("billing.invoice_items_count", int64(len(items))),
	)
	s.recordAudit(ctx, "invoice.generate", "invoice", resp.Invoice.ID.String(), nil, resp.Invoice, nil)
	return resp, nil
}

func (s *service) CreateAdjustmentInvoice(ctx context.Context, req domain.CreateAdjustmentInvoiceRequest) (domain.CreateAdjustmentInvoiceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidOrganization
	}
	subID, err := parseID(req.SubscriptionID)
	if err != nil {
		return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidSubscription
	}
	if req.PeriodEnd.Before(req.PeriodStart) || req.PeriodStart.IsZero() || req.PeriodEnd.IsZero() {
		return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidPeriod
	}
	if len(req.Lines) == 0 {
		return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidPeriod
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindInvoiceByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.CreateAdjustmentInvoiceResponse{}, err
		}
		if existing != nil {
			items, _ := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, existing.ID)
			return domain.CreateAdjustmentInvoiceResponse{Invoice: *existing, Items: derefItems(items)}, nil
		}
	}

	sub, err := s.subRepo.FindSubscriptionByID(ctx, orgID, subID)
	if err != nil {
		return domain.CreateAdjustmentInvoiceResponse{}, err
	}
	if sub == nil {
		return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, sub.CustomerID)
	if err != nil {
		return domain.CreateAdjustmentInvoiceResponse{}, err
	}

	periodStart := req.PeriodStart.UTC()
	periodEnd := req.PeriodEnd.UTC()

	now := s.clock.Now(ctx)
	issueAt := now
	if req.IssueAt != nil {
		issueAt = req.IssueAt.UTC()
	}

	config, err := s.getInvoiceNumberConfig(ctx, orgID, issueAt)
	if err != nil {
		return domain.CreateAdjustmentInvoiceResponse{}, err
	}

	issueAtLocal := issueAt
	if config.Timezone != "" {
		if loc, err := time.LoadLocation(config.Timezone); err == nil {
			issueAtLocal = issueAt.In(loc)
		}
	}

	periodKey := buildSequencePeriodKey(config.SequenceScope, issueAtLocal)
	inv := domain.Invoice{
		ID:             uuid.New(),
		OrgID:          orgID,
		CustomerID:     sub.CustomerID,
		SubscriptionID: &sub.ID,
		Number:         "",
		Status:         domain.StatusDraft,
		Currency:       sub.Currency,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if req.IssueAt != nil {
		t := issueAt
		inv.IssuedAt = &t
	}
	if req.DueAt != nil {
		t := req.DueAt.UTC()
		inv.DueAt = &t
	}
	if idempotencyKey != "" {
		inv.IdempotencyKey = &idempotencyKey
	}

	meta := map[string]interface{}{
		"adjustment": true,
	}
	if base := strings.TrimSpace(req.BaseInvoiceID); base != "" {
		meta["base_invoice_id"] = base
	}
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		meta["reason"] = reason
	}
	if metaJSON, err := json.Marshal(meta); err == nil {
		inv.Metadata = metaJSON
	}

	items := make([]domain.InvoiceItem, 0, len(req.Lines))
	for _, line := range req.Lines {
		if line.AmountCents <= 0 {
			continue
		}
		lineCurrency := strings.TrimSpace(strings.ToUpper(line.Currency))
		if lineCurrency == "" {
			lineCurrency = sub.Currency
		}
		if lineCurrency != sub.Currency {
			return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidPeriod
		}

		var planPriceID *uuid.UUID
		if line.PlanPriceID != nil && strings.TrimSpace(*line.PlanPriceID) != "" {
			id, err := parseID(*line.PlanPriceID)
			if err != nil {
				return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidPeriod
			}
			planPriceID = &id
		}
		var meterID *uuid.UUID
		if line.MeterID != nil && strings.TrimSpace(*line.MeterID) != "" {
			id, err := parseID(*line.MeterID)
			if err != nil {
				return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidPeriod
			}
			meterID = &id
		}
		var ratingResultID *uuid.UUID
		if line.RatingResultID != nil && strings.TrimSpace(*line.RatingResultID) != "" {
			id, err := parseID(*line.RatingResultID)
			if err != nil {
				return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidPeriod
			}
			ratingResultID = &id
		}

		var windowStart *time.Time
		if line.WindowStart != nil {
			t := line.WindowStart.UTC()
			windowStart = &t
		}
		var windowEnd *time.Time
		if line.WindowEnd != nil {
			t := line.WindowEnd.UTC()
			windowEnd = &t
		}

		desc := strings.TrimSpace(line.Description)
		if desc == "" {
			desc = "Usage adjustment"
		}

		items = append(items, domain.InvoiceItem{
			ID:              uuid.New(),
			OrgID:           orgID,
			CustomerID:      sub.CustomerID,
			SubscriptionID:  &sub.ID,
			PlanPriceID:     planPriceID,
			MeterID:         meterID,
			RatingResultID:  ratingResultID,
			LineType:        domain.LineTypeAdjustment,
			Description:     desc,
			Quantity:        line.Quantity,
			UnitAmountCents: line.UnitAmountCents,
			AmountCents:     line.AmountCents,
			Currency:        sub.Currency,
			PeriodStart:     windowStart,
			PeriodEnd:       windowEnd,
			Metadata:        json.RawMessage(`{}`),
			CreatedAt:       now,
		})
	}
	if len(items) == 0 {
		return domain.CreateAdjustmentInvoiceResponse{}, domain.ErrInvalidPeriod
	}

	var subtotal int64
	for _, item := range items {
		subtotal += item.AmountCents
	}
	taxItems, taxTotals, err := s.buildTaxLines(ctx, orgID, sub, subtotal, periodStart, periodEnd)
	if err != nil {
		return domain.CreateAdjustmentInvoiceResponse{}, err
	}
	items = append(items, taxItems...)

	inv.SubtotalCents = subtotal
	inv.TaxCents = taxTotals.Total
	inv.TotalCents = subtotal + taxTotals.Exclusive
	inv.AmountDueCents = inv.TotalCents
	inv.Checksum = buildInvoiceChecksum(inv, items)

	var createdItems []domain.InvoiceItem
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockInvoiceKey(ctx, tx, buildAdjustmentLockKey(orgID, sub.ID, periodStart, periodEnd, idempotencyKey)); err != nil {
			return err
		}
		repo := s.repo.WithTx(tx)
		sequence, err := nextInvoiceSequenceWithDB(ctx, tx, orgID, periodKey)
		if err != nil {
			return err
		}
		inv.Number = renderInvoiceNumber(config.Format, config.Prefix, issueAtLocal, sequence, orgID)
		if err := repo.CreateInvoice(ctx, inv); err != nil {
			return err
		}
		for i := range items {
			items[i].InvoiceID = inv.ID
		}
		if err := repo.CreateInvoiceItems(ctx, items); err != nil {
			return err
		}
		createdItems = items
		return nil
	}); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindInvoiceByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.CreateAdjustmentInvoiceResponse{}, findErr
			}
			if existing != nil {
				itemList, _ := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, existing.ID)
				return domain.CreateAdjustmentInvoiceResponse{Invoice: *existing, Items: derefItems(itemList)}, nil
			}
		}
		return domain.CreateAdjustmentInvoiceResponse{}, err
	}

	openResp, err := s.OpenInvoice(ctx, domain.OpenInvoiceRequest{ID: inv.ID.String()})
	if err != nil {
		return domain.CreateAdjustmentInvoiceResponse{}, err
	}

	resp := domain.CreateAdjustmentInvoiceResponse{Invoice: openResp.Invoice, Items: openResp.Items}
	if len(resp.Items) == 0 && len(createdItems) > 0 {
		resp.Items = createdItems
	}
	s.recordAudit(ctx, "invoice.adjustment.create", "invoice", resp.Invoice.ID.String(), nil, resp.Invoice, meta)
	return resp, nil
}

func (s *service) RecalculateDraftInvoice(ctx context.Context, req domain.RecalculateDraftInvoiceRequest) (domain.RecalculateDraftInvoiceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.RecalculateDraftInvoiceResponse{}, domain.ErrInvalidOrganization
	}
	invoiceID, err := parseID(req.ID)
	if err != nil {
		return domain.RecalculateDraftInvoiceResponse{}, domain.ErrNotFound
	}

	inv, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.RecalculateDraftInvoiceResponse{}, err
	}
	if inv == nil {
		return domain.RecalculateDraftInvoiceResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, inv.CustomerID)
	if err != nil {
		return domain.RecalculateDraftInvoiceResponse{}, err
	}
	if inv.Status != domain.StatusDraft {
		items, _ := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, invoiceID)
		return domain.RecalculateDraftInvoiceResponse{Invoice: *inv, Items: derefItems(items)}, nil
	}

	subID := uuid.Nil
	if inv.SubscriptionID != nil {
		subID = *inv.SubscriptionID
	}
	if subID == uuid.Nil {
		return domain.RecalculateDraftInvoiceResponse{}, domain.ErrNotFound
	}
	sub, err := s.subRepo.FindSubscriptionByID(ctx, orgID, subID)
	if err != nil {
		return domain.RecalculateDraftInvoiceResponse{}, err
	}
	if sub == nil {
		return domain.RecalculateDraftInvoiceResponse{}, domain.ErrNotFound
	}

	periodStart := inv.PeriodStart.UTC()
	periodEnd := inv.PeriodEnd.UTC()
	items, subtotal, taxTotals, err := s.buildDraftInvoiceItems(ctx, orgID, sub, periodStart, periodEnd)
	if err != nil {
		return domain.RecalculateDraftInvoiceResponse{}, err
	}

	updates := map[string]interface{}{
		"subtotal_cents":   subtotal,
		"tax_cents":        taxTotals.Total,
		"total_cents":      subtotal + taxTotals.Exclusive,
		"amount_due_cents": subtotal + taxTotals.Exclusive,
		"updated_at":       s.clock.Now(ctx),
	}

	inv.SubtotalCents = subtotal
	inv.TaxCents = taxTotals.Total
	inv.TotalCents = subtotal + taxTotals.Exclusive
	inv.AmountDueCents = inv.TotalCents
	inv.Checksum = buildInvoiceChecksum(*inv, items)
	updates["checksum"] = inv.Checksum

	var outItems []domain.InvoiceItem
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockInvoiceKey(ctx, tx, buildInvoiceLockKey(orgID, sub.ID, periodStart, periodEnd)); err != nil {
			return err
		}
		repo := s.repo.WithTx(tx)
		if err := repo.DeleteInvoiceItemsByInvoice(ctx, orgID, invoiceID); err != nil {
			return err
		}
		for i := range items {
			items[i].InvoiceID = inv.ID
		}
		if err := repo.CreateInvoiceItems(ctx, items); err != nil {
			return err
		}
		if err := repo.UpdateInvoice(ctx, orgID, invoiceID, updates); err != nil {
			return err
		}
		outItems = items
		return nil
	}); err != nil {
		return domain.RecalculateDraftInvoiceResponse{}, err
	}

	updated, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.RecalculateDraftInvoiceResponse{}, err
	}
	if updated == nil {
		return domain.RecalculateDraftInvoiceResponse{}, domain.ErrNotFound
	}

	resp := domain.RecalculateDraftInvoiceResponse{Invoice: *updated, Items: outItems}
	s.recordAudit(ctx, "invoice.recalculate", "invoice", resp.Invoice.ID.String(), *inv, resp.Invoice, nil)
	return resp, nil
}

func (s *service) GetInvoice(ctx context.Context, req domain.GetInvoiceRequest) (domain.GetInvoiceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.GetInvoiceResponse{}, domain.ErrInvalidOrganization
	}
	invoiceID, err := parseID(req.ID)
	if err != nil {
		return domain.GetInvoiceResponse{}, domain.ErrNotFound
	}

	inv, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.GetInvoiceResponse{}, err
	}
	if inv == nil {
		return domain.GetInvoiceResponse{}, domain.ErrNotFound
	}
	items, err := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, invoiceID)
	if err != nil {
		return domain.GetInvoiceResponse{}, err
	}
	return domain.GetInvoiceResponse{Invoice: *inv, Items: derefItems(items)}, nil
}

func (s *service) OpenInvoice(ctx context.Context, req domain.OpenInvoiceRequest) (resp domain.OpenInvoiceResponse, err error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.OpenInvoiceResponse{}, domain.ErrInvalidOrganization
	}

	ctx, span := telemetry.StartSpan(
		ctx,
		"invoice.open",
		telemetry.UUIDAttr("billing.org_id", orgID),
		telemetry.StringAttr("billing.invoice_id", strings.TrimSpace(req.ID)),
	)
	defer func() { telemetry.EndSpan(span, err) }()

	startedAt := time.Now()
	defer func() { telemetry.ObserveOperation("invoice.open", time.Since(startedAt), err) }()

	invoiceID, err := parseID(req.ID)
	if err != nil {
		return domain.OpenInvoiceResponse{}, domain.ErrInvalidSubscription
	}

	inv, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.OpenInvoiceResponse{}, err
	}
	if inv == nil {
		return domain.OpenInvoiceResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, inv.CustomerID)
	if err != nil {
		return domain.OpenInvoiceResponse{}, err
	}
	if inv.Status != domain.StatusDraft {
		return domain.OpenInvoiceResponse{}, domain.ErrInvalidStatus
	}

	now := s.clock.Now(ctx)
	updates := map[string]interface{}{
		"status":     domain.StatusOpen,
		"updated_at": now,
	}
	if inv.IssuedAt == nil {
		updates["issued_at"] = now
	}

	if err := s.repo.UpdateInvoice(ctx, orgID, invoiceID, updates); err != nil {
		return domain.OpenInvoiceResponse{}, err
	}

	updated, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.OpenInvoiceResponse{}, err
	}
	if updated == nil {
		return domain.OpenInvoiceResponse{}, domain.ErrNotFound
	}

	items, err := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, invoiceID)
	if err != nil {
		return domain.OpenInvoiceResponse{}, err
	}

	if err := s.postLedgerForInvoiceOpen(ctx, *updated, derefItems(items)); err != nil {
		_ = s.repo.UpdateInvoice(ctx, orgID, invoiceID, map[string]interface{}{
			"status":     inv.Status,
			"issued_at":  inv.IssuedAt,
			"updated_at": s.clock.Now(ctx),
		})
		return domain.OpenInvoiceResponse{}, err
	}

	finalInvoice, finalItems, err := s.applyCreditDrawOnOpen(ctx, *updated, derefItems(items))
	if err != nil {
		return domain.OpenInvoiceResponse{}, err
	}

	resp = domain.OpenInvoiceResponse{Invoice: finalInvoice, Items: finalItems}
	span.SetAttributes(
		telemetry.UUIDAttr("billing.invoice_id", finalInvoice.ID),
		telemetry.UUIDAttr("billing.customer_id", finalInvoice.CustomerID),
		telemetry.Int64Attr("billing.invoice_total_cents", finalInvoice.TotalCents),
	)
	s.recordAudit(ctx, "invoice.open", "invoice", resp.Invoice.ID.String(), *inv, resp.Invoice, nil)
	return resp, nil
}

func (s *service) PayInvoice(ctx context.Context, req domain.PayInvoiceRequest) (resp domain.PayInvoiceResponse, err error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PayInvoiceResponse{}, domain.ErrInvalidOrganization
	}

	ctx, span := telemetry.StartSpan(
		ctx,
		"invoice.pay",
		telemetry.UUIDAttr("billing.org_id", orgID),
		telemetry.StringAttr("billing.invoice_id", strings.TrimSpace(req.ID)),
	)
	defer func() { telemetry.EndSpan(span, err) }()

	startedAt := time.Now()
	defer func() { telemetry.ObserveOperation("invoice.pay", time.Since(startedAt), err) }()

	invoiceID, err := parseID(req.ID)
	if err != nil {
		return domain.PayInvoiceResponse{}, domain.ErrInvalidSubscription
	}

	inv, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.PayInvoiceResponse{}, err
	}
	if inv == nil {
		return domain.PayInvoiceResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, inv.CustomerID)
	if err != nil {
		return domain.PayInvoiceResponse{}, err
	}
	if inv.Status != domain.StatusOpen {
		return domain.PayInvoiceResponse{}, domain.ErrInvalidStatus
	}
	paymentAmount := inv.AmountDueCents
	if paymentAmount <= 0 {
		paymentAmount = inv.TotalCents - inv.AmountPaidCents
	}
	if paymentAmount < 0 {
		paymentAmount = 0
	}

	now := s.clock.Now(ctx)
	updates := map[string]interface{}{
		"status":            domain.StatusPaid,
		"updated_at":        now,
		"paid_at":           now,
		"amount_paid_cents": inv.AmountPaidCents + paymentAmount,
		"amount_due_cents":  int64(0),
	}

	if err := s.repo.UpdateInvoice(ctx, orgID, invoiceID, updates); err != nil {
		return domain.PayInvoiceResponse{}, err
	}

	updated, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.PayInvoiceResponse{}, err
	}
	if updated == nil {
		return domain.PayInvoiceResponse{}, domain.ErrNotFound
	}

	items, err := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, invoiceID)
	if err != nil {
		return domain.PayInvoiceResponse{}, err
	}

	if err := s.postLedgerForInvoicePayment(ctx, *updated, paymentAmount); err != nil {
		_ = s.repo.UpdateInvoice(ctx, orgID, invoiceID, map[string]interface{}{
			"status":            inv.Status,
			"paid_at":           inv.PaidAt,
			"amount_paid_cents": inv.AmountPaidCents,
			"amount_due_cents":  inv.AmountDueCents,
			"updated_at":        s.clock.Now(ctx),
		})
		return domain.PayInvoiceResponse{}, err
	}

	resp = domain.PayInvoiceResponse{Invoice: *updated, Items: derefItems(items)}
	span.SetAttributes(
		telemetry.UUIDAttr("billing.invoice_id", updated.ID),
		telemetry.UUIDAttr("billing.customer_id", updated.CustomerID),
		telemetry.Int64Attr("billing.amount_paid_cents", updated.AmountPaidCents),
	)
	s.recordAudit(ctx, "invoice.pay", "invoice", resp.Invoice.ID.String(), *inv, resp.Invoice, nil)
	return resp, nil
}

func (s *service) VoidInvoice(ctx context.Context, req domain.VoidInvoiceRequest) (domain.VoidInvoiceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.VoidInvoiceResponse{}, domain.ErrInvalidOrganization
	}

	invoiceID, err := parseID(req.ID)
	if err != nil {
		return domain.VoidInvoiceResponse{}, domain.ErrInvalidSubscription
	}

	inv, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.VoidInvoiceResponse{}, err
	}
	if inv == nil {
		return domain.VoidInvoiceResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, inv.CustomerID)
	if err != nil {
		return domain.VoidInvoiceResponse{}, err
	}
	if inv.Status != domain.StatusDraft && inv.Status != domain.StatusOpen {
		return domain.VoidInvoiceResponse{}, domain.ErrInvalidStatus
	}

	now := s.clock.Now(ctx)
	updates := map[string]interface{}{
		"status":           domain.StatusVoid,
		"updated_at":       now,
		"voided_at":        now,
		"amount_due_cents": int64(0),
	}

	if err := s.repo.UpdateInvoice(ctx, orgID, invoiceID, updates); err != nil {
		return domain.VoidInvoiceResponse{}, err
	}

	updated, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.VoidInvoiceResponse{}, err
	}
	if updated == nil {
		return domain.VoidInvoiceResponse{}, domain.ErrNotFound
	}

	items, err := s.repo.ListInvoiceItemsByInvoice(ctx, orgID, invoiceID)
	if err != nil {
		return domain.VoidInvoiceResponse{}, err
	}

	if err := s.postLedgerForInvoiceVoid(ctx, *inv); err != nil {
		_ = s.repo.UpdateInvoice(ctx, orgID, invoiceID, map[string]interface{}{
			"status":           inv.Status,
			"voided_at":        inv.VoidedAt,
			"amount_due_cents": inv.AmountDueCents,
			"updated_at":       s.clock.Now(ctx),
		})
		return domain.VoidInvoiceResponse{}, err
	}

	resp := domain.VoidInvoiceResponse{Invoice: *updated, Items: derefItems(items)}
	s.recordAudit(ctx, "invoice.void", "invoice", resp.Invoice.ID.String(), *inv, resp.Invoice, nil)
	return resp, nil
}

func (s *service) ResendInvoice(ctx context.Context, req domain.ResendInvoiceRequest) (domain.ResendInvoiceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ResendInvoiceResponse{}, domain.ErrInvalidOrganization
	}
	invoiceID, err := parseID(req.ID)
	if err != nil {
		return domain.ResendInvoiceResponse{}, domain.ErrNotFound
	}
	inv, err := s.repo.FindInvoiceByID(ctx, orgID, invoiceID)
	if err != nil {
		return domain.ResendInvoiceResponse{}, err
	}
	if inv == nil {
		return domain.ResendInvoiceResponse{}, domain.ErrNotFound
	}

	meta := map[string]interface{}{"channel": "email"}
	s.recordAudit(ctx, "invoice.resend", "invoice", inv.ID.String(), *inv, *inv, meta)

	return domain.ResendInvoiceResponse{Status: "queued"}, nil
}

func (s *service) ListInvoices(ctx context.Context, req domain.ListInvoicesRequest) (domain.ListInvoicesResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListInvoicesResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.InvoiceListFilter{
		Status: strings.ToLower(strings.TrimSpace(req.Status)),
		Number: strings.TrimSpace(req.Number),
	}
	if req.CustomerID != "" {
		id, err := parseUUID(req.CustomerID, domain.ErrInvalidCustomer)
		if err != nil {
			return domain.ListInvoicesResponse{}, err
		}
		filter.CustomerID = id
	}
	if req.SubscriptionID != "" {
		id, err := parseUUID(req.SubscriptionID, domain.ErrInvalidSubscription)
		if err != nil {
			return domain.ListInvoicesResponse{}, err
		}
		filter.SubscriptionID = id
	}
	if filter.Status != "" && !isValidStatus(filter.Status) {
		return domain.ListInvoicesResponse{}, domain.ErrInvalidStatus
	}

	if req.PeriodStartFrom != nil && req.PeriodStartTo != nil && req.PeriodStartTo.Before(*req.PeriodStartFrom) {
		return domain.ListInvoicesResponse{}, domain.ErrInvalidPeriod
	}
	filter.PeriodStartFrom = req.PeriodStartFrom
	filter.PeriodStartTo = req.PeriodStartTo

	if req.IssuedFrom != nil && req.IssuedTo != nil && req.IssuedTo.Before(*req.IssuedFrom) {
		return domain.ListInvoicesResponse{}, domain.ErrInvalidPeriod
	}
	filter.IssuedFrom = req.IssuedFrom
	filter.IssuedTo = req.IssuedTo

	if req.CreatedFrom != nil && req.CreatedTo != nil && req.CreatedTo.Before(*req.CreatedFrom) {
		return domain.ListInvoicesResponse{}, domain.ErrInvalidPeriod
	}
	filter.CreatedFrom = req.CreatedFrom
	filter.CreatedTo = req.CreatedTo

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListInvoicesResponse{}, err
	}

	items, err := s.repo.ListInvoices(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListInvoicesResponse{}, err
	}

	resp := domain.ListInvoicesResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Invoice) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(items) > pageSize {
			items = items[:pageSize]
		}
	}

	resp.Invoices = make([]domain.Invoice, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Invoices = append(resp.Invoices, *item)
	}

	return resp, nil
}

func (s *service) buildDraftInvoiceItems(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription, periodStart, periodEnd time.Time) ([]domain.InvoiceItem, int64, taxTotals, error) {
	items, err := s.buildInvoiceItems(ctx, orgID, sub, periodStart, periodEnd)
	if err != nil {
		return nil, 0, taxTotals{}, err
	}

	var grossSubtotal int64
	for _, item := range items {
		grossSubtotal += item.AmountCents
	}

	discountItems, discountTotal, err := s.buildDiscountLines(ctx, orgID, sub, grossSubtotal, periodStart, periodEnd)
	if err != nil {
		return nil, 0, taxTotals{}, err
	}
	items = append(items, discountItems...)

	netSubtotal := grossSubtotal - discountTotal
	if netSubtotal < 0 {
		netSubtotal = 0
	}

	taxItems, taxes, err := s.buildTaxLines(ctx, orgID, sub, netSubtotal, periodStart, periodEnd)
	if err != nil {
		return nil, 0, taxTotals{}, err
	}
	items = append(items, taxItems...)

	return items, netSubtotal, taxes, nil
}

func (s *service) buildInvoiceItems(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription, periodStart, periodEnd time.Time) ([]domain.InvoiceItem, error) {
	usageItems, err := s.aggregateUsageCharges(ctx, orgID, sub, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	flatItems, err := s.buildFlatCharges(ctx, orgID, sub, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	items := make([]domain.InvoiceItem, 0, len(usageItems)+len(flatItems))
	items = append(items, flatItems...)
	items = append(items, usageItems...)
	return items, nil
}

func (s *service) buildDiscountLines(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription, subtotal int64, periodStart, periodEnd time.Time) ([]domain.InvoiceItem, int64, error) {
	if s.coupons == nil || sub == nil || subtotal <= 0 {
		return nil, 0, nil
	}
	type discountCandidate struct {
		coupon    coupondomain.Coupon
		appliedAt time.Time
		source    string
	}
	var candidates []discountCandidate
	details, err := s.coupons.GetAttachedCouponDetails(ctx, sub.ID)
	if err != nil {
		return nil, 0, err
	}
	if details != nil {
		candidates = append(candidates, discountCandidate{
			coupon:    details.Coupon,
			appliedAt: details.AppliedAt,
			source:    coupondomain.CouponApplicationSourceSubscription,
		})
	}

	autoCoupons, err := s.coupons.ListAutoApplyCoupons(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, 0, err
	}
	customerSegment, err := s.discountSegment(ctx, orgID, sub)
	if err != nil {
		return nil, 0, err
	}
	attachedCouponID := uuid.Nil
	if details != nil {
		attachedCouponID = details.Coupon.ID
	}
	for _, coupon := range autoCoupons {
		if coupon.ID == attachedCouponID {
			continue
		}
		candidates = append(candidates, discountCandidate{
			coupon:    coupon,
			appliedAt: couponAppliedAt(coupon, periodStart),
			source:    coupondomain.CouponApplicationSourceAutoApply,
		})
	}

	now := s.clock.Now(ctx)
	items := make([]domain.InvoiceItem, 0, len(candidates))
	var total int64
	remaining := subtotal
	for _, candidate := range candidates {
		if remaining <= 0 {
			break
		}
		coupon := candidate.coupon
		if !couponMatchesSegment(coupon, customerSegment) {
			continue
		}
		if !couponAppliesToPeriod(coupon, candidate.appliedAt, periodStart, periodEnd) {
			continue
		}
		amount := couponDiscountAmount(coupon, sub.Currency, remaining)
		if amount <= 0 {
			continue
		}
		if amount > remaining {
			amount = remaining
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"coupon_id":   coupon.ID.String(),
			"coupon_type": coupon.Type,
			"applied_at":  candidate.appliedAt.UTC().Format(time.RFC3339),
			"source":      candidate.source,
		})
		item := domain.InvoiceItem{
			ID:              uuid.New(),
			OrgID:           orgID,
			CustomerID:      sub.CustomerID,
			SubscriptionID:  &sub.ID,
			LineType:        domain.LineTypeDiscount,
			Description:     "Coupon discount: " + coupon.Name,
			Quantity:        1,
			UnitAmountCents: float64(amount),
			AmountCents:     amount,
			Currency:        sub.Currency,
			PeriodStart:     &periodStart,
			PeriodEnd:       &periodEnd,
			Metadata:        json.RawMessage(meta),
			CreatedAt:       now,
		}
		items = append(items, item)
		total += amount
		remaining -= amount
	}
	return items, total, nil
}

func (s *service) discountSegment(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription) (string, error) {
	if sub == nil {
		return "", nil
	}
	var segment string
	err := s.db.WithContext(ctx).
		Raw(
			`SELECT COALESCE(NULLIF(c.metadata->>'segment', ''), NULLIF(s.metadata->>'segment', ''), '')
			 FROM subscriptions s
			 JOIN customers c ON c.org_id = s.org_id AND c.id = s.customer_id
			 WHERE s.org_id = ? AND s.id = ? AND s.customer_id = ?
			 LIMIT 1`,
			orgID, sub.ID, sub.CustomerID,
		).
		Scan(&segment).Error
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(segment), nil
}

func couponMatchesSegment(coupon coupondomain.Coupon, segment string) bool {
	if coupon.TargetSegment == nil || strings.TrimSpace(*coupon.TargetSegment) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(*coupon.TargetSegment), strings.TrimSpace(segment))
}

func couponDiscountAmount(coupon coupondomain.Coupon, currency string, subtotal int64) int64 {
	amount := int64(0)
	switch strings.ToUpper(strings.TrimSpace(coupon.Type)) {
	case coupondomain.CouponTypeFixed:
		if coupon.AmountCents == nil || *coupon.AmountCents <= 0 {
			return 0
		}
		if coupon.Currency != nil && strings.TrimSpace(strings.ToUpper(*coupon.Currency)) != strings.TrimSpace(strings.ToUpper(currency)) {
			return 0
		}
		amount = *coupon.AmountCents
	case coupondomain.CouponTypePercent:
		if coupon.Percentage == nil || *coupon.Percentage <= 0 {
			return 0
		}
		amount = roundCents(float64(subtotal) * (*coupon.Percentage / 100.0))
	default:
		return 0
	}
	return amount
}

func couponAppliesToPeriod(coupon coupondomain.Coupon, appliedAt, periodStart, periodEnd time.Time) bool {
	applied := appliedAt.UTC()
	start := periodStart.UTC()
	end := periodEnd.UTC()
	if !couponIntersectsValidWindow(coupon, start, end) {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(coupon.Duration)) {
	case coupondomain.CouponDurationForever:
		return true
	case coupondomain.CouponDurationOnce:
		return (start.Before(applied) || start.Equal(applied)) && end.After(applied)
	case coupondomain.CouponDurationRepeating:
		if coupon.DurationMonths == nil || *coupon.DurationMonths <= 0 {
			return false
		}
		if end.Before(applied) || end.Equal(applied) {
			return false
		}
		return start.Before(applied.AddDate(0, *coupon.DurationMonths, 0))
	default:
		return false
	}
}

func couponIntersectsValidWindow(coupon coupondomain.Coupon, periodStart, periodEnd time.Time) bool {
	if coupon.ValidFrom != nil {
		validFrom := coupon.ValidFrom.UTC()
		if periodEnd.Before(validFrom) || periodEnd.Equal(validFrom) {
			return false
		}
	}
	if coupon.ValidUntil != nil {
		validUntil := coupon.ValidUntil.UTC()
		if periodStart.After(validUntil) || periodStart.Equal(validUntil) {
			return false
		}
	}
	return true
}

func couponAppliedAt(coupon coupondomain.Coupon, fallback time.Time) time.Time {
	if coupon.ValidFrom != nil {
		return coupon.ValidFrom.UTC()
	}
	if !coupon.CreatedAt.IsZero() {
		return coupon.CreatedAt.UTC()
	}
	return fallback.UTC()
}

func (s *service) buildFlatCharges(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription, periodStart, periodEnd time.Time) ([]domain.InvoiceItem, error) {
	var items []*subscriptiondomain.SubscriptionItem
	err := s.db.WithContext(ctx).
		Where("org_id = ? AND subscription_id = ?", orgID, sub.ID).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	now := s.clock.Now(ctx)
	subStart := sub.StartAt
	if sub.TrialEnd != nil && sub.TrialEnd.After(subStart) {
		subStart = sub.TrialEnd.UTC()
	}
	subEnd := sub.EndedAt
	if sub.CancelAt != nil {
		cancelAt := sub.CancelAt.UTC()
		if subEnd == nil || cancelAt.Before(*subEnd) {
			subEnd = &cancelAt
		}
	}
	var out []domain.InvoiceItem
	for _, item := range items {
		if item == nil {
			continue
		}
		price, err := s.planRepo.FindPlanPriceByID(ctx, orgID, item.PlanPriceID)
		if err != nil {
			return nil, err
		}
		if price == nil {
			return nil, fmt.Errorf("plan price %s not found for subscription item %s", item.PlanPriceID, item.ID)
		}
		if price.PriceType != plandomain.PriceTypeFlat {
			continue
		}

		activeStart, activeEnd, active := proration.EffectiveWindow(periodStart, periodEnd, subStart, subEnd, sub.CanceledAt, item.StartAt, item.EndAt)
		if !active {
			zap.L().With(
				zap.String("org_id", orgID.String()),
				zap.String("subscription_id", sub.ID.String()),
				zap.String("item_id", item.ID.String()),
				zap.Time("period_start", periodStart),
				zap.Time("period_end", periodEnd),
				zap.Time("sub_start", subStart),
				zap.Time("item_start", item.StartAt),
			).Info("skipping flat charge: effective window not active")
			continue
		}

		amounts, err := s.listPlanAmountsForPeriod(ctx, orgID, price.ID, sub.Currency, periodStart, periodEnd)
		if err != nil {
			return nil, err
		}
		if len(amounts) == 0 {
			return nil, domain.ErrNotFound
		}

		for _, amount := range amounts {
			if amount == nil {
				continue
			}
			windowStart, windowEnd, ok := intersectWindow(activeStart, activeEnd, amount.EffectiveFrom, amount.EffectiveTo)
			if !ok {
				zap.L().With(
					zap.String("item_id", item.ID.String()),
					zap.String("amount_id", amount.ID.String()),
					zap.Time("active_start", activeStart),
					zap.Time("active_end", activeEnd),
					zap.Time("effective_from", amount.EffectiveFrom),
				).Info("skipping flat charge amount: window does not intersect")
				continue
			}
			factor := proration.Factor(periodStart, periodEnd, windowStart, windowEnd)
			base := float64(amount.UnitAmountCents) * item.Quantity
			total := int64(round(base * factor))

			out = append(out, domain.InvoiceItem{
				ID:              uuid.New(),
				OrgID:           orgID,
				CustomerID:      sub.CustomerID,
				SubscriptionID:  &sub.ID,
				PlanPriceID:     &price.ID,
				LineType:        domain.LineTypeSubscription,
				Description:     price.Name,
				Quantity:        item.Quantity,
				UnitAmountCents: amount.UnitAmountCents,
				AmountCents:     total,
				Currency:        sub.Currency,
				PeriodStart:     &periodStart,
				PeriodEnd:       &periodEnd,
				Metadata:        json.RawMessage(`{}`),
				CreatedAt:       now,
			})
		}
	}

	return out, nil
}

type taxTotals struct {
	Inclusive int64
	Exclusive int64
	Total     int64
}

type taxRate struct {
	ID         uuid.UUID
	Code       string
	Name       string
	Percentage float64
	Inclusive  bool
}

func (s *service) buildTaxLines(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription, subtotal int64, periodStart, periodEnd time.Time) ([]domain.InvoiceItem, taxTotals, error) {
	if subtotal <= 0 || sub == nil {
		return nil, taxTotals{}, nil
	}

	rates, err := s.listActiveTaxRates(ctx, orgID)
	if err != nil {
		return nil, taxTotals{}, err
	}
	if len(rates) == 0 {
		return nil, taxTotals{}, nil
	}

	inclusiveRates := make([]taxRate, 0, len(rates))
	exclusiveRates := make([]taxRate, 0, len(rates))
	sumInclusive := 0.0
	for _, rate := range rates {
		if rate.Inclusive {
			inclusiveRates = append(inclusiveRates, rate)
			sumInclusive += rate.Percentage / 100.0
		} else {
			exclusiveRates = append(exclusiveRates, rate)
		}
	}

	inclusiveBase := float64(subtotal)
	if sumInclusive > 0 {
		inclusiveBase = float64(subtotal) / (1.0 + sumInclusive)
	}
	inclusiveBaseCents := roundCents(inclusiveBase)
	inclusiveTarget := int64(0)
	if sumInclusive > 0 {
		inclusiveTarget = subtotal - inclusiveBaseCents
	}

	now := s.clock.Now(ctx)
	items := make([]domain.InvoiceItem, 0, len(rates))
	totals := taxTotals{}

	if len(inclusiveRates) > 0 {
		inclusiveSum := int64(0)
		for idx, rate := range inclusiveRates {
			amount := roundCents(float64(inclusiveBaseCents) * (rate.Percentage / 100.0))
			if idx == len(inclusiveRates)-1 {
				amount += inclusiveTarget - inclusiveSum - amount
			}
			if amount < 0 {
				amount = 0
			}
			if amount == 0 {
				continue
			}
			inclusiveSum += amount
			items = append(items, taxInvoiceItem(sub, rate, amount, periodStart, periodEnd, now))
		}
		totals.Inclusive = inclusiveSum
	}

	if len(exclusiveRates) > 0 {
		exclusiveSum := int64(0)
		for _, rate := range exclusiveRates {
			amount := roundCents(float64(subtotal) * (rate.Percentage / 100.0))
			if amount <= 0 {
				continue
			}
			exclusiveSum += amount
			items = append(items, taxInvoiceItem(sub, rate, amount, periodStart, periodEnd, now))
		}
		totals.Exclusive = exclusiveSum
	}

	totals.Total = totals.Inclusive + totals.Exclusive
	return items, totals, nil
}

func (s *service) listActiveTaxRates(ctx context.Context, orgID uuid.UUID) ([]taxRate, error) {
	if s.db == nil {
		return nil, nil
	}
	var rows []taxRate
	err := s.db.WithContext(ctx).Raw(
		`SELECT id, code, name, percentage, inclusive
		 FROM tax_rates
		 WHERE org_id = ? AND active = true
		 ORDER BY code ASC`, orgID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *service) listPlanAmountsForPeriod(ctx context.Context, orgID, priceID uuid.UUID, currency string, periodStart, periodEnd time.Time) ([]*plandomain.PlanAmount, error) {
	if s.db == nil {
		return nil, nil
	}
	var amounts []*plandomain.PlanAmount
	err := s.db.WithContext(ctx).
		Where("org_id = ? AND plan_price_id = ? AND currency = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)",
			orgID, priceID, currency, periodEnd, periodStart).
		Order("effective_from asc").
		Find(&amounts).Error
	if err != nil {
		return nil, err
	}
	return amounts, nil
}

func intersectWindow(activeStart, activeEnd, effectiveFrom time.Time, effectiveTo *time.Time) (time.Time, time.Time, bool) {
	start := activeStart
	if effectiveFrom.After(start) {
		start = effectiveFrom
	}
	end := activeEnd
	if effectiveTo != nil && effectiveTo.Before(end) {
		end = *effectiveTo
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func taxInvoiceItem(sub *subscriptiondomain.Subscription, rate taxRate, amount int64, periodStart, periodEnd, now time.Time) domain.InvoiceItem {
	meta := map[string]interface{}{
		"tax_rate_id": rate.ID.String(),
		"tax_code":    rate.Code,
		"percentage":  rate.Percentage,
		"inclusive":   rate.Inclusive,
	}
	metaJSON, _ := json.Marshal(meta)

	return domain.InvoiceItem{
		ID:              uuid.New(),
		OrgID:           sub.OrgID,
		CustomerID:      sub.CustomerID,
		SubscriptionID:  &sub.ID,
		LineType:        domain.LineTypeTax,
		Description:     rate.Name,
		Quantity:        1,
		UnitAmountCents: float64(amount),
		AmountCents:     amount,
		Currency:        sub.Currency,
		PeriodStart:     &periodStart,
		PeriodEnd:       &periodEnd,
		Metadata:        metaJSON,
		CreatedAt:       now,
	}
}

func (s *service) aggregateUsageCharges(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription, periodStart, periodEnd time.Time) ([]domain.InvoiceItem, error) {
	if sub == nil {
		return nil, nil
	}

	usageStart := periodStart
	if sub.TrialEnd != nil {
		trialEnd := sub.TrialEnd.UTC()
		if trialEnd.After(usageStart) {
			usageStart = trialEnd
		}
	}

	usageEnd := periodEnd
	if sub.CancelAt != nil {
		cancelAt := sub.CancelAt.UTC()
		if cancelAt.Before(usageEnd) {
			usageEnd = cancelAt
		}
	}
	if sub.CanceledAt != nil {
		canceledAt := sub.CanceledAt.UTC()
		if canceledAt.Before(usageEnd) {
			usageEnd = canceledAt
		}
	}
	if sub.EndedAt != nil {
		endedAt := sub.EndedAt.UTC()
		if endedAt.Before(usageEnd) {
			usageEnd = endedAt
		}
	}

	if !usageEnd.After(usageStart) {
		return nil, nil
	}

	type row struct {
		PlanPriceID uuid.UUID
		MeterID     uuid.UUID
		Currency    string
		Quantity    float64
		AmountCents int64
	}
	var rows []row
	err := s.db.WithContext(ctx).Raw(
		`SELECT plan_price_id, meter_id, currency, SUM(quantity) as quantity, SUM(amount_cents) as amount_cents
		 FROM rating_results
		 WHERE org_id = ? AND subscription_id = ? AND window_start >= ? AND window_end <= ?
		 GROUP BY plan_price_id, meter_id, currency`,
		orgID, sub.ID, usageStart, usageEnd,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	now := s.clock.Now(ctx)
	customerID := sub.CustomerID
	subscriptionID := sub.ID
	items := make([]domain.InvoiceItem, 0, len(rows))
	for _, r := range rows {
		priceID := r.PlanPriceID
		meterID := r.MeterID
		items = append(items, domain.InvoiceItem{
			ID:              uuid.New(),
			OrgID:           orgID,
			CustomerID:      customerID,
			SubscriptionID:  &subscriptionID,
			PlanPriceID:     &priceID,
			MeterID:         &meterID,
			LineType:        domain.LineTypeUsage,
			Description:     "Usage charges",
			Quantity:        r.Quantity,
			UnitAmountCents: 0,
			AmountCents:     r.AmountCents,
			Currency:        r.Currency,
			PeriodStart:     &periodStart,
			PeriodEnd:       &periodEnd,
			Metadata:        json.RawMessage(`{}`),
			CreatedAt:       now,
		})
	}
	return items, nil
}

func (s *service) hasPendingUsage(ctx context.Context, orgID uuid.UUID, sub *subscriptiondomain.Subscription, periodStart, periodEnd time.Time) (bool, error) {
	if sub == nil {
		return false, nil
	}
	var meterIDs []uuid.UUID
	if err := s.db.WithContext(ctx).Raw(
		`SELECT DISTINCT p.meter_id
		 FROM subscription_items i
		 JOIN plan_prices p ON p.id = i.plan_price_id
		 WHERE i.org_id = ? AND i.subscription_id = ? AND p.meter_id IS NOT NULL
		   AND i.start_at <= ? AND (i.end_at IS NULL OR i.end_at >= ?)`,
		orgID, sub.ID, periodEnd, periodStart,
	).Scan(&meterIDs).Error; err != nil {
		return false, err
	}
	if len(meterIDs) == 0 {
		return false, nil
	}

	var events []usagedomain.UsageEvent
	if err := s.db.WithContext(ctx).
		Model(&usagedomain.UsageEvent{}).
		Select("id").
		Where("org_id = ? AND customer_id = ? AND recorded_at >= ? AND recorded_at <= ? AND status != ?",
			orgID, sub.CustomerID, periodStart, periodEnd, usagedomain.StatusRated).
		Where("meter_id IN ?", meterIDs).
		Limit(1).
		Find(&events).Error; err != nil {
		return false, err
	}
	return len(events) > 0, nil
}

func (s *service) withCustomerTestClock(ctx context.Context, orgID, customerID uuid.UUID) (context.Context, error) {
	if s.customerRepo == nil || customerID == uuid.Nil {
		return ctx, nil
	}
	customer, err := s.customerRepo.FindByID(ctx, orgID, customerID)
	if err != nil {
		return ctx, err
	}
	if customer == nil || customer.TestClockID == nil {
		return ctx, nil
	}
	if s.testClockRepo == nil {
		return ctx, nil
	}
	testClock, err := s.testClockRepo.GetByID(ctx, orgID, *customer.TestClockID)
	if err != nil {
		return ctx, err
	}
	if testClock == nil || testClock.Status != testclockdomain.StatusActive {
		return ctx, nil
	}
	return clock.WithTestClock(ctx, testClock.ID, testClock.CurrentTime), nil
}

const (
	ledgerAccountReceivable = "1100_accounts_receivable"
	ledgerAccountRevenue    = "4000_revenue"
	ledgerAccountTaxPayable = "2100_tax_payable"
	ledgerAccountCash       = "1000_cash"
	ledgerAccountCredits    = "credits"
)

func (s *service) applyCreditDrawOnOpen(ctx context.Context, inv domain.Invoice, items []domain.InvoiceItem) (domain.Invoice, []domain.InvoiceItem, error) {
	if s.ledgerRepo == nil || inv.CustomerID == uuid.Nil || inv.AmountDueCents <= 0 {
		return inv, items, nil
	}

	var updated domain.Invoice
	outItems := items
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockInvoiceKey(ctx, tx, buildCreditDrawLockKey(inv.OrgID, inv.CustomerID, inv.Currency)); err != nil {
			return err
		}

		repo := s.repo.WithTx(tx)
		ledgerRepo := s.ledgerRepo.WithTx(tx)
		current, err := repo.FindInvoiceByID(ctx, inv.OrgID, inv.ID)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrNotFound
		}
		if current.AmountDueCents <= 0 {
			updated = *current
			outItems, _ = s.invoiceItemsWithRepo(ctx, repo, inv.OrgID, inv.ID)
			return nil
		}

		idempotencyKey := "credit_draw:" + inv.ID.String()
		var existingItems int64
		if err := tx.WithContext(ctx).
			Model(&domain.InvoiceItem{}).
			Where("org_id = ? AND invoice_id = ? AND line_type = ? AND idempotency_key = ?", inv.OrgID, inv.ID, domain.LineTypeAdjustment, idempotencyKey).
			Count(&existingItems).Error; err != nil {
			return err
		}
		if existingItems > 0 {
			updated = *current
			outItems, _ = s.invoiceItemsWithRepo(ctx, repo, inv.OrgID, inv.ID)
			return nil
		}

		balance, err := ledgerRepo.GetBalance(ctx, inv.OrgID, inv.CustomerID, ledgerAccountCredits, inv.Currency)
		if err != nil {
			return err
		}
		if balance <= 0 {
			updated = *current
			outItems, _ = s.invoiceItemsWithRepo(ctx, repo, inv.OrgID, inv.ID)
			return nil
		}
		amount := balance
		if amount > current.AmountDueCents {
			amount = current.AmountDueCents
		}
		if amount <= 0 {
			updated = *current
			outItems, _ = s.invoiceItemsWithRepo(ctx, repo, inv.OrgID, inv.ID)
			return nil
		}

		now := s.clock.Now(ctx)
		meta, _ := json.Marshal(map[string]interface{}{
			"source":       "ledger_credit_draw",
			"account_code": ledgerAccountCredits,
		})
		item := domain.InvoiceItem{
			ID:              uuid.New(),
			InvoiceID:       inv.ID,
			OrgID:           inv.OrgID,
			CustomerID:      inv.CustomerID,
			SubscriptionID:  inv.SubscriptionID,
			LineType:        domain.LineTypeAdjustment,
			Description:     "Customer credit applied",
			Quantity:        1,
			UnitAmountCents: float64(amount),
			AmountCents:     amount,
			Currency:        inv.Currency,
			IdempotencyKey:  &idempotencyKey,
			Metadata:        json.RawMessage(meta),
			CreatedAt:       now,
		}
		if err := repo.CreateInvoiceItems(ctx, []domain.InvoiceItem{item}); err != nil {
			return err
		}
		itemList, err := s.invoiceItemsWithRepo(ctx, repo, inv.OrgID, inv.ID)
		if err != nil {
			return err
		}

		newPaid := current.AmountPaidCents + amount
		newDue := current.AmountDueCents - amount
		if newDue < 0 {
			newDue = 0
		}
		updates := map[string]interface{}{
			"amount_paid_cents": newPaid,
			"amount_due_cents":  newDue,
			"updated_at":        now,
			"checksum":          buildInvoiceChecksum(*current, itemList),
		}
		if newDue == 0 {
			updates["status"] = domain.StatusPaid
			updates["paid_at"] = now
		}

		if err := s.createCreditUseLedgerTransactionTx(ctx, ledgerRepo, *current, item.ID, amount, now); err != nil {
			return err
		}
		if err := repo.UpdateInvoice(ctx, inv.OrgID, inv.ID, updates); err != nil {
			return err
		}

		refreshed, err := repo.FindInvoiceByID(ctx, inv.OrgID, inv.ID)
		if err != nil {
			return err
		}
		if refreshed == nil {
			return domain.ErrNotFound
		}
		updated = *refreshed
		outItems = itemList
		return nil
	})
	if err != nil {
		return domain.Invoice{}, nil, err
	}
	if updated.ID == uuid.Nil {
		updated = inv
	}
	return updated, outItems, nil
}

func (s *service) invoiceItemsWithRepo(ctx context.Context, repo domain.Repository, orgID, invoiceID uuid.UUID) ([]domain.InvoiceItem, error) {
	itemPtrs, err := repo.ListInvoiceItemsByInvoice(ctx, orgID, invoiceID)
	if err != nil {
		return nil, err
	}
	return derefItems(itemPtrs), nil
}

func (s *service) createCreditUseLedgerTransactionTx(ctx context.Context, repo ledgerdomain.Repository, inv domain.Invoice, invoiceItemID uuid.UUID, amount int64, occurredAt time.Time) error {
	if amount <= 0 {
		return nil
	}
	if err := ensureLedgerAccount(ctx, repo, inv.OrgID, ledgerAccountCredits, ledgerdomain.LedgerAccountTypeLiability, "Customer Credits"); err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, repo, inv.OrgID, ledgerAccountReceivable, ledgerdomain.LedgerAccountTypeAssets, "Accounts Receivable"); err != nil {
		return err
	}

	idempotencyKey := "invoice_credit_draw:" + inv.ID.String()
	existing, err := repo.FindTransactionByIdempotencyKey(ctx, inv.OrgID, idempotencyKey)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	now := s.clock.Now(ctx)
	sourceType := string(ledgerdomain.SourceTypeCreditUse)
	tx := ledgerdomain.LedgerTransaction{
		ID:             uuid.New(),
		OrgID:          inv.OrgID,
		Currency:       inv.Currency,
		SourceType:     sourceType,
		SourceID:       inv.ID,
		CustomerID:     &inv.CustomerID,
		SubscriptionID: inv.SubscriptionID,
		InvoiceID:      &inv.ID,
		InvoiceItemID:  &invoiceItemID,
		OccurredAt:     occurredAt,
		PostedAt:       now,
		IdempotencyKey: &idempotencyKey,
		Metadata:       json.RawMessage(`{"source":"invoice_credit_draw"}`),
		CreatedAt:      now,
	}
	meta, _ := json.Marshal(map[string]interface{}{
		"invoice_item_id": invoiceItemID.String(),
	})
	entries := []ledgerdomain.LedgerEntry{
		{
			ID:            uuid.New(),
			TransactionID: tx.ID,
			OrgID:         inv.OrgID,
			AccountCode:   ledgerAccountCredits,
			EntryType:     ledgerdomain.LedgerEntryTypeDebit,
			AmountCents:   amount,
			Currency:      inv.Currency,
			Metadata:      json.RawMessage(meta),
			CreatedAt:     now,
		},
		{
			ID:            uuid.New(),
			TransactionID: tx.ID,
			OrgID:         inv.OrgID,
			AccountCode:   ledgerAccountReceivable,
			EntryType:     ledgerdomain.LedgerEntryTypeCredit,
			AmountCents:   amount,
			Currency:      inv.Currency,
			Metadata:      json.RawMessage(meta),
			CreatedAt:     now,
		},
	}

	if err := repo.CreateTransaction(ctx, tx); err != nil {
		return err
	}
	return repo.CreateEntries(ctx, entries)
}

func ensureLedgerAccount(ctx context.Context, repo ledgerdomain.Repository, orgID uuid.UUID, code string, accountType ledgerdomain.LedgerAccountType, name string) error {
	account, err := repo.FindAccountByCode(ctx, orgID, code)
	if err != nil {
		return err
	}
	if account != nil {
		return nil
	}
	return repo.CreateAccount(ctx, ledgerdomain.LedgerAccount{
		ID:        uuid.New(),
		OrgID:     orgID,
		Code:      code,
		Type:      accountType,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	})
}

func (s *service) postLedgerForInvoiceOpen(ctx context.Context, inv domain.Invoice, items []domain.InvoiceItem) error {
	if s.ledger == nil {
		return nil
	}
	total := inv.TotalCents
	if total <= 0 {
		return nil
	}
	taxPayable := inv.TaxCents
	revenue := total - taxPayable
	if revenue < 0 {
		revenue = 0
	}

	invoiceID := inv.ID.String()
	customerID := inv.CustomerID.String()
	var subscriptionID *string
	if inv.SubscriptionID != nil {
		value := inv.SubscriptionID.String()
		subscriptionID = &value
	}
	occurredAt := s.clock.Now(ctx)
	if inv.IssuedAt != nil {
		occurredAt = inv.IssuedAt.UTC()
	}

	entries := []ledgerdomain.LedgerEntryInput{
		{
			AccountCode: ledgerAccountReceivable,
			EntryType:   ledgerdomain.LedgerEntryTypeDebit,
			AmountCents: total,
			Currency:    inv.Currency,
		},
	}
	if revenue > 0 {
		entries = append(entries, ledgerdomain.LedgerEntryInput{
			AccountCode: ledgerAccountRevenue,
			EntryType:   ledgerdomain.LedgerEntryTypeCredit,
			AmountCents: revenue,
			Currency:    inv.Currency,
		})
	}
	if taxPayable > 0 {
		entries = append(entries, ledgerdomain.LedgerEntryInput{
			AccountCode: ledgerAccountTaxPayable,
			EntryType:   ledgerdomain.LedgerEntryTypeCredit,
			AmountCents: taxPayable,
			Currency:    inv.Currency,
		})
	}

	_, err := s.ledger.CreateTransaction(ctx, ledgerdomain.CreateTransactionRequest{
		Currency:       inv.Currency,
		SourceType:     string(ledgerdomain.SourceTypeBillingCycle),
		SourceID:       invoiceID,
		CustomerID:     &customerID,
		SubscriptionID: subscriptionID,
		InvoiceID:      &invoiceID,
		OccurredAt:     &occurredAt,
		IdempotencyKey: "invoice_open:" + invoiceID,
		Entries:        entries,
	})
	return err
}

func (s *service) postLedgerForInvoicePayment(ctx context.Context, inv domain.Invoice, paymentAmount int64) error {
	if s.ledger == nil {
		return nil
	}
	amount := paymentAmount
	if amount <= 0 {
		return nil
	}

	invoiceID := inv.ID.String()
	customerID := inv.CustomerID.String()
	var subscriptionID *string
	if inv.SubscriptionID != nil {
		value := inv.SubscriptionID.String()
		subscriptionID = &value
	}
	occurredAt := s.clock.Now(ctx)
	if inv.PaidAt != nil {
		occurredAt = inv.PaidAt.UTC()
	}

	entries := []ledgerdomain.LedgerEntryInput{
		{
			AccountCode: ledgerAccountCash,
			EntryType:   ledgerdomain.LedgerEntryTypeDebit,
			AmountCents: amount,
			Currency:    inv.Currency,
		},
		{
			AccountCode: ledgerAccountReceivable,
			EntryType:   ledgerdomain.LedgerEntryTypeCredit,
			AmountCents: amount,
			Currency:    inv.Currency,
		},
	}

	_, err := s.ledger.CreateTransaction(ctx, ledgerdomain.CreateTransactionRequest{
		Currency:       inv.Currency,
		SourceType:     string(ledgerdomain.SourceTypePayment),
		SourceID:       invoiceID,
		CustomerID:     &customerID,
		SubscriptionID: subscriptionID,
		InvoiceID:      &invoiceID,
		OccurredAt:     &occurredAt,
		IdempotencyKey: "invoice_pay:" + invoiceID,
		Entries:        entries,
	})
	return err
}

func (s *service) postLedgerForInvoiceVoid(ctx context.Context, inv domain.Invoice) error {
	if s.ledger == nil {
		return nil
	}
	if inv.Status != domain.StatusOpen {
		return nil
	}
	total := inv.TotalCents
	if total <= 0 {
		return nil
	}
	taxPayable := inv.TaxCents
	revenue := total - taxPayable
	if revenue < 0 {
		revenue = 0
	}

	invoiceID := inv.ID.String()
	customerID := inv.CustomerID.String()
	var subscriptionID *string
	if inv.SubscriptionID != nil {
		value := inv.SubscriptionID.String()
		subscriptionID = &value
	}
	occurredAt := s.clock.Now(ctx)
	if inv.VoidedAt != nil {
		occurredAt = inv.VoidedAt.UTC()
	}

	entries := []ledgerdomain.LedgerEntryInput{}
	if revenue > 0 {
		entries = append(entries, ledgerdomain.LedgerEntryInput{
			AccountCode: ledgerAccountRevenue,
			EntryType:   ledgerdomain.LedgerEntryTypeDebit,
			AmountCents: revenue,
			Currency:    inv.Currency,
		})
	}
	if taxPayable > 0 {
		entries = append(entries, ledgerdomain.LedgerEntryInput{
			AccountCode: ledgerAccountTaxPayable,
			EntryType:   ledgerdomain.LedgerEntryTypeDebit,
			AmountCents: taxPayable,
			Currency:    inv.Currency,
		})
	}
	entries = append(entries, ledgerdomain.LedgerEntryInput{
		AccountCode: ledgerAccountReceivable,
		EntryType:   ledgerdomain.LedgerEntryTypeCredit,
		AmountCents: total,
		Currency:    inv.Currency,
	})

	_, err := s.ledger.CreateTransaction(ctx, ledgerdomain.CreateTransactionRequest{
		Currency:       inv.Currency,
		SourceType:     string(ledgerdomain.SourceTypeAdjustment),
		SourceID:       invoiceID,
		CustomerID:     &customerID,
		SubscriptionID: subscriptionID,
		InvoiceID:      &invoiceID,
		OccurredAt:     &occurredAt,
		IdempotencyKey: "invoice_void:" + invoiceID,
		Entries:        entries,
	})
	return err
}

func (s *service) recordAudit(ctx context.Context, action, resourceType, resourceID string, before, after interface{}, meta map[string]interface{}) {
	if s.audit == nil {
		return
	}
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return
	}

	actorType, actorID := auditlog.ActorFromContext(ctx)
	if strings.TrimSpace(actorType) == "" {
		actorType = "system"
	}

	var beforeJSON []byte
	if before != nil {
		beforeJSON, _ = json.Marshal(before)
	}
	var afterJSON []byte
	if after != nil {
		afterJSON, _ = json.Marshal(after)
	}

	var metaJSON []byte
	merged := mergeMetadata(meta, auditlog.MetadataFromContext(ctx))
	if merged != nil {
		metaJSON, _ = json.Marshal(merged)
	}

	var resourcePtr *string
	if strings.TrimSpace(resourceID) != "" {
		resourcePtr = &resourceID
	}

	requestID := strings.TrimSpace(auditlog.RequestIDFromContext(ctx))
	var requestPtr *string
	if requestID != "" {
		requestPtr = &requestID
	}

	reason := strings.TrimSpace(auditlog.ReasonFromContext(ctx))
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}

	_ = s.audit.Record(ctx, auditlog.RecordInput{
		OrgID:        orgID,
		ActorType:    actorType,
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourcePtr,
		BeforeData:   beforeJSON,
		AfterData:    afterJSON,
		Metadata:     metaJSON,
		Reason:       reasonPtr,
		RequestID:    requestPtr,
	})
}

func mergeMetadata(primary, secondary map[string]interface{}) map[string]interface{} {
	if len(primary) == 0 && len(secondary) == 0 {
		return nil
	}
	merged := map[string]interface{}{}
	for key, value := range secondary {
		merged[key] = value
	}
	for key, value := range primary {
		merged[key] = value
	}
	return merged
}

func (s *service) pickPlanAmount(ctx context.Context, orgID, planPriceID uuid.UUID, currency string, at time.Time) (*plandomain.PlanAmount, error) {
	var amount plandomain.PlanAmount
	err := s.db.WithContext(ctx).
		Where("org_id = ? AND plan_price_id = ? AND currency = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)",
			orgID, planPriceID, currency, at, at).
		Order("effective_from desc").
		First(&amount).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &amount, nil
}

type invoiceNumberConfig struct {
	Prefix        string
	Format        string
	SequenceScope string
	Timezone      string
}

func (s *service) getInvoiceNumberConfig(ctx context.Context, orgID uuid.UUID, issueAt time.Time) (invoiceNumberConfig, error) {
	type prefsRow struct {
		InvoicePrefix        string
		InvoiceNumberFormat  string
		InvoiceSequenceScope string
		Timezone             string
	}

	var prefs prefsRow
	err := s.db.WithContext(ctx).
		Raw(`SELECT invoice_prefix, invoice_number_format, invoice_sequence_scope, timezone
		     FROM organization_billing_preferences
		     WHERE org_id = ?`, orgID).
		Scan(&prefs).Error
	if err != nil {
		return invoiceNumberConfig{}, err
	}

	config := invoiceNumberConfig{
		Prefix:        strings.TrimSpace(prefs.InvoicePrefix),
		Format:        strings.TrimSpace(prefs.InvoiceNumberFormat),
		SequenceScope: strings.TrimSpace(prefs.InvoiceSequenceScope),
		Timezone:      strings.TrimSpace(prefs.Timezone),
	}

	type formatRow struct {
		Format        string
		SequenceScope string
	}
	var format formatRow
	err = s.db.WithContext(ctx).
		Raw(`SELECT format, sequence_scope
		     FROM organization_invoice_number_formats
		     WHERE org_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)
		     ORDER BY effective_from DESC
		     LIMIT 1`, orgID, issueAt, issueAt).
		Scan(&format).Error
	if err != nil {
		return invoiceNumberConfig{}, err
	}

	format.Format = strings.TrimSpace(format.Format)
	format.SequenceScope = strings.TrimSpace(format.SequenceScope)
	if format.Format != "" {
		config.Format = format.Format
	}
	if format.SequenceScope != "" {
		config.SequenceScope = format.SequenceScope
	}

	if config.Prefix == "" {
		config.Prefix = "INV"
	}
	if config.Format == "" {
		config.Format = defaultInvoiceNumberFormat
	}
	if config.SequenceScope == "" {
		config.SequenceScope = defaultInvoiceSequenceScope
	}

	return config, nil
}

func (s *service) nextInvoiceSequence(ctx context.Context, orgID uuid.UUID, periodKey string) (int64, error) {
	return nextInvoiceSequenceWithDB(ctx, s.db, orgID, periodKey)
}

func nextInvoiceSequenceWithDB(ctx context.Context, db *gorm.DB, orgID uuid.UUID, periodKey string) (int64, error) {
	var seq int64
	err := db.WithContext(ctx).Raw(
		`INSERT INTO invoice_sequences (org_id, period_key, last_value, updated_at)
		 VALUES (?, ?, 1, ?)
		 ON CONFLICT (org_id, period_key)
		 DO UPDATE SET last_value = invoice_sequences.last_value + 1,
		               updated_at = EXCLUDED.updated_at
		 RETURNING last_value`,
		orgID,
		periodKey,
		time.Now().UTC(),
	).Scan(&seq).Error
	if err != nil {
		return 0, err
	}
	return seq, nil
}

func buildInvoiceLockKey(orgID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time) string {
	return fmt.Sprintf(
		"invoice.generate:%s:%s:%s:%s",
		orgID.String(),
		subscriptionID.String(),
		periodStart.UTC().Format(time.RFC3339Nano),
		periodEnd.UTC().Format(time.RFC3339Nano),
	)
}

func buildCreditDrawLockKey(orgID, customerID uuid.UUID, currency string) string {
	return fmt.Sprintf(
		"invoice.credit_draw:%s:%s:%s",
		orgID.String(),
		customerID.String(),
		strings.ToUpper(strings.TrimSpace(currency)),
	)
}

func lockInvoiceKey(ctx context.Context, tx *gorm.DB, key string) error {
	if tx == nil || tx.Dialector == nil || tx.Dialector.Name() != "postgres" {
		return nil
	}
	lockKey := int64(crc32.ChecksumIEEE([]byte(key)))
	return tx.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", lockKey).Error
}

func buildInvoiceChecksum(inv domain.Invoice, items []domain.InvoiceItem) string {
	type checksumItem struct {
		LineType    string
		PlanPriceID string
		MeterID     string
		Quantity    float64
		UnitAmount  float64
		Amount      int64
		Currency    string
		PeriodStart string
		PeriodEnd   string
		Description string
	}

	checkItems := make([]checksumItem, 0, len(items))
	for _, item := range items {
		planPriceID := ""
		if item.PlanPriceID != nil {
			planPriceID = item.PlanPriceID.String()
		}
		meterID := ""
		if item.MeterID != nil {
			meterID = item.MeterID.String()
		}
		periodStart := ""
		if item.PeriodStart != nil {
			periodStart = item.PeriodStart.UTC().Format(time.RFC3339Nano)
		}
		periodEnd := ""
		if item.PeriodEnd != nil {
			periodEnd = item.PeriodEnd.UTC().Format(time.RFC3339Nano)
		}
		checkItems = append(checkItems, checksumItem{
			LineType:    item.LineType,
			PlanPriceID: planPriceID,
			MeterID:     meterID,
			Quantity:    item.Quantity,
			UnitAmount:  item.UnitAmountCents,
			Amount:      item.AmountCents,
			Currency:    item.Currency,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Description: item.Description,
		})
	}

	sort.Slice(checkItems, func(i, j int) bool {
		a, b := checkItems[i], checkItems[j]
		if a.LineType != b.LineType {
			return a.LineType < b.LineType
		}
		if a.PlanPriceID != b.PlanPriceID {
			return a.PlanPriceID < b.PlanPriceID
		}
		if a.MeterID != b.MeterID {
			return a.MeterID < b.MeterID
		}
		if a.Amount != b.Amount {
			return a.Amount < b.Amount
		}
		if a.Quantity != b.Quantity {
			return a.Quantity < b.Quantity
		}
		if a.UnitAmount != b.UnitAmount {
			return a.UnitAmount < b.UnitAmount
		}
		if a.PeriodStart != b.PeriodStart {
			return a.PeriodStart < b.PeriodStart
		}
		if a.PeriodEnd != b.PeriodEnd {
			return a.PeriodEnd < b.PeriodEnd
		}
		return a.Description < b.Description
	})

	payload := struct {
		OrgID          string         `json:"org_id"`
		CustomerID     string         `json:"customer_id"`
		SubscriptionID string         `json:"subscription_id,omitempty"`
		Currency       string         `json:"currency"`
		PeriodStart    string         `json:"period_start"`
		PeriodEnd      string         `json:"period_end"`
		Subtotal       int64          `json:"subtotal_cents"`
		Tax            int64          `json:"tax_cents"`
		Total          int64          `json:"total_cents"`
		Items          []checksumItem `json:"items"`
	}{
		OrgID:       inv.OrgID.String(),
		CustomerID:  inv.CustomerID.String(),
		Currency:    inv.Currency,
		PeriodStart: inv.PeriodStart.UTC().Format(time.RFC3339Nano),
		PeriodEnd:   inv.PeriodEnd.UTC().Format(time.RFC3339Nano),
		Subtotal:    inv.SubtotalCents,
		Tax:         inv.TaxCents,
		Total:       inv.TotalCents,
		Items:       checkItems,
	}
	if inv.SubscriptionID != nil {
		payload.SubscriptionID = inv.SubscriptionID.String()
	}

	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func buildAdjustmentLockKey(orgID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time, idempotencyKey string) string {
	return fmt.Sprintf(
		"invoice.adjustment:%s:%s:%s:%s:%s",
		orgID.String(),
		subscriptionID.String(),
		periodStart.UTC().Format(time.RFC3339Nano),
		periodEnd.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(idempotencyKey),
	)
}

func buildSequencePeriodKey(scope string, issueAt time.Time) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "org_year":
		return issueAt.Format("2006")
	case "org_global":
		return "global"
	default:
		return issueAt.Format("200601")
	}
}

func renderInvoiceNumber(format, prefix string, issueAt time.Time, sequence int64, orgID uuid.UUID) string {
	out := strings.TrimSpace(format)
	if out == "" {
		out = defaultInvoiceNumberFormat
	}

	out = strings.ReplaceAll(out, "{PREFIX}", prefix)
	out = strings.ReplaceAll(out, "{YYYY}", issueAt.Format("2006"))
	out = strings.ReplaceAll(out, "{YY}", issueAt.Format("06"))
	out = strings.ReplaceAll(out, "{MM}", issueAt.Format("01"))
	out = strings.ReplaceAll(out, "{DD}", issueAt.Format("02"))
	out = strings.ReplaceAll(out, "{ORG}", strings.ToUpper(orgID.String()[:8]))
	out = replaceSequenceToken(out, sequence)

	return out
}

func replaceSequenceToken(value string, sequence int64) string {
	for {
		start := strings.Index(value, "{SEQ")
		if start == -1 {
			return value
		}
		end := strings.Index(value[start:], "}")
		if end == -1 {
			return value
		}

		token := value[start : start+end+1]
		width := 0
		if strings.HasPrefix(token, "{SEQ:") {
			raw := strings.TrimSuffix(strings.TrimPrefix(token, "{SEQ:"), "}")
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				width = parsed
			}
		}

		seqText := strconv.FormatInt(sequence, 10)
		if width > 0 {
			seqText = fmt.Sprintf("%0*d", width, sequence)
		}

		value = strings.Replace(value, token, seqText, 1)
	}
}

const (
	defaultInvoiceNumberFormat  = "{PREFIX}-{YYYY}{MM}-{SEQ:6}"
	defaultInvoiceSequenceScope = "org_month"
)

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidSubscription
	}
	return id, nil
}

func parseUUID(value string, invalidErr error) (uuid.UUID, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return uuid.Nil, invalidErr
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, invalidErr
	}
	return id, nil
}

func decodeCursor(token string) (*domain.ListCursor, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil
	}
	decoded, err := pagination.DecodeCursor(raw)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}
	if decoded == nil || decoded.ID == "" || decoded.CreatedAt == "" {
		return nil, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}
	parsedID, err := uuid.Parse(decoded.ID)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}
	return &domain.ListCursor{
		ID:        parsedID,
		CreatedAt: parsedTime,
	}, nil
}

func isValidStatus(value string) bool {
	switch value {
	case domain.StatusDraft, domain.StatusOpen, domain.StatusPaid, domain.StatusVoid, domain.StatusUncollectible:
		return true
	default:
		return false
	}
}

func round(value float64) float64 {
	if value < 0 {
		return 0
	}
	return float64(int64(value + 0.5))
}

func roundCents(value float64) int64 {
	if value < 0 {
		return 0
	}
	return int64(value + 0.5)
}

func shortID() string {
	return strings.ToUpper(uuid.New().String()[:8])
}

func derefItems(items []*domain.InvoiceItem) []domain.InvoiceItem {
	out := make([]domain.InvoiceItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, *item)
	}
	return out
}
