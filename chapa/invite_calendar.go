package chapa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var scopes = []string{
	calendar.CalendarScope,
}

type CalendarInviteInput struct {
	Summary        string
	Description    string
	AttendeeEmails []string
	StartAt        string // RFC3339 format
	EndAt          string // RFC3339 format
}

type CalendarInviteResult struct {
	EventID  string
	HTMLLink string
	MeetLink string
}

type CalendarInviteService interface {
	CreateInvite(input CalendarInviteInput) (*CalendarInviteResult, error)
}

type GoogleCalendarInviteService struct {
	CalendarID string
	TokenPath  string
}

func NewGoogleCalendarInviteService() *GoogleCalendarInviteService {
	return &GoogleCalendarInviteService{
		CalendarID: "natibir400@gmail.com",
		TokenPath:  "chapa/oauth-token.json",
	}
}

func (s *GoogleCalendarInviteService) CreateInvite(
	input CalendarInviteInput,
) (*CalendarInviteResult, error) {

	ctx := context.Background()

	client, err := s.getOAuthClient(ctx)
	if err != nil {
		return nil, err
	}

	calendarService, err := calendar.NewService(
		ctx,
		option.WithHTTPClient(client),
	)

	if err != nil {
		return nil, err
	}

	var attendees []*calendar.EventAttendee

	for _, email := range input.AttendeeEmails {

		email = strings.TrimSpace(email)

		if email != "" {
			attendees = append(attendees, &calendar.EventAttendee{
				Email: email,
			})
		}
	}

	if len(attendees) == 0 {
		return nil, fmt.Errorf("at least one attendee email is required")
	}

	event := &calendar.Event{
		Summary:     input.Summary,
		Description: input.Description,

		Start: &calendar.EventDateTime{
			DateTime: input.StartAt,
			TimeZone: "Africa/Addis_Ababa",
		},

		End: &calendar.EventDateTime{
			DateTime: input.EndAt,
			TimeZone: "Africa/Addis_Ababa",
		},

		Attendees: attendees,

		ConferenceData: &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId: fmt.Sprintf(
					"meet-%d",
					time.Now().UnixNano(),
				),

				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{
					Type: "hangoutsMeet",
				},
			},
		},
	}

	createdEvent, err := calendarService.Events.
		Insert(s.CalendarID, event).
		ConferenceDataVersion(1).
		SendUpdates("all").
		Do()

	if err != nil {
		return nil, err
	}

	return &CalendarInviteResult{
		EventID:  createdEvent.Id,
		HTMLLink: createdEvent.HtmlLink,
		MeetLink: createdEvent.HangoutLink,
	}, nil
}

func (s *GoogleCalendarInviteService) getOAuthClient(
	ctx context.Context,
) (*http.Client, error) {

	clientID := getRequiredEnv("GOOGLE_OAUTH_CLIENT_ID")

	clientSecret := getRequiredEnv(
		"GOOGLE_OAUTH_CLIENT_SECRET",
	)

	redirectURI := getEnv(
		"GOOGLE_OAUTH_REDIRECT_URI",
		"http://localhost",
	)

	tokenRaw, err := os.ReadFile(s.TokenPath)

	if err != nil {
		return nil, fmt.Errorf(
			"oauth token file not found: %w",
			err,
		)
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	token := &oauth2.Token{}

	if err := json.Unmarshal(tokenRaw, token); err != nil {
		return nil, err
	}

	return config.Client(ctx, token), nil
}

func getEnv(key string, fallback string) string {

	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return fallback
	}

	return value
}
