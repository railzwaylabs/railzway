package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/clock"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/subscription/domain"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

const subscriptionCacheTTL = 15 * time.Second

type service struct {
	db        *gorm.DB
	repo      domain.Repository
	customers customerdomain.Repository
	clocks    testclockdomain.Repository
	audit     *auditlog.Service
	clock     clock.Clock
	cache     *redis.Client
}

type Params struct {
	fx.In

	DB        *gorm.DB
	Repo      domain.Repository
	Customers customerdomain.Repository  `optional:"true"`
	Clocks    testclockdomain.Repository `optional:"true"`
	Audit     *auditlog.Service          `optional:"true"`
	Clock     clock.Clock                `optional:"true"`
	Cache     *redis.Client              `name:"redis_cache" optional:"true"`
}

func NewService(p Params) domain.Service {
	clk := p.Clock
	if clk == nil {
		clk = clock.SystemClock{}
	}
	return &service{
		db:        p.DB,
		repo:      p.Repo,
		customers: p.Customers,
		clocks:    p.Clocks,
		audit:     p.Audit,
		clock:     clk,
		cache:     p.Cache,
	}
}

func (s *service) CreateSubscription(ctx context.Context, req domain.CreateSubscriptionRequest) (resp domain.SubscriptionResponse, err error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidOrganization
	}

	ctx, span := telemetry.StartSpan(
		ctx,
		"subscription.create",
		telemetry.UUIDAttr("billing.org_id", orgID),
		telemetry.StringAttr("billing.customer_id", strings.TrimSpace(req.CustomerID)),
		telemetry.StringAttr("billing.plan_id", strings.TrimSpace(req.PlanID)),
		telemetry.Int64Attr("billing.subscription_items_count", int64(len(req.Items))),
		telemetry.StringAttr("billing.idempotency_key", strings.TrimSpace(req.IdempotencyKey)),
	)
	defer func() { telemetry.EndSpan(span, err) }()

	startedAt := time.Now()
	defer func() { telemetry.ObserveOperation("subscription.create", time.Since(startedAt), err) }()

	if len(req.Items) == 0 {
		return domain.SubscriptionResponse{}, domain.ErrMissingItems
	}

	customerID, err := parseID(req.CustomerID)
	if err != nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidCustomer
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, customerID)
	if err != nil {
		return domain.SubscriptionResponse{}, err
	}

	planID, err := parseID(req.PlanID)
	if err != nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidPlan
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = domain.StatusActive
	}

	if !isValidStatus(status) {
		return domain.SubscriptionResponse{}, domain.ErrInvalidStatus
	}

	currency := normalizeCurrency(req.Currency)
	if !isValidCurrency(currency) {
		return domain.SubscriptionResponse{}, domain.ErrInvalidCurrency
	}

	startAt := s.clock.Now(ctx)
	if req.StartAt != nil {
		startAt = req.StartAt.UTC()
	}

	if req.TrialEnd != nil {
		trialEnd := req.TrialEnd.UTC()
		if trialEnd.Before(startAt) {
			return domain.SubscriptionResponse{}, domain.ErrInvalidPeriod
		}
		if req.Status == "" {
			status = domain.StatusTrialing
		}
		if status == domain.StatusTrialing && trialEnd.Before(s.clock.Now(ctx)) {
			return domain.SubscriptionResponse{}, domain.ErrInvalidPeriod
		}
	}
	if status == domain.StatusTrialing && req.TrialEnd == nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidPeriod
	}

	if req.CurrentPeriodStart.IsZero() || req.CurrentPeriodEnd.IsZero() {
		return domain.SubscriptionResponse{}, domain.ErrInvalidPeriod
	}

	if req.CurrentPeriodEnd.Before(req.CurrentPeriodStart) {
		return domain.SubscriptionResponse{}, domain.ErrInvalidPeriod
	}

	if req.TrialEnd != nil && req.TrialEnd.UTC().Before(req.CurrentPeriodStart.UTC()) {
		return domain.SubscriptionResponse{}, domain.ErrInvalidPeriod
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindSubscriptionByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.SubscriptionResponse{}, err
		}
		if existing != nil {
			resp = toSubscriptionResponse(*existing)
			items, err := s.listSubscriptionItemsBySubscription(ctx, orgID, existing.ID)
			if err != nil {
				return domain.SubscriptionResponse{}, err
			}

			resp.Items = items
			span.SetAttributes(
				telemetry.UUIDAttr("billing.subscription_id", existing.ID),
				telemetry.BoolAttr("billing.idempotent_hit", true),
			)

			return resp, nil
		}
	}

	now := s.clock.Now(ctx)
	sub := domain.Subscription{
		ID:                 uuid.New(),
		OrgID:              orgID,
		CustomerID:         customerID,
		PlanID:             planID,
		Status:             status,
		Currency:           currency,
		StartAt:            startAt,
		CurrentPeriodStart: req.CurrentPeriodStart.UTC(),
		CurrentPeriodEnd:   req.CurrentPeriodEnd.UTC(),
		TrialEnd:           req.TrialEnd,
		CancelAt:           req.CancelAt,
		Metadata:           json.RawMessage(`{}`),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if req.TrialEnd != nil {
		t := req.TrialEnd.UTC()
		sub.TrialEnd = &t
	}
	if req.CancelAt != nil {
		t := req.CancelAt.UTC()
		sub.CancelAt = &t
	}
	if idempotencyKey != "" {
		sub.IdempotencyKey = &idempotencyKey
	}

	var itemResponses []domain.SubscriptionItemResponse
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)
		if err := repo.CreateSubscription(ctx, sub); err != nil {
			return err
		}

		period := domain.SubscriptionPeriod{
			ID:             uuid.New(),
			OrgID:          orgID,
			SubscriptionID: sub.ID,
			Status:         domain.PeriodStatusOpen,
			PeriodStart:    sub.CurrentPeriodStart,
			PeriodEnd:      sub.CurrentPeriodEnd,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := repo.CreateSubscriptionPeriod(ctx, period); err != nil {
			return err
		}

		itemResponses = make([]domain.SubscriptionItemResponse, 0, len(req.Items))
		for _, itemReq := range req.Items {
			priceID, err := parseID(itemReq.PlanPriceID)
			if err != nil {
				return domain.ErrInvalidPlanPrice
			}
			if itemReq.Quantity < 0 {
				return domain.ErrInvalidQuantity
			}

			startAtItem := sub.CurrentPeriodStart
			if itemReq.StartAt != nil {
				startAtItem = itemReq.StartAt.UTC()
			}
			var endAtItem *time.Time
			if itemReq.EndAt != nil {
				t := itemReq.EndAt.UTC()
				if t.Before(startAtItem) {
					return domain.ErrInvalidPeriod
				}
				endAtItem = &t
			}

			itemIdempotencyKey := strings.TrimSpace(itemReq.IdempotencyKey)
			var item domain.SubscriptionItem
			if itemIdempotencyKey != "" {
				existingItem, err := repo.FindSubscriptionItemByIdempotencyKey(ctx, orgID, itemIdempotencyKey)
				if err != nil {
					return err
				}
				if existingItem != nil {
					item = *existingItem
				}
			}

			if item.ID == uuid.Nil {
				item = domain.SubscriptionItem{
					ID:             uuid.New(),
					OrgID:          orgID,
					SubscriptionID: sub.ID,
					PlanPriceID:    priceID,
					Quantity:       itemReq.Quantity,
					StartAt:        startAtItem,
					EndAt:          endAtItem,
					Metadata:       json.RawMessage(`{}`),
					CreatedAt:      now,
					UpdatedAt:      now,
				}
				if itemIdempotencyKey != "" {
					item.IdempotencyKey = &itemIdempotencyKey
				}

				if err := repo.CreateSubscriptionItem(ctx, item); err != nil {
					if itemIdempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
						existingItem, findErr := repo.FindSubscriptionItemByIdempotencyKey(ctx, orgID, itemIdempotencyKey)
						if findErr != nil {
							return findErr
						}
						if existingItem != nil {
							item = *existingItem
						} else {
							return err
						}
					} else {
						return err
					}
				}
			}

			itemResp := toSubscriptionItemResponse(item)
			itemResponses = append(itemResponses, itemResp)
		}

		return nil
	}); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindSubscriptionByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.SubscriptionResponse{}, findErr
			}
			if existing != nil {
				resp = toSubscriptionResponse(*existing)
				span.SetAttributes(
					telemetry.UUIDAttr("billing.subscription_id", existing.ID),
					telemetry.BoolAttr("billing.idempotent_hit", true),
				)
				return resp, nil
			}
		}
		return domain.SubscriptionResponse{}, err
	}

	resp = toSubscriptionResponse(sub)
	if len(itemResponses) > 0 {
		resp.Items = itemResponses
	}
	span.SetAttributes(
		telemetry.UUIDAttr("billing.subscription_id", sub.ID),
		telemetry.UUIDAttr("billing.customer_id", sub.CustomerID),
		telemetry.UUIDAttr("billing.plan_id", sub.PlanID),
	)
	s.setSubscriptionCache(ctx, orgID, sub.ID, resp)
	s.recordAudit(ctx, "subscription.create", "subscription", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) UpdateSubscription(ctx context.Context, id string, req domain.UpdateSubscriptionRequest) (domain.SubscriptionResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidOrganization
	}

	subID, err := parseID(id)
	if err != nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidID
	}

	beforeSub, err := s.repo.FindSubscriptionByID(ctx, orgID, subID)
	if err != nil {
		return domain.SubscriptionResponse{}, err
	}
	if beforeSub == nil {
		return domain.SubscriptionResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, beforeSub.CustomerID)
	if err != nil {
		return domain.SubscriptionResponse{}, err
	}

	now := s.clock.Now(ctx)
	updates := map[string]interface{}{
		"updated_at": now,
	}

	desiredStatus := beforeSub.Status
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if !isValidStatus(status) {
			return domain.SubscriptionResponse{}, domain.ErrInvalidStatus
		}
		desiredStatus = status
	}

	if req.CanceledAt != nil || req.EndedAt != nil {
		if req.Status != nil && desiredStatus != domain.StatusCanceled {
			return domain.SubscriptionResponse{}, domain.ErrInvalidStatus
		}
		desiredStatus = domain.StatusCanceled
	}

	if !isValidTransition(beforeSub.Status, desiredStatus) {
		return domain.SubscriptionResponse{}, domain.ErrInvalidStatus
	}

	if desiredStatus != beforeSub.Status {
		updates["status"] = desiredStatus
	}

	if desiredStatus == domain.StatusCanceled {
		canceledAt := now
		if req.CanceledAt != nil {
			canceledAt = req.CanceledAt.UTC()
		}
		endedAt := canceledAt
		if req.EndedAt != nil {
			endedAt = req.EndedAt.UTC()
		}
		updates["canceled_at"] = canceledAt
		updates["ended_at"] = endedAt
	} else if req.CancelAt != nil {
		t := req.CancelAt.UTC()
		if t.Before(beforeSub.CurrentPeriodStart) {
			return domain.SubscriptionResponse{}, domain.ErrInvalidPeriod
		}
		updates["cancel_at"] = t
	}

	if err := s.repo.UpdateSubscription(ctx, orgID, subID, updates); err != nil {
		return domain.SubscriptionResponse{}, err
	}
	s.deleteSubscriptionCache(ctx, orgID, subID)

	item, err := s.repo.FindSubscriptionByID(ctx, orgID, subID)
	if err != nil {
		return domain.SubscriptionResponse{}, err
	}
	if item == nil {
		return domain.SubscriptionResponse{}, domain.ErrNotFound
	}

	resp := toSubscriptionResponse(*item)
	s.setSubscriptionCache(ctx, orgID, subID, resp)
	s.recordAudit(ctx, "subscription.update", "subscription", resp.ID, toSubscriptionResponse(*beforeSub), resp, nil)
	return resp, nil
}

