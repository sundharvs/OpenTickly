/**
 * TEMPORARY placeholder data for the external-calendar-events visual UI.
 *
 * Google Calendar (and other provider) integration is not implemented yet —
 * the backend endpoints are intentionally stubbed out, see
 * apps/backend/internal/bootstrap/public_track_openapi_unimplemented.go:67-128.
 * Delete this file and wire a real query (e.g. useExternalCalendarEventsQuery)
 * once that lands; ConnectedCalendarView.tsx is the only call site to update.
 */
export type ExternalCalendarEventInput = {
  id: string;
  title: string;
  start: Date;
  end: Date;
};

function atHour(day: Date, hour: number, minute: number): Date {
  const d = new Date(day);
  d.setHours(hour, minute, 0, 0);
  return d;
}

/** Builds a week of recurring mock external events anchored to `weekDays[0]`'s week. */
export function buildMockExternalCalendarEvents(weekDays: Date[]): ExternalCalendarEventInput[] {
  return weekDays.flatMap((day, index) => [
    {
      id: `mock-breakfast-${index}`,
      title: "Breakfast",
      start: atHour(day, 7, 30),
      end: atHour(day, 8, 0),
    },
    {
      id: `mock-commute-${index}`,
      title: "Commute",
      start: atHour(day, 9, 30),
      end: atHour(day, 10, 30),
    },
  ]);
}
