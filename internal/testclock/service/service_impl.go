package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/testclock/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type service struct {
	db   *gorm.DB
	repo domain.Repository
	audit *auditlog.Service
}

type Params struct {
	fx.In

	DB   *gorm.DB
	Repo domain.Repository
	Audit *auditlog.Service `optional:"true"`
}

func NewService(p Params) domain.Service {
	return &service{
		db:   p.DB,
		repo: p.Repo,
		audit: p.Audit,
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

func (s *service) Upsert(ctx context.Context, req domain.UpsertTestClockRequest) (domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TestClock{}, domain.ErrInvalidOrganization
	}
	if req.CurrentTime.IsZero() {
		return domain.TestClock{}, domain.ErrInvalidTime
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = domain.StatusActive
	}
	if status != domain.StatusActive && status != domain.StatusPaused {
		return domain.TestClock{}, domain.ErrInvalidStatus
	}

	before, _ := s.repo.GetByOrgID(ctx, orgID)
	now := time.Now().UTC()
	record := domain.TestClock{
		ID:          uuid.New(),
		OrgID:       orgID,
		Status:      status,
		CurrentTime: req.CurrentTime.UTC(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Upsert(ctx, record); err != nil {
		return domain.TestClock{}, err
	}

	out, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if out == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	s.recordAudit(ctx, "testclock.upsert", "test_clock", out.ID.String(), before, *out, nil)
	return *out, nil
}

func (s *service) Advance(ctx context.Context, req domain.AdvanceTestClockRequest) (domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TestClock{}, domain.ErrInvalidOrganization
	}
	if req.AdvanceBy <= 0 {
		return domain.TestClock{}, domain.ErrInvalidAdvance
	}

	clock, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if clock == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}

	nextTime := clock.CurrentTime.Add(req.AdvanceBy)
	updates := map[string]interface{}{
		"clock_time": nextTime.UTC(),
		"updated_at": time.Now().UTC(),
		"status":     domain.StatusActive,
	}
	if err := s.repo.Update(ctx, orgID, updates); err != nil {
		return domain.TestClock{}, err
	}

	out, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if out == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	s.recordAudit(ctx, "testclock.advance", "test_clock", out.ID.String(), *clock, *out, map[string]interface{}{
		"advance_seconds": int64(req.AdvanceBy.Seconds()),
	})
	return *out, nil
}

func (s *service) Pause(ctx context.Context) (domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TestClock{}, domain.ErrInvalidOrganization
	}
	clock, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if clock == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	if err := s.repo.Update(ctx, orgID, map[string]interface{}{
		"status":     domain.StatusPaused,
		"updated_at": time.Now().UTC(),
	}); err != nil {
		return domain.TestClock{}, err
	}
	out, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if out == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	s.recordAudit(ctx, "testclock.pause", "test_clock", out.ID.String(), *clock, *out, nil)
	return *out, nil
}

func (s *service) Resume(ctx context.Context) (domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.TestClock{}, domain.ErrInvalidOrganization
	}
	clock, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if clock == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	if err := s.repo.Update(ctx, orgID, map[string]interface{}{
		"status":     domain.StatusActive,
		"updated_at": time.Now().UTC(),
	}); err != nil {
		return domain.TestClock{}, err
	}
	out, err := s.repo.GetByOrgID(ctx, orgID)
	if err != nil {
		return domain.TestClock{}, err
	}
	if out == nil {
		return domain.TestClock{}, domain.ErrNotFound
	}
	s.recordAudit(ctx, "testclock.resume", "test_clock", out.ID.String(), *clock, *out, nil)
	return *out, nil
}

func (s *service) ListActive(ctx context.Context) ([]domain.TestClock, error) {
	return s.repo.ListActive(ctx)
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
