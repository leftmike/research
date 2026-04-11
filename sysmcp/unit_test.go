package main

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestToolsUnit tests all four sysmcp tools by running the go-sdk server
// in-process and connecting to it with a mark3labs/mcp-go client via io.Pipe.
func TestToolsUnit(t *testing.T) {
	ctx := context.Background()

	// Two pipes bridge the go-sdk server and the mark3labs client.
	// clientWriter → serverReader: client sends requests to server.
	// serverWriter → clientReader: server sends responses to client.
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	// Run the go-sdk server on one end of the pipes.
	go func() {
		s := newServer()
		_ = s.Run(ctx, &mcp.IOTransport{
			Reader: serverReader,
			Writer: serverWriter,
		})
	}()

	// Connect the mark3labs client to the other ends.
	tr := transport.NewIO(clientReader, clientWriter, io.NopCloser(bytes.NewReader(nil)))
	c := mcpclient.NewClient(tr)
	defer c.Close()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("starting client: %v", err)
	}

	initReq := mcpgo.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcpgo.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcpgo.Implementation{
		Name:    "sysmcp-unit-test",
		Version: "0.0.1",
	}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		t.Fatalf("initializing: %v", err)
	}

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
			req := mcpgo.CallToolRequest{}
			req.Params.Name = tc.tool

			result, err := c.CallTool(ctx, req)
			if err != nil {
				t.Fatalf("calling tool %q: %v", tc.tool, err)
			}

			var text string
			for _, content := range result.Content {
				if tc2, ok := content.(mcpgo.TextContent); ok {
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
