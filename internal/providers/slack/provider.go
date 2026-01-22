package slack

import "context"

type Provider interface {
	PostMessage(ctx context.Context, channelID string, message string) error
}

type NoOpProvider struct{}

func (p *NoOpProvider) PostMessage(ctx context.Context, channelID string, message string) error {
	return nil
}
