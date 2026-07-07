/* @vitest-environment jsdom */
import { describe, expect, it } from "vitest";

import { buildExternalEvents } from "./calendar-events-builder.ts";
import type { ExternalCalendarEventInput } from "./external-calendar-types.ts";

describe("buildExternalEvents", () => {
  it("maps external calendar inputs onto the external resource lane", () => {
    const input: ExternalCalendarEventInput[] = [
      {
        id: "breakfast-1",
        title: "Breakfast",
        start: new Date("2026-04-09T07:30:00.000Z"),
        end: new Date("2026-04-09T08:00:00.000Z"),
      },
    ];

    const [event] = buildExternalEvents(input);

    expect(event).toMatchObject({
      resource: { kind: "external" },
      resourceId: "external",
      title: "Breakfast",
    });
    expect(event!.start).toEqual(input[0]!.start);
    expect(event!.end).toEqual(input[0]!.end);
  });

  it("splits an external event that crosses midnight into per-day segments", () => {
    const input: ExternalCalendarEventInput[] = [
      {
        id: "overnight-1",
        title: "Overnight flight",
        start: new Date(2026, 3, 9, 22, 0),
        end: new Date(2026, 3, 10, 2, 0),
      },
    ];

    const events = buildExternalEvents(input);

    expect(events).toHaveLength(2);
    expect(events[0]!.start).toEqual(input[0]!.start);
    expect(events[1]!.end).toEqual(input[0]!.end);
    expect(new Set(events.map((e) => e.id)).size).toBe(2);
  });

  it("produces unique ids across events sharing the same source id", () => {
    const input: ExternalCalendarEventInput[] = [
      {
        id: "recurring-1",
        title: "Breakfast",
        start: new Date("2026-04-09T07:30:00.000Z"),
        end: new Date("2026-04-09T08:00:00.000Z"),
      },
      {
        id: "recurring-1",
        title: "Breakfast",
        start: new Date("2026-04-10T07:30:00.000Z"),
        end: new Date("2026-04-10T08:00:00.000Z"),
      },
    ];

    const events = buildExternalEvents(input);

    expect(new Set(events.map((e) => e.id)).size).toBe(2);
  });
});
