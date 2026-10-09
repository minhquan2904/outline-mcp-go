package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/server"
)

func main() {
	baseURL := os.Getenv("OUTLINE_API_URL")
	apiKey := os.Getenv("OUTLINE_API_KEY")

	client := NewClient(baseURL, apiKey)

	// Create MCP server
	s := server.NewServer(&mcp.ServerCapabilities{
		Tools: &mcp.ToolCapabilities{},
	})

	// Register tools
	for _, tool := range Tools {
		toolName := tool.Name
		toolDesc := tool.Description
		handler := tool.Handler // capture handler

		s.Tool(
			toolName,
			toolDesc,
			map[string]interface{}{}, // TODO: add proper schema
			func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.ToolResultText, error) {
				// Execute handler
				output := handler(ctx, request.Arguments, client)

				// Check if output is an error
				isError := strings.HasPrefix(output, "Error:")

				return &mcp.ToolResultText{
					Content: output,
					IsError: isError,
				}, nil
			},
		)
	}

	// Connect stdio transport
	transport := server.NewStdioServerTransport()
	_ = s.Connect(transport)

	// Log startup
	fmt.Fprintf(os.Stderr, "[outline] MCP server started (stdio transport)\n")
}
