package main

import (
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Println("Starting GitHub MCP Server...")

	ghService, err := newGithubService()
	if err != nil {
		log.Fatalf("Failed to create GitHub service: %v", err)
	}
	log.Println("GitHub service initialized successfully.")

	s := server.NewMCPServer("GitHub MCP", "1.0.0")

	// 3. Define the tool for listing PRs
	listPRsTool := mcp.NewTool(
		"list_pull_requests",
		mcp.WithDescription("Lists pull requests authored by the authenticated user."),

		// Add an optional string argument for "state"
		mcp.WithString(
			"state",
			mcp.Description("The state of the pull requests to list (open, closed, or all). Defaults to 'open'."),
			mcp.Enum("open", "closed", "all"), // This helps Claude know the valid options
		),
	)

	// 4. Add the tool to the server, passing our service's handler function.
	s.AddTool(listPRsTool, ghService.listPullRequestsHandler)

	getUnresolvedCommentsTool := mcp.NewTool(
		"get_unresolved_comments",
		mcp.WithDescription("Gets all unresolved review comments from a specific GitHub pull request."),
		mcp.WithString(
			"pull_request_url",
			mcp.Required(),
			mcp.Description("The full URL of the pull request (e.g., https://github.com/owner/repo/pull/123)"),
		),
	)

	// 6. Add the new comments tool to the server
	s.AddTool(getUnresolvedCommentsTool, ghService.getUnresolvedCommentsHandler)

	// 7. Tool to get full comment details without truncation
	getFullCommentsTool := mcp.NewTool(
		"get_full_comments",
		mcp.WithDescription("Gets all review comments from a specific GitHub pull request with full content (no truncation)."),
		mcp.WithString(
			"pull_request_url",
			mcp.Required(),
			mcp.Description("The full URL of the pull request (e.g., https://github.com/owner/repo/pull/123)"),
		),
		mcp.WithBoolean(
			"unresolved_only",
			mcp.Description("If true, only show unresolved comments. If false, show all comments. Defaults to false."),
		),
	)

	// 8. Add the full comments tool to the server
	s.AddTool(getFullCommentsTool, ghService.getFullCommentsHandler)

	log.Println("MCP server running. Waiting for requests from Claude CLI...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
