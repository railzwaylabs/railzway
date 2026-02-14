package vault

import (
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/fx"
)

var Module = fx.Module("security.vault",
	fx.Provide(
		func(cfg config.Config) (Provider, error) {
			vaultCfg := Config{
				Provider:   cfg.Vault.Provider,
				AESKey:     cfg.Vault.AESKey,
				VaultAddr:  cfg.Vault.Addr,
				VaultToken: cfg.Vault.Token,
			}
			return NewFactory(vaultCfg)
		},
	),
)
