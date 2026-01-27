package service

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/testclock/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	scheduler "github.com/railzwaylabs/railzway/internal/scheduler"
)

type Params struct {
	fx.In

	DB        *gorm.DB
	Log       *zap.Logger
	Clock     clock.Clock
	GenID     *snowflake.Node
	Repo      domain.Repository
	Scheduler *scheduler.Scheduler
}

type Service struct {
	db        *gorm.DB
	log       *zap.Logger
	clock     clock.Clock
	genID     *snowflake.Node
	repo      domain.Repository
	scheduler *scheduler.Scheduler
}

func New(p Params) domain.Service {
	return &Service{
		db:        p.DB,
		log:       p.Log.Named("testclock.service"),
		clock:     p.Clock,
		genID:     p.GenID,
		repo:      p.Repo,
		scheduler: p.Scheduler,
	}
}

func (s *Service) Create(ctx context.Context, req domain.CreateTestClockRequest) (*domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidStatus // Reuse or create new ErrInvalidOrganization
	}

	initialTime := req.InitialTime
	if initialTime.IsZero() {
		initialTime = s.clock.Now(ctx).UTC()
	} else {
		initialTime = initialTime.UTC()
	}

	clock := &domain.TestClock{
		ID:            s.genID.Generate(),
		OrgID:         orgID,
		Name:          req.Name,
		SimulatedTime: initialTime,
		Status:        domain.TestClockStatusActive,
		CreatedAt:     s.clock.Now(ctx).UTC(),
		UpdatedAt:     s.clock.Now(ctx).UTC(),
	}

	var created *domain.TestClock
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithTrx(tx).Create(ctx, clock); err != nil {
			return err
		}

		state := &domain.TestClockState{
			TestClockID: clock.ID,
			OrgID:       orgID,
			CurrentTime: clock.SimulatedTime,
			Status:      domain.TestClockStateStatusIdle,
			UpdatedAt:   clock.UpdatedAt,
		}

		if err := tx.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(state).Error; err != nil {
			return err
		}

		created = clock
		return nil
	}); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) Advance(ctx context.Context, id string, duration time.Duration) (*domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidStatus
	}

	clockID, err := snowflake.ParseString(id)
	if err != nil {
		return nil, domain.ErrTestClockNotFound
	}

	var (
		updatedClock *domain.TestClock
		advancingTo  time.Time
	)

	now := s.clock.Now(ctx).UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		clock, err := s.repo.FindByIDAndOrg(ctx, tx, clockID, orgID)
		if err != nil {
			return err
		}
		if clock == nil {
			return domain.ErrTestClockNotFound
		}

		if clock.Status != domain.TestClockStatusActive {
			// For now only active clocks can be advanced
			return domain.ErrInvalidStatus
		}

		state, err := s.loadStateForUpdate(ctx, tx, clockID, orgID)
		if err != nil {
			return err
		}
		if state == nil {
			return domain.ErrStateNotFound
		}
		if state.Status == domain.TestClockStateStatusAdvancing {
			return domain.ErrAdvanceInProgress
		}
		if !state.CurrentTime.UTC().Equal(clock.SimulatedTime.UTC()) {
			return domain.ErrStateMismatch
		}

		advancingTo = state.CurrentTime.Add(duration)
		if err := s.updateState(ctx, tx, clockID, orgID, map[string]any{
			"status":       string(domain.TestClockStateStatusAdvancing),
			"advancing_to": advancingTo,
			"last_error":   nil,
			"updated_at":   now,
		}); err != nil {
			return err
		}

		if err := s.repo.WithTrx(tx).Update(ctx, clock.ID.String(), map[string]any{
			"status":     domain.TestClockStatusAdvancing,
			"updated_at": now,
		}); err != nil {
			return err
		}

		clock.Status = domain.TestClockStatusAdvancing
		clock.UpdatedAt = now
		updatedClock = clock
		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := s.scheduler.TriggerSimulationStep(ctx, updatedClock.ID, advancingTo); err != nil {
		if updateErr := s.markAdvanceFailed(ctx, updatedClock.ID, orgID, err); updateErr != nil {
			s.log.Error("failed to mark test clock advance failed",
				zap.Error(updateErr),
				zap.String("test_clock_id", updatedClock.ID.String()),
			)
		}
		return nil, err
	}

	if err := s.markAdvanceSucceeded(ctx, updatedClock.ID, orgID, advancingTo); err != nil {
		return nil, err
	}

	updatedClock.SimulatedTime = advancingTo
	updatedClock.Status = domain.TestClockStatusActive
	updatedClock.UpdatedAt = s.clock.Now(ctx).UTC()
	return updatedClock, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidStatus
	}

	clockID, err := snowflake.ParseString(id)
	if err != nil {
		return nil, domain.ErrTestClockNotFound
	}

	clock, err := s.repo.FindByIDAndOrg(ctx, s.db, clockID, orgID)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		return nil, domain.ErrTestClockNotFound
	}

	return clock, nil
}

