package http

import (
	"errors"
	"net/http"
	"testing"
)

func TestBuildAIPromptBlocks_UnwrapsJSONFence(t *testing.T) {
	blocks := buildAIPromptBlocks("```json\n{\"blocks\":[{\"type\":\"heading\",\"text\":\"Hello there!\"},{\"type\":\"quote\",\"text\":\"I'm online.\"}]}\n```")

	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(blocks))
	}
	if blocks[0].Type != "heading" || blocks[0].Text != "Hello there!" {
		t.Fatalf("blocks[0] = %#v, want heading Hello there!", blocks[0])
	}
	if blocks[1].Type != "quote" || blocks[1].Text != "I'm online." {
		t.Fatalf("blocks[1] = %#v, want quote I'm online.", blocks[1])
	}
}

func TestClassifyAIPromptErrorQuotaExceeded(t *testing.T) {
	err := errors.New("Error 429, Message: You exceeded your current quota. Please retry in 58.955253213s., Status: RESOURCE_EXHAUSTED")

	got := classifyAIPromptError(err)

	if got.Status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", got.Status, http.StatusTooManyRequests)
	}
	if got.Code != "ai_quota_exceeded" {
		t.Fatalf("code = %q, want %q", got.Code, "ai_quota_exceeded")
	}
	if got.RetryAfterSeconds == nil || *got.RetryAfterSeconds != 59 {
		t.Fatalf("retry_after_seconds = %v, want 59", got.RetryAfterSeconds)
	}
}

func TestClassifyAIPromptErrorTimeout(t *testing.T) {
	err := errors.New("request failed: deadline exceeded")

	got := classifyAIPromptError(err)

	if got.Status != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", got.Status, http.StatusGatewayTimeout)
	}
	if got.Code != "ai_timeout" {
		t.Fatalf("code = %q, want %q", got.Code, "ai_timeout")
	}
}

func TestExtractRetryAfterSeconds(t *testing.T) {
	got := extractRetryAfterSeconds("Details: retryDelay:58s")
	if got == nil || *got != 58 {
		t.Fatalf("extractRetryAfterSeconds() = %v, want 58", got)
	}
}
