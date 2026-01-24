package service

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func setupConfigTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	err = db.AutoMigrate(&domain.PaymentMethodConfig{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	return db
}

func TestGetAvailablePaymentMethods(t *testing.T) {
	db := setupConfigTestDB(t)
	node, _ := snowflake.NewNode(1)
	svc := NewPaymentMethodConfigService(db)

	orgID := node.Generate()

	// Seed Configs
	configs := []domain.PaymentMethodConfig{
		{
			ID:                node.Generate(),
			OrgID:             orgID,
			MethodType:        "card",
			MethodName:        "card_global",
			AvailabilityRules: datatypes.JSON([]byte(`{"countries": ["*"], "currencies": ["USD", "EUR"]}`)),
			Provider:          "stripe",
			DisplayName:       "Global Card",
			Priority:          100,
			IsActive:          true,
		},
		{
			ID:                node.Generate(),
			OrgID:             orgID,
			MethodType:        "card",
			MethodName:        "card_idr",
			AvailabilityRules: datatypes.JSON([]byte(`{"countries": ["ID"], "currencies": ["IDR"]}`)),
			Provider:          "xendit",
			DisplayName:       "IDR Card",
			Priority:          90,
			IsActive:          true,
		},
		{
			ID:                node.Generate(),
			OrgID:             orgID,
			MethodType:        "ewallet",
			MethodName:        "gopay",
			AvailabilityRules: datatypes.JSON([]byte(`{"countries": ["ID"], "currencies": ["IDR"]}`)),
			Provider:          "xendit",
			DisplayName:       "GoPay",
			Priority:          80, // Lower priority than card_idr
			IsActive:          true,
		},
		{
			ID:                node.Generate(),
			OrgID:             orgID,
			MethodType:        "ewallet",
			MethodName:        "inactive_method",
			AvailabilityRules: datatypes.JSON([]byte(`{"countries": ["ID"], "currencies": ["IDR"]}`)),
			Provider:          "xendit",
			DisplayName:       "Inactive",
			Priority:          1000,
			IsActive:          false, // Inactive
		},
		{
			ID:                node.Generate(),
			OrgID:             orgID,
			MethodType:        "card",
			MethodName:        "card_ph",
			AvailabilityRules: datatypes.JSON([]byte(`{"countries": ["PH"], "currencies": ["PHP"]}`)),
			Provider:          "xendit",
			DisplayName:       "PH Card",
			Priority:          95,
			IsActive:          true,
		},
	}

	for _, c := range configs {
		c.CreatedAt = time.Now()
		c.UpdatedAt = time.Now()
		shouldBeActive := c.IsActive

		// Force IsActive to be respected (even if false)
		// Also ensure AvailabilityRules is stored
		if err := db.Select("ID", "OrgID", "MethodType", "MethodName", "AvailabilityRules", "Provider", "DisplayName", "Priority", "IsActive", "CreatedAt", "UpdatedAt").Create(&c).Error; err != nil {
			t.Fatalf("failed to seed config: %v", err)
		}
		// Workaround for GORM default value on zero-value bool
		if !shouldBeActive {
			t.Logf("Updating Inactive Config ID: %v", c.ID)
			if err := db.Exec("UPDATE payment_method_configs SET is_active = 0 WHERE id = ?", c.ID).Error; err != nil {
				t.Fatalf("failed to set is_active false: %v", err)
			}
			var check domain.PaymentMethodConfig
			if err := db.First(&check, c.ID).Error; err != nil {
				t.Fatalf("failed to verification fetch: %v", err)
			}
			t.Logf("After Update Config %s: IsActive=%v (Raw ID: %v)", check.MethodName, check.IsActive, check.ID)
		}
	}

	tests := []struct {
		name     string
		country  string
		currency string
		want     []string // MethodNames
	}{
		{
			name: "ID/IDR should match global (no, restricted by currency) and ID methods",
			// card_global allows URL/EUR. card_idr allows ID/IDR. gopay allows ID/IDR.
			country:  "ID",
			currency: "IDR",
			want:     []string{"card_idr", "gopay"},
		},
		{
			name:     "Should filter by priority",
			country:  "ID",
			currency: "IDR",
			want:     []string{"card_idr", "gopay"}, // card_idr(90) > gopay(80)
		},
		{
			name:     "US/USD should match global",
			country:  "US",
			currency: "USD",
			want:     []string{"card_global"},
		},
		{
			name:     "PH/PHP should match PH method",
			country:  "PH",
			currency: "PHP",
			want:     []string{"card_ph"},
		},
		{
			name:     "Inactive method should be ignored",
			country:  "ID",
			currency: "IDR",
			// Inactive has priority 1000 but inactive
			want: []string{"card_idr", "gopay"},
		},
		{
			name:     "No match",
			country:  "SG",
			currency: "SGD",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetAvailablePaymentMethods(context.Background(), orgID, tt.country, tt.currency)
			if err != nil {
				t.Errorf("GetAvailablePaymentMethods() error = %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("GetAvailablePaymentMethods() got %d items, want %d", len(got), len(tt.want))
				for _, g := range got {
					t.Logf("Got: %s", g.MethodName)
				}
				return
			}

			for i, method := range got {
				if method.MethodName != tt.want[i] {
					t.Errorf("GetAvailablePaymentMethods() item[%d] = %v, want %v", i, method.MethodName, tt.want[i])
				}
			}
		})
	}
}
