package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/config"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	subscriptionscheduler "github.com/railzwaylabs/railzway/internal/subscription/scheduler"
	"github.com/railzwaylabs/railzway/internal/testclock/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

const defaultSimulationBatchSize = 200

type service struct {
	db             *gorm.DB
	repo           domain.Repository
	audit          *auditlog.Service
	closePeriodJob *subscriptionscheduler.ClosePeriodJob
	batchSize      int
}

type Params struct {
	fx.In

	DB         *gorm.DB
	Repo       domain.Repository
	Audit      *auditlog.Service             `optional:"true"`
	Config     *config.Config                `optional:"true"`
	SubRepo    subscriptiondomain.Repository `optional:"true"`
	InvoiceSvc invoicedomain.Service         `optional:"true"`
}

func NewService(p Params) domain.Service {
	batchSize := defaultSimulationBatchSize
	gracePeriod := time.Duration(0)
	if p.Config != nil {
		if p.Config.Billing.SubscriptionClosePeriodBatchSize > 0 {
			batchSize = p.Config.Billing.SubscriptionClosePeriodBatchSize
		}
		if p.Config.Billing.LateUsageGraceHours > 0 {
			gracePeriod = time.Duration(p.Config.Billing.LateUsageGraceHours) * time.Hour
		}
	}
	var closePeriodJob *subscriptionscheduler.ClosePeriodJob
	if p.DB != nil && p.SubRepo != nil && p.InvoiceSvc != nil {
		closePeriodJob = subscriptionscheduler.NewClosePeriodJob(p.DB, p.SubRepo, p.InvoiceSvc, gracePeriod)
	}
	return &service{
		db:             p.DB,
		repo:           p.Repo,
		audit:          p.Audit,
		closePeriodJob: closePeriodJob,
		batchSize:      batchSize,
	}
}

func (s *service) Get(ctx context.Context) (*domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, domain.ErrInvalidOrganization
	}
	clock, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		return nil, domain.ErrNotFound
	}
	return clock, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, domain.ErrInvalidOrganization
	}
	clockID, err := parseTestClockID(id)
	if err != nil {
		return nil, err
	}
	clock, err := s.repo.GetByID(ctx, orgID, clockID)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		return nil, domain.ErrNotFound
	}
	return clock, nil
}

func (s *service) List(ctx context.Context) ([]domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, domain.ErrInvalidOrganization
	}
	return s.repo.ListByOrgID(ctx, orgID)
}

func (s *service) Upsert(ctx context.Context, req domain.UpsertTestClockRequest) (domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TestClock{}, domain.ErrInvalidOrganization
	}
	if req.FrozenTime.IsZero() {
		return domain.TestClock{}, domain.ErrInvalidTime
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = domain.StatusActive
	}
	if status != domain.StatusActive && status != domain.StatusPaused {
		return domain.TestClock{}, domain.ErrInvalidStatus
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Test clock"
	}

	now := time.Now().UTC()
	record := domain.TestClock{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        name,
		Status:      status,
		CurrentTime: req.FrozenTime.UTC(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, record); err != nil {
		return domain.TestClock{}, err
	}

	out, err := s.repo.GetByID(ctx, orgID, record.ID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if out == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	s.recordAudit(ctx, "testclock.create", "test_clock", out.ID.String(), nil, *out, nil)
	return *out, nil
}

func (s *service) Advance(ctx context.Context, req domain.AdvanceTestClockRequest) (domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TestClock{}, domain.ErrInvalidOrganization
	}
	if req.FrozenTime.IsZero() {
		return domain.TestClock{}, domain.ErrInvalidTime
	}

	clock, err := s.findTargetClock(ctx, orgID, req.ID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if !req.FrozenTime.After(clock.CurrentTime) {
		return domain.TestClock{}, domain.ErrInvalidAdvance
	}

	nextTime := req.FrozenTime.UTC()
	updates := map[string]interface{}{
		"clock_time": nextTime,
		"updated_at": time.Now().UTC(),
		"status":     domain.StatusActive,
	}
	if err := s.repo.Update(ctx, orgID, clock.ID, updates); err != nil {
		return domain.TestClock{}, err
	}

	out, err := s.repo.GetByID(ctx, orgID, clock.ID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if out == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	closedPeriods, generatedInvoices, err := s.runSimulation(ctx, *out)
	if err != nil {
		return domain.TestClock{}, err
	}
	s.recordAudit(ctx, "testclock.advance", "test_clock", out.ID.String(), *clock, *out, map[string]interface{}{
		"from_time":          clock.CurrentTime.UTC().Format(time.RFC3339),
		"to_time":            out.CurrentTime.UTC().Format(time.RFC3339),
		"closed_periods":     closedPeriods,
		"generated_invoices": generatedInvoices,
	})
	return *out, nil
}

func (s *service) Pause(ctx context.Context) (domain.TestClock, error) {
	return s.PauseByID(ctx, "")
}

func (s *service) Resume(ctx context.Context) (domain.TestClock, error) {
	return s.ResumeByID(ctx, "")
}

func (s *service) PauseByID(ctx context.Context, id string) (domain.TestClock, error) {
	return s.updateStatus(ctx, id, domain.StatusPaused, "testclock.pause")
}

func (s *service) ResumeByID(ctx context.Context, id string) (domain.TestClock, error) {
	return s.updateStatus(ctx, id, domain.StatusActive, "testclock.resume")
}

func (s *service) updateStatus(ctx context.Context, id string, status string, auditAction string) (domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TestClock{}, domain.ErrInvalidOrganization
	}
	clock, err := s.findTargetClock(ctx, orgID, id)
	if err != nil {
		return domain.TestClock{}, err
	}
	if err := s.repo.Update(ctx, orgID, clock.ID, map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}); err != nil {
		return domain.TestClock{}, err
	}
	out, err := s.repo.GetByID(ctx, orgID, clock.ID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if out == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	s.recordAudit(ctx, auditAction, "test_clock", out.ID.String(), *clock, *out, nil)
	return *out, nil
}

func (s *service) ListActive(ctx context.Context) ([]domain.TestClock, error) {
	return s.repo.ListActive(ctx)
}

func (s *service) findTargetClock(ctx context.Context, orgID uuid.UUID, id string) (*domain.TestClock, error) {
	rawID := strings.TrimSpace(id)
	if rawID == "" {
		clock, err := s.repo.GetByOrgID(ctx, orgID)
		if err != nil {
			return nil, err
		}
		if clock == nil {
			return nil, domain.ErrNotFound
		}
		return clock, nil
	}
	clockID, err := parseTestClockID(rawID)
	if err != nil {
		return nil, err
	}
	clock, err := s.repo.GetByID(ctx, orgID, clockID)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		return nil, domain.ErrNotFound
	}
	return clock, nil
}