func (s *service) GetSubscriptionByID(ctx context.Context, req domain.GetSubscriptionRequest) (domain.SubscriptionResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidOrganization
	}

	id, err := parseID(req.ID)
	if err != nil {
		return domain.SubscriptionResponse{}, domain.ErrInvalidID
	}
	if cached, ok := s.getSubscriptionCache(ctx, orgID, id); ok {
		return cached, nil
	}

	item, err := s.repo.FindSubscriptionByID(ctx, orgID, id)
	if err != nil {
		return domain.SubscriptionResponse{}, err
	}
	if item == nil {
		return domain.SubscriptionResponse{}, domain.ErrNotFound
	}

	resp := toSubscriptionResponse(*item)
	s.setSubscriptionCache(ctx, orgID, id, resp)
	return resp, nil
}

func subscriptionCacheKey(orgID uuid.UUID, subscriptionID uuid.UUID) string {
	return fmt.Sprintf("cache:subscription:%s:%s", orgID.String(), subscriptionID.String())
}

func (s *service) getSubscriptionCache(ctx context.Context, orgID, subscriptionID uuid.UUID) (domain.SubscriptionResponse, bool) {
	if s.cache == nil {
		return domain.SubscriptionResponse{}, false
	}
	raw, err := s.cache.Get(ctx, subscriptionCacheKey(orgID, subscriptionID)).Result()
	if err != nil {
		return domain.SubscriptionResponse{}, false
	}
	var cached domain.SubscriptionResponse
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return domain.SubscriptionResponse{}, false
	}
	return cached, true
}

