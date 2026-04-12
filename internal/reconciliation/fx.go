package reconciliation

import (
	"github.com/railzwaylabs/railzway/internal/reconciliation/scheduler"
	"go.uber.org/fx"
)

// Module provides reconciliation scheduler wiring.
var Module = fx.Module("reconciliation",
	fx.Invoke(scheduler.StartReconciliationScheduler),
)
