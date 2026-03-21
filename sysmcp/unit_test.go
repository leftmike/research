package main

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newTestServer builds a mark3labs/mcp-go MCPServer whose tool handlers call
// the same getDate/getTime/getOS/getHardware helpers used by the real server.
func newTestServer() *server.MCPServer {
	s := server.NewMCPServer("sysmcp-test", "0.1.0")

	s.AddTool(
		mcp.NewTool("date", mcp.WithDescription("Return the current date (YYYY-MM-DD)")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(getDate()), nil
		},
	)
	s.AddTool(
		mcp.NewTool("time", mcp.WithDescription("Return the current time (HH:MM:SS timezone)")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(getTime()), nil
		},
	)
	s.AddTool(
		mcp.NewTool("os", mcp.WithDescription("Return OS name and kernel version (Linux)")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(getOS()), nil
		},
	)
	s.AddTool(
		mcp.NewTool("hardware", mcp.WithDescription("Return CPU model, core count, and total RAM (Linux)")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(getHardware()), nil
		},
	)

	return s
}

func TestToolsUnit(t *testing.T) {
	s := newTestServer()
	ctx := context.Background()

	// Initialize the server (required before calling tools).
	_ = s.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 0,
		"method": "initialize",
		"params": {
			"protocolVersion": "2024-11-05",
			"clientInfo": {"name": "unit-test", "version": "0.0.1"},
			"capabilities": {}
		}
	}`))

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
			msg := fmt.Sprintf(`{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "tools/call",
				"params": {"name": %q, "arguments": {}}
			}`, tc.tool)

			response := s.HandleMessage(ctx, []byte(msg))

			resp, ok := response.(mcp.JSONRPCResponse)
			if !ok {
				t.Fatalf("tool %q: expected JSONRPCResponse, got %T", tc.tool, response)
			}
			result, ok := resp.Result.(*mcp.CallToolResult)
			if !ok {
				t.Fatalf("tool %q: expected *CallToolResult, got %T", tc.tool, resp.Result)
			}

			var text string
			for _, c := range result.Content {
				if tc2, ok := c.(mcp.TextContent); ok {
					text += tc2.Text
				}
			}

			re := regexp.MustCompile(tc.pattern)
			if !re.MatchString(text) {
				t.Errorf("tool %q: output %q did not match pattern %q", tc.tool, text, tc.pattern)
			}
		})
	}
}
