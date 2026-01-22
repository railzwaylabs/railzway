package domain_test

import (
	"os"
	"testing"

	"github.com/smallbiznis/railzway/internal/quota/domain"
	"github.com/stretchr/testify/assert"
)

func TestLoadFromEnv(t *testing.T) {
	// 1. Default fallback
	os.Clearenv()
	cfg := domain.LoadFromEnv()
	assert.True(t, cfg.Enabled)
	assert.Equal(t, 100, cfg.OrgCustomer)
	assert.Equal(t, 10, cfg.OrgSubscription)
	assert.Equal(t, 100000, cfg.OrgUsageMonthly)

	// 2. Custom Env
	os.Setenv("QUOTA_ENABLED", "false")
	os.Setenv("QUOTA_ORG_CUSTOMER", "50")
	os.Setenv("QUOTA_ORG_SUBSCRIPTION", "5")
	os.Setenv("QUOTA_ORG_USAGE_MONTHLY", "500")

	cfg = domain.LoadFromEnv()
	assert.False(t, cfg.Enabled)
	assert.Equal(t, 50, cfg.OrgCustomer)
	assert.Equal(t, 5, cfg.OrgSubscription)
	assert.Equal(t, 500, cfg.OrgUsageMonthly)

	// 3. Cleanup
	os.Clearenv()
}