func (s *service) setSubscriptionCache(ctx context.Context, orgID, subscriptionID uuid.UUID, resp domain.SubscriptionResponse) {
	if s.cache == nil {
		return
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, subscriptionCacheKey(orgID, subscriptionID), raw, subscriptionCacheTTL).Err()
}

func (s *service) deleteSubscriptionCache(ctx context.Context, orgID, subscriptionID uuid.UUID) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, subscriptionCacheKey(orgID, subscriptionID)).Err()
}

func (s *service) ListSubscriptions(ctx context.Context, req domain.ListSubscriptionRequest) (domain.ListSubscriptionResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListSubscriptionResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.SubscriptionListFilter{
		Status: strings.ToLower(strings.TrimSpace(req.Status)),
	}
	if req.CustomerID != "" {
		id, err := parseID(req.CustomerID)
		if err != nil {
			return domain.ListSubscriptionResponse{}, domain.ErrInvalidCustomer
		}
		filter.CustomerID = id
	}
	if filter.Status != "" && !isValidStatus(filter.Status) {
		return domain.ListSubscriptionResponse{}, domain.ErrInvalidStatus
	}

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListSubscriptionResponse{}, err
	}

	items, err := s.repo.ListSubscriptions(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListSubscriptionResponse{}, err
	}

	resp := domain.ListSubscriptionResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Subscription) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(items) > pageSize {
			items = items[:pageSize]
		}
	}

	resp.Subscriptions = make([]domain.SubscriptionResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Subscriptions = append(resp.Subscriptions, toSubscriptionResponse(*item))
	}

	return resp, nil
}

