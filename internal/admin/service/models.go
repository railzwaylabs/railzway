package service

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SummaryAlert struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Tag      string `json:"tag"`
}

type SummaryHighlight struct {
	Name string `json:"name"`
	Note string `json:"note"`
	Tag  string `json:"tag"`
}

type InvoiceHighlight struct {
	Number string `json:"number"`
	Note   string `json:"note"`
	Tag    string `json:"tag"`
}

type AuditLogEntry struct {
	Title string `json:"title"`
	Note  string `json:"note"`
	Tag   string `json:"tag"`
}

type DashboardSummary struct {
	MRRCents     int64          `json:"mrrCents"`
	UsageCents   int64          `json:"usageCents"`
	OpenInvoices int64          `json:"openInvoices"`
	LateEvents   int64          `json:"lateEvents"`
	Alerts       []SummaryAlert `json:"alerts"`
}

type CustomersSummary struct {
	Active     int64              `json:"active"`
	AtRisk     int64              `json:"atRisk"`
	NRRPct     float64            `json:"nrrPct"`
	Highlights []SummaryHighlight `json:"highlights"`
}

type PlansSummary struct {
	Active     int64              `json:"active"`
	Draft      int64              `json:"draft"`
	Tiered     int64              `json:"tiered"`
	Highlights []SummaryHighlight `json:"highlights"`
}

type SubscriptionsSummary struct {
	Active     int64              `json:"active"`
	Trialing   int64              `json:"trialing"`
	PastDue    int64              `json:"pastDue"`
	Highlights []SummaryHighlight `json:"highlights"`
}

type UsageSummary struct {
	EventsPerHour int64              `json:"eventsPerHour"`
	LatePct       float64            `json:"latePct"`
	ActiveMeters  int64              `json:"activeMeters"`
	Highlights    []SummaryHighlight `json:"highlights"`
}

type RatingSummary struct {
	RatedEvents   int64              `json:"ratedEvents"`
	AvgLatencySec float64            `json:"avgLatencySec"`
	ReplaysToday  int64              `json:"replaysToday"`
	Highlights    []SummaryHighlight `json:"highlights"`
}

type InvoicesSummary struct {
	Draft      int64              `json:"draft"`
	Open       int64              `json:"open"`
	PaidCents  int64              `json:"paidCents"`
	Highlights []InvoiceHighlight `json:"highlights"`
}

type PaymentsSummary struct {
	CollectedCents int64              `json:"collectedCents"`
	Failed         int64              `json:"failed"`
	Retries        int64              `json:"retries"`
	Highlights     []SummaryHighlight `json:"highlights"`
}

type TaxesSummary struct {
	Profiles        int64              `json:"profiles"`
	ExemptCustomers int64              `json:"exemptCustomers"`
	Highlights      []SummaryHighlight `json:"highlights"`
}

type AuditLogsSummary struct {
	Entries []AuditLogEntry `json:"entries"`
}

type SettingsSummary struct {
	APIKeys       int64              `json:"apiKeys"`
	InvoiceFormat string             `json:"invoiceFormat"`
	Timezone      string             `json:"timezone"`
	Highlights    []SummaryHighlight `json:"highlights"`
}

type ConfigWarning struct {
	Module  string `json:"module"`
	Code    string `json:"code"`
	Link    string `json:"link,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ConfigWarningsResponse struct {
	Warnings []ConfigWarning `json:"warnings"`
}

type ReconciliationMismatch struct {
	Action    string          `json:"action"`
	InvoiceID string          `json:"invoiceId"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"createdAt"`
}

type ReconciliationSummaryResponse struct {
	WindowDays       int                      `json:"windowDays"`
	UsageMismatches  int64                    `json:"usageMismatches"`
	LedgerMismatches int64                    `json:"ledgerMismatches"`
	TotalMismatches  int64                    `json:"totalMismatches"`
	Latest           []ReconciliationMismatch `json:"latest"`
}

type SummarySnapshot struct {
	OrgID         uuid.UUID            `json:"orgId"`
	Dashboard     DashboardSummary     `json:"dashboard"`
	Customers     CustomersSummary     `json:"customers"`
	Plans         PlansSummary         `json:"plans"`
	Subscriptions SubscriptionsSummary `json:"subscriptions"`
	Usage         UsageSummary         `json:"usage"`
	Rating        RatingSummary        `json:"rating"`
	Invoices      InvoicesSummary      `json:"invoices"`
	Payments      PaymentsSummary      `json:"payments"`
	Taxes         TaxesSummary         `json:"taxes"`
	AuditLogs     AuditLogsSummary     `json:"auditLogs"`
	Settings      SettingsSummary      `json:"settings"`
	Source        string               `json:"source"`
	RefreshedAt   time.Time            `json:"refreshedAt"`
}

type AdminSummary struct {
	OrgID         uuid.UUID       `gorm:"primaryKey" json:"org_id"`
	Dashboard     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"dashboard"`
	Customers     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"customers"`
	Plans         json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"plans"`
	Subscriptions json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"subscriptions"`
	Usage         json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"usage"`
	Rating        json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"rating"`
	Invoices      json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"invoices"`
	Payments      json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"payments"`
	Taxes         json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"taxes"`
	AuditLogs     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"audit_logs"`
	Settings      json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	Source        string          `gorm:"type:text;not null;default:'materialized'" json:"source"`
	RefreshedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"refreshed_at"`
}

func (AdminSummary) TableName() string { return "admin_summaries" }
