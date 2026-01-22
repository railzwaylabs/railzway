package reference

import (
	"context"
	"database/sql"

	"github.com/smallbiznis/railzway/internal/reference/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

func (r *repository) ListCountries(ctx context.Context) ([]domain.Country, error) {
	type row struct {
		Code string `gorm:"column:code"`
		Name string `gorm:"column:name"`
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT code, name FROM countries ORDER BY name`).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	countries := make([]domain.Country, 0, len(rows))
	for _, item := range rows {
		countries = append(countries, domain.Country{
			Code: item.Code,
			Name: item.Name,
		})
	}

	return countries, nil
}

func (r *repository) ListTimezonesByCountry(ctx context.Context, countryCode string) ([]domain.Timezone, error) {
	var timezones []domain.Timezone
	err := r.db.WithContext(ctx).
		Raw(`SELECT DISTINCT timezone_name AS name FROM country_timezones WHERE country_code = ? ORDER BY timezone_name`, countryCode).
		Scan(&timezones).Error
	if err != nil {
		return nil, err
	}

	return timezones, nil
}

func (r *repository) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	type row struct {
		Code      string         `gorm:"column:code"`
		Name      string         `gorm:"column:name"`
		Symbol    sql.NullString `gorm:"column:symbol"`
		MinorUnit int16          `gorm:"column:minor_unit"`
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT code, name, symbol, minor_unit FROM currencies WHERE is_active = true ORDER BY code`).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	currencies := make([]domain.Currency, 0, len(rows))
	for _, item := range rows {
		var symbol *string
		if item.Symbol.Valid {
			value := item.Symbol.String
			symbol = &value
		}
		currencies = append(currencies, domain.Currency{
			Code:      item.Code,
			Name:      item.Name,
			Symbol:    symbol,
			MinorUnit: item.MinorUnit,
		})
	}

	return currencies, nil
}
