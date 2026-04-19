package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	aiassistant "github.com/railzwaylabs/railzway/internal/ai/assistant"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type aiPromptTokenRequest struct {
	ID             string                 `json:"id"`
	Kind           string                 `json:"kind"`
	Type           string                 `json:"type"`
	Key            string                 `json:"key"`
	Label          string                 `json:"label"`
	SecondaryLabel string                 `json:"secondary_label"`
	ResourceID     string                 `json:"resource_id"`
	Value          string                 `json:"value"`
	From           string                 `json:"from"`
	To             string                 `json:"to"`
	Timezone       string                 `json:"timezone"`
	Preset         string                 `json:"preset"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type createAIPromptRequest struct {
	Prompt         string                 `json:"prompt"`
	Tokens         []aiPromptTokenRequest `json:"tokens"`
	ConversationID string                 `json:"conversation_id"`
	ThreadID       string                 `json:"thread_id"`
}

type aiPromptMessageBlockResponse struct {
	Type  string      `json:"type"`
	Tone  string      `json:"tone,omitempty"`
	Title string      `json:"title,omitempty"`
	Text  string      `json:"text,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type structuredAIResponse struct {
	Blocks []aiPromptMessageBlockResponse `json:"blocks"`
}

type aiPromptMessageResponse struct {
	ID        string                         `json:"id"`
	Role      string                         `json:"role"`
	Prompt    string                         `json:"prompt,omitempty"`
	Blocks    []aiPromptMessageBlockResponse `json:"blocks,omitempty"`
	Usage     *aiassistant.Usage             `json:"usage,omitempty"`
	LatencyMs float64                        `json:"latency_ms,omitempty"`
	CreatedAt time.Time                      `json:"created_at"`
}

type createAIPromptResponse struct {
	ConversationID string                  `json:"conversation_id,omitempty"`
	ThreadID       string                  `json:"thread_id,omitempty"`
	Message        aiPromptMessageResponse `json:"message"`
}

type createAIThreadRequest struct {
	Title string `json:"title"`
}

type aiThreadResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type aiThreadListResponse struct {
	Threads []aiThreadResponse `json:"threads"`
}

type aiThreadDetailResponse struct {
	Thread   aiThreadResponse          `json:"thread"`
	Messages []aiPromptMessageResponse `json:"messages"`
}

type aiErrorResponse struct {
	Error             string `json:"error"`
	Message           string `json:"message"`
	RetryAfterSeconds *int   `json:"retry_after_seconds,omitempty"`
}

type classifiedAIError struct {
	Status            int
	Code              string
	Message           string
	RetryAfterSeconds *int
}

func (h *Handler) ListAIThreads(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.aiThreadStore == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ai_thread_store_not_configured"})
		return
	}

	limit := int(parseInt32(c.Query("page_size")))
	threads, err := h.aiThreadStore.ListThreads(c.Request.Context(), orgID, userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	resp := make([]aiThreadResponse, 0, len(threads))
	for _, thread := range threads {
		resp = append(resp, toAIThreadResponse(thread))
	}
	c.JSON(http.StatusOK, aiThreadListResponse{Threads: resp})
}

func (h *Handler) CreateAIThread(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.aiThreadStore == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ai_thread_store_not_configured"})
		return
	}

	var payload createAIThreadRequest
	if !bindOptionalJSONOrAbort(c, &payload) {
		return
	}

	thread, err := h.aiThreadStore.CreateThread(c.Request.Context(), aiassistant.CreateThreadInput{
		OrgID:  orgID,
		UserID: userID,
		Title:  strings.TrimSpace(payload.Title),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"thread": toAIThreadResponse(thread)})
}

func (h *Handler) GetAIThread(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.aiThreadStore == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ai_thread_store_not_configured"})
		return
	}

	threadID, err := uuid.Parse(strings.TrimSpace(c.Param("thread_id")))
	if err != nil || threadID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_thread_id"})
		return
	}

	detail, err := h.aiThreadStore.GetThreadDetail(c.Request.Context(), orgID, userID, threadID)
	if err != nil {
		if err == aiassistant.ErrThreadNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "thread_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	messages := make([]aiPromptMessageResponse, 0, len(detail.Messages))
	for _, message := range detail.Messages {
		messages = append(messages, toAIPromptMessageResponse(message))
	}

	c.JSON(http.StatusOK, aiThreadDetailResponse{
		Thread:   toAIThreadResponse(detail.Thread),
		Messages: messages,
	})
}

