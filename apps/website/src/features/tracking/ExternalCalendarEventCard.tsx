import { memo } from "react";

import type { ExternalCalendarEvent } from "./calendar-types.ts";

// Mirrors CalendarEventCard's memo boundary: react-big-calendar re-invokes
// components.event(props) multiple times per interaction with fresh element
// references, so a memo boundary avoids redundant re-renders.
export const ExternalCalendarEventCard = memo(ExternalCalendarEventCardImpl);

function ExternalCalendarEventCardImpl({ event }: { event: ExternalCalendarEvent }) {
  return (
    <div
      className="flex h-full flex-col justify-start overflow-hidden rounded-[4px] px-1.5 py-1 text-left text-[12px] text-[var(--track-text)]"
      data-testid={`calendar-external-event-${event.id}`}
      style={{ backgroundColor: "var(--track-external-event-bg)" }}
    >
      <span className="truncate font-semibold leading-tight">{event.title}</span>
    </div>
  );
}
