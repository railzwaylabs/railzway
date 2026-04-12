package proration

import "time"

// EffectiveWindow calculates the active window within a billing period.
// It intersects the billing period with subscription and item lifetimes.
func EffectiveWindow(
	periodStart, periodEnd time.Time,
	subStart time.Time,
	subEnd, subCancel *time.Time,
	itemStart time.Time,
	itemEnd *time.Time,
) (time.Time, time.Time, bool) {
	start := periodStart
	if subStart.After(start) {
		start = subStart
	}
	if itemStart.After(start) {
		start = itemStart
	}

	end := periodEnd
	if subEnd != nil && subEnd.Before(end) {
		end = *subEnd
	}
	if subCancel != nil && subCancel.Before(end) {
		end = *subCancel
	}
	if itemEnd != nil && itemEnd.Before(end) {
		end = *itemEnd
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// Factor returns the proration factor for an active window within a period.
func Factor(periodStart, periodEnd, activeStart, activeEnd time.Time) float64 {
	periodSeconds := periodEnd.Sub(periodStart).Seconds()
	if periodSeconds <= 0 {
		return 0
	}
	activeSeconds := activeEnd.Sub(activeStart).Seconds()
	if activeSeconds <= 0 {
		return 0
	}
	factor := activeSeconds / periodSeconds
	if factor > 1 {
		return 1
	}
	if factor < 0 {
		return 0
	}
	return factor
}
