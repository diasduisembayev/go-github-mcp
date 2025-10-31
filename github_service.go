package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v62/github"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var prURLRegex = regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)

type githubService struct {
	restClient    *github.Client
	graphqlClient *githubv4.Client
}

func newGithubService() (*githubService, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	authorizedClient := oauth2.NewClient(ctx, tokenSource)

	githubClient := github.NewClient(authorizedClient)
	graphqlClient := githubv4.NewClient(authorizedClient)

	if err := validateCredentials(ctx, githubClient); err != nil {
		return nil, fmt.Errorf("GitHub authentication failed: %v", err)
	}

	return &githubService{
		restClient:    githubClient,
		graphqlClient: graphqlClient,
	}, nil
}

func validateCredentials(ctx context.Context, client *github.Client) error {
	_, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("authentication test failed: %v", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected API response: %s", resp.Status)
	}

	return nil
}

func (s *githubService) listPullRequestsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	state := req.GetString("state", "open")
	var queryParts []string
	queryParts = append(queryParts, "is:pr", "author:@me")

	if state == "open" || state == "closed" {
		queryParts = append(queryParts, fmt.Sprintf("is:%s", state))
	}

	query := strings.Join(queryParts, " ")

	opts := &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 15,
		},
	}

	result, resp, err := s.restClient.Search.Issues(ctx, query, opts)
	if err != nil {
		log.Printf("Error searching GitHub: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error searching GitHub: %v", err)), nil
	}

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("GitHub API returned non-200 status: %s", resp.Status)), nil
	}

	if result.GetTotal() == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No pull requests found with state: %s", state)), nil
	}

	log.Printf("Found %d PRs.", result.GetTotal())
	var responseBuilder strings.Builder
	responseBuilder.WriteString(fmt.Sprintf("Found %d pull requests (state: %s):\n\n", result.GetTotal(), state))

	for _, issue := range result.Issues {
		responseBuilder.WriteString(fmt.Sprintf("- [State: %s] %s\n  %s\n",
			issue.GetState(),
			issue.GetTitle(),
			issue.GetHTMLURL(),
		))
	}

	return mcp.NewToolResultText(responseBuilder.String()), nil
}

func parsePRURL(url string) (owner string, repo string, number int, err error) {
	matches := prURLRegex.FindStringSubmatch(url)
	if len(matches) != 4 {
		return "", "", 0, fmt.Errorf("invalid PR URL format. Expected: .../owner/repo/pull/123")
	}

	owner = matches[1]
	repo = matches[2]
	number, err = strconv.Atoi(matches[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number: %s", matches[3])
	}

	return owner, repo, number, nil
}

func (s *githubService) getUnresolvedCommentsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prURL, err := req.RequireString("pull_request_url")
	if err != nil {
		return mcp.NewToolResultError("Missing required argument: pull_request_url"), nil
	}

	owner, repo, prNumber, err := parsePRURL(prURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid PR URL: %v", err)), nil
	}

	var query prCommentsQuery
	variables := map[string]interface{}{
		"owner":    githubv4.String(owner),
		"repo":     githubv4.String(repo),
		"prNumber": githubv4.Int(prNumber),
	}

	if err := s.graphqlClient.Query(ctx, &query, variables); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("GitHub GraphQL query failed: %v", err)), nil
	}

	var responseBuilder strings.Builder
	unresolvedCount := 0
	for _, thread := range query.Repository.PullRequest.ReviewThreads.Nodes {
		if !thread.IsResolved {
			unresolvedCount++
			if len(thread.Comments.Nodes) > 0 {
				firstComment := thread.Comments.Nodes[0]
				responseBuilder.WriteString(fmt.Sprintf(
					"Unresolved Thread on: %s (Line %d)\n",
					string(firstComment.Path),
					int(firstComment.Line),
				))

				for _, comment := range thread.Comments.Nodes {
					fullBody := string(comment.Body)
					if len(fullBody) > 200 {
						lines := strings.Split(fullBody, "\n")
						if len(lines) > 3 {
							preview := strings.Join(lines[:3], "\n")
							responseBuilder.WriteString(fmt.Sprintf(
								"  - @%s: %s\n… +%d lines (ctrl+o to expand)\n",
								string(comment.Author.Login),
								preview,
								len(lines)-3,
							))
						} else {
							responseBuilder.WriteString(fmt.Sprintf(
								"  - @%s: %s\n",
								string(comment.Author.Login),
								fullBody,
							))
						}
					} else {
						responseBuilder.WriteString(fmt.Sprintf(
							"  - @%s: %s\n",
							string(comment.Author.Login),
							fullBody,
						))
					}
				}
				responseBuilder.WriteString(fmt.Sprintf("  (Thread Link: %s)\n\n", firstComment.URL.String())) // .URL is a URI
			}
		}
	}

	if unresolvedCount == 0 {
		return mcp.NewToolResultText("No unresolved comments found on that PR."), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Found %d unresolved comment threads:\n\n%s", unresolvedCount, responseBuilder.String())), nil
}

func (s *githubService) getFullCommentsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prURL, err := req.RequireString("pull_request_url")
	if err != nil {
		return mcp.NewToolResultError("Missing required argument: pull_request_url"), nil
	}

	unresolvedOnlyStr := req.GetString("unresolved_only", "false")
	unresolvedOnly := unresolvedOnlyStr == "true"

	owner, repo, prNumber, err := parsePRURL(prURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid PR URL: %v", err)), nil
	}

	var query prCommentsQuery
	variables := map[string]interface{}{
		"owner":    githubv4.String(owner),
		"repo":     githubv4.String(repo),
		"prNumber": githubv4.Int(prNumber),
	}

	if err := s.graphqlClient.Query(ctx, &query, variables); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("GitHub GraphQL query failed: %v", err)), nil
	}

	var responseBuilder strings.Builder
	threadCount := 0
	for _, thread := range query.Repository.PullRequest.ReviewThreads.Nodes {
		isResolved := bool(thread.IsResolved)

		if unresolvedOnly && isResolved {
			continue
		}

		threadCount++
		if len(thread.Comments.Nodes) > 0 {
			firstComment := thread.Comments.Nodes[0]
			status := "Resolved"
			if !isResolved {
				status = "Unresolved"
			}

			responseBuilder.WriteString(fmt.Sprintf(
				"=== %s Thread on: %s (Line %d) ===\n",
				status,
				string(firstComment.Path),
				int(firstComment.Line),
			))

			for i, comment := range thread.Comments.Nodes {
				if i > 0 {
					responseBuilder.WriteString("\n--- Reply ---\n")
				}
				responseBuilder.WriteString(fmt.Sprintf(
					"@%s:\n%s\n",
					string(comment.Author.Login),
					string(comment.Body),
				))
			}
			responseBuilder.WriteString(fmt.Sprintf("\n(Thread Link: %s)\n\n", firstComment.URL.String()))
		}
	}

	if threadCount == 0 {
		if unresolvedOnly {
			return mcp.NewToolResultText("No unresolved comments found on that PR."), nil
		} else {
			return mcp.NewToolResultText("No comments found on that PR."), nil
		}
	}

	filterText := ""
	if unresolvedOnly {
		filterText = " unresolved"
	}

	return mcp.NewToolResultText(fmt.Sprintf("Found %d%s comment threads:\n\n%s", threadCount, filterText, responseBuilder.String())), nil
}
