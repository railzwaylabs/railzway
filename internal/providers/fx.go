package providers

import (
	"github.com/smallbiznis/railzway/internal/payment"
	"github.com/smallbiznis/railzway/internal/providers/email"
	"github.com/smallbiznis/railzway/internal/providers/pdf"
	"go.uber.org/fx"
)

var Module = fx.Module("providers",
	email.Module,
	payment.Module,
	pdf.Module,
)
