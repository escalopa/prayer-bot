package domain

import "time"

// ResolvedLocation is what a location resolver derives from raw coordinates.
// The formatted street address is deliberately absent: only the timezone,
// place identifier, city label, and country ever cross the port.
type ResolvedLocation struct {
	Timezone    string
	PlaceID     string
	City        string
	CountryCode string
}

// MetricCount is one keyed counter in the owner dashboard.
type MetricCount struct {
	Key   string
	Count int64
}

// AdminDashboard aggregates the owner dashboard metrics.
type AdminDashboard struct {
	Users                   int64
	Groups                  int64
	ConfiguredUsers         int64
	NewUsers24Hours         int64
	NewUsers7Days           int64
	NewUsers30Days          int64
	ActiveUsers24Hours      int64
	ActiveUsers7Days        int64
	ActiveUsers30Days       int64
	ReminderUsers           int64
	EnabledRules            int64
	PendingSchedules        int64
	QueuedTasks             int64
	SentDeliveries24Hours   int64
	FailedDeliveries24Hours int64
	StaleDeliveries24Hours  int64
	ProcessingDeliveries    int64
	FailedUpdates24Hours    int64
	Languages               []MetricCount
	Methods                 []MetricCount
	ReminderKinds           []MetricCount
}

// OutboxItem is one pending transactional-outbox row awaiting Cloud Tasks
// enqueueing.
type OutboxItem struct {
	ID          int64
	DeliveryKey string
	Endpoint    string
	RunAt       time.Time
	Payload     []byte
}
