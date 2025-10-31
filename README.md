# go-github-mcp

A Model Context Protocol (MCP) server implementation in Go that provides GitHub integration for Claude.  
This project showcases how to build custom MCP servers using the [`github.com/mark3labs/mcp-go/mcp`](https://github.com/mark3labs/mcp-go) package.

---

## Features

- **List Pull Requests**: Get your authored pull requests with filtering by state (open, closed, all)
- **Get Unresolved Comments**: Retrieve unresolved review comments with smart previews
- **Get Full Comments**: Access complete comment threads with full content and no truncation
- **GitHub Authentication**: Secure token-based authentication
- **GraphQL Integration**: Efficient data fetching using GitHub's GraphQL API

---

## Installation

### Prerequisites

- Go 1.19 or later
- GitHub Personal Access Token with appropriate permissions
- Claude CLI or compatible MCP client

### Building

1. Clone the repository:

   ```bash
   git clone <repository-url>
   cd gogithub
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Build the MCP server:

   ```bash
   go build -o github-mcp-server
   ```

---

## Configuration

### 1. GitHub Token Setup

Create a GitHub Personal Access Token with the following permissions:

- `repo` — Full control of private repositories
- `read:user` — Read access to profile

Set the token as an environment variable:

```bash
export GITHUB_TOKEN=your_github_token_here
```

---

### 2. Claude MCP Configuration

Add the MCP server to your Claude configuration file:

~/.claude.json

```json
{
  "mcpServers": {
    "my-github-tools": {
      "command": "/path/to/your/gogithub/github-mcp-server",
      "env": {
        "GITHUB_TOKEN": "your_github_token_here"
      }
    }
  }
}
```

> Replace `/path/to/your/gogithub/github-mcp-server` with the absolute path to your built binary.

---

### 3. Restart Claude

Restart Claude CLI to load the new MCP server configuration.

---

## Usage

Once configured, you can use the following tools in Claude.

### List Pull Requests

```bash
list my open pull requests
```

**Parameters:**
- `state` (optional): `"open"`, `"closed"`, or `"all"` (default: `"open"`)

---

### Get Unresolved Comments

```bash
get unresolved comments from https://github.com/owner/repo/pull/123
```

**Parameters:**
- `pull_request_url` (required): Full GitHub PR URL

---

### Get Full Comments

```bash
get full comments from https://github.com/owner/repo/pull/123
```

**Parameters:**
- `pull_request_url` (required): Full GitHub PR URL
- `unresolved_only` (optional): `"true"` or `"false"` (default: `"false"`)

---

## Example Workflow

1. **Find your PRs:**
   ```bash
   list my open prs
   ```

2. **Check for unresolved comments:**
   ```bash
   get unresolved comments from the first open PR
   ```

3. **Get full comment details:**
   ```bash
   get full comments from this PR
   ```

4. **Ask Claude for assistance:**
   ```bash
   help me fix those comments step by step
   ```

---

## Architecture

### Project Structure

```
gogithub/
├── main.go             # MCP server setup and tool registration
├── github_service.go   # GitHub API integration and handlers
├── go.mod              # Go module dependencies
├── go.sum              # Dependency checksums
└── README.md           # This file
```

### Key Components

- **MCP Server** — Built using `github.com/mark3labs/mcp-go/mcp`
- **GitHub REST API** — Pull request discovery via search
- **GitHub GraphQL API** — Efficient comment thread retrieval
- **OAuth2 Authentication** — Secure GitHub token handling

---

## Code Examples

### Tool Registration

```go
listPRsTool := mcp.NewTool(
    "list_pull_requests",
    mcp.WithDescription("Lists pull requests authored by the authenticated user."),
    mcp.WithString(
        "state",
        mcp.Description("The state of the pull requests to list (open, closed, or all). Defaults to 'open'."),
        mcp.Enum("open", "closed", "all"),
    ),
)

s.AddTool(listPRsTool, ghService.listPullRequestsHandler)
```

### Handler Implementation

```go
func (s *githubService) listPullRequestsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    state := req.GetString("state", "open")

    // GitHub API integration
    query := strings.Join(queryParts, " ")
    result, resp, err := s.restClient.Search.Issues(ctx, query, opts)

    // Return formatted results
    return mcp.NewToolResultText(responseBuilder.String()), nil
}
```

---

## Related Resources

- [MCP Go Package](https://github.com/mark3labs/mcp-go)
- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [GitHub API Documentation](https://docs.github.com/en/rest)
- [Claude MCP Documentation](https://docs.anthropic.com/claude/docs/mcp)
