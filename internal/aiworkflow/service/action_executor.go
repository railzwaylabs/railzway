package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiworkflow/domain"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

type actionExecutor interface {
	Execute(ctx context.Context, workflow domain.Workflow, action domain.WorkflowActionRow) error
}

type actionExecutorImpl struct {
	products productdomain.Service
	plans    plandomain.Service
	features featuredomain.Service
	usage    usagedomain.Service
}

func NewActionExecutor(products productdomain.Service, plans plandomain.Service, features featuredomain.Service, usage usagedomain.Service) actionExecutor {
	return &actionExecutorImpl{
		products: products,
		plans:    plans,
		features: features,
		usage:    usage,
	}
}

type actionPayload struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`

	FeatureType string `json:"feature_type"`
	MeterID     string `json:"meter_id"`

	Aggregation string `json:"aggregation"`
	Unit        string `json:"unit"`

	ProductID string `json:"product_id"`
}

func (e *actionExecutorImpl) Execute(ctx context.Context, workflow domain.Workflow, action domain.WorkflowActionRow) error {
	switch action.Type {
	case domain.ActionNavigate, domain.ActionNotify:
		return nil
	case domain.ActionCreateProduct:
		return e.createProduct(ctx, workflow, action)
	case domain.ActionCreatePlan:
		return e.createPlan(ctx, workflow, action)
	case domain.ActionCreateMeter:
		return e.createMeter(ctx, workflow, action)
	case domain.ActionCreateFeature:
		return e.createFeature(ctx, workflow, action)
	default:
		return fmt.Errorf("unsupported_action:%s", action.Type)
	}
}

func (e *actionExecutorImpl) createProduct(ctx context.Context, workflow domain.Workflow, action domain.WorkflowActionRow) error {
	payload := parsePayload(action.Payload)
	name := pickName(payload.Name, action.Label, "AI Product")
	code := pickCode(payload.Code, name)
	desc := trimOptional(payload.Description)
	idempotencyKey := buildIdempotencyKey(workflow.ID, action.ID)

	_, err := e.products.Create(ctx, productdomain.CreateProductRequest{
		Code:           code,
		Name:           name,
		Description:    desc,
		IdempotencyKey: idempotencyKey,
	})
	return err
}

func (e *actionExecutorImpl) createPlan(ctx context.Context, workflow domain.Workflow, action domain.WorkflowActionRow) error {
	payload := parsePayload(action.Payload)
	name := pickName(payload.Name, action.Label, "AI Plan")
	code := pickCode(payload.Code, name)
	desc := strings.TrimSpace(payload.Description)
	idempotencyKey := buildIdempotencyKey(workflow.ID, action.ID)
	productID := trimOptional(payload.ProductID)

	_, err := e.plans.CreatePlan(ctx, plandomain.CreatePlanRequest{
		Code:           code,
		Name:           name,
		Description:    desc,
		ProductID:      productID,
		IdempotencyKey: idempotencyKey,
	})
	return err
}

func (e *actionExecutorImpl) createMeter(ctx context.Context, workflow domain.Workflow, action domain.WorkflowActionRow) error {
	payload := parsePayload(action.Payload)
	name := pickName(payload.Name, action.Label, "AI Meter")
	code := pickCode(payload.Code, name)
	aggregation := strings.TrimSpace(payload.Aggregation)
	if aggregation == "" {
		aggregation = usagedomain.AggregationSum
	}
	unit := strings.TrimSpace(payload.Unit)
	if unit == "" {
		unit = "unit"
	}
	idempotencyKey := buildIdempotencyKey(workflow.ID, action.ID)

	_, err := e.usage.CreateMeter(ctx, usagedomain.CreateMeterRequest{
		Code:           code,
		Name:           name,
		Aggregation:    aggregation,
		Unit:           unit,
		IdempotencyKey: idempotencyKey,
	})
	return err
}

func (e *actionExecutorImpl) createFeature(ctx context.Context, workflow domain.Workflow, action domain.WorkflowActionRow) error {
	payload := parsePayload(action.Payload)
	name := pickName(payload.Name, action.Label, "AI Feature")
	code := pickCode(payload.Code, name)
	featureType := strings.TrimSpace(payload.FeatureType)
	if featureType == "" {
		featureType = string(featuredomain.FeatureTypeBoolean)
	}

	var meterID *string
	if strings.TrimSpace(payload.MeterID) != "" {
		meterID = &payload.MeterID
	}

	idempotencyKey := buildIdempotencyKey(workflow.ID, action.ID)

	_, err := e.features.Create(ctx, featuredomain.CreateFeatureRequest{
		Code:           code,
		Name:           name,
		Description:    trimOptional(payload.Description),
		FeatureType:    featureType,
		MeterID:        meterID,
		IdempotencyKey: idempotencyKey,
	})
	return err
}

func parsePayload(raw json.RawMessage) actionPayload {
	if len(raw) == 0 || string(raw) == "null" {
		return actionPayload{}
	}
	var payload actionPayload
	_ = json.Unmarshal(raw, &payload)
	return payload
}

func pickName(value, fallback, defaultName string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	trimmed = strings.TrimSpace(fallback)
	if trimmed != "" {
		return trimmed
	}
	return defaultName
}

func pickCode(code, name string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed != "" {
		return normalizeCode(trimmed)
	}
	return normalizeCode(name)
}

func normalizeCode(value string) string {
	code := strings.ToLower(strings.TrimSpace(value))
	code = strings.ReplaceAll(code, " ", "-")
	return code
}

func trimOptional(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func buildIdempotencyKey(workflowID, actionID uuid.UUID) string {
	if workflowID == uuid.Nil || actionID == uuid.Nil {
		return ""
	}
	return fmt.Sprintf("aiwf-%s-%s", workflowID.String(), actionID.String())
}
