/* @vitest-environment jsdom */
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";

import { ExternalCalendarEventPopover } from "./ExternalCalendarEventPopover.tsx";
import type { ExternalCalendarEvent } from "./calendar-types.ts";

function makeEvent(overrides: Partial<ExternalCalendarEvent> = {}): ExternalCalendarEvent {
  return {
    allDay: false,
    end: new Date(Date.UTC(2026, 3, 9, 8, 0)),
    id: "breakfast-1",
    resource: { kind: "external" },
    resourceId: "external",
    start: new Date(Date.UTC(2026, 3, 9, 7, 30)),
    title: "Breakfast",
    ...overrides,
  };
}

describe("ExternalCalendarEventPopover", () => {
  test("renders the event title, source badge, and time range", () => {
    render(
      <ExternalCalendarEventPopover
        event={makeEvent()}
        onClose={vi.fn()}
        onCopyAsTimeEntry={vi.fn()}
        onStart={vi.fn()}
        position={{ x: 0, y: 0 }}
        timeFormat="h:mm A"
        timezone="UTC"
      />,
    );

    expect(screen.getByText("Breakfast")).toBeInTheDocument();
    expect(screen.getByText("Google")).toBeInTheDocument();
    expect(screen.getByText(/7:30 AM/)).toBeInTheDocument();
    expect(screen.getByText(/8:00 AM/)).toBeInTheDocument();
  });

  test("clicking start calls onStart and clicking copy calls onCopyAsTimeEntry", () => {
    const onStart = vi.fn();
    const onCopyAsTimeEntry = vi.fn();

    render(
      <ExternalCalendarEventPopover
        event={makeEvent()}
        onClose={vi.fn()}
        onCopyAsTimeEntry={onCopyAsTimeEntry}
        onStart={onStart}
        position={{ x: 0, y: 0 }}
        timeFormat="h:mm A"
        timezone="UTC"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Start Time Entry" }));
    expect(onStart).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "Copy as Time Entry" }));
    expect(onCopyAsTimeEntry).toHaveBeenCalledTimes(1);
  });

  test("clicking outside the popover calls onClose", () => {
    const onClose = vi.fn();

    render(
      <ExternalCalendarEventPopover
        event={makeEvent()}
        onClose={onClose}
        onCopyAsTimeEntry={vi.fn()}
        onStart={vi.fn()}
        position={{ x: 0, y: 0 }}
        timeFormat="h:mm A"
        timezone="UTC"
      />,
    );

    fireEvent.mouseDown(document.body);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
