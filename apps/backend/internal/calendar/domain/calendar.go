// Package domain models a user's connected external calendar integrations
// (Google Calendar today) and the calendars OpenTickly imports events from.
//
// Scope is intentionally one-way: OpenTickly reads events from selected
// calendars and lets the user manually turn one into a time entry. It never
// writes time entries or anything else back to the provider. Disconnecting an
// integration soft-deletes it so any time entries already created from its
// events keep their history.
package domain

import (
	"errors"
	"time"
)

var (
	ErrIntegrationNotFound = errors.New("calendar integration not found")
	ErrCalendarNotFound    = errors.New("calendar not found")
	ErrNotConfigured       = errors.New("calendar provider is not configured")
	ErrUnsupportedProvider = errors.New("unsupported calendar provider")
	ErrInvalidState        = errors.New("invalid or expired oauth state")
)

const ProviderGoogle = "google"

// Integration is a user's OAuth connection to a calendar provider account.
type Integration struct {
	ID             int64
	UserID         int64
	Provider       string
	Email          string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
	HasWriteScope  bool
	ErrorStatus    string
	Scopes         []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Calendar is one calendar under an integration (e.g. the account's primary
// calendar, or a secondary calendar it has access to). Selected controls
// whether events are imported from it.
type Calendar struct {
	ID              int64
	IntegrationID   int64
	ExternalID      string
	Name            string
	BackgroundColor string
	ForegroundColor string
	Selected        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SelectedCalendar pairs a selected Calendar with enough of its parent
// Integration to call the provider on the user's behalf.
type SelectedCalendar struct {
	Calendar       Calendar
	IntegrationID  int64
	Provider       string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
}

// RemoteCalendar is a calendar as reported by the provider, before it has
// been reconciled against locally stored selection state.
type RemoteCalendar struct {
	ExternalID      string
	Name            string
	BackgroundColor string
	ForegroundColor string
	Primary         bool
}

// Event is a single event read from a selected calendar. It is never
// persisted — ListEvents fetches live from the provider on each call.
type Event struct {
	CalendarID int64
	ExternalID string
	Title      string
	Provider   string
	StartTime  time.Time
	EndTime    time.Time
	AllDay     bool
	HTMLLink   string
	ICalUID    string
}
