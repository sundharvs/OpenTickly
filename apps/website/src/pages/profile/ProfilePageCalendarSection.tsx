import { AppButton, AppSwitch } from "@opentickly/web-ui";
import { type ReactElement, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  connectCalendarProvider,
  useCalendarCalendarsQuery,
  useCalendarIntegrationsQuery,
  useDisconnectCalendarIntegrationMutation,
  useToggleCalendarSelectionMutation,
} from "../../shared/query/web-shell.ts";
import { PreferenceCard } from "./ProfilePagePrimitives.tsx";

export function ExternalCalendarsSection(props: {
  returnToPath: string;
  siteUrl: string;
}): ReactElement {
  const { t } = useTranslation("profile");
  const integrationsQuery = useCalendarIntegrationsQuery();

  return (
    <PreferenceCard
      description={t("externalCalendarsDescription")}
      id="external-calendars"
      title={t("externalCalendars")}
    >
      <div className="space-y-4 px-5 py-[15px]">
        <div className="flex flex-wrap gap-3">
          <ProviderConnectButton
            baseUrl={props.siteUrl}
            provider="google"
            returnToPath={props.returnToPath}
          />
        </div>

        {(integrationsQuery.data ?? []).map((integration) => (
          <IntegrationRow
            integrationId={integration.calendar_integration_id ?? 0}
            key={integration.calendar_integration_id}
            label={integration.email || integration.provider || ""}
          />
        ))}
      </div>
    </PreferenceCard>
  );
}

function ProviderConnectButton(props: {
  baseUrl: string;
  provider: "google";
  returnToPath: string;
}): ReactElement {
  const { t } = useTranslation("profile");
  const [connecting, setConnecting] = useState(false);

  return (
    <div className="min-w-[220px] rounded-[8px] border border-[var(--track-border)] px-4 py-3">
      <p className="text-[11px] font-semibold uppercase tracking-[0.4px] text-[var(--track-text-soft)]">
        {t("googleCalendar")}
      </p>
      <button
        className="text-[14px] font-medium text-[var(--track-accent-text)] underline disabled:opacity-60"
        disabled={connecting}
        onClick={() => {
          setConnecting(true);
          connectCalendarProvider(props.baseUrl, props.provider, props.returnToPath);
        }}
        type="button"
      >
        {connecting ? t("connecting") : t("connect")}
      </button>
    </div>
  );
}

function IntegrationRow(props: { integrationId: number; label: string }): ReactElement {
  const { t } = useTranslation("profile");
  const [showUnselected, setShowUnselected] = useState(false);
  const calendarsQuery = useCalendarCalendarsQuery(props.integrationId);
  const toggleSelectionMutation = useToggleCalendarSelectionMutation();
  const disconnectMutation = useDisconnectCalendarIntegrationMutation();

  const calendars = calendarsQuery.data?.calendars ?? [];
  const visibleCalendars = showUnselected
    ? calendars
    : calendars.filter((calendar) => calendar.selected);

  return (
    <div className="overflow-hidden rounded-[8px] border border-[var(--track-border)]">
      <div className="flex items-center justify-between border-b border-[var(--track-border)] px-4 py-3">
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.4px] text-[var(--track-text-soft)]">
            {t("googleCalendar")}
          </p>
          <p className="text-[14px] font-medium text-white">{props.label}</p>
        </div>
        <AppButton
          disabled={disconnectMutation.isPending}
          onClick={() => {
            disconnectMutation.mutate(props.integrationId);
          }}
          type="button"
        >
          {disconnectMutation.isPending ? t("removing") : t("remove")}
        </AppButton>
      </div>

      {calendars.length === 0 ? (
        <p className="px-4 py-3 text-[14px] text-[var(--track-text-muted)]">
          {t("noCalendarsFound")}
        </p>
      ) : (
        <>
          {visibleCalendars.map((calendar) => (
            <div
              className="flex items-center gap-3 border-b border-[var(--track-border)] px-4 py-2 last:border-b-0"
              key={calendar.calendar_id}
            >
              <AppSwitch
                aria-label={calendar.name ?? ""}
                checked={Boolean(calendar.selected)}
                onChange={(selected) => {
                  toggleSelectionMutation.mutate({
                    calendarId: calendar.calendar_id ?? 0,
                    integrationId: props.integrationId,
                    selected,
                  });
                }}
                size="sm"
              />
              <span className="text-[14px] text-[var(--track-text)]">{calendar.name}</span>
            </div>
          ))}
          <button
            className="w-full px-4 py-2 text-left text-[13px] font-semibold text-[var(--track-accent-text)]"
            onClick={() => {
              setShowUnselected((current) => !current);
            }}
            type="button"
          >
            {showUnselected ? t("hideUnselectedCalendars") : t("showUnselectedCalendars")}
          </button>
        </>
      )}
    </div>
  );
}
