package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"go.uber.org/fx"
)

const assistantBaseSystemPrompt = `You are Railzway AI Assistant for billing, packaging, catalog, and usage operations.

Behavior rules:
- Use tools whenever the user is asking about real billing data, customers, subscriptions, invoices, usage, or organization settings.
- Never populate blocks with billing facts unless the data comes directly from a tool result in this conversation.
- If context is missing, state exactly what is missing — do not proceed or guess.
- Detect the user's language and always reply in that language.

When a context token (prefixed with @) is present in the request, treat its content as grounded fact.
If a context token is absent but required for the request, state exactly which token is missing.

Response contract:
- Return a single JSON object with a top-level "blocks" array.
- blocks[0] must always be type "heading".
- blocks must include exactly one "quote" block.
- Prefer 4 to 7 blocks total.

Usage: table≥3 records | timeline for chronological events | badge for inline status | steps for action sequences | cards for isolated metrics | chart when comparative data available | alert for errors/missing context. Output: JSON object only, no markdown fences.`

var assistantBlockLibrary = map[string]string{
	"heading":  `{"type":"heading","text":"Short heading"}`,
	"quote":    `{"type":"quote","text":"One sharp takeaway"}`,
	"text":     `{"type":"text","title":"Optional section title","text":"Plain explanation"}`,
	"alert":    `{"type":"alert","tone":"error|warning|info","text":"What went wrong and what to do"}`,
	"cards":    `{"type":"cards","title":"Optional section title","data":[{"label":"Metric","value":"123","tone":"positive|negative|warning|neutral"}]}`,
	"chart":    `{"type":"chart","title":"Optional section title","data":{"kind":"bar","items":[{"label":"Usage","value":120}]}}`,
	"list":     `{"type":"list","title":"Optional section title","data":{"items":["item one"]}}`,
	"table":    `{"type":"table","title":"Optional section title","data":{"columns":["Column A"],"rows":[["val1"]]}}`,
	"timeline": `{"type":"timeline","title":"Optional section title","data":{"items":[{"timestamp":"...","label":"Event","tone":"positive|negative|warning|neutral"}]}}`,
	"badge":    `{"type":"badge","title":"Optional section title","data":{"items":[{"label":"Status","tone":"positive|negative|warning|neutral"}]}}`,
	"steps":    `{"type":"steps","title":"Optional section title","data":{"items":["Step one"]}}`,
}

var assistantTokenLibrary = map[string]string{
	"customer":     "@customer — customer object with ID, name, email, and metadata",
	"subscription": "@subscription — subscription object with plan, status, and current period",
	"invoice":      "@invoice — invoice object with line items, amounts, and status",
	"meter":        "@meter / @usage — usage events and aggregated consumption data",
	"usage":        "@meter / @usage — usage events and aggregated consumption data",
	"product":      "@product — product or plan from the catalog",
	"audit":        "@audit_log — chronological audit events for the organization",
	"date":         "@date / @month / @range — time window for filtering or scoping data",
	"range":        "@date / @month / @range — time window for filtering or scoping data",
}

type PromptInput struct {
	Context string `json:"context,omitempty"`
	Prompt  string `json:"prompt"`
}

