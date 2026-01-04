package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	ghErrors "github.com/github/github-mcp-server/pkg/errors"
	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v74/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MinimalEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Actor     *MinimalUser    `json:"actor"`
	Repo      *MinimalRepo    `json:"repo"`
	CreatedAt string          `json:"created_at"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type MinimalRepo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// GetUserActivity creates a tool to get activity for a GitHub user.
func GetUserActivity(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_user_activity",
			mcp.WithDescription(t("TOOL_GET_USER_ACTIVITY_DESCRIPTION", "Get recent activity for a GitHub user. Returns a list of events performed by the user.")),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        t("TOOL_GET_USER_ACTIVITY_USER_TITLE", "Get user activity"),
				ReadOnlyHint: ToBoolPtr(true),
			}),
			mcp.WithString("username",
				mcp.Required(),
				mcp.Description("The GitHub username to get activity for."),
			),
			WithPagination(),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			username, err := RequiredParam[string](request, "username")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			pagination, err := OptionalPaginationParams(request)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			opts := &github.ListOptions{
				PerPage: pagination.PerPage,
				Page:    pagination.Page,
			}

			client, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitHub client: %w", err)
			}

			events, resp, err := client.Activity.ListEventsPerformedByUser(ctx, username, false, opts)
			if err != nil {
				return ghErrors.NewGitHubAPIErrorResponse(ctx,
					fmt.Sprintf("failed to get activity for user '%s'", username),
					resp,
					err,
				), nil
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to get user activity: %s", string(body))), nil
			}

			minimalEvents := make([]MinimalEvent, 0, len(events))
			for _, event := range events {
				me := MinimalEvent{
					ID:   event.GetID(),
					Type: event.GetType(),
					Actor: &MinimalUser{
						Login:      event.Actor.GetLogin(),
						ID:         event.Actor.GetID(),
						AvatarURL:  event.Actor.GetAvatarURL(),
						ProfileURL: event.Actor.GetURL(),
					},
					Repo: &MinimalRepo{
						ID:   event.Repo.GetID(),
						Name: event.Repo.GetName(),
						URL:  event.Repo.GetURL(),
					},
					CreatedAt: event.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
				}

				// Include payload if available, but keep it raw as it varies by event type
				if event.RawPayload != nil {
					me.Payload = *event.RawPayload
				}

				minimalEvents = append(minimalEvents, me)
			}

			r, err := json.Marshal(minimalEvents)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response: %w", err)
			}

			return mcp.NewToolResultText(string(r)), nil
		}
}
