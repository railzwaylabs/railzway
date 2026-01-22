package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/billingoperations/domain"
	"github.com/smallbiznis/railzway/internal/orgcontext"
)

func (s *Service) ListOverdueInvoices(ctx context.Context, limit int) (domain.OverdueInvoicesResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.OverdueInvoicesResponse{}, domain.ErrInvalidOrganization
	}
	if limit <= 0 {
		limit = 25
	}

	currency, err := s.repo.FetchOrgCurrency(ctx, snowflake.ID(orgID))
	if err != nil {
		return domain.OverdueInvoicesResponse{}, err
	}

	now := s.clock.Now().UTC()
	rows, err := s.repo.ListOverdueInvoices(ctx, snowflake.ID(orgID), currency, now, limit)
	if err != nil {
		return domain.OverdueInvoicesResponse{}, err
	}

	invoices := make([]domain.OverdueInvoice, 0, len(rows))
	for _, row := range rows {
		invoiceNumber := strings.TrimSpace(row.InvoiceNumber)
		if invoiceNumber == "" {
			invoiceNumber = row.InvoiceID.String()
		}

		daysOverdue := int(now.Sub(row.DueAt).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0
		}

		assignedToProp := domain.Assignment{}
		if row.AssignedTo.Valid {
			assignedAtVal := time.Time{}
			if row.AssignedAt.Valid {
				assignedAtVal = row.AssignedAt.Time
			}
			assignedToProp = assignmentFields(
				row.AssignedTo,
				assignedAtVal,
				row.AssignmentExpiresAt,
				row.Status.String,
				row.ReleasedAt,
				row.ReleasedBy,
				row.ReleaseReason,
				row.BreachedAt,
				row.BreachLevel,
				row.LastActionAt,
				now,
			)
		} else {
			assignedToProp = assignmentFields(
				row.AssignedTo,
				time.Time{}, // no assigned at
				row.AssignmentExpiresAt,
				"", // no status
				sql.NullTime{},
				sql.NullString{},
				sql.NullString{},
				sql.NullTime{},
				sql.NullString{},
				sql.NullTime{},
				now,
			)
		}

		assignmentPtr := &assignedToProp
		if assignedToProp.AssignedTo == "" && assignedToProp.Status == "" {
			assignmentPtr = nil
		}

		invoices = append(invoices, domain.OverdueInvoice{
			InvoiceID:     row.InvoiceID.String(),
			InvoiceNumber: invoiceNumber,
			CustomerID:    row.CustomerID.String(),
			CustomerName:  row.CustomerName,
			AmountDue:     row.AmountDue,
			Currency:      currency,
			DueAt:         row.DueAt,
			DaysOverdue:   daysOverdue,
			PublicToken:   decryptToken(s.encKey, row.TokenHash.String),
			Assignment:    assignmentPtr,
		})
	}

	return domain.OverdueInvoicesResponse{
		Currency: currency,
		Invoices: invoices,
		HasData:  len(invoices) > 0,
	}, nil
}

