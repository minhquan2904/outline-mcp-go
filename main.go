package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	baseURL := os.Getenv("OUTLINE_API_URL")
	apiKey := os.Getenv("OUTLINE_API_KEY")

	client := NewClient(baseURL, apiKey)

	s := mcp.NewServer(&mcp.Implementation{Name: "outline-mcp-go", Version: "0.1.0"}, nil)

	for _, tool := range Tools {
		handler := tool.Handler
		s.AddTool(
			&mcp.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: toolSchemas[tool.Name],
			},
			func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := map[string]interface{}{}
				if len(req.Params.Arguments) > 0 {
					if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
						return textResult("Error: invalid arguments: "+err.Error(), true), nil
					}
				}
				output := handler(ctx, args, client)
				return textResult(output, strings.HasPrefix(output, "Error:")), nil
			},
		)
	}

	fmt.Fprintf(os.Stderr, "[outline] MCP server started (stdio transport)\n")
	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "[outline] server stopped: %v\n", err)
		os.Exit(1)
	}
}

func textResult(text string, isError bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: isError,
	}
}
