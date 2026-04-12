package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidCode         = errors.New("invalid_code")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidType         = errors.New("invalid_type")
	ErrInvalidCurrency     = errors.New("invalid_currency")
	ErrInvalidEntry        = errors.New("invalid_entry")
	ErrInvalidAmount       = errors.New("invalid_amount")
	ErrInvalidSource       = errors.New("invalid_source")
	ErrInvalidCursor       = errors.New("invalid_cursor")
	ErrUnbalancedEntry     = errors.New("unbalanced_entry")
	ErrNotFound            = errors.New("not_found")
)

type CreateAccountRequest struct {
	Code string
	Type string
	Name string
}

type CreateAccountResponse struct {
	Account LedgerAccount `json:"account"`
}

type LedgerEntryInput struct {
	AccountCode string          `json:"account_code"`
	EntryType   LedgerEntryType `json:"entry_type"`
	AmountCents int64           `json:"amount_cents"`
	Currency    string          `json:"currency"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type CreateTransactionRequest struct {
	Currency       string
	SourceType     string
	SourceID       string
	ReferenceType  *string
	ReferenceID    *string
	CustomerID     *string
	SubscriptionID *string
	InvoiceID      *string
	OccurredAt     *time.Time
	IdempotencyKey string
	Entries        []LedgerEntryInput
}

type CreateTransactionResponse struct {
	Transaction LedgerTransaction `json:"transaction"`
	Entries     []LedgerEntry     `json:"entries"`
}

type ListAccountsResponse struct {
	Accounts []LedgerAccount `json:"accounts"`
}

type ListTransactionsRequest struct {
	PageToken      string
	PageSize       int32
	SourceType     string
	SourceID       string
	CustomerID     string
	SubscriptionID string
	InvoiceID      string
	PlanPriceID    string
	MeterID        string
	OccurredFrom   *time.Time
	OccurredTo     *time.Time
	PostedFrom     *time.Time
	PostedTo       *time.Time
}

type ListTransactionsResponse struct {
	pagination.PageInfo
	Transactions []LedgerTransaction `json:"transactions"`
}

type GetTransactionRequest struct {
	ID string
}

type GetTransactionResponse struct {
	Transaction LedgerTransaction `json:"transaction"`
	Entries     []LedgerEntry     `json:"entries"`
}

type Service interface {
	CreateAccount(ctx context.Context, req CreateAccountRequest) (CreateAccountResponse, error)
	CreateTransaction(ctx context.Context, req CreateTransactionRequest) (CreateTransactionResponse, error)
	ListAccounts(ctx context.Context) (ListAccountsResponse, error)
	GetTransaction(ctx context.Context, req GetTransactionRequest) (GetTransactionResponse, error)
	ListTransactions(ctx context.Context, req ListTransactionsRequest) (ListTransactionsResponse, error)
}
