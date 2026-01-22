package domain

import (
	"context"
	"errors"
	"time"

	"github.com/smallbiznis/railzway/pkg/db/pagination"
)

type ListCustomerRequest struct {
	PageToken   string
	PageSize    int32
	Name        string
	Email       string
	Currency    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type ListCustomerFilter struct {
	Name        string
	Email       string
	Currency    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type ListCustomerResponse struct {
	pagination.PageInfo
	Customers []Customer `json:"customers"`
}

type CreateCustomerRequest struct {
	Name  string
	Email string
}

type GetCustomerRequest struct {
	ID string
}

type Service interface {
	Create(context.Context, CreateCustomerRequest) (Customer, error)
	List(context.Context, ListCustomerRequest) (ListCustomerResponse, error)
	GetByID(context.Context, GetCustomerRequest) (Customer, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidName         = errors.New("invalid_name")
	ErrInvalidEmail        = errors.New("invalid_email")
	ErrInvalidID           = errors.New("invalid_id")
	ErrNotFound            = errors.New("not_found")
)
