package domain

import "errors"

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidID           = errors.New("invalid_id")
	ErrNotFound            = errors.New("not_found")
	ErrInvalidTaxCode      = errors.New("invalid_tax_code")
	ErrInvalidTaxMode      = errors.New("invalid_tax_mode")
	ErrInvalidTaxRate      = errors.New("invalid_tax_rate")
)