func (s *service) listSubscriptionItemsBySubscription(ctx context.Context, orgID, subscriptionID uuid.UUID) ([]domain.SubscriptionItemResponse, error) {
	const chunkSize = 200
	filter := domain.SubscriptionItemListFilter{SubscriptionID: subscriptionID}
	var cursor *domain.ListCursor
	out := make([]domain.SubscriptionItemResponse, 0)

	for {
		items, err := s.repo.ListSubscriptionItems(ctx, orgID, filter, chunkSize+1, cursor)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		hasMore := false
		if len(items) > chunkSize {
			hasMore = true
			items = items[:chunkSize]
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			out = append(out, toSubscriptionItemResponse(*item))
		}
		if !hasMore {
			break
		}
		last := items[len(items)-1]
		if last == nil {
			break
		}
		cursor = &domain.ListCursor{ID: last.ID, CreatedAt: last.CreatedAt}
	}

	return out, nil
}

func (s *service) CreateSubscriptionItem(ctx context.Context, req domain.CreateSubscriptionItemRequest) (domain.SubscriptionItemResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.SubscriptionItemResponse{}, domain.ErrInvalidOrganization
	}

	subID, err := parseID(req.SubscriptionID)
	if err != nil {
		return domain.SubscriptionItemResponse{}, domain.ErrInvalidID
	}
	sub, err := s.repo.FindSubscriptionByID(ctx, orgID, subID)
	if err != nil {
		return domain.SubscriptionItemResponse{}, err
	}
	if sub == nil {
		return domain.SubscriptionItemResponse{}, domain.ErrNotFound
	}
	ctx, err = s.withCustomerTestClock(ctx, orgID, sub.CustomerID)
	if err != nil {
		return domain.SubscriptionItemResponse{}, err
	}

	priceID, err := parseID(req.PlanPriceID)
	if err != nil {
		return domain.SubscriptionItemResponse{}, domain.ErrInvalidPlanPrice
	}
	if req.Quantity < 0 {
		return domain.SubscriptionItemResponse{}, domain.ErrInvalidQuantity
	}

	startAt := s.clock.Now(ctx)
	if req.StartAt != nil {
		startAt = req.StartAt.UTC()
	}
	var endAt *time.Time
	if req.EndAt != nil {
		t := req.EndAt.UTC()
		if t.Before(startAt) {
			return domain.SubscriptionItemResponse{}, domain.ErrInvalidPeriod
		}
		endAt = &t
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindSubscriptionItemByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.SubscriptionItemResponse{}, err
		}
		if existing != nil {
			return toSubscriptionItemResponse(*existing), nil
		}
	}

	now := s.clock.Now(ctx)
	item := domain.SubscriptionItem{
		ID:             uuid.New(),
		OrgID:          orgID,
		SubscriptionID: subID,
		PlanPriceID:    priceID,
		Quantity:       req.Quantity,
		StartAt:        startAt,
		EndAt:          endAt,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if idempotencyKey != "" {
		item.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.CreateSubscriptionItem(ctx, item); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindSubscriptionItemByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.SubscriptionItemResponse{}, findErr
			}
			if existing != nil {
				return toSubscriptionItemResponse(*existing), nil
			}
		}
		return domain.SubscriptionItemResponse{}, err
	}

	resp := toSubscriptionItemResponse(item)
	s.recordAudit(ctx, "subscription_item.create", "subscription_item", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) GetSubscriptionItemByID(ctx context.Context, req domain.GetSubscriptionItemRequest) (domain.SubscriptionItemResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.SubscriptionItemResponse{}, domain.ErrInvalidOrganization
	}

	id, err := parseID(req.ID)
	if err != nil {
		return domain.SubscriptionItemResponse{}, domain.ErrInvalidID
	}

	item, err := s.repo.FindSubscriptionItemByID(ctx, orgID, id)
	if err != nil {
		return domain.SubscriptionItemResponse{}, err
	}
	if item == nil {
		return domain.SubscriptionItemResponse{}, domain.ErrNotFound
	}

	return toSubscriptionItemResponse(*item), nil
}