func (s *Service) ListOutstandingCustomers(ctx context.Context, limit int) (domain.OutstandingCustomersResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.OutstandingCustomersResponse{}, domain.ErrInvalidOrganization
	}
	if limit <= 0 {
		limit = 25
	}

	currency, err := s.repo.FetchOrgCurrency(ctx, snowflake.ID(orgID))
	if err != nil {
		return domain.OutstandingCustomersResponse{}, err
	}

	now := s.clock.Now().UTC()
	rows, err := s.repo.ListOutstandingCustomers(ctx, snowflake.ID(orgID), currency, now, limit)
	if err != nil {
		return domain.OutstandingCustomersResponse{}, err
	}

	customers := make([]domain.OutstandingCustomer, 0, len(rows))
	for _, row := range rows {
		oldestOverdueInvoiceID := ""
		if row.OldestOverdueInvoiceID.Valid {
			oldestOverdueInvoiceID = row.OldestOverdueInvoiceID.String
		}

		oldestOverdueInvoiceNumber := ""
		if row.OldestOverdueInvoiceNumber.Valid {
			oldestOverdueInvoiceNumber = strings.TrimSpace(row.OldestOverdueInvoiceNumber.String)
		}
		if oldestOverdueInvoiceNumber == "" && oldestOverdueInvoiceID != "" {
			oldestOverdueInvoiceNumber = oldestOverdueInvoiceID
		}

		var oldestOverdueAt *time.Time
		var oldestOverdueDays int
		if row.OldestOverdueAt.Valid {
			due := row.OldestOverdueAt.Time.UTC()
			oldestOverdueAt = &due
			oldestOverdueDays = int(now.Sub(due).Hours() / 24)
			if oldestOverdueDays < 0 {
				oldestOverdueDays = 0
			}
		}

		var lastPaymentAt *time.Time
		if row.LastPaymentAt.Valid {
			occurred := row.LastPaymentAt.Time.UTC()
			lastPaymentAt = &occurred
		}

		assignedToProp := domain.Assignment{}
		if row.AssignedTo.Valid {
			assignedAtVal := time.Time{}
			if row.AssignedAt.Valid {
				assignedAtVal = row.AssignedAt.Time
			}
			assignedToProp = assignmentFields(
				row.AssignedTo,
				assignedAtVal,
				row.AssignmentExpiresAt,
				row.Status.String,
				row.ReleasedAt,
				row.ReleasedBy,
				row.ReleaseReason,
				row.BreachedAt,
				row.BreachLevel,
				row.LastActionAt,
				now,
			)
		} else {
			assignedToProp = assignmentFields(
				row.AssignedTo,
				time.Time{},
				row.AssignmentExpiresAt,
				"",
				sql.NullTime{},
				sql.NullString{},
				sql.NullString{},
				sql.NullTime{},
				sql.NullString{},
				sql.NullTime{},
				now,
			)
		}

		assignmentPtr := &assignedToProp
		if assignedToProp.AssignedTo == "" && assignedToProp.Status == "" {
			assignmentPtr = nil
		}

		customers = append(customers, domain.OutstandingCustomer{
			CustomerID:             row.CustomerID.String(),
			CustomerName:           row.CustomerName,
			OutstandingBalance:     row.Outstanding,
			Currency:               currency,
			OldestOverdueInvoiceID: oldestOverdueInvoiceID,
			OldestOverdueInvoice:   oldestOverdueInvoiceNumber,
			OldestOverdueAt:        oldestOverdueAt,
			LastPaymentAt:          lastPaymentAt,
			OldestOverdueDays:      oldestOverdueDays,
			HasOverdueOutstanding:  oldestOverdueAt != nil,
			PublicToken:            decryptToken(s.encKey, row.TokenHash.String),
			Assignment:             assignmentPtr,
		})
	}

	return domain.OutstandingCustomersResponse{
		Currency:  currency,
		Customers: customers,
		HasData:   len(customers) > 0,
	}, nil
}