func parseTestClockID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidTime
	}
	return id, nil
}

func (s *service) runSimulation(ctx context.Context, testClock domain.TestClock) (int, int, error) {
	if s.closePeriodJob == nil {
		return 0, 0, nil
	}
	batchSize := s.batchSize
	if batchSize <= 0 {
		batchSize = defaultSimulationBatchSize
	}
	simCtx := orgcontext.WithOrgID(ctx, testClock.OrgID)
	simCtx = clock.WithTestClock(simCtx, testClock.ID, testClock.CurrentTime)

	totalClosed := 0
	totalGenerated := 0
	for {
		result, err := s.closePeriodJob.RunForTestClock(simCtx, testClock.OrgID, testClock.ID, testClock.CurrentTime, batchSize)
		if err != nil {
			return totalClosed, totalGenerated, fmt.Errorf("test clock simulation failed: %w", err)
		}
		totalClosed += result.ClosedPeriods
		totalGenerated += result.GeneratedInvoices
		if result.ClosedPeriods == 0 || result.ClosedPeriods < batchSize {
			break
		}
	}
	return totalClosed, totalGenerated, nil
}

func (s *service) recordAudit(ctx context.Context, action, resourceType, resourceID string, before, after interface{}, meta map[string]interface{}) {
	if s.audit == nil {
		return
	}
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return
	}

	actorType, actorID := auditlog.ActorFromContext(ctx)
	if strings.TrimSpace(actorType) == "" {
		actorType = "system"
	}

	var beforeJSON []byte
	if before != nil {
		beforeJSON, _ = json.Marshal(before)
	}
	var afterJSON []byte
	if after != nil {
		afterJSON, _ = json.Marshal(after)
	}

	var metaJSON []byte
	merged := mergeMetadata(meta, auditlog.MetadataFromContext(ctx))
	if merged != nil {
		metaJSON, _ = json.Marshal(merged)
	}

	var resourcePtr *string
	if strings.TrimSpace(resourceID) != "" {
		resourcePtr = &resourceID
	}

	requestID := strings.TrimSpace(auditlog.RequestIDFromContext(ctx))
	var requestPtr *string
	if requestID != "" {
		requestPtr = &requestID
	}

	reason := strings.TrimSpace(auditlog.ReasonFromContext(ctx))
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}

	_ = s.audit.Record(ctx, auditlog.RecordInput{
		OrgID:        orgID,
		ActorType:    actorType,
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourcePtr,
		BeforeData:   beforeJSON,
		AfterData:    afterJSON,
		Metadata:     metaJSON,
		Reason:       reasonPtr,
		RequestID:    requestPtr,
	})
}

func mergeMetadata(primary, secondary map[string]interface{}) map[string]interface{} {
	if len(primary) == 0 && len(secondary) == 0 {
		return nil
	}
	merged := map[string]interface{}{}
	for key, value := range secondary {
		merged[key] = value
	}
	for key, value := range primary {
		merged[key] = value
	}
	return merged
}