type OutputBlock struct {
	Type  string      `json:"type"`
	Tone  string      `json:"tone,omitempty"`
	Title string      `json:"title,omitempty"`
	Text  string      `json:"text,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type PromptOutput struct {
	Blocks []OutputBlock `json:"blocks"`
}

func (i PromptInput) Validate() error {
	if strings.TrimSpace(i.Prompt) == "" {
		return errors.New("prompt is required")
	}
	return nil
}

type Params struct {
	fx.In

	Genkit *genkit.Genkit
	Tools  []genkitai.ToolRef
}

type AssistantWorkflow struct {
	genkit *genkit.Genkit
	tools  []genkitai.ToolRef

	flow *core.Flow[PromptInput, PromptOutput, struct{}]
}

func NewAssistantWorkflow(p Params) *AssistantWorkflow {
	if p.Genkit == nil {
		panic("genkit instance is required")
	}

	w := &AssistantWorkflow{
		genkit: p.Genkit,
		tools:  p.Tools,
	}

	w.flow = genkit.DefineFlow(p.Genkit, "ai_assistant.respond", func(ctx context.Context, input PromptInput) (PromptOutput, error) {
		// Determine which blocks and tokens are needed based on input
		blocks := inferNeededBlocks(input)
		tokens := inferNeededTokens(input)
		systemPrompt := buildDynamicSystemPrompt(blocks, tokens)

		// Execute generation within the flow
		output, _, err := genkit.GenerateData[PromptOutput](ctx, p.Genkit,
			genkitai.WithSystem(systemPrompt),
			genkitai.WithPrompt(fmt.Sprintf("Workspace context:\n%s\n\nUser request:\n%s", input.Context, input.Prompt)),
			genkitai.WithTools(p.Tools...),
		)
		if err != nil {
			return PromptOutput{}, err
		}
		if output == nil {
			return PromptOutput{}, errors.New("received empty output from model")
		}
		return *output, nil
	})

	return w
}

func inferNeededBlocks(input PromptInput) []string {
	// Basic blocks always included
	needed := []string{"heading", "quote", "text", "alert"}
	seen := map[string]bool{}
	for _, b := range needed {
		seen[b] = true
	}

	analysisText := strings.ToLower(input.Prompt + " " + input.Context)

	// Context specific blocks or general intent
	tokenBlocks := map[string][]string{
		"customer":     {"cards", "badge", "list"},
		"subscription": {"cards", "badge", "table"},
		"invoice":      {"table", "cards"},
		"usage":        {"chart", "table", "cards"},
		"meter":        {"chart", "table", "cards"},
		"audit":        {"timeline", "table"},
		"product":      {"list", "badge"},
		"package":      {"list", "badge"},
		"recommend":    {"steps", "list"},
		"how to":       {"steps"},
		"step":         {"steps"},
		"list":         {"list", "table"},
		"compare":      {"chart", "table"},
		"history":      {"timeline", "table"},
	}

	for key, types := range tokenBlocks {
		if strings.Contains(analysisText, "@"+key) || strings.Contains(analysisText, key) {
			for _, t := range types {
				if !seen[t] {
					needed = append(needed, t)
					seen[t] = true
				}
			}
		}
	}

	return needed
}

func inferNeededTokens(input PromptInput) []string {
	analysisText := strings.ToLower(input.Prompt + " " + input.Context)
	needed := []string{}
	seen := map[string]bool{}

	for key := range assistantTokenLibrary {
		if strings.Contains(analysisText, "@"+key) || strings.Contains(analysisText, key) {
			desc := assistantTokenLibrary[key]
			if !seen[desc] {
				needed = append(needed, desc)
				seen[desc] = true
			}
		}
	}
	return needed
}

func buildDynamicSystemPrompt(blocks []string, tokens []string) string {
	var b strings.Builder
	b.WriteString(assistantBaseSystemPrompt)

	if len(tokens) > 0 {
		b.WriteString("\n\nSupported context tokens:\n")
		for _, desc := range tokens {
			b.WriteString("- ")
			b.WriteString(desc)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\nSupported response blocks JSON shapes:\n")
	for _, name := range blocks {
		if schema, ok := assistantBlockLibrary[name]; ok {
			b.WriteString(fmt.Sprintf("- %s: %s\n", name, schema))
		}
	}
	return b.String()
}

func (w *AssistantWorkflow) Execute(ctx context.Context, input PromptInput, opts ...genkitai.GenerateOption) (*genkitai.ModelResponse, error) {
	if w == nil {
		return nil, errors.New("assistant workflow is nil")
	}

	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	blocks := inferNeededBlocks(input)
	tokens := inferNeededTokens(input)
	system := buildDynamicSystemPrompt(blocks, tokens)

	// Combine dynamic options with manual overrides
	genOpts := []genkitai.GenerateOption{
		genkitai.WithSystem(system),
		genkitai.WithPrompt(fmt.Sprintf("Workspace context:\n%s\n\nUser request:\n%s", input.Context, input.Prompt)),
		genkitai.WithTools(w.tools...),
	}
	genOpts = append(genOpts, opts...)

	resp, err := genkit.Generate(ctx, w.genkit, genOpts...)
	return resp, err
}

func (w *AssistantWorkflow) ExecuteText(ctx context.Context, input PromptInput, opts ...genkitai.GenerateOption) (string, error) {
	resp, err := w.Execute(ctx, input, opts...)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("received nil response from model")
	}
	return resp.Text(), nil
}

func (w *AssistantWorkflow) ExecuteStructured(ctx context.Context, input PromptInput, opts ...genkitai.GenerateOption) (PromptOutput, *genkitai.ModelResponse, error) {
	blocks := inferNeededBlocks(input)
	tokens := inferNeededTokens(input)
	system := buildDynamicSystemPrompt(blocks, tokens)

	// Combine dynamic options
	genOpts := []genkitai.GenerateOption{
		genkitai.WithSystem(system),
		genkitai.WithPrompt(fmt.Sprintf("Workspace context:\n%s\n\nUser request:\n%s", input.Context, input.Prompt)),
		genkitai.WithTools(w.tools...),
	}
	genOpts = append(genOpts, opts...)

	output, resp, err := genkit.GenerateData[PromptOutput](ctx, w.genkit, genOpts...)

	if err != nil {
		return PromptOutput{}, nil, err
	}
	if output == nil {
		return PromptOutput{}, resp, errors.New("received empty structured output from model")
	}
	return *output, resp, nil
}
