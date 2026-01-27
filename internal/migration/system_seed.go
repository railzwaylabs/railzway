package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type currencySeed struct {
	Code      string
	Name      string
	Symbol    *string
	MinorUnit int
	IsActive  bool
}

type namedSeed struct {
	Code string
	Name string
}

func seedSystemImmutableData(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("system seed requires database handle")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin system seed transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := seedCurrencies(ctx, tx); err != nil {
		return err
	}
	if err := seedBillingCycleTypes(ctx, tx); err != nil {
		return err
	}
	if err := seedLedgerAccountTypes(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit system seed transaction: %w", err)
	}
	return nil
}

func seedCurrencies(ctx context.Context, tx *sql.Tx) error {
	usd := "$"
	eur := "EUR"
	gbp := "GBP"
	jpy := "JPY"
	idr := "IDR"
	php := "PHP"
	sgd := "SGD"
	aud := "AUD"
	cad := "CAD"

	seeds := []currencySeed{
		{Code: "USD", Name: "US Dollar", Symbol: &usd, MinorUnit: 2, IsActive: true},
		{Code: "EUR", Name: "Euro", Symbol: &eur, MinorUnit: 2, IsActive: true},
		{Code: "GBP", Name: "British Pound", Symbol: &gbp, MinorUnit: 2, IsActive: true},
		{Code: "JPY", Name: "Japanese Yen", Symbol: &jpy, MinorUnit: 0, IsActive: true},
		{Code: "IDR", Name: "Indonesian Rupiah", Symbol: &idr, MinorUnit: 2, IsActive: true},
		{Code: "PHP", Name: "Philippine Peso", Symbol: &php, MinorUnit: 2, IsActive: true},
		{Code: "SGD", Name: "Singapore Dollar", Symbol: &sgd, MinorUnit: 2, IsActive: true},
		{Code: "AUD", Name: "Australian Dollar", Symbol: &aud, MinorUnit: 2, IsActive: true},
		{Code: "CAD", Name: "Canadian Dollar", Symbol: &cad, MinorUnit: 2, IsActive: true},
	}

	const stmt = `
		INSERT INTO currencies (code, name, symbol, minor_unit, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name,
		    symbol = EXCLUDED.symbol,
		    minor_unit = EXCLUDED.minor_unit,
		    is_active = EXCLUDED.is_active
	`

	for _, seed := range seeds {
		if _, err := tx.ExecContext(ctx, stmt, seed.Code, seed.Name, seed.Symbol, seed.MinorUnit, seed.IsActive); err != nil {
			return fmt.Errorf("seed currency %s: %w", seed.Code, err)
		}
	}
	return nil
}

func seedBillingCycleTypes(ctx context.Context, tx *sql.Tx) error {
	seeds := []namedSeed{
		{Code: "monthly", Name: "Monthly"},
		{Code: "weekly", Name: "Weekly"},
	}

	const stmt = `
		INSERT INTO billing_cycle_types (code, name)
		VALUES ($1, $2)
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name
	`

	for _, seed := range seeds {
		if _, err := tx.ExecContext(ctx, stmt, seed.Code, seed.Name); err != nil {
			return fmt.Errorf("seed billing cycle type %s: %w", seed.Code, err)
		}
	}
	return nil
}

func seedLedgerAccountTypes(ctx context.Context, tx *sql.Tx) error {
	seeds := []namedSeed{
		{Code: "asset", Name: "Asset"},
		{Code: "liability", Name: "Liability"},
		{Code: "income", Name: "Income"},
		{Code: "expense", Name: "Expense"},
		{Code: "equity", Name: "Equity"},
	}

	const stmt = `
		INSERT INTO ledger_account_types (code, name)
		VALUES ($1, $2)
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name
	`

	for _, seed := range seeds {
		if _, err := tx.ExecContext(ctx, stmt, seed.Code, seed.Name); err != nil {
			return fmt.Errorf("seed ledger account type %s: %w", seed.Code, err)
		}
	}
	return nil
}
