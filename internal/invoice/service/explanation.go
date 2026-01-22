package service

import (
	"context"

	"github.com/bwmarrin/snowflake"
	invoicedomain "github.com/smallbiznis/railzway/internal/invoice/domain"
	"gorm.io/gorm"
)

type ExplanationService struct {
	db *gorm.DB
}

func NewExplanationService(db *gorm.DB) *ExplanationService {
	return &ExplanationService{db: db}
}

type InvoiceExplanation struct {
	InvoiceID string                 `json:"invoice_id"`
	Total     int64                  `json:"total"`
	Breakdown []LineItemExplanation  `json:"breakdown"`
}

type LineItemExplanation struct {
	LineItemID  string                 `json:"line_item_id"`
	Description string                 `json:"description"`
	Amount      int64                  `json:"amount"`
	Explanation *DetailedExplanation   `json:"explanation,omitempty"`
}

type DetailedExplanation struct {
	Quantity      float64 `json:"quantity"`
	UnitPrice     int64   `json:"unit_price"`
	PeriodStart   string  `json:"period_start"`
	PeriodEnd     string  `json:"period_end"`
	SourceEvents  int64   `json:"source_events"`
	MeterID       *string `json:"meter_id,omitempty"`
	MeterName     *string `json:"meter_name,omitempty"`
	FeatureCode   *string `json:"feature_code,omitempty"`
	RatingResultID *string `json:"rating_result_id,omitempty"`
}

func (s *ExplanationService) ExplainInvoice(ctx context.Context, invoiceID snowflake.ID) (*InvoiceExplanation, error) {
	// Fetch invoice
	var invoice invoicedomain.Invoice
	if err := s.db.WithContext(ctx).Where("id = ?", invoiceID).First(&invoice).Error; err != nil {
		return nil, err
	}

	// Fetch invoice items with rating results
	var items []struct {
		invoicedomain.InvoiceItem
		RatingQuantity  *float64 `gorm:"column:rating_quantity"`
		RatingUnitPrice *int64   `gorm:"column:rating_unit_price"`
		PeriodStart     *string  `gorm:"column:period_start"`
		PeriodEnd       *string  `gorm:"column:period_end"`
		MeterID         *string  `gorm:"column:meter_id"`
		MeterName       *string  `gorm:"column:meter_name"`
		FeatureCode     *string  `gorm:"column:feature_code"`
	}

	if err := s.db.WithContext(ctx).Raw(`
		SELECT 
			ii.*,
			rr.quantity AS rating_quantity,
			rr.unit_price AS rating_unit_price,
			rr.period_start,
			rr.period_end,
			rr.meter_id,
			m.name AS meter_name,
			rr.feature_code
		FROM invoice_items ii
		LEFT JOIN rating_results rr ON ii.rating_result_id = rr.id
		LEFT JOIN meters m ON rr.meter_id = m.id
		WHERE ii.invoice_id = ?
		ORDER BY ii.created_at ASC
	`, invoiceID).Scan(&items).Error; err != nil {
		return nil, err
	}

	// Build explanation
	breakdown := make([]LineItemExplanation, 0, len(items))
	for _, item := range items {
		lineItem := LineItemExplanation{
			LineItemID:  item.ID.String(),
			Description: item.Description,
			Amount:      item.Amount,
		}

		// Add detailed explanation if this is a usage-based line item
		if item.RatingResultID != nil && item.RatingQuantity != nil {
			// Count source events
			var eventCount int64
			if item.MeterID != nil {
				s.db.WithContext(ctx).Raw(`
					SELECT COUNT(*)
					FROM usage_events
					WHERE meter_id = ?
					  AND subscription_id = ?
					  AND recorded_at >= ?
					  AND recorded_at < ?
				`, item.MeterID, invoice.SubscriptionID, item.PeriodStart, item.PeriodEnd).Scan(&eventCount)
			}

			ratingResultIDStr := item.RatingResultID.String()
			lineItem.Explanation = &DetailedExplanation{
				Quantity:       *item.RatingQuantity,
				UnitPrice:      *item.RatingUnitPrice,
				PeriodStart:    stringOrEmpty(item.PeriodStart),
				PeriodEnd:      stringOrEmpty(item.PeriodEnd),
				SourceEvents:   eventCount,
				MeterID:        item.MeterID,
				MeterName:      item.MeterName,
				FeatureCode:    item.FeatureCode,
				RatingResultID: &ratingResultIDStr,
			}
		}

		breakdown = append(breakdown, lineItem)
	}

	return &InvoiceExplanation{
		InvoiceID: invoiceID.String(),
		Total:     invoice.TotalAmount,
		Breakdown: breakdown,
	}, nil
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
