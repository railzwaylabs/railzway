package service

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetDefaultPaymentMethod_NotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&paymentdomain.PaymentMethod{}))

	svc := &PaymentMethodServiceImpl{db: db}
	_, err = svc.GetDefaultPaymentMethod(context.Background(), 123)
	require.ErrorIs(t, err, paymentdomain.ErrPaymentMethodNotFound)
}
