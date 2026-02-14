package service

import (
	"testing"

	pricetierdomain "github.com/railzwaylabs/railzway/internal/pricetier/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTieredVolumeAmount(t *testing.T) {
	tiers := []pricetierdomain.PriceTier{
		{
			StartQuantity:   1,
			EndQuantity:     floatPtr(100),
			UnitAmountCents: int64Ptr(10),
		},
		{
			StartQuantity:   101,
			EndQuantity:     nil,
			UnitAmountCents: int64Ptr(8),
			FlatAmountCents: int64Ptr(100),
		},
	}

	amount, unitPrice, err := calculateTieredVolumeAmount(150, tiers)
	require.NoError(t, err)
	assert.Equal(t, int64(1300), amount) // 150*8 + 100 flat
	assert.Equal(t, int64(9), unitPrice) // round(1300/150) = 9
}

func TestTieredGraduatedAmount(t *testing.T) {
	tiers := []pricetierdomain.PriceTier{
		{
			StartQuantity:   1,
			EndQuantity:     floatPtr(100),
			UnitAmountCents: int64Ptr(10),
		},
		{
			StartQuantity:   101,
			EndQuantity:     floatPtr(200),
			UnitAmountCents: int64Ptr(8),
		},
		{
			StartQuantity:   201,
			EndQuantity:     nil,
			UnitAmountCents: int64Ptr(6),
		},
	}

	amount, unitPrice, err := calculateTieredGraduatedAmount(250, tiers)
	require.NoError(t, err)
	assert.Equal(t, int64(2100), amount) // 100*10 + 100*8 + 50*6
	assert.Equal(t, int64(8), unitPrice) // round(2100/250) = 8
}

func floatPtr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
