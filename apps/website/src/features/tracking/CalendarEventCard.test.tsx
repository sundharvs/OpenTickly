/* @vitest-environment jsdom */
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { GithubComTogglTogglApiInternalModelsTimeEntry } from "../../shared/api/generated/public-track/types.gen.ts";
import type { CalendarEvent } from "./calendar-types.ts";
import { CalendarEventCard } from "./CalendarEventCard.tsx";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
  initReactI18next: { type: "3rdParty", init: () => undefined },
}));

vi.mock("../../shared/query/useUserPreferences.ts", () => ({
  useUserPreferences: () => ({ durationFormat: "improved" }),
}));

afterEach(cleanup);

const DRAFT_ENTRY_ID = -1;

function makeDraftEvent(start: string, stop: string): CalendarEvent {
  const entry: GithubComTogglTogglApiInternalModelsTimeEntry & { id: number } = {
    id: DRAFT_ENTRY_ID,
    workspace_id: 1,
    description: "",
    start,
    stop,
    duration: 1800,
    billable: false,
    tag_ids: [],
  };
  return {
    allDay: false,
    end: new Date(stop),
    entry,
    id: DRAFT_ENTRY_ID,
    resource: { color: "#000000", isDraft: true, isLocked: false, isRunning: false },
    start: new Date(start),
    title: "",
  };
}

describe("CalendarEventCard draft auto-open", () => {
  it("re-opens the editor for a second drag even when the component instance is reused", () => {
    const onEditEntry = vi.fn();

    const firstDraft = makeDraftEvent("2026-03-10T09:00:00Z", "2026-03-10T09:30:00Z");
    const { rerender } = render(<CalendarEventCard event={firstDraft} onEditEntry={onEditEntry} />);
    expect(onEditEntry).toHaveBeenCalledTimes(1);
    expect(onEditEntry).toHaveBeenLastCalledWith(firstDraft.entry, expect.anything());

    const secondDraft = makeDraftEvent("2026-03-10T14:00:00Z", "2026-03-10T14:30:00Z");
    rerender(<CalendarEventCard event={secondDraft} onEditEntry={onEditEntry} />);

    expect(onEditEntry).toHaveBeenCalledTimes(2);
    expect(onEditEntry).toHaveBeenLastCalledWith(secondDraft.entry, expect.anything());
  });
});
