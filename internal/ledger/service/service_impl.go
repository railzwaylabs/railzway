package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/ledger/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type service struct {
	db    *gorm.DB
	repo  domain.Repository
	audit *auditlog.Service
}

type Params struct {
	fx.In

	DB    *gorm.DB
	Repo  domain.Repository
	Audit *auditlog.Service `optional:"true"`
}

func NewService(p Params) domain.Service {
	return &service{
		db:    p.DB,
		repo:  p.Repo,
		audit: p.Audit,
	}
}

func (s *service) CreateAccount(ctx context.Context, req domain.CreateAccountRequest) (domain.CreateAccountResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.CreateAccountResponse{}, domain.ErrInvalidOrganization
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return domain.CreateAccountResponse{}, domain.ErrInvalidCode
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.CreateAccountResponse{}, domain.ErrInvalidName
	}
	accountType := strings.TrimSpace(strings.ToLower(req.Type))
	if !isValidAccountType(accountType) {
		return domain.CreateAccountResponse{}, domain.ErrInvalidType
	}

	existing, err := s.repo.FindAccountByCode(ctx, orgID, code)
	if err != nil {
		return domain.CreateAccountResponse{}, err
	}
	if existing != nil {
		return domain.CreateAccountResponse{Account: *existing}, nil
	}

	now := time.Now().UTC()
	account := domain.LedgerAccount{
		ID:        uuid.New(),
		OrgID:     orgID,
		Code:      code,
		Type:      domain.LedgerAccountType(accountType),
		Name:      name,
		CreatedAt: now,
	}

	if err := s.repo.CreateAccount(ctx, account); err != nil {
		return domain.CreateAccountResponse{}, err
	}

	resp := domain.CreateAccountResponse{Account: account}
	s.recordAudit(ctx, "ledger.account.create", "ledger_account", account.ID.String(), nil, account, nil)
	return resp, nil
}

