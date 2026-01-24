package providers

import (
	"github.com/railzwaylabs/railzway/internal/payment"
	"github.com/railzwaylabs/railzway/internal/providers/email"
	"github.com/railzwaylabs/railzway/internal/providers/pdf"
	"go.uber.org/fx"
)

var Module = fx.Module("providers",
	email.Module,
	payment.Module,
	pdf.Module,
)
