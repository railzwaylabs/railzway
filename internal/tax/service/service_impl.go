package service

import (
	"context"
	"math"

	"github.com/bwmarrin/snowflake"
	taxdomain "github.com/smallbiznis/railzway/internal/tax/domain"
	"go.uber.org/fx"
)

type resolverParam struct {
	fx.In

	Repository taxdomain.Repository
}

type resolver struct {
	repo taxdomain.Repository
}

func NewResolver(p resolverParam) taxdomain.TaxResolver {
	return &resolver{repo: p.Repository}
}

func (r *resolver) ResolveForInvoice(ctx context.Context, orgID, customerID snowflake.ID) (*taxdomain.TaxDefinition, error) {
	def, err := r.repo.GetActiveTaxDefinition(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if def == nil || def.Rate == nil || *def.Rate <= 0 {
		return nil, nil
	}
	// customerID reserved for future customer-specific tax logic.
	return def, nil
}

// ComputeTaxExclusive calculates tax added on top of subtotal.
// Rounding happens only here to keep stored values integer-safe.
func ComputeTaxExclusive(subtotal int64, rate *float64) int64 {
	return computeTaxExclusive(subtotal, rate)
}

// ComputeTaxInclusive calculates the tax portion included in subtotal.
// Rounding happens only here to keep stored values integer-safe.
func ComputeTaxInclusive(subtotal int64, rate *float64) int64 {
	return computeTaxInclusive(subtotal, rate)
}

func computeTaxExclusive(subtotal int64, rate *float64) int64 {
	if subtotal <= 0 || rate == nil || *rate <= 0 {
		return 0
	}

	tax := float64(subtotal) * (*rate)
	result := int64(math.Round(tax))
	if result < 0 {
		return 0
	}
	return result
}

func computeTaxInclusive(subtotal int64, rate *float64) int64 {
	if subtotal <= 0 || rate == nil || *rate <= 0 {
		return 0
	}

	tax := float64(subtotal) * (*rate / (1 + *rate))
	result := int64(math.Round(tax))
	if result < 0 {
		return 0
	}
	return result
}