func (s *service) CreateTransaction(ctx context.Context, req domain.CreateTransactionRequest) (resp domain.CreateTransactionResponse, err error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidOrganization
	}

	ctx, span := telemetry.StartSpan(
		ctx,
		"ledger.transaction.create",
		telemetry.UUIDAttr("billing.org_id", orgID),
		telemetry.StringAttr("billing.source_type", strings.TrimSpace(req.SourceType)),
		telemetry.StringAttr("billing.source_id", strings.TrimSpace(req.SourceID)),
		telemetry.Int64Attr("billing.entries_count", int64(len(req.Entries))),
		telemetry.StringAttr("billing.idempotency_key", strings.TrimSpace(req.IdempotencyKey)),
	)
	defer func() { telemetry.EndSpan(span, err) }()

	startedAt := time.Now()
	defer func() { telemetry.ObserveOperation("ledger.transaction.create", time.Since(startedAt), err) }()

	currency := strings.TrimSpace(strings.ToUpper(req.Currency))
	if currency == "" || len(currency) != 3 {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidCurrency
	}
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidSource
	}
	sourceID, err := parseID(req.SourceID)
	if err != nil {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidSource
	}
	if len(req.Entries) < 2 {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidEntry
	}

	customerID, err := parseOptionalUUID(req.CustomerID)
	if err != nil {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidSource
	}
	subscriptionID, err := parseOptionalUUID(req.SubscriptionID)
	if err != nil {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidSource
	}
	invoiceID, err := parseOptionalUUID(req.InvoiceID)
	if err != nil {
		return domain.CreateTransactionResponse{}, domain.ErrInvalidSource
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindTransactionByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.CreateTransactionResponse{}, err
		}
		if existing != nil {
			entries, _ := s.repo.ListEntriesByTransaction(ctx, orgID, existing.ID)
			resp = domain.CreateTransactionResponse{Transaction: *existing, Entries: entries}
			span.SetAttributes(
				telemetry.UUIDAttr("billing.transaction_id", existing.ID),
				telemetry.BoolAttr("billing.idempotent_hit", true),
			)
			return resp, nil
		}
	}

	occurredAt := time.Now().UTC()
	if req.OccurredAt != nil {
		occurredAt = req.OccurredAt.UTC()
	}

	now := time.Now().UTC()
	tx := domain.LedgerTransaction{
		ID:             uuid.New(),
		OrgID:          orgID,
		Currency:       currency,
		SourceType:     sourceType,
		SourceID:       sourceID,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		CustomerID:     customerID,
		SubscriptionID: subscriptionID,
		InvoiceID:      invoiceID,
		OccurredAt:     occurredAt,
		PostedAt:       now,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		IdempotencyKey: nil,
	}
	if idempotencyKey != "" {
		tx.IdempotencyKey = &idempotencyKey
	}

	debitTotal := int64(0)
	creditTotal := int64(0)
	entries := make([]domain.LedgerEntry, 0, len(req.Entries))
	for _, entry := range req.Entries {
		accountCode := strings.TrimSpace(entry.AccountCode)
		if accountCode == "" {
			return domain.CreateTransactionResponse{}, domain.ErrInvalidEntry
		}
		entryType := strings.TrimSpace(strings.ToLower(string(entry.EntryType)))
		if entryType != string(domain.LedgerEntryTypeDebit) && entryType != string(domain.LedgerEntryTypeCredit) {
			return domain.CreateTransactionResponse{}, domain.ErrInvalidEntry
		}
		if entry.AmountCents <= 0 {
			return domain.CreateTransactionResponse{}, domain.ErrInvalidAmount
		}
		entryCurrency := strings.TrimSpace(strings.ToUpper(entry.Currency))
		if entryCurrency == "" {
			entryCurrency = currency
		}
		if entryCurrency != currency {
			return domain.CreateTransactionResponse{}, domain.ErrInvalidCurrency
		}

		account, err := s.repo.FindAccountByCode(ctx, orgID, accountCode)
		if err != nil {
			return domain.CreateTransactionResponse{}, err
		}
		if account == nil {
			return domain.CreateTransactionResponse{}, domain.ErrNotFound
		}

		if entryType == string(domain.LedgerEntryTypeDebit) {
			debitTotal += entry.AmountCents
		} else {
			creditTotal += entry.AmountCents
		}

		meta := entry.Metadata
		if meta == nil {
			meta = json.RawMessage(`{}`)
		}

		entryRecord := domain.LedgerEntry{
			ID:            uuid.New(),
			TransactionID: tx.ID,
			OrgID:         orgID,
			AccountID:     &account.ID,
			AccountCode:   account.Code,
			EntryType:     domain.LedgerEntryType(entryType),
			AmountCents:   entry.AmountCents,
			Currency:      currency,
			Description:   entry.Description,
			Metadata:      meta,
			CreatedAt:     now,
		}
		entries = append(entries, entryRecord)
	}

	if debitTotal != creditTotal {
		return domain.CreateTransactionResponse{}, domain.ErrUnbalancedEntry
	}

	if err := s.db.WithContext(ctx).Transaction(func(txDb *gorm.DB) error {
		repo := s.repo.WithTx(txDb)
		if err := repo.CreateTransaction(ctx, tx); err != nil {
			return err
		}
		return repo.CreateEntries(ctx, entries)
	}); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindTransactionByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.CreateTransactionResponse{}, findErr
			}
			if existing != nil {
				entryList, _ := s.repo.ListEntriesByTransaction(ctx, orgID, existing.ID)
				resp = domain.CreateTransactionResponse{Transaction: *existing, Entries: entryList}
				span.SetAttributes(
					telemetry.UUIDAttr("billing.transaction_id", existing.ID),
					telemetry.BoolAttr("billing.idempotent_hit", true),
				)
				return resp, nil
			}
		}
		return domain.CreateTransactionResponse{}, err
	}

	resp = domain.CreateTransactionResponse{Transaction: tx, Entries: entries}
	span.SetAttributes(
		telemetry.UUIDAttr("billing.transaction_id", tx.ID),
		telemetry.Int64Attr("billing.debit_total_cents", debitTotal),
		telemetry.Int64Attr("billing.credit_total_cents", creditTotal),
	)
	s.recordAudit(ctx, "ledger.transaction.create", "ledger_transaction", tx.ID.String(), nil, tx, nil)
	return resp, nil
}

func (s *service) ListAccounts(ctx context.Context) (domain.ListAccountsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListAccountsResponse{}, domain.ErrInvalidOrganization
	}

	accounts, err := s.repo.ListAccounts(ctx, orgID)
	if err != nil {
		return domain.ListAccountsResponse{}, err
	}

	return domain.ListAccountsResponse{Accounts: accounts}, nil
}

