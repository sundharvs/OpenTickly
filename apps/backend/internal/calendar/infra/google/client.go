package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"opentoggl/backend/apps/backend/internal/calendar/domain"
)

const (
	calendarListURL = "https://www.googleapis.com/calendar/v3/users/me/calendarList"
	calendarBaseURL = "https://www.googleapis.com/calendar/v3/calendars/"
)

// Client calls the Google Calendar REST API with an already-valid access
// token; callers are responsible for refreshing an expired token first.
type Client struct {
	HTTPClient *http.Client
}

func (client Client) httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return http.DefaultClient
}

func (client Client) get(ctx context.Context, accessToken string, requestURL string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build google calendar request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := client.httpClient().Do(request)
	if err != nil {
		return fmt.Errorf("call google calendar api: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read google calendar api response: %w", err)
	}
	if response.StatusCode == http.StatusUnauthorized {
		return domain.ErrInvalidState
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("google calendar api returned %d: %s", response.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode google calendar api response: %w", err)
	}
	return nil
}

// ListCalendars lists every calendar the authenticated account can see.
func (client Client) ListCalendars(ctx context.Context, accessToken string) ([]domain.RemoteCalendar, error) {
	var payload struct {
		Items []struct {
			ID              string `json:"id"`
			Summary         string `json:"summary"`
			BackgroundColor string `json:"backgroundColor"`
			ForegroundColor string `json:"foregroundColor"`
			Primary         bool   `json:"primary"`
		} `json:"items"`
	}
	if err := client.get(ctx, accessToken, calendarListURL, &payload); err != nil {
		return nil, fmt.Errorf("list google calendars: %w", err)
	}

	calendars := make([]domain.RemoteCalendar, 0, len(payload.Items))
	for _, item := range payload.Items {
		calendars = append(calendars, domain.RemoteCalendar{
			ExternalID:      item.ID,
			Name:            item.Summary,
			BackgroundColor: item.BackgroundColor,
			ForegroundColor: item.ForegroundColor,
			Primary:         item.Primary,
		})
	}
	return calendars, nil
}

// ListEvents lists events on one calendar within [start, end), expanding
// recurring events into their individual instances.
func (client Client) ListEvents(ctx context.Context, accessToken string, calendarExternalID string, start, end time.Time) ([]domain.Event, error) {
	values := url.Values{}
	values.Set("timeMin", start.UTC().Format(time.RFC3339))
	values.Set("timeMax", end.UTC().Format(time.RFC3339))
	values.Set("singleEvents", "true")
	values.Set("orderBy", "startTime")
	requestURL := calendarBaseURL + url.PathEscape(calendarExternalID) + "/events?" + values.Encode()

	var payload struct {
		Items []struct {
			ID       string `json:"id"`
			Status   string `json:"status"`
			Summary  string `json:"summary"`
			ICalUID  string `json:"iCalUID"`
			HTMLLink string `json:"htmlLink"`
			Start    struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"end"`
		} `json:"items"`
	}
	if err := client.get(ctx, accessToken, requestURL, &payload); err != nil {
		return nil, fmt.Errorf("list google events for calendar %s: %w", calendarExternalID, err)
	}

	events := make([]domain.Event, 0, len(payload.Items))
	for _, item := range payload.Items {
		if item.Status == "cancelled" {
			continue
		}
		allDay := item.Start.DateTime == "" && item.Start.Date != ""
		startTime, endTime, ok := parseEventBounds(item.Start.DateTime, item.Start.Date, item.End.DateTime, item.End.Date)
		if !ok {
			continue
		}
		events = append(events, domain.Event{
			ExternalID: item.ID,
			Title:      item.Summary,
			Provider:   domain.ProviderGoogle,
			StartTime:  startTime,
			EndTime:    endTime,
			AllDay:     allDay,
			HTMLLink:   item.HTMLLink,
			ICalUID:    item.ICalUID,
		})
	}
	return events, nil
}

func parseEventBounds(startDateTime, startDate, endDateTime, endDate string) (time.Time, time.Time, bool) {
	start, ok := parseEventTime(startDateTime, startDate)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	end, ok := parseEventTime(endDateTime, endDate)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func parseEventTime(dateTime, date string) (time.Time, bool) {
	if dateTime != "" {
		parsed, err := time.Parse(time.RFC3339, dateTime)
		if err != nil {
			return time.Time{}, false
		}
		return parsed, true
	}
	if date != "" {
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil {
			return time.Time{}, false
		}
		return parsed, true
	}
	return time.Time{}, false
}
