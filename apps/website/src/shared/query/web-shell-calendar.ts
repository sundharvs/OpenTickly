import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ExternalCalendarEventInput } from "../../features/tracking/external-calendar-types.ts";
import { unwrapWebApiResult } from "../api/web-client.ts";
import {
  deleteIntegrationsCalendarByIntegrationId,
  getIntegrationsCalendar,
  getIntegrationsCalendarByIntegrationIdCalendars,
  getIntegrationsCalendarEvents,
  patchIntegrationsCalendarByIntegrationIdCalendarsByCalendarId,
} from "../api/public/track/index.ts";

const calendarIntegrationsQueryKey = ["calendar-integrations"] as const;
const calendarCalendarsQueryKey = (integrationId: number) =>
  ["calendar-integrations", integrationId, "calendars"] as const;

export function useCalendarIntegrationsQuery() {
  return useQuery({
    queryFn: () => unwrapWebApiResult(getIntegrationsCalendar()),
    queryKey: calendarIntegrationsQueryKey,
  });
}

export function useCalendarCalendarsQuery(integrationId: number) {
  return useQuery({
    queryFn: () =>
      unwrapWebApiResult(
        getIntegrationsCalendarByIntegrationIdCalendars({
          path: { integration_id: integrationId },
        }),
      ),
    queryKey: calendarCalendarsQueryKey(integrationId),
  });
}

export function useToggleCalendarSelectionMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: { calendarId: number; integrationId: number; selected: boolean }) =>
      unwrapWebApiResult(
        patchIntegrationsCalendarByIntegrationIdCalendarsByCalendarId({
          body: { selected: request.selected },
          path: { calendar_id: request.calendarId, integration_id: request.integrationId },
        }),
      ),
    onSuccess: (_data, request) => {
      void queryClient.invalidateQueries({
        queryKey: calendarCalendarsQueryKey(request.integrationId),
      });
    },
  });
}

export function useDisconnectCalendarIntegrationMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (integrationId: number) =>
      unwrapWebApiResult(
        deleteIntegrationsCalendarByIntegrationId({ path: { integration_id: integrationId } }),
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: calendarIntegrationsQueryKey });
    },
  });
}

export function useExternalCalendarEventsQuery(startDateIso: string, endDateIso: string) {
  return useQuery({
    queryFn: async (): Promise<ExternalCalendarEventInput[]> => {
      const response = await unwrapWebApiResult(
        getIntegrationsCalendarEvents({
          query: { start_date: startDateIso, end_date: endDateIso },
        }),
      );
      return (response.events ?? []).flatMap((event) => {
        if (event.all_day) {
          return [];
        }
        if (!event.external_id || !event.title || !event.start_time || !event.end_time) {
          return [];
        }
        return [
          {
            end: new Date(event.end_time),
            id: event.external_id,
            start: new Date(event.start_time),
            title: event.title,
          },
        ];
      });
    },
    queryKey: ["calendar-events", startDateIso, endDateIso],
  });
}

/** Navigates the browser to the OAuth connect endpoint for a provider (a real
 * redirect, not an XHR) — mirrors the plain "Connect" link on track.toggl.com. */
export function connectCalendarProvider(
  baseUrl: string,
  provider: "google",
  returnToPath: string,
): void {
  const url = new URL(
    `${baseUrl || globalThis.location.origin}/api/v9/integrations/calendar/setup`,
  );
  url.searchParams.set("provider", provider);
  url.searchParams.set("return_to", returnToPath);
  globalThis.location.href = url.toString();
}