func (s *service) ListTransactions(ctx context.Context, req domain.ListTransactionsRequest) (domain.ListTransactionsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListTransactionsResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.TransactionListFilter{
		SourceType: strings.TrimSpace(req.SourceType),
	}
	if req.SourceID != "" {
		id, err := parseUUID(req.SourceID, domain.ErrInvalidSource)
		if err != nil {
			return domain.ListTransactionsResponse{}, err
		}
		filter.SourceID = id
	}
	if req.CustomerID != "" {
		id, err := parseUUID(req.CustomerID, domain.ErrInvalidSource)
		if err != nil {
			return domain.ListTransactionsResponse{}, err
		}
		filter.CustomerID = id
	}
	if req.SubscriptionID != "" {
		id, err := parseUUID(req.SubscriptionID, domain.ErrInvalidSource)
		if err != nil {
			return domain.ListTransactionsResponse{}, err
		}
		filter.SubscriptionID = id
	}
	if req.InvoiceID != "" {
		id, err := parseUUID(req.InvoiceID, domain.ErrInvalidSource)
		if err != nil {
			return domain.ListTransactionsResponse{}, err
		}
		filter.InvoiceID = id
	}
	if req.PlanPriceID != "" {
		id, err := parseUUID(req.PlanPriceID, domain.ErrInvalidSource)
		if err != nil {
			return domain.ListTransactionsResponse{}, err
		}
		filter.PlanPriceID = id
	}
	if req.MeterID != "" {
		id, err := parseUUID(req.MeterID, domain.ErrInvalidSource)
		if err != nil {
			return domain.ListTransactionsResponse{}, err
		}
		filter.MeterID = id
	}
	if req.OccurredFrom != nil && req.OccurredTo != nil && req.OccurredTo.Before(*req.OccurredFrom) {
		return domain.ListTransactionsResponse{}, domain.ErrInvalidSource
	}
	filter.OccurredFrom = req.OccurredFrom
	filter.OccurredTo = req.OccurredTo
	if req.PostedFrom != nil && req.PostedTo != nil && req.PostedTo.Before(*req.PostedFrom) {
		return domain.ListTransactionsResponse{}, domain.ErrInvalidSource
	}
	filter.PostedFrom = req.PostedFrom
	filter.PostedTo = req.PostedTo

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListTransactionsResponse{}, err
	}

	items, err := s.repo.ListTransactions(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListTransactionsResponse{}, err
	}

	resp := domain.ListTransactionsResponse{}
	itemPtrs := make([]*domain.LedgerTransaction, len(items))
	for i := range items {
		itemPtrs[i] = &items[i]
	}
	pageInfo := pagination.BuildCursorPageInfo(itemPtrs, int32(pageSize), func(item *domain.LedgerTransaction) string {
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

	resp.Transactions = items
	return resp, nil
}

func (s *service) GetTransaction(ctx context.Context, req domain.GetTransactionRequest) (domain.GetTransactionResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.GetTransactionResponse{}, domain.ErrInvalidOrganization
	}

	txID, err := parseID(req.ID)
	if err != nil {
		return domain.GetTransactionResponse{}, domain.ErrInvalidSource
	}

	tx, err := s.repo.FindTransactionByID(ctx, orgID, txID)
	if err != nil {
		return domain.GetTransactionResponse{}, err
	}
	if tx == nil {
		return domain.GetTransactionResponse{}, domain.ErrNotFound
	}

	entries, err := s.repo.ListEntriesByTransaction(ctx, orgID, txID)
	if err != nil {
		return domain.GetTransactionResponse{}, err
	}

	return domain.GetTransactionResponse{Transaction: *tx, Entries: entries}, nil
}

func isValidAccountType(value string) bool {
	switch value {
	case string(domain.LedgerAccountTypeAssets),
		string(domain.LedgerAccountTypeLiability),
		string(domain.LedgerAccountTypeIncome),
		string(domain.LedgerAccountTypeExpense),
		string(domain.LedgerAccountTypeEquity):
		return true
	default:
		return false
	}
}

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidSource
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

func parseOptionalUUID(value *string) (*uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return nil, domain.ErrInvalidSource
	}
	return &id, nil
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