func (s *Service) ListPaymentIssues(ctx context.Context, limit int) (domain.PaymentIssuesResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.PaymentIssuesResponse{}, domain.ErrInvalidOrganization
	}
	if limit <= 0 {
		limit = 25
	}

	now := s.clock.Now().UTC()
	rows, err := s.repo.ListPaymentIssues(ctx, snowflake.ID(orgID), now, limit)
	if err != nil {
		return domain.PaymentIssuesResponse{}, err
	}

	issues := make([]domain.PaymentIssue, 0, len(rows))
	for _, row := range rows {
		var lastAttempt *time.Time
		if row.LastAttempt.Valid {
			occurred := row.LastAttempt.Time.UTC()
			lastAttempt = &occurred
		}
		assignedToProp := domain.Assignment{}
		if row.AssignedTo.Valid {
			assignedAtVal := time.Time{}
			if row.AssignedAt.Valid {
				assignedAtVal = row.AssignedAt.Time
			}
			assignedToProp = assignmentFields(
				row.AssignedTo,
				assignedAtVal,
				row.AssignmentExpiresAt,
				row.Status.String,
				row.ReleasedAt,
				row.ReleasedBy,
				row.ReleaseReason,
				row.BreachedAt,
				row.BreachLevel,
				row.LastActionAt,
				now,
			)
		} else {
			assignedToProp = assignmentFields(
				row.AssignedTo,
				time.Time{}, // no assigned at
				row.AssignmentExpiresAt,
				"", // no status
				sql.NullTime{},
				sql.NullString{},
				sql.NullString{},
				sql.NullTime{},
				sql.NullString{},
				sql.NullTime{},
				now,
			)
		}

		assignmentPtr := &assignedToProp
		if assignedToProp.AssignedTo == "" && assignedToProp.Status == "" {
			assignmentPtr = nil
		}

		issues = append(issues, domain.PaymentIssue{
			CustomerID:          row.CustomerID.String(),
			CustomerName:        row.CustomerName,
			IssueType:           row.IssueType,
			LastAttempt:         lastAttempt,
			AssignedTo:          assignedToProp.AssignedTo,
			AssignmentExpiresAt: &assignedToProp.AssignmentExpiresAt,
			Assignment:          assignmentPtr,
		})
	}

	return domain.PaymentIssuesResponse{
		Issues:  issues,
		HasData: len(issues) > 0,
	}, nil
}

