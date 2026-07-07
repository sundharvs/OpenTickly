import { type ReactElement, useRef } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";

import { CloseIcon, CopyIcon } from "../../shared/ui/icons.tsx";
import { TimerActionButton } from "../../shared/ui/TimerActionButton.tsx";
import { useDismiss } from "../../shared/ui/useDismiss.ts";
import type { ExternalCalendarEvent } from "./calendar-types.ts";
import { formatClockTime } from "./overview-data.ts";
import type { TimeFormat } from "./overview-data.ts";

export type ExternalEventPopoverPosition = { x: number; y: number };

/**
 * Read-only preview for an external calendar event, opened on click.
 * Rendered via portal at the clicked event's position, matching
 * CalendarEntryContextMenu's positioning approach.
 */
export function ExternalCalendarEventPopover({
  event,
  onClose,
  onCopyAsTimeEntry,
  onStart,
  position,
  timeFormat,
  timezone,
}: {
  event: ExternalCalendarEvent;
  onClose: () => void;
  onCopyAsTimeEntry: () => void;
  onStart: () => void;
  position: ExternalEventPopoverPosition;
  timeFormat: TimeFormat;
  timezone: string;
}): ReactElement {
  const { t } = useTranslation("tracking");
  const popoverRef = useRef<HTMLDivElement>(null);
  useDismiss(popoverRef, true, onClose);

  return createPortal(
    <div
      className="fixed z-[201] w-64 rounded-[8px] border border-[var(--track-border)] bg-[var(--track-surface)] p-3 shadow-[0_4px_16px_var(--track-shadow-popover)]"
      data-testid={`calendar-external-event-popover-${event.id}`}
      ref={popoverRef}
      style={{ left: position.x, top: position.y }}
    >
      <div className="flex items-center gap-1">
        <TimerActionButton
          ariaLabel={t("startTimeEntryAria")}
          isRunning={false}
          onClick={onStart}
          size="xs"
        />
        <button
          aria-label={t("copyAsTimeEntryAria")}
          className="inline-flex size-7 items-center justify-center rounded-[6px] text-[var(--track-text-muted)] transition-colors hover:bg-[var(--track-row-hover)] hover:text-[var(--track-text)]"
          onClick={onCopyAsTimeEntry}
          type="button"
        >
          <CopyIcon className="size-4" />
        </button>
        <button
          aria-label={t("closeAria")}
          className="ml-auto inline-flex size-7 items-center justify-center rounded-[6px] text-[var(--track-text-muted)] transition-colors hover:bg-[var(--track-row-hover)] hover:text-[var(--track-text)]"
          onClick={onClose}
          type="button"
        >
          <CloseIcon className="size-4" />
        </button>
      </div>
      <div className="mt-2 flex items-center justify-between gap-2">
        <span className="truncate text-[14px] font-semibold text-[var(--track-text)]">
          {event.title}
        </span>
        <span className="flex shrink-0 items-center gap-1 text-[12px] text-[var(--track-text-muted)]">
          <span
            aria-hidden
            className="size-2.5 rounded-full"
            style={{ backgroundColor: "var(--track-brand-google-calendar)" }}
          />
          {t("externalEventSourceGoogle")}
        </span>
      </div>
      <div className="mt-0.5 text-[12px] text-[var(--track-text-muted)]">
        {formatClockTime(event.start, timezone, timeFormat)}
        {" – "}
        {formatClockTime(event.end, timezone, timeFormat)}
      </div>
    </div>,
    document.body,
  );
}
