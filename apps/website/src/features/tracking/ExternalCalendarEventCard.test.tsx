/* @vitest-environment jsdom */
import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";

import { ExternalCalendarEventCard } from "./ExternalCalendarEventCard.tsx";
import type { ExternalCalendarEvent } from "./calendar-types.ts";

function makeEvent(overrides: Partial<ExternalCalendarEvent> = {}): ExternalCalendarEvent {
  return {
    allDay: false,
    end: new Date(2026, 3, 9, 8, 0),
    id: "breakfast-1",
    resource: { kind: "external" },
    resourceId: "external",
    start: new Date(2026, 3, 9, 7, 30),
    title: "Breakfast",
    ...overrides,
  };
}

describe("ExternalCalendarEventCard", () => {
  test("renders only the title, with no duration or play button", () => {
    render(<ExternalCalendarEventCard event={makeEvent()} />);

    expect(screen.getByText("Breakfast")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.queryByText(/\d+:\d{2}/)).not.toBeInTheDocument();
  });

  test("is tagged with a stable per-event test id", () => {
    render(<ExternalCalendarEventCard event={makeEvent({ id: "commute-2" })} />);

    expect(screen.getByTestId("calendar-external-event-commute-2")).toBeInTheDocument();
  });
});
