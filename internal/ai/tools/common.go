package tools

import (
	"strings"
	"time"

	genkitai "github.com/firebase/genkit/go/ai"
)

const (
	defaultToolPageSize = int32(5)
	maxToolPageSize     = int32(20)
)

type AssistantToolset struct {
	refs []genkitai.ToolRef
}

type idInput struct {
	ID string `json:"id" jsonschema:"description=Resource identifier"`
}

type pageInput struct {
	PageSize int32 `json:"page_size" jsonschema:"description=Maximum number of items to return"`
}

type customerListInput struct {
	PageSize int32  `json:"page_size" jsonschema:"description=Maximum number of customers to return"`
	Name     string `json:"name,omitempty" jsonschema:"description=Optional customer name filter"`
	Email    string `json:"email,omitempty" jsonschema:"description=Optional customer email filter"`
	Currency string `json:"currency,omitempty" jsonschema:"description=Optional ISO currency filter"`
}

type productListInput struct {
	PageSize int32  `json:"page_size" jsonschema:"description=Maximum number of products to return"`
	Code     string `json:"code,omitempty" jsonschema:"description=Optional product code filter"`
	Name     string `json:"name,omitempty" jsonschema:"description=Optional product name filter"`
	Active   *bool  `json:"active,omitempty" jsonschema:"description=Optional active status filter"`
}

type createProductInput struct {
	Code        string  `json:"code" jsonschema:"description=Unique product code, lowercase or kebab case"`
	Name        string  `json:"name" jsonschema:"description=Human-readable product name"`
	Description *string `json:"description,omitempty" jsonschema:"description=Optional product description"`
	Active      *bool   `json:"active,omitempty" jsonschema:"description=Optional active status flag"`
}

type featureListInput struct {
	PageSize    int32  `json:"page_size" jsonschema:"description=Maximum number of features to return"`
	Code        string `json:"code,omitempty" jsonschema:"description=Optional feature code filter"`
	Name        string `json:"name,omitempty" jsonschema:"description=Optional feature name filter"`
	FeatureType string `json:"feature_type,omitempty" jsonschema:"description=Optional feature type filter"`
	Active      *bool  `json:"active,omitempty" jsonschema:"description=Optional active status filter"`
}

type planListInput struct {
	PageSize  int32   `json:"page_size" jsonschema:"description=Maximum number of plans to return"`
	ProductID *string `json:"product_id,omitempty" jsonschema:"description=Optional product identifier"`
	Code      string  `json:"code,omitempty" jsonschema:"description=Optional plan code filter"`
	Name      string  `json:"name,omitempty" jsonschema:"description=Optional plan name filter"`
	Active    *bool   `json:"active,omitempty" jsonschema:"description=Optional active status filter"`
}

type subscriptionListInput struct {
	PageSize   int32  `json:"page_size" jsonschema:"description=Maximum number of subscriptions to return"`
	CustomerID string `json:"customer_id,omitempty" jsonschema:"description=Optional customer identifier"`
	Status     string `json:"status,omitempty" jsonschema:"description=Optional subscription status filter"`
}

type invoiceListInput struct {
	PageSize       int32  `json:"page_size" jsonschema:"description=Maximum number of invoices to return"`
	CustomerID     string `json:"customer_id,omitempty" jsonschema:"description=Optional customer identifier"`
	SubscriptionID string `json:"subscription_id,omitempty" jsonschema:"description=Optional subscription identifier"`
	Status         string `json:"status,omitempty" jsonschema:"description=Optional invoice status filter"`
	Number         string `json:"number,omitempty" jsonschema:"description=Optional invoice number filter"`
}

type meterListInput struct {
	PageSize int32  `json:"page_size" jsonschema:"description=Maximum number of meters to return"`
	Code     string `json:"code,omitempty" jsonschema:"description=Optional meter code filter"`
	Name     string `json:"name,omitempty" jsonschema:"description=Optional meter name filter"`
	Active   *bool  `json:"active,omitempty" jsonschema:"description=Optional active status filter"`
}

type usageListInput struct {
	PageSize   int32  `json:"page_size" jsonschema:"description=Maximum number of usage events to return"`
	MeterID    string `json:"meter_id,omitempty" jsonschema:"description=Optional meter identifier"`
	CustomerID string `json:"customer_id,omitempty" jsonschema:"description=Optional customer identifier"`
	Status     string `json:"status,omitempty" jsonschema:"description=Optional usage status filter"`
	Days       int    `json:"days,omitempty" jsonschema:"description=Optional number of recent days to inspect"`
}

type customerRecommendationInput struct {
	CustomerID string `json:"customer_id" jsonschema:"description=Customer identifier"`
	Days       int    `json:"days,omitempty" jsonschema:"description=Recent day window for usage and invoice signals"`
}

type organizationMembersInput struct {
	OrgID string `json:"org_id" jsonschema:"description=Organization identifier"`
}

type organizationIDInput struct {
	ID    string `json:"id,omitempty" jsonschema:"description=Organization identifier"`
	OrgID string `json:"org_id,omitempty" jsonschema:"description=Organization identifier"`
}

func (t *AssistantToolset) All() []genkitai.ToolRef {
	if t == nil || len(t.refs) == 0 {
		return nil
	}
	out := make([]genkitai.ToolRef, len(t.refs))
	copy(out, t.refs)
	return out
}

func normalizedPageSize(v int32) int32 {
	switch {
	case v <= 0:
		return defaultToolPageSize
	case v > maxToolPageSize:
		return maxToolPageSize
	default:
		return v
	}
}

func recentWindow(days int) (*time.Time, *time.Time) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -days)
	return &from, &now
}

func (i organizationIDInput) identifier() string {
	if v := strings.TrimSpace(i.OrgID); v != "" {
		return v
	}
	return strings.TrimSpace(i.ID)
}