func (s *service) ListSubscriptionItems(ctx context.Context, req domain.ListSubscriptionItemRequest) (domain.ListSubscriptionItemResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListSubscriptionItemResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.SubscriptionItemListFilter{}
	if req.SubscriptionID != "" {
		id, err := parseID(req.SubscriptionID)
		if err != nil {
			return domain.ListSubscriptionItemResponse{}, domain.ErrInvalidID
		}
		filter.SubscriptionID = id
	}
	if req.PlanPriceID != "" {
		id, err := parseID(req.PlanPriceID)
		if err != nil {
			return domain.ListSubscriptionItemResponse{}, domain.ErrInvalidPlanPrice
		}
		filter.PlanPriceID = id
	}

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListSubscriptionItemResponse{}, err
	}

	items, err := s.repo.ListSubscriptionItems(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListSubscriptionItemResponse{}, err
	}

	resp := domain.ListSubscriptionItemResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.SubscriptionItem) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(items) > pageSize {
			items = items[:pageSize]
		}
	}

	resp.Items = make([]domain.SubscriptionItemResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Items = append(resp.Items, toSubscriptionItemResponse(*item))
	}

	return resp, nil
}

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidID
	}
	return id, nil
}

func (s *service) withCustomerTestClock(ctx context.Context, orgID, customerID uuid.UUID) (context.Context, error) {
	if s.customers == nil {
		return ctx, nil
	}
	customer, err := s.customers.FindByID(ctx, orgID, customerID)
	if err != nil {
		return ctx, err
	}
	if customer == nil {
		return ctx, domain.ErrInvalidCustomer
	}
	if customer.TestClockID == nil {
		return ctx, nil
	}
	if s.clocks == nil {
		return ctx, nil
	}
	testClock, err := s.clocks.GetByID(ctx, orgID, *customer.TestClockID)
	if err != nil {
		return ctx, err
	}
	if testClock == nil || testClock.Status != testclockdomain.StatusActive {
		return ctx, nil
	}
	return clock.WithTestClock(ctx, testClock.ID, testClock.CurrentTime), nil
}

