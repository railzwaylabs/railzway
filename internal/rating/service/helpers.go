package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/bwmarrin/snowflake"
)

func resolveEffectiveWindow(
	cycleStart, cycleEnd time.Time,
	subStartAt time.Time,
	subEndedAt, subCanceledAt *time.Time,
	entEffectiveFrom time.Time,
	entEffectiveTo *time.Time,
) (time.Time, time.Time, bool) {
	start := cycleStart
	if subStartAt.After(start) {
		start = subStartAt
	}
	if !entEffectiveFrom.IsZero() && entEffectiveFrom.After(start) {
		start = entEffectiveFrom
	}

	end := cycleEnd
	if subEndedAt != nil && subEndedAt.Before(end) {
		end = *subEndedAt
	}
	if subCanceledAt != nil && subCanceledAt.Before(end) {
		end = *subCanceledAt
	}
	if entEffectiveTo != nil && entEffectiveTo.Before(end) {
		end = *entEffectiveTo
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, false
	}

	return start, end, true
}

func calculateProrationFactor(start, end time.Time, cycleDurationSeconds float64) float64 {
	if cycleDurationSeconds <= 0 {
		return 0
	}
	activeSeconds := end.Sub(start).Seconds()
	factor := activeSeconds / cycleDurationSeconds
	if factor > 1.0 {
		return 1.0
	}
	if factor < 0.0 {
		return 0.0
	}
	return factor
}

func buildRatingChecksum(
	billingCycleID snowflake.ID,
	subscriptionID snowflake.ID,
	priceID snowflake.ID,
	meterID *snowflake.ID,
	featureCode string,
	periodStart, periodEnd time.Time,
) string {
	meterPart := "flat"
	if meterID != nil && *meterID != 0 {
		meterPart = meterID.String()
	}

	payload := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|%s",
		billingCycleID.String(),
		subscriptionID.String(),
		meterPart,
		priceID.String(),
		featureCode,
		periodStart.UTC().Format(time.RFC3339Nano),
		periodEnd.UTC().Format(time.RFC3339Nano),
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func roundRatingAmount(raw float64) int64 {
	return int64(math.Floor(raw + 0.5))
}
