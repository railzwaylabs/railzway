package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	auditdomain "github.com/railzwaylabs/railzway/internal/audit/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const orgBootstrapStateTable = "org_bootstrap_state"

type OrgBootstrapStatus string

const (
	OrgStatusInitializing OrgBootstrapStatus = "initializing"
	OrgStatusActive       OrgBootstrapStatus = "active"
	OrgStatusSuspended    OrgBootstrapStatus = "suspended"
	OrgStatusTerminated   OrgBootstrapStatus = "terminated"
)

var (
	ErrOrgBootstrapStateNotFound = errors.New("org bootstrap state not found")
	ErrOrgNotActive              = errors.New("organization is not active")
	ErrOrgInvalidTransition      = errors.New("invalid organization state transition")
)

type OrgBootstrapState struct {
	OrgID        snowflake.ID `gorm:"column:org_id"`
	Status       string       `gorm:"column:status"`
	CreatedAt    time.Time    `gorm:"column:created_at"`
	ActivatedAt  *time.Time   `gorm:"column:activated_at"`
	SuspendedAt  *time.Time   `gorm:"column:suspended_at"`
	TerminatedAt *time.Time   `gorm:"column:terminated_at"`
}

type OrgStateService interface {
	WithTx(tx *gorm.DB) OrgStateService
	Initialize(ctx context.Context, orgID snowflake.ID, createdAt time.Time) error
	Activate(ctx context.Context, orgID snowflake.ID, activatedAt time.Time) error
	Suspend(ctx context.Context, orgID snowflake.ID, suspendedAt time.Time) error
	Terminate(ctx context.Context, orgID snowflake.ID, terminatedAt time.Time) error
	Get(ctx context.Context, orgID snowflake.ID) (*OrgBootstrapState, error)
}

type orgStateService struct {
	db       *gorm.DB
	auditSvc auditdomain.Service
}

type OrgStateParams struct {
	fx.In

	DB       *gorm.DB
	AuditSvc auditdomain.Service `optional:"true"`
}

func NewOrgStateService(p OrgStateParams) OrgStateService {
	return &orgStateService{db: p.DB, auditSvc: p.AuditSvc}
}

func (s *orgStateService) WithTx(tx *gorm.DB) OrgStateService {
	if tx == nil {
		return s
	}
	return &orgStateService{db: tx, auditSvc: s.auditSvc}
}

func (s *orgStateService) Initialize(ctx context.Context, orgID snowflake.ID, createdAt time.Time) error {
	if orgID == 0 {
		return errors.New("org id is required")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	result := s.db.WithContext(ctx).Exec(
		`INSERT INTO org_bootstrap_state (org_id, status, created_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT (org_id) DO NOTHING`,
		orgID,
		string(OrgStatusInitializing),
		createdAt.UTC(),
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		if err := s.auditTransition(ctx, orgID, "", string(OrgStatusInitializing), createdAt.UTC()); err != nil {
			return err
		}
	}
	return nil
}

func (s *orgStateService) Activate(ctx context.Context, orgID snowflake.ID, activatedAt time.Time) error {
	return s.transition(ctx, orgID, OrgStatusActive, activatedAt)
}

func (s *orgStateService) Suspend(ctx context.Context, orgID snowflake.ID, suspendedAt time.Time) error {
	return s.transition(ctx, orgID, OrgStatusSuspended, suspendedAt)
}

func (s *orgStateService) Terminate(ctx context.Context, orgID snowflake.ID, terminatedAt time.Time) error {
	return s.transition(ctx, orgID, OrgStatusTerminated, terminatedAt)
}

func (s *orgStateService) Get(ctx context.Context, orgID snowflake.ID) (*OrgBootstrapState, error) {
	if orgID == 0 {
		return nil, errors.New("org id is required")
	}
	var state OrgBootstrapState
	result := s.db.WithContext(ctx).Table(orgBootstrapStateTable).
		Select("org_id, status, created_at, activated_at, suspended_at, terminated_at").
		Where("org_id = ?", orgID).
		Limit(1).
		Scan(&state)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrOrgBootstrapStateNotFound
	}
	state.Status = strings.ToLower(strings.TrimSpace(state.Status))
	return &state, nil
}

func (s *orgStateService) transition(ctx context.Context, orgID snowflake.ID, target OrgBootstrapStatus, when time.Time) error {
	if orgID == 0 {
		return errors.New("org id is required")
	}
	if when.IsZero() {
		when = time.Now().UTC()
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var state OrgBootstrapState
		if err := tx.WithContext(ctx).
			Table(orgBootstrapStateTable).
			Select("org_id, status, activated_at, suspended_at, terminated_at").
			Where("org_id = ?", orgID).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Scan(&state).Error; err != nil {
			return err
		}
		if state.OrgID == 0 {
			return ErrOrgBootstrapStateNotFound
		}

		current := OrgBootstrapStatus(strings.ToLower(strings.TrimSpace(state.Status)))
		if current == target {
			return nil
		}
		if !isOrgTransitionAllowed(current, target) {
			return fmt.Errorf("%w: %s -> %s", ErrOrgInvalidTransition, current, target)
		}

		updates := map[string]any{"status": string(target)}
		if target == OrgStatusActive && state.ActivatedAt == nil {
			updates["activated_at"] = when.UTC()
		}
		if target == OrgStatusSuspended && state.SuspendedAt == nil {
			updates["suspended_at"] = when.UTC()
		}
		if target == OrgStatusTerminated && state.TerminatedAt == nil {
			updates["terminated_at"] = when.UTC()
		}

		if err := tx.WithContext(ctx).
			Table(orgBootstrapStateTable).
			Where("org_id = ?", orgID).
			Updates(updates).Error; err != nil {
			return err
		}

		return s.auditTransition(ctx, orgID, string(current), string(target), when.UTC())
	})
}

func isOrgTransitionAllowed(current OrgBootstrapStatus, target OrgBootstrapStatus) bool {
	switch current {
	case OrgStatusInitializing:
		return target == OrgStatusActive || target == OrgStatusTerminated
	case OrgStatusActive:
		return target == OrgStatusSuspended || target == OrgStatusTerminated
	case OrgStatusSuspended:
		return target == OrgStatusActive || target == OrgStatusTerminated
	case OrgStatusTerminated:
		return false
	default:
		return false
	}
}

func (s *orgStateService) auditTransition(ctx context.Context, orgID snowflake.ID, from string, to string, at time.Time) error {
	if s.auditSvc == nil {
		return nil
	}
	if orgID == 0 {
		return errors.New("org id is required for audit transition")
	}
	targetID := orgID.String()
	metadata := map[string]any{
		"from_status": strings.TrimSpace(from),
		"to_status":   strings.TrimSpace(to),
		"at":          at.UTC().Format(time.RFC3339),
	}
	action := "org.bootstrap.transition"
	if strings.TrimSpace(from) == "" && strings.TrimSpace(to) != "" {
		action = "org.bootstrap.initialized"
	}
	if strings.TrimSpace(from) != "" && strings.TrimSpace(to) == string(OrgStatusActive) {
		action = "org.bootstrap.activated"
	}
	if strings.TrimSpace(to) == string(OrgStatusSuspended) {
		action = "org.bootstrap.suspended"
	}
	if strings.TrimSpace(to) == string(OrgStatusTerminated) {
		action = "org.bootstrap.terminated"
	}
	return s.auditSvc.AuditLog(ctx, &orgID, string(auditdomain.ActorTypeSystem), nil, action, "organization", &targetID, metadata)
}