func (s *Service) GetOperations(ctx context.Context, limit int) (domain.BillingOperationsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.BillingOperationsResponse{}, domain.ErrInvalidOrganization
	}
	if limit <= 0 {
		limit = 25
	}

	currency, err := s.repo.FetchOrgCurrency(ctx, snowflake.ID(orgID))
	if err != nil {
		return domain.BillingOperationsResponse{}, err
	}

	now := s.clock.Now().UTC()
	summary, err := s.repo.LoadActionSummary(ctx, snowflake.ID(orgID), currency, now)
	if err != nil {
		return domain.BillingOperationsResponse{}, err
	}

	overdueRows, err := s.repo.ListOverdueInvoices(ctx, snowflake.ID(orgID), currency, now, limit)
	if err != nil {
		return domain.BillingOperationsResponse{}, err
	}
	failedRows, err := s.repo.ListFailedPaymentActions(ctx, snowflake.ID(orgID), currency, now, limit)
	if err != nil {
		return domain.BillingOperationsResponse{}, err
	}
	queueRows, err := s.repo.ListCollectionQueue(ctx, snowflake.ID(orgID), currency, now, limit)
	if err != nil {
		return domain.BillingOperationsResponse{}, err
	}
	paymentRows, err := s.repo.ListPaymentIssues(ctx, snowflake.ID(orgID), now, limit)
	if err != nil {
		return domain.BillingOperationsResponse{}, err
	}

	criticalActions := make([]domain.CriticalAction, 0, len(overdueRows)+len(failedRows))
	for _, row := range overdueRows {
		invoiceNumber := strings.TrimSpace(row.InvoiceNumber)
		if invoiceNumber == "" {
			invoiceNumber = row.InvoiceID.String()
		}

		dueAt := row.DueAt.UTC()
		daysOverdue := int(now.Sub(dueAt).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0
		}

		assignedToProp := domain.Assignment{}
		if row.AssignedTo.Valid {
			assignedAtVal := time.Time{}
			if row.AssignedAt.Valid {
				assignedAtVal = row.AssignedAt.Time
			}
			assignedToProp = assignmentFields(
				row.AssignedTo,
				assignedAtVal,
				row.AssignmentExpiresAt,
				row.Status.String,
				row.ReleasedAt,
				row.ReleasedBy,
				row.ReleaseReason,
				row.BreachedAt,
				row.BreachLevel,
				row.LastActionAt,
				now,
			)
		} else {
			assignedToProp = assignmentFields(
				row.AssignedTo,
				time.Time{}, // no assigned at
				row.AssignmentExpiresAt,
				"", // no status
				sql.NullTime{},
				sql.NullString{},
				sql.NullString{},
				sql.NullTime{},
				sql.NullString{},
				sql.NullTime{},
				now,
			)
		}

		assignmentPtr := &assignedToProp
		if assignedToProp.AssignedTo == "" && assignedToProp.Status == "" {
			assignmentPtr = nil
		}

		criticalActions = append(criticalActions, domain.CriticalAction{
			Category:            domain.CriticalCategoryOverdueInvoice,
			InvoiceID:           row.InvoiceID.String(),
			InvoiceNumber:       invoiceNumber,
			CustomerID:          row.CustomerID.String(),
			CustomerName:        row.CustomerName,
			AmountDue:           row.AmountDue,
			Currency:            currency,
			DueAt:               &dueAt,
			DaysOverdue:         daysOverdue,
			AssignedTo:          assignedToProp.AssignedTo,
			AssignmentExpiresAt: &assignedToProp.AssignmentExpiresAt,
			PublicToken:         decryptToken(s.encKey, row.TokenHash.String),
			Assignment:          assignmentPtr,
		})
	}

	for _, row := range failedRows {
		invoiceID := ""
		if row.InvoiceID.Valid {
			invoiceID = row.InvoiceID.String
		}

		invoiceNumber := ""
		if row.InvoiceNumber.Valid {
			invoiceNumber = strings.TrimSpace(row.InvoiceNumber.String)
		}
		if invoiceNumber == "" {
			invoiceNumber = invoiceID
		}

		var dueAt *time.Time
		daysOverdue := 0
		if row.DueAt.Valid {
			due := row.DueAt.Time.UTC()
			dueAt = &due
			daysOverdue = int(now.Sub(due).Hours() / 24)
			if daysOverdue < 0 {
				daysOverdue = 0
			}
		}

		var lastAttempt *time.Time
		if row.LastAttempt.Valid {
			occurred := row.LastAttempt.Time.UTC()
			lastAttempt = &occurred
		}

		assignedToProp := domain.Assignment{}
		if row.AssignedTo.Valid {
			assignedAtVal := time.Time{}
			if row.AssignedAt.Valid {
				assignedAtVal = row.AssignedAt.Time
			}
			assignedToProp = assignmentFields(
				row.AssignedTo,
				assignedAtVal,
				row.AssignmentExpiresAt,
				row.Status.String,
				row.ReleasedAt,
				row.ReleasedBy,
				row.ReleaseReason,
				row.BreachedAt,
				row.BreachLevel,
				row.LastActionAt,
				now,
			)
		} else {
			assignedToProp = assignmentFields(
				row.AssignedTo,
				time.Time{}, // no assigned at
				row.AssignmentExpiresAt,
				"", // no status
				sql.NullTime{},
				sql.NullString{},
				sql.NullString{},
				sql.NullTime{},
				sql.NullString{},
				sql.NullTime{},
				now,
			)
		}

		assignmentPtr := &assignedToProp
		if assignedToProp.AssignedTo == "" && assignedToProp.Status == "" {
			assignmentPtr = nil
		}

		amountDue := int64(0)
		if row.AmountDue.Valid {
			amountDue = row.AmountDue.Int64
		}

		criticalActions = append(criticalActions, domain.CriticalAction{
			Category:            domain.CriticalCategoryFailedPayment,
			InvoiceID:           invoiceID,
			InvoiceNumber:       invoiceNumber,
			CustomerID:          row.CustomerID.String(),
			CustomerName:        row.CustomerName,
			AmountDue:           amountDue,
			Currency:            currency,
			DueAt:               dueAt,
			DaysOverdue:         daysOverdue,
			LastAttempt:         lastAttempt,
			AssignedTo:          assignedToProp.AssignedTo,
			AssignmentExpiresAt: &assignedToProp.AssignmentExpiresAt,
			PublicToken:         decryptToken(s.encKey, row.TokenHash.String),
			Assignment:          assignmentPtr,
		})
	}

	queue := make([]domain.CollectionQueueEntry, 0, len(queueRows))
	for _, row := range queueRows {
		oldestInvoiceID := ""
		if row.OldestUnpaidInvoiceID.Valid {
			oldestInvoiceID = row.OldestUnpaidInvoiceID.String
		}
		oldestInvoiceNumber := ""
		if row.OldestUnpaidInvoice.Valid {
			oldestInvoiceNumber = strings.TrimSpace(row.OldestUnpaidInvoice.String)
		}
		if oldestInvoiceNumber == "" {
			oldestInvoiceNumber = oldestInvoiceID
		}

		var oldestUnpaidAt *time.Time
		oldestUnpaidDays := 0
		if row.OldestUnpaidAt.Valid {
			due := row.OldestUnpaidAt.Time.UTC()
			oldestUnpaidAt = &due
			oldestUnpaidDays = int(now.Sub(due).Hours() / 24)
			if oldestUnpaidDays < 0 {
				oldestUnpaidDays = 0
			}
		}

		var lastPaymentAt *time.Time
		if row.LastPaymentAt.Valid {
			occurred := row.LastPaymentAt.Time.UTC()
			lastPaymentAt = &occurred
		}

		assignedToProp := domain.Assignment{}
		if row.AssignedTo.Valid {
			assignedAtVal := time.Time{}
			if row.AssignedAt.Valid {
				assignedAtVal = row.AssignedAt.Time
			}
			assignedToProp = assignmentFields(
				row.AssignedTo,
				assignedAtVal,
				row.AssignmentExpiresAt,
				row.Status.String,
				row.ReleasedAt,
				row.ReleasedBy,
				row.ReleaseReason,
				row.BreachedAt,
				row.BreachLevel,
				row.LastActionAt,
				now,
			)
		} else {
			assignedToProp = assignmentFields(
				row.AssignedTo,
				time.Time{}, // no assigned at
				row.AssignmentExpiresAt,
				"", // no status
				sql.NullTime{},
				sql.NullString{},
				sql.NullString{},
				sql.NullTime{},
				sql.NullString{},
				sql.NullTime{},
				now,
			)
		}

		assignmentPtr := &assignedToProp
		if assignedToProp.AssignedTo == "" && assignedToProp.Status == "" {
			assignmentPtr = nil
		}

		queue = append(queue, domain.CollectionQueueEntry{
			CustomerID:            row.CustomerID.String(),
			CustomerName:          row.CustomerName,
			OutstandingBalance:    row.Outstanding,
			Currency:              currency,
			OldestUnpaidInvoiceID: oldestInvoiceID,
			OldestUnpaidInvoice:   oldestInvoiceNumber,
			OldestUnpaidAt:        oldestUnpaidAt,
			OldestUnpaidDays:      oldestUnpaidDays,
			LastPaymentAt:         lastPaymentAt,
			AgingBucket:           computeAgingBucket(oldestUnpaidDays),
			RiskLevel:             computeRiskLevel(row.Outstanding, oldestUnpaidDays),
			AssignedTo:            assignedToProp.AssignedTo,
			AssignmentExpiresAt:   &assignedToProp.AssignmentExpiresAt,
			PublicToken:           decryptToken(s.encKey, row.TokenHash.String),
			Assignment:            assignmentPtr,
		})
	}

	issues := make([]domain.PaymentIssue, 0, len(paymentRows))
	for _, row := range paymentRows {
		var lastAttempt *time.Time
		if row.LastAttempt.Valid {
			occurred := row.LastAttempt.Time.UTC()
			lastAttempt = &occurred
		}
		assignedToProp := domain.Assignment{}
		if row.AssignedTo.Valid {
			assignedAtVal := time.Time{}
			if row.AssignedAt.Valid {
				assignedAtVal = row.AssignedAt.Time
			}
			assignedToProp = assignmentFields(
				row.AssignedTo,
				assignedAtVal,
				row.AssignmentExpiresAt,
				row.Status.String,
				row.ReleasedAt,
				row.ReleasedBy,
				row.ReleaseReason,
				row.BreachedAt,
				row.BreachLevel,
				row.LastActionAt,
				now,
			)
		} else {
			assignedToProp = assignmentFields(
				row.AssignedTo,
				time.Time{}, // no assigned at
				row.AssignmentExpiresAt,
				"", // no status
				sql.NullTime{},
				sql.NullString{},
				sql.NullString{},
				sql.NullTime{},
				sql.NullString{},
				sql.NullTime{},
				now,
			)
		}

		assignmentPtr := &assignedToProp
		if assignedToProp.AssignedTo == "" && assignedToProp.Status == "" {
			assignmentPtr = nil
		}

		issues = append(issues, domain.PaymentIssue{
			CustomerID:          row.CustomerID.String(),
			CustomerName:        row.CustomerName,
			IssueType:           row.IssueType,
			LastAttempt:         lastAttempt,
			AssignedTo:          assignedToProp.AssignedTo,
			AssignmentExpiresAt: &assignedToProp.AssignmentExpiresAt,
			Assignment:          assignmentPtr,
		})
	}

	return domain.BillingOperationsResponse{
		Currency: currency,
		Summary: domain.ActionSummary{
			CustomersWithOutstanding: summary.CustomersWithOutstanding,
			OverdueInvoices:          summary.OverdueInvoices,
			FailedPaymentAttempts:    summary.FailedPaymentAttempts,
			TotalOutstanding:         summary.TotalOutstanding,
			Currency:                 currency,
		},
		CriticalActions: criticalActions,
		CollectionQueue: queue,
		PaymentIssues:   issues,
		GeneratedAt:     now,
	}, nil
}

