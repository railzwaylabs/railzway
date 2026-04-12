package domain

import "context"

type Repository interface {
	ListCountries(ctx context.Context) ([]Country, error)
	ListTimezonesByCountry(ctx context.Context, countryCode string) ([]Timezone, error)
	ListCurrencies(ctx context.Context) ([]Currency, error)
}
