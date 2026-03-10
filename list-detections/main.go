// Command list-detections is an MCP server that exposes ExtraHop RevealX 360
// detections search as a single list_detections tool.
//
// Configuration is via environment variables:
//
//	EXTRAHOP_BASE_URL      – API base URL (defaults to the RevealX 360 Cloud endpoint)
//	EXTRAHOP_CLIENT_ID     – OAuth2 client ID (required)
//	EXTRAHOP_CLIENT_SECRET – OAuth2 client secret (required)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	extrahop "github.com/leftmike/extrahop-go-client"
)

func main() {
	baseURL := os.Getenv("EXTRAHOP_BASE_URL")
	clientID := os.Getenv("EXTRAHOP_CLIENT_ID")
	clientSecret := os.Getenv("EXTRAHOP_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("EXTRAHOP_CLIENT_ID and EXTRAHOP_CLIENT_SECRET environment variables must be set")
	}

	client, err := extrahop.NewClient(baseURL, clientID, clientSecret, nil)
	if err != nil {
		log.Fatalf("failed to create ExtraHop client: %v", err)
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "extrahop-detections", Version: "1.0.0"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_detections",
		Description: `Search and retrieve security detections from ExtraHop RevealX 360.

Returns network security events detected by ExtraHop sensors. Each detection includes:
- A human-readable title and narrative description
- A risk score from 0 (low) to 99 (critical)
- MITRE ATT&CK technique mappings (e.g. T1046 Network Service Scanning)
- Participant devices or users with their role: offender or victim
- Lifecycle status and, when closed, the resolution taken
- Detection-type-specific properties for deeper context

Use this tool to:
- Surface active threats and anomalies on the network
- Triage unreviewed alerts ranked by risk score or MITRE category
- Scope detections to a specific time window for incident response
- Review detections currently under investigation
- Audit resolved detections and the actions taken`,
	}, newListDetectionsHandler(client))

	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// listDetectionsInput defines the arguments for the list_detections tool.
// Field descriptions are used by the SDK to populate the tool's input schema.
type listDetectionsInput struct {
	Limit  int   `json:"limit,omitempty"  jsonschema:"Maximum number of detections to return. Defaults to 100 when omitted; maximum is 1000. Use with offset to page through large result sets."`
	Offset int   `json:"offset,omitempty" jsonschema:"Number of detections to skip before returning results. Combine with limit to retrieve subsequent pages. Defaults to 0."`
	From   int64 `json:"from,omitempty"   jsonschema:"Earliest detection start time as Unix milliseconds. Negative values express a relative window from now: -3600000 = last hour, -86400000 = last 24 hours, -604800000 = last 7 days. Omit to include detections of any age."`
	Until  int64 `json:"until,omitempty"  jsonschema:"Latest detection start time as Unix milliseconds. Use 0 or omit to include detections up to the current moment."`

	// Statuses filters by detection lifecycle status.
	// Valid values: "new" (unreviewed), "in_progress" (under investigation), "closed" (resolved).
	// Omit to return detections across all statuses.
	Statuses []string `json:"statuses,omitempty" jsonschema:"Filter by detection lifecycle status. Valid values: 'new' (unreviewed, awaiting triage), 'in_progress' (actively being investigated), 'closed' (resolved). Omit to return all statuses."`
}

// listDetectionsResult is the JSON envelope written to the tool response.
type listDetectionsResult struct {
	Count      int                   `json:"count"`
	Detections []*extrahop.Detection `json:"detections"`
}

// newListDetectionsHandler returns a typed tool handler that calls the ExtraHop
// detections API and returns an indented JSON response. Using 'any' as the output
// type suppresses output schema generation; content is returned as plain text.
func newListDetectionsHandler(client *extrahop.Client) func(context.Context, *mcp.CallToolRequest, listDetectionsInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in listDetectionsInput) (*mcp.CallToolResult, any, error) {
		params := &extrahop.DetectionListParams{
			Limit:    in.Limit,
			Offset:   in.Offset,
			From:     in.From,
			Until:    in.Until,
			Statuses: in.Statuses,
		}

		detections, err := client.Detections.List(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("ExtraHop API error: %v", err)}},
			}, nil, nil
		}

		if detections == nil {
			detections = []*extrahop.Detection{}
		}

		out := listDetectionsResult{
			Count:      len(detections),
			Detections: detections,
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to encode detections: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
		}, nil, nil
	}
}
