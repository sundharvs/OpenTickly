// Package calendar implements external calendar integration (Google Calendar
// today). It is deliberately scoped to one-way import, matching the product
// intent that was documented in docs/product/tracking.md § Calendar
// Integrations before that section — and later the entire docs/ tree — was
// deleted from this repository (commits b40f2466, 486b9eaf). Recovered here
// since it no longer has a home in docs/:
//
//   - Sync direction is calendar -> OpenTickly only. OpenTickly never writes
//     time entries, availability, or anything else back to the provider.
//   - Conflict rule: a manually-edited time entry always wins over a later
//     calendar overwrite. Since this integration never re-syncs a time entry
//     once created, this is satisfied by construction rather than by an
//     explicit merge rule.
//   - Disconnecting an integration (soft delete) must not delete time entries
//     already created from its events. Historical time entries are ordinary
//     rows with no foreign key to the calendar integration, so removing the
//     integration cannot cascade into them.
//
// Deviations from the upstream Toggl Track contract this package's HTTP
// handlers satisfy (see openapi/toggl-track-api-v9.swagger.json, tag
// "calendar" — read-only, upstream-owned):
//
//   - Upstream's GetEvents only serves events "fetched from provider before
//     the request is made" via a separate POST .../events/update background
//     step. This deployment has no job scheduler wired up for that, so
//     ListEvents calls the provider live within the request instead.
//   - Only the minimal endpoint slice needed by the Profile "External
//     calendars" section and the timer calendar's external-event overlay is
//     implemented: connect/disconnect, list integrations, list calendars +
//     toggle selection, list events for a date range. Attendees, details
//     suggestions, the deprecated per-calendar events route, calendar
//     defaults (PatchCalendar's default_project_id etc., which exist upstream
//     to support auto-created time entries), and the Outlook provider are
//     out of scope — see the remaining stubs in
//     apps/backend/internal/bootstrap/public_track_openapi_unimplemented.go.
package calendar
