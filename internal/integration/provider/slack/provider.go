package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/railzwaylabs/railzway/internal/integration/domain"
)

type Provider struct {
	client *http.Client
}

func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *Provider) Send(ctx context.Context, input domain.NotificationInput) error {
	webhookURL, ok := input.Data["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("missing_webhook_url")
	}

	// Simple Slack message format
	// In the future, we can use sophisticated blocks
	msg := map[string]any{
		"text": fmt.Sprintf("*Notification: %s*\nChannel: %s\nData: %v", input.TemplateID, input.ChannelID, input.Data),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack_api_error: status=%d", resp.StatusCode)
	}

	return nil
}