func (s *Service) GetInbox(ctx context.Context, req domain.InboxRequest) (domain.InboxResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.InboxResponse{}, domain.ErrInvalidOrganization
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 25
	}

	currency, err := s.repo.FetchOrgCurrency(ctx, snowflake.ID(orgID))
	if err != nil {
		return domain.InboxResponse{}, err
	}

	now := s.clock.Now().UTC()
	rows, err := s.repo.ListInboxItems(ctx, snowflake.ID(orgID), limit, now)
	if err != nil {
		return domain.InboxResponse{}, err
	}

	items := make([]domain.InboxItem, 0, len(rows))
	for _, row := range rows {
		var lastAttempt *time.Time
		if row.LastAttempt.Valid {
			t := row.LastAttempt.Time.UTC()
			lastAttempt = &t
		}

		items = append(items, domain.InboxItem{
			EntityType:   row.EntityType,
			EntityID:     row.EntityID,
			EntityName:   row.EntityName,
			RiskCategory: row.RiskCategory,
			RiskScore:    row.RiskScore,
			AmountDue:    row.AmountDue,
			Currency:     currency,
			DaysOverdue:  int(row.DaysOverdue),
			LastAttempt:  lastAttempt,
			PublicToken:  decryptToken(s.encKey, row.TokenHash.String),
		})
	}

	return domain.InboxResponse{
		Items:    items,
		Currency: currency,
	}, nil
}