func (s *Service) List(ctx context.Context) ([]domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidStatus
	}

	clocks, err := s.repo.FindActiveByOrg(ctx, s.db, orgID)
	if err != nil {
		return nil, err
	}

	return clocks, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.ErrInvalidStatus
	}

	clockID, err := snowflake.ParseString(id)
	if err != nil {
		return domain.ErrTestClockNotFound
	}

	// Verify existence and ownership
	clock, err := s.repo.FindByIDAndOrg(ctx, s.db, clockID, orgID)
	if err != nil {
		return err
	}
	if clock == nil {
		return domain.ErrTestClockNotFound
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, req domain.UpdateTestClockRequest) (*domain.TestClock, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidStatus
	}

	clockID, err := snowflake.ParseString(id)
	if err != nil {
		return nil, domain.ErrTestClockNotFound
	}

	clock, err := s.repo.FindByIDAndOrg(ctx, s.db, clockID, orgID)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		return nil, domain.ErrTestClockNotFound
	}

	clock.Name = req.Name
	clock.UpdatedAt = s.clock.Now(ctx).UTC()

	if err := s.repo.Update(ctx, id, clock); err != nil {
		return nil, err
	}

	return clock, nil
}

func (s *Service) loadStateForUpdate(ctx context.Context, tx *gorm.DB, testClockID, orgID snowflake.ID) (*domain.TestClockState, error) {
	var state domain.TestClockState
	result := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("test_clock_id = ? AND org_id = ?", testClockID, orgID).
		First(&state)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &state, nil
}

func (s *Service) updateState(ctx context.Context, tx *gorm.DB, testClockID, orgID snowflake.ID, updates map[string]any) error {
	return tx.WithContext(ctx).
		Table(domain.TestClockState{}.TableName()).
		Where("test_clock_id = ? AND org_id = ?", testClockID, orgID).
		Updates(updates).Error
}

func (s *Service) markAdvanceSucceeded(ctx context.Context, testClockID, orgID snowflake.ID, advancingTo time.Time) error {
	now := s.clock.Now(ctx).UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.updateState(ctx, tx, testClockID, orgID, map[string]any{
			"simulated_time": advancingTo,
			"advancing_to":   nil,
			"status":         string(domain.TestClockStateStatusSucceeded),
			"last_error":     nil,
			"updated_at":     now,
		}); err != nil {
			return err
		}

		return s.repo.WithTrx(tx).Update(ctx, testClockID.String(), map[string]any{
			"simulated_time": advancingTo,
			"status":         domain.TestClockStatusActive,
			"updated_at":     now,
		})
	})
}

func (s *Service) markAdvanceFailed(ctx context.Context, testClockID, orgID snowflake.ID, cause error) error {
	now := s.clock.Now(ctx).UTC()
	lastError := cause.Error()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.updateState(ctx, tx, testClockID, orgID, map[string]any{
			"status":     string(domain.TestClockStateStatusFailed),
			"last_error": lastError,
			"updated_at": now,
		}); err != nil {
			return err
		}

		return s.repo.WithTrx(tx).Update(ctx, testClockID.String(), map[string]any{
			"status":     domain.TestClockStatusActive,
			"updated_at": now,
		})
	})
}
