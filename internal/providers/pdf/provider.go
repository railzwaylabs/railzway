package pdf

import (
	"context"
	"io"
)

// TODO: Import the actual Invoice domain model when bridging this.
// For now using interface{} for flexibility in this initial step.
type Provider interface {
	GenerateInvoice(ctx context.Context, data interface{}) (io.Reader, error)
	GenerateReceipt(ctx context.Context, data interface{}) (io.Reader, error)
}

type NoOpProvider struct{}

func (p *NoOpProvider) GenerateInvoice(ctx context.Context, data interface{}) (io.Reader, error) {
	return nil, nil
}

func (p *NoOpProvider) GenerateReceipt(ctx context.Context, data interface{}) (io.Reader, error) {
	return nil, nil
}