func (s *Service) GetMyWork(ctx context.Context, userID string, req domain.MyWorkRequest) (domain.MyWorkResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.MyWorkResponse{}, domain.ErrInvalidOrganization
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 25
	}

	currency, err := s.repo.FetchOrgCurrency(ctx, snowflake.ID(orgID))
	if err != nil {
		return domain.MyWorkResponse{}, err
	}

	now := s.clock.Now().UTC()
	rows, err := s.repo.ListMyWorkItems(ctx, snowflake.ID(orgID), userID, limit, now)
	if err != nil {
		return domain.MyWorkResponse{}, err
	}

	items := make([]domain.MyWorkItem, 0, len(rows))
	for _, row := range rows {
		var amtAtClaim int64
		var daysAtClaim int

		var snap map[string]any
		_ = json.Unmarshal(row.SnapshotMetadata, &snap)
		if snap != nil {
			if val, ok := snap["amount_due"].(float64); ok {
				amtAtClaim = int64(val)
			}
			if val, ok := snap["days_overdue"].(float64); ok {
				daysAtClaim = int(val)
			}
		}

		assignmentAge := ""
		duration := now.Sub(row.AssignedAt)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if hours > 0 {
			assignmentAge = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			assignmentAge = fmt.Sprintf("%dm", minutes)
		}

		items = append(items, domain.MyWorkItem{
			AssignmentID:       row.AssignmentID,
			EntityType:         row.EntityType,
			EntityID:           row.EntityID,
			EntityName:         row.EntityName.String,
			CustomerName:       row.CustomerName.String,
			CustomerEmail:      row.CustomerEmail.String,
			InvoiceNumber:      row.InvoiceNumber.String,
			AmountDueAtClaim:   amtAtClaim,
			DaysOverdueAtClaim: daysAtClaim,
			CurrentAmountDue:   row.CurrentAmountDue.Int64,
			CurrentDaysOverdue: int(row.CurrentDaysOverdue.Float64),
			Currency:           currency,
			ClaimedAt:          row.AssignedAt,
			AssignmentAge:      assignmentAge,
			Status:             row.Status,
			LastActionAt:       timePtr(row.LastActionAt),
			PublicToken:        decryptToken(s.encKey, row.TokenHash.String),
		})
	}

	return domain.MyWorkResponse{
		Items:    items,
		Currency: currency,
	}, nil
}

