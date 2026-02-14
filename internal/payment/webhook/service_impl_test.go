package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockPaymentService is a mock implementation of payment service
type MockPaymentService struct {
	mock.Mock
}

func (m *MockPaymentService) ProcessEvent(ctx context.Context, event *domain.PaymentEvent, rawPayload []byte) error {
	args := m.Called(ctx, event, rawPayload)
	return args.Error(0)
}

func TestNewService(t *testing.T) {
	// Test that Vault is initialized with a secret
	params := Params{
		Log: zap.NewNop(),
		Cfg: config.Config{
			PaymentProviderConfigSecret: "12345678901234567890123456789012", // 32 bytes
		}, 
	}
	// We can't access private fields easily, but we can checking if NewService panics or valid return
	svc := NewService(params)
	assert.NotNil(t, svc)
}


func TestMaskPayload(t *testing.T) {
	raw := `{"card": "4242", "user": {"billing_details": "secret"}, "other": "ok"}`
	masked := maskPayload([]byte(raw))

	var output map[string]interface{}
	json.Unmarshal(masked, &output)

	assert.Equal(t, "***", output["card"])
	assert.Equal(t, "ok", output["other"])
	
	user, _ := output["user"].(map[string]interface{})
	assert.Equal(t, "***", user["billing_details"])
}