func (h *Handler) DeleteAIThread(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.aiThreadStore == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ai_thread_store_not_configured"})
		return
	}

	threadID, err := uuid.Parse(strings.TrimSpace(c.Param("thread_id")))
	if err != nil || threadID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_thread_id"})
		return
	}
	if err := h.aiThreadStore.DeleteThread(c.Request.Context(), orgID, userID, threadID); err != nil {
		if err == aiassistant.ErrThreadNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "thread_not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) CreateAIPrompt(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	if h.aiAssistant == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ai_not_configured"})
		return
	}
	if h.aiThreadStore == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ai_thread_store_not_configured"})
		return
	}
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var payload createAIPromptRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	if err := h.validateAIPromptTokens(payload.Tokens); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_tokens", "message": err.Error()})
		return
	}

	payload.Prompt = strings.TrimSpace(payload.Prompt)
	if payload.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_prompt"})
		return
	}

	threadID, err := resolveAIThreadID(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_thread_id"})
		return
	}
	if threadID == uuid.Nil {
		thread, err := h.aiThreadStore.CreateThread(c.Request.Context(), aiassistant.CreateThreadInput{
			OrgID:  orgID,
			UserID: userID,
			Title:  summarizeAIPromptTitle(payload.Prompt),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		threadID = thread.ID
	} else {
		if _, err := h.aiThreadStore.GetThread(c.Request.Context(), orgID, userID, threadID); err != nil {
			if err == aiassistant.ErrThreadNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "thread_not_found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
	}

	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	
	// Fetch recent history for context
	history, _ := h.aiThreadStore.GetRecentMessages(ctx, orgID, threadID, 10)
	isFirstMessage := len(history) == 0

	userMessage, err := h.aiThreadStore.CreateMessage(ctx, aiassistant.CreateMessageInput{
		ThreadID: threadID,
		OrgID:    orgID,
		UserID:   &userID,
		Role:     "user",
		Prompt:   payload.Prompt,
		Tokens:   toAssistantTokens(payload.Tokens),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	contextText := buildAIPromptContext(orgID, payload.Tokens)
	output, modelResp, err := h.aiAssistant.ExecuteStructured(ctx, aiassistant.PromptInput{
		Context: contextText,
		Prompt:  payload.Prompt,
	}, genkitai.WithMessages(toGenkitMessages(history)...))
	if err != nil {
		aiErr := classifyAIPromptError(err)
		_, _ = h.aiThreadStore.CreateMessage(ctx, aiassistant.CreateMessageInput{
			ThreadID: threadID,
			OrgID:    orgID,
			Role:     "assistant",
			Prompt:   aiErr.Message,
			Blocks:   buildAIPromptErrorBlocks(aiErr.Message),
		})
		if aiErr.RetryAfterSeconds != nil && *aiErr.RetryAfterSeconds > 0 {
			c.Header("Retry-After", strconv.Itoa(*aiErr.RetryAfterSeconds))
		}
		
		h.recordAIPromptAudit(ctx, orgID, userID, threadID, payload.Prompt, nil, 0, err)

		c.JSON(aiErr.Status, aiErrorResponse{
			Error:             aiErr.Code,
			Message:           aiErr.Message,
			RetryAfterSeconds: aiErr.RetryAfterSeconds,
		})
		return
	}

	blocks := toStructuredPromptBlocks(output.Blocks)
	assistantMessage, err := h.aiThreadStore.CreateMessage(ctx, aiassistant.CreateMessageInput{
		ThreadID:  threadID,
		OrgID:     orgID,
		Role:      "assistant",
		Prompt:    buildAssistantPromptText(blocks),
		Blocks:    toAssistantBlocks(blocks),
		Usage:     toAssistantUsage(modelResp),
		LatencyMs: modelResp.LatencyMs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	response := createAIPromptResponse{
		ConversationID: threadID.String(),
		ThreadID:       threadID.String(),
		Message:        toAIPromptMessageResponse(assistantMessage),
	}

	h.recordAIPromptAudit(ctx, orgID, userID, threadID, payload.Prompt, assistantMessage.Usage, assistantMessage.LatencyMs, nil)

	if isFirstMessage && userMessage.Role == "user" {
		_ = h.aiThreadStore.UpdateThreadTitle(ctx, orgID, userID, threadID, summarizeAIPromptTitle(userMessage.Prompt))
	}
	c.JSON(http.StatusOK, response)
}

func toStructuredPromptBlocks(blocks []aiassistant.OutputBlock) []aiPromptMessageBlockResponse {
	out := make([]aiPromptMessageBlockResponse, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, aiPromptMessageBlockResponse{
			Type:  strings.TrimSpace(block.Type),
			Tone:  strings.TrimSpace(block.Tone),
			Title: strings.TrimSpace(block.Title),
			Text:  strings.TrimSpace(block.Text),
			Data:  block.Data,
		})
	}
	return normalizeAIPromptBlocks(out)
}

func buildAIPromptContext(orgID uuid.UUID, tokens []aiPromptTokenRequest) string {
	var b strings.Builder
	b.WriteString("Organization context:\n")
	b.WriteString("- org_id: ")
	b.WriteString(orgID.String())
	b.WriteString("\n")

	if len(tokens) == 0 {
		b.WriteString("- tokens: none\n")
		return b.String()
	}

	b.WriteString("Structured tokens:\n")
	for _, token := range tokens {
		key := strings.TrimSpace(token.Key)
		if key == "" {
			key = "@token"
		}

		switch strings.TrimSpace(token.Kind) {
		case "resource":
			parts := []string{fmt.Sprintf("- resource %s => type=%s", key, strings.TrimSpace(token.Type))}
			if rid := strings.TrimSpace(token.ResourceID); rid != "" {
				parts = append(parts, fmt.Sprintf("resource_id=%s", rid))
			}
			if lbl := strings.TrimSpace(token.Label); lbl != "" {
				parts = append(parts, fmt.Sprintf("label=%q", lbl))
			}
			if slbl := strings.TrimSpace(token.SecondaryLabel); slbl != "" {
				parts = append(parts, fmt.Sprintf("secondary_label=%q", slbl))
			}
			b.WriteString(strings.Join(parts, " ") + "\n")
		case "time":
			parts := []string{fmt.Sprintf("- time %s => type=%s", key, strings.TrimSpace(token.Type))}
			if val := strings.TrimSpace(token.Value); val != "" {
				parts = append(parts, fmt.Sprintf("value=%q", val))
			}
			if from := strings.TrimSpace(token.From); from != "" {
				parts = append(parts, fmt.Sprintf("from=%q", from))
			}
			if to := strings.TrimSpace(token.To); to != "" {
				parts = append(parts, fmt.Sprintf("to=%q", to))
			}
			if preset := strings.TrimSpace(token.Preset); preset != "" {
				parts = append(parts, fmt.Sprintf("preset=%q", preset))
			}
			if tz := strings.TrimSpace(token.Timezone); tz != "" {
				parts = append(parts, fmt.Sprintf("timezone=%q", tz))
			}
			b.WriteString(strings.Join(parts, " ") + "\n")
		default:
			b.WriteString(fmt.Sprintf("- token %s => type=%s label=%q\n",
				key,
				strings.TrimSpace(token.Type),
				strings.TrimSpace(token.Label),
			))
		}

		if len(token.Metadata) > 0 {
			encoded, err := json.Marshal(token.Metadata)
			if err == nil {
				b.WriteString("  metadata: ")
				b.Write(encoded)
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

func buildAIPromptBlocks(text string) []aiPromptMessageBlockResponse {
	normalizedText := unwrapJSONCodeFence(strings.TrimSpace(text))
	if normalizedText == "" {
		return []aiPromptMessageBlockResponse{{
			Type:  "text",
			Title: "Answer",
			Text:  "",
		}}
	}

	if strings.HasPrefix(normalizedText, "{") {
		var parsed structuredAIResponse
		if err := json.Unmarshal([]byte(normalizedText), &parsed); err == nil && len(parsed.Blocks) > 0 {
			return normalizeAIPromptBlocks(parsed.Blocks)
		}

		var fallback interface{}
		if err := json.Unmarshal([]byte(normalizedText), &fallback); err == nil {
			return []aiPromptMessageBlockResponse{{
				Type:  "json",
				Title: "Response",
				Data:  fallback,
			}}
		}
	}

	return []aiPromptMessageBlockResponse{{
		Type:  "text",
		Title: "Answer",
		Text:  normalizedText,
	}}
}

func unwrapJSONCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 {
		return trimmed
	}

	first := strings.TrimSpace(lines[0])
	last := strings.TrimSpace(lines[len(lines)-1])
	if last != "```" {
		return trimmed
	}

	if first != "```" && !strings.HasPrefix(strings.ToLower(first), "```json") {
		return trimmed
	}

	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
}

func buildAIPromptErrorBlocks(text string) []aiassistant.MessageBlock {
	return []aiassistant.MessageBlock{
		{Type: "heading", Text: "AI request failed"},
		{Type: "quote", Text: strings.TrimSpace(text)},
	}
}

func normalizeAIPromptBlocks(blocks []aiPromptMessageBlockResponse) []aiPromptMessageBlockResponse {
	normalized := make([]aiPromptMessageBlockResponse, 0, len(blocks))
	for _, block := range blocks {
		blockType := strings.ToLower(strings.TrimSpace(block.Type))
		switch blockType {
		case "heading", "quote", "text", "markdown", "json", "cards", "chart", "list", "alert", "table", "timeline", "badge", "steps":
			block.Type = blockType
			block.Tone = strings.ToLower(strings.TrimSpace(block.Tone))
			block.Title = strings.TrimSpace(block.Title)
			block.Text = strings.TrimSpace(block.Text)
			normalized = append(normalized, block)
		default:
			normalized = append(normalized, aiPromptMessageBlockResponse{
				Type:  "text",
				Title: block.Title,
				Text:  strings.TrimSpace(block.Text),
				Data:  block.Data,
			})
		}
	}
	if len(normalized) == 0 {
		return []aiPromptMessageBlockResponse{{
			Type:  "text",
			Title: "Answer",
			Text:  "",
		}}
	}
	return normalized
}

func classifyAIPromptError(err error) classifiedAIError {
	raw := strings.TrimSpace(strings.ToLower(err.Error()))
	retryAfter := extractRetryAfterSeconds(err.Error())

	switch {
	case strings.Contains(raw, "quota exceeded") || strings.Contains(raw, "resource_exhausted") || strings.Contains(raw, "exceeded your current quota"):
		message := "AI quota is exhausted for the configured provider."
		if retryAfter != nil && *retryAfter > 0 {
			message = fmt.Sprintf("AI quota is exhausted for the configured provider. Retry in about %d seconds.", *retryAfter)
		}
		return classifiedAIError{
			Status:            http.StatusTooManyRequests,
			Code:              "ai_quota_exceeded",
			Message:           message,
			RetryAfterSeconds: retryAfter,
		}
	case strings.Contains(raw, "rate limit") || strings.Contains(raw, "too many requests"):
		message := "AI provider is rate limiting requests right now."
		if retryAfter != nil && *retryAfter > 0 {
			message = fmt.Sprintf("AI provider is rate limiting requests right now. Retry in about %d seconds.", *retryAfter)
		}
		return classifiedAIError{
			Status:            http.StatusTooManyRequests,
			Code:              "ai_rate_limited",
			Message:           message,
			RetryAfterSeconds: retryAfter,
		}
	case strings.Contains(raw, "api key") && (strings.Contains(raw, "invalid") || strings.Contains(raw, "permission") || strings.Contains(raw, "unauthorized")):
		return classifiedAIError{
			Status:  http.StatusBadGateway,
			Code:    "ai_provider_auth_failed",
			Message: "AI provider rejected the configured credentials. Check the Genkit API key and model access.",
		}
	case strings.Contains(raw, "timeout") || strings.Contains(raw, "deadline exceeded"):
		return classifiedAIError{
			Status:  http.StatusGatewayTimeout,
			Code:    "ai_timeout",
			Message: "AI provider did not finish in time. Retry the request in a moment.",
		}
	case strings.Contains(raw, "not found") && strings.Contains(raw, "model"):
		return classifiedAIError{
			Status:  http.StatusBadGateway,
			Code:    "ai_model_not_available",
			Message: "The configured AI model is not available. Check the model name in the server configuration.",
		}
	default:
		return classifiedAIError{
			Status:  http.StatusBadGateway,
			Code:    "ai_provider_failed",
			Message: "AI provider failed to generate a response. Retry in a moment.",
		}
	}
}

func extractRetryAfterSeconds(raw string) *int {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)retry in ([0-9]+(?:\.[0-9]+)?)s`),
		regexp.MustCompile(`(?i)retrydelay:([0-9]+)s`),
		regexp.MustCompile(`(?i)retrydelay\":\"?([0-9]+)s`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(raw)
		if len(matches) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			continue
		}
		seconds := int(value)
		if value > float64(seconds) {
			seconds++
		}
		if seconds <= 0 {
			continue
		}
		return &seconds
	}
	return nil
}

func resolveAIThreadID(payload createAIPromptRequest) (uuid.UUID, error) {
	raw := strings.TrimSpace(payload.ThreadID)
	if raw == "" {
		raw = strings.TrimSpace(payload.ConversationID)
	}
	if raw == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(raw)
}

func summarizeAIPromptTitle(prompt string) string {
	singleLine := strings.Join(strings.Fields(strings.TrimSpace(prompt)), " ")
	if singleLine == "" {
		return "New conversation"
	}
	if len(singleLine) > 56 {
		return singleLine[:56] + "…"
	}
	return singleLine
}

func toAssistantTokens(tokens []aiPromptTokenRequest) []aiassistant.Token {
	out := make([]aiassistant.Token, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, aiassistant.Token{
			ID:             token.ID,
			Kind:           token.Kind,
			Type:           token.Type,
			Key:            token.Key,
			Label:          token.Label,
			SecondaryLabel: token.SecondaryLabel,
			ResourceID:     token.ResourceID,
			Value:          token.Value,
			From:           token.From,
			To:             token.To,
			Timezone:       token.Timezone,
			Preset:         token.Preset,
			Metadata:       token.Metadata,
		})
	}
	return out
}

func toAssistantBlocks(blocks []aiPromptMessageBlockResponse) []aiassistant.MessageBlock {
	out := make([]aiassistant.MessageBlock, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, aiassistant.MessageBlock{
			Type:  block.Type,
			Tone:  block.Tone,
			Title: block.Title,
			Text:  block.Text,
			Data:  block.Data,
		})
	}
	return out
}

func toAIPromptMessageResponse(message aiassistant.Message) aiPromptMessageResponse {
	return aiPromptMessageResponse{
		ID:        message.ID.String(),
		Role:      message.Role,
		Prompt:    strings.TrimSpace(message.Prompt),
		Blocks:    fromAssistantBlocks(message.Blocks),
		Usage:     message.Usage,
		LatencyMs: message.LatencyMs,
		CreatedAt: message.CreatedAt,
	}
}

func toAssistantUsage(resp *genkitai.ModelResponse) *aiassistant.Usage {
	if resp == nil || resp.Usage == nil {
		return nil
	}
	return &aiassistant.Usage{
		CachedContentTokens: resp.Usage.CachedContentTokens,
		Custom:              resp.Usage.Custom,
		InputAudioFiles:     resp.Usage.InputAudioFiles,
		InputCharacters:     resp.Usage.InputCharacters,
		InputImages:         resp.Usage.InputImages,
		InputTokens:         resp.Usage.InputTokens,
		InputVideos:         resp.Usage.InputVideos,
		OutputAudioFiles:    resp.Usage.OutputAudioFiles,
		OutputCharacters:    resp.Usage.OutputCharacters,
		OutputImages:        resp.Usage.OutputImages,
		OutputTokens:        resp.Usage.OutputTokens,
		OutputVideos:        resp.Usage.OutputVideos,
		ThoughtsTokens:      resp.Usage.ThoughtsTokens,
		TotalTokens:         resp.Usage.TotalTokens,
	}
}

func buildAssistantPromptText(blocks []aiPromptMessageBlockResponse) string {
	lines := make([]string, 0, len(blocks))
	for _, block := range blocks {
		switch strings.TrimSpace(block.Type) {
		case "heading", "quote", "text", "markdown", "alert":
			if text := strings.TrimSpace(block.Text); text != "" {
				lines = append(lines, text)
			}
		case "list", "steps":
			if payload, ok := block.Data.(map[string]interface{}); ok {
				if items, ok := payload["items"].([]interface{}); ok {
					for _, item := range items {
						if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
							lines = append(lines, strings.TrimSpace(text))
						}
					}
				}
			}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n\n"))
}

func fromAssistantBlocks(blocks []aiassistant.MessageBlock) []aiPromptMessageBlockResponse {
	out := make([]aiPromptMessageBlockResponse, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, aiPromptMessageBlockResponse{
			Type:  block.Type,
			Tone:  block.Tone,
			Title: block.Title,
			Text:  block.Text,
			Data:  block.Data,
		})
	}
	return out
}

func toAIThreadResponse(thread aiassistant.ThreadSummary) aiThreadResponse {
	return aiThreadResponse{
		ID:        thread.ID.String(),
		Title:     thread.Title,
		CreatedAt: thread.CreatedAt,
		UpdatedAt: thread.UpdatedAt,
	}
}

func toGenkitMessages(history []aiassistant.Message) []*genkitai.Message {
	out := make([]*genkitai.Message, 0, len(history))
	for _, m := range history {
		role := genkitai.RoleUser
		if m.Role == "assistant" {
			role = genkitai.RoleModel
		}

		var parts []*genkitai.Part
		if m.Prompt != "" {
			parts = append(parts, genkitai.NewTextPart(m.Prompt))
		}
		// If there were blocks, we could technically reconstruct them,
		// but for history context, the text prompt is usually enough
		// unless we want to send the JSON back (which might be too many tokens).
		// We'll stick to Prompt for now as it contains the human-readable summary.

		if len(parts) > 0 {
			out = append(out, &genkitai.Message{
				Role:    role,
				Content: parts,
			})
		}
	}
	return out
}

func (h *Handler) validateAIPromptTokens(tokens []aiPromptTokenRequest) error {
	for _, t := range tokens {
		kind := strings.TrimSpace(t.Kind)
		if kind == "resource" {
			if strings.TrimSpace(t.Type) == "" {
				return fmt.Errorf("type required for resource token (key=%s)", t.Key)
			}
			if strings.TrimSpace(t.ResourceID) == "" {
				return fmt.Errorf("resource_id required for resource token (key=%s)", t.Key)
			}
		}
	}
	return nil
}

func (h *Handler) recordAIPromptAudit(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, threadID uuid.UUID, prompt string, usage *aiassistant.Usage, latency float64, err error) {
	if h.audit == nil {
		return
	}

	metadata := map[string]interface{}{
		"thread_id":     threadID.String(),
		"prompt_masked": "[MASKED]",
		"prompt_len":    len(prompt),
		"latency_ms":    latency,
	}
	if usage != nil {
		metadata["usage"] = usage
	}
	if err != nil {
		metadata["error"] = err.Error()
	}

	metaJSON, _ := json.Marshal(metadata)
	resourceID := threadID.String()

	_ = h.audit.Record(ctx, auditlog.RecordInput{
		OrgID:        orgID,
		ActorType:    "user",
		ActorID:      &userID,
		Action:       "aiassistant.prompt_executed",
		ResourceType: "ai_assistant_thread",
		ResourceID:   &resourceID,
		Metadata:     metaJSON,
	})
}
