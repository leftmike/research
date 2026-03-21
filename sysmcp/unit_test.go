package main

import (
	"context"
	"regexp"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

// TestToolsUnit tests all four sysmcp tools using the mark3labs/mcp-go client
// package by launching the server as a subprocess and communicating over stdio.
func TestToolsUnit(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		tool    string
		pattern string
	}{
		{"date", `\d{4}-\d{2}-\d{2}`},
		{"time", `\d{2}:\d{2}:\d{2}`},
		{"os", `(?i)OS:`},
		{"hardware", `(?i)(CPU:|Cores:|RAM:)`},
	}

	for _, tc := range tests {
		t.Run(tc.tool, func(t *testing.T) {
			c, err := client.NewStdioMCPClient(binary, nil)
			if err != nil {
				t.Fatalf("creating MCP client: %v", err)
			}
			defer c.Close()

			ctx := context.Background()

			_, err = c.Initialize(ctx, mcpgo.InitializeRequest{
				Params: mcpgo.InitializeParams{
					ProtocolVersion: mcpgo.LATEST_PROTOCOL_VERSION,
					ClientInfo: mcpgo.Implementation{
						Name:    "sysmcp-unit-test",
						Version: "0.0.1",
					},
				},
			})
			if err != nil {
				t.Fatalf("initializing MCP connection: %v", err)
			}

			result, err := c.CallTool(ctx, mcpgo.CallToolRequest{
				Params: mcpgo.CallToolParams{
					Name: tc.tool,
				},
			})
			if err != nil {
				t.Fatalf("calling tool %q: %v", tc.tool, err)
			}

			var text string
			for _, content := range result.Content {
				switch v := content.(type) {
				case mcpgo.TextContent:
					text += v.Text
				case *mcpgo.TextContent:
					text += v.Text
				}
			}

			re := regexp.MustCompile(tc.pattern)
			if !re.MatchString(text) {
				t.Errorf("tool %q: output %q did not match pattern %q", tc.tool, text, tc.pattern)
			}
		})
	}
}
