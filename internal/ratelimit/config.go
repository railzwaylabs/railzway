package ratelimit

type Config struct {
	WindowSec                     int
	UsageEventsSubscriptionPerMin int
	UsageEventsCustomerPerMin     int
	UsageEventsOrgPerMin          int
	UsageEventsConcurrencyPerCustomerMeter int
	UsageEventsConcurrencyTTLSeconds       int
}