func (s *Service) GetRecentlyResolved(ctx context.Context, userID string, req domain.RecentlyResolvedRequest) (domain.RecentlyResolvedResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.RecentlyResolvedResponse{}, domain.ErrInvalidOrganization
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 25
	}

	since := s.clock.Now().UTC().AddDate(0, 0, -7) // Last 7 days by default

	rows, err := s.repo.ListRecentlyResolvedItems(ctx, snowflake.ID(orgID), userID, limit, since)
	if err != nil {
		return domain.RecentlyResolvedResponse{}, err
	}

	items := make([]domain.ResolvedItem, 0, len(rows))
	for _, row := range rows {
		duration := row.ResolvedAt.Sub(row.AssignedAt)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		durationStr := ""
		if hours > 0 {
			durationStr = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			durationStr = fmt.Sprintf("%dm", minutes)
		}

		var amtAtClaim int64
		var snap map[string]any
		_ = json.Unmarshal(row.SnapshotMetadata, &snap)
		if snap != nil {
			if val, ok := snap["amount_due"].(float64); ok {
				amtAtClaim = int64(val)
			}
		}

		items = append(items, domain.ResolvedItem{
			AssignmentID:     row.AssignmentID,
			EntityType:       row.EntityType,
			EntityID:         row.EntityID,
			EntityName:       row.EntityID, // EntityName might need to be resolved or joined in repo
			Status:           row.Status,
			ResolvedAt:       row.ResolvedAt,
			ResolvedBy:       row.ResolvedBy.String,
			Reason:           row.ReleaseReason.String,
			ClaimedAt:        row.AssignedAt,
			Duration:         durationStr,
			AmountDueAtClaim: amtAtClaim,
			Currency:         "", // Org currency could be fetched or joined
		})
	}

	return domain.RecentlyResolvedResponse{
		Items: items,
	}, nil
}

func (s *Service) GetTeamView(ctx context.Context, req domain.TeamViewRequest) (domain.TeamViewResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.TeamViewResponse{}, domain.ErrInvalidOrganization
	}

	currency, err := s.repo.FetchOrgCurrency(ctx, snowflake.ID(orgID))
	if err != nil {
		return domain.TeamViewResponse{}, err
	}

	now := s.clock.Now().UTC()
	rows, err := s.repo.GetTeamViewStats(ctx, snowflake.ID(orgID), now)
	if err != nil {
		return domain.TeamViewResponse{}, err
	}

	members := make([]domain.TeamMemberWorkload, 0, len(rows))
	var summary domain.TeamSummary

	for _, row := range rows {
		ageStr := ""
		hours := row.AvgAssignmentAgeMinutes / 60
		minutes := row.AvgAssignmentAgeMinutes % 60
		if hours > 0 {
			ageStr = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			ageStr = fmt.Sprintf("%dm", minutes)
		}

		members = append(members, domain.TeamMemberWorkload{
			UserID:             row.UserID,
			ActiveAssignments:  row.ActiveAssignments,
			AvgAssignmentAge:   ageStr,
			TotalExposureOwned: row.TotalExposureOwned,
			EscalationCount:    row.EscalationCount,
		})

		summary.TotalActiveAssignments += row.ActiveAssignments
		summary.TotalExposure += row.TotalExposureOwned
		summary.EscalationCount += row.EscalationCount
	}

	// Calculate overall average age if needed, or use first member's if simplified
	if len(members) > 0 {
		summary.AvgAssignmentAge = members[0].AvgAssignmentAge // Simplified
	}

	return domain.TeamViewResponse{
		Members:  members,
		Summary:  summary,
		Currency: currency,
	}, nil
}