func decodeCursor(token string) (*domain.ListCursor, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil
	}
	decoded, err := pagination.DecodeCursor(raw)
	if err != nil {
		return nil, domain.ErrInvalidID
	}
	if decoded == nil || decoded.ID == "" || decoded.CreatedAt == "" {
		return nil, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
	if err != nil {
		return nil, domain.ErrInvalidID
	}
	parsedID, err := uuid.Parse(decoded.ID)
	if err != nil {
		return nil, domain.ErrInvalidID
	}
	return &domain.ListCursor{
		ID:        parsedID,
		CreatedAt: parsedTime,
	}, nil
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isValidCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func isValidStatus(value string) bool {
	switch value {
	case domain.StatusTrialing, domain.StatusActive, domain.StatusPastDue, domain.StatusCanceled, domain.StatusPaused:
		return true
	default:
		return false
	}
}

func isValidTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case domain.StatusTrialing:
		return to == domain.StatusActive || to == domain.StatusPastDue || to == domain.StatusCanceled || to == domain.StatusPaused
	case domain.StatusActive:
		return to == domain.StatusPastDue || to == domain.StatusCanceled || to == domain.StatusPaused
	case domain.StatusPastDue:
		return to == domain.StatusActive || to == domain.StatusCanceled || to == domain.StatusPaused
	case domain.StatusPaused:
		return to == domain.StatusActive || to == domain.StatusCanceled
	case domain.StatusCanceled:
		return false
	default:
		return false
	}
}

func toSubscriptionResponse(sub domain.Subscription) domain.SubscriptionResponse {
	return domain.SubscriptionResponse{
		ID:                 sub.ID.String(),
		OrgID:              sub.OrgID.String(),
		CustomerID:         sub.CustomerID.String(),
		PlanID:             sub.PlanID.String(),
		Status:             sub.Status,
		Currency:           sub.Currency,
		StartAt:            sub.StartAt,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		TrialEnd:           sub.TrialEnd,
		CancelAt:           sub.CancelAt,
		CanceledAt:         sub.CanceledAt,
		EndedAt:            sub.EndedAt,
		Metadata:           sub.Metadata,
		CreatedAt:          sub.CreatedAt,
		UpdatedAt:          sub.UpdatedAt,
	}
}

func toSubscriptionItemResponse(item domain.SubscriptionItem) domain.SubscriptionItemResponse {
	return domain.SubscriptionItemResponse{
		ID:             item.ID.String(),
		OrgID:          item.OrgID.String(),
		SubscriptionID: item.SubscriptionID.String(),
		PlanPriceID:    item.PlanPriceID.String(),
		Quantity:       item.Quantity,
		StartAt:        item.StartAt,
		EndAt:          item.EndAt,
		Metadata:       item.Metadata,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
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