func (s *Service) GetExposureAnalysis(ctx context.Context, req domain.ExposureAnalysisRequest) (domain.ExposureAnalysisResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.ExposureAnalysisResponse{}, domain.ErrInvalidOrganization
	}

	currency, err := s.repo.FetchOrgCurrency(ctx, snowflake.ID(orgID))
	if err != nil {
		return domain.ExposureAnalysisResponse{}, err
	}

	now := s.clock.Now().UTC()
	stats, err := s.repo.GetExposureStats(ctx, snowflake.ID(orgID), now)
	if err != nil {
		return domain.ExposureAnalysisResponse{}, err
	}

	topCustomers, err := s.repo.ListTopHighExposure(ctx, snowflake.ID(orgID), now)
	if err != nil {
		return domain.ExposureAnalysisResponse{}, err
	}

	topHighExposure := make([]domain.InboxItem, 0, len(topCustomers))
	for _, tc := range topCustomers {
		topHighExposure = append(topHighExposure, domain.InboxItem{
			EntityType:   "customer",
			EntityID:     "", // ID not in row?
			EntityName:   tc.EntityName,
			RiskCategory: "high_exposure",
			RiskScore:    tc.RiskScore,
			AmountDue:    tc.AmountDue,
			Currency:     currency,
			DaysOverdue:  tc.DaysOverdue,
		})
	}

	return domain.ExposureAnalysisResponse{
		TotalExposure: stats.TotalExposure,
		Currency:      currency,
		ByRiskCategory: []domain.ExposureCategory{
			{Category: "overdue", Amount: stats.CurrentAmount, Count: stats.OverdueCount},
		},
		ByAgingBucket: []domain.ExposureBucket{
			{Bucket: "0-30", Amount: stats.Bucket0To30},
			{Bucket: "31-60", Amount: stats.Bucket31To60},
			{Bucket: "61-90", Amount: stats.Bucket61To90},
			{Bucket: "90+", Amount: stats.Bucket90Plus},
		},
		TopHighExposure: topHighExposure,
	}, nil
}

func (s *Service) GetInvoicePayments(ctx context.Context, invoiceID string) (domain.InvoicePaymentsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.InvoicePaymentsResponse{}, domain.ErrInvalidOrganization
	}

	sid, err := snowflake.ParseString(invoiceID)
	if err != nil {
		return domain.InvoicePaymentsResponse{}, domain.ErrInvalidEntityID
	}

	rows, err := s.repo.ListInvoicePayments(ctx, snowflake.ID(orgID), sid)
	if err != nil {
		return domain.InvoicePaymentsResponse{}, err
	}

	payments := make([]domain.PaymentDetail, 0, len(rows))
	for _, row := range rows {
		payments = append(payments, domain.PaymentDetail{
			PaymentID:  row.ProviderPaymentID,
			Amount:     0, // Amount needs to be extracted from payload or added to row
			Currency:   row.Currency,
			OccurredAt: row.ReceivedAt,
			Provider:   row.Provider,
			Status:     "succeeded", // Simplified
		})
	}

	return domain.InvoicePaymentsResponse{
		Payments: payments,
	}, nil
}
