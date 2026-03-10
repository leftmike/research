// Command list-detections is an MCP server that exposes ExtraHop RevealX 360
// detections search as a single list_detections tool.
//
// Configuration is via environment variables:
//
//	EXTRAHOP_BASE_URL    – API base URL (defaults to the RevealX 360 Cloud endpoint)
//	EXTRAHOP_CLIENT_ID   – OAuth2 client ID (required)
//	EXTRAHOP_CLIENT_SECRET – OAuth2 client secret (required)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

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

	s := server.NewMCPServer("extrahop-detections", "1.0.0")
	s.AddTool(listDetectionsTool(), newListDetectionsHandler(client))

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// listDetectionsTool returns the MCP tool definition for list_detections.
func listDetectionsTool() mcp.Tool {
	return mcp.NewTool("list_detections",
		mcp.WithDescription(`Search and retrieve security detections from ExtraHop RevealX 360.

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
- Audit resolved detections and the actions taken`),

		mcp.WithNumber("limit",
			mcp.Description("Maximum number of detections to return. Defaults to 100 when omitted; maximum is 1000. Use together with offset to page through large result sets."),
		),

		mcp.WithNumber("offset",
			mcp.Description("Number of detections to skip before returning results. Combine with limit to retrieve subsequent pages. Defaults to 0."),
		),

		mcp.WithNumber("from",
			mcp.Description("Earliest detection start time as Unix milliseconds. "+
				"Negative values express a relative window from now: "+
				"-3600000 = last hour, -86400000 = last 24 hours, -604800000 = last 7 days. "+
				"Omit to include detections of any age."),
		),

		mcp.WithNumber("until",
			mcp.Description("Latest detection start time as Unix milliseconds. "+
				"Use 0 or omit to include detections up to the current moment."),
		),

		mcp.WithArray("statuses",
			mcp.Description(`Filter by detection lifecycle status. Accepted values:
- "new"         – unreviewed detections awaiting triage
- "in_progress" – detections actively being investigated
- "closed"      – detections that have been resolved

Omit to return detections across all statuses.`),
			mcp.Items(map[string]any{
				"type": "string",
				"enum": []string{"new", "in_progress", "closed"},
			}),
		),
	)
}

// listDetectionsResult is the JSON envelope returned by the tool.
type listDetectionsResult struct {
	// Count is the number of detections returned in this response.
	Count int `json:"count"`

	// Detections is the list of matching detection records.
	Detections []*extrahop.Detection `json:"detections"`
}

// newListDetectionsHandler returns an MCP tool handler that calls the
// ExtraHop detections API and returns results as indented JSON.
func newListDetectionsHandler(client *extrahop.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := buildParams(req)

		detections, err := client.Detections.List(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("ExtraHop API error: %v", err)), nil
		}

		// Ensure the JSON array is never null.
		if detections == nil {
			detections = []*extrahop.Detection{}
		}

		out := listDetectionsResult{
			Count:      len(detections),
			Detections: detections,
		}

		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to encode detections: %w", err)
		}

		return mcp.NewToolResultText(string(b)), nil
	}
}

// buildParams translates MCP call arguments into DetectionListParams using the
// request's typed accessor methods. JSON numbers arrive as float64 and are
// cast to int/int64 as needed.
func buildParams(req mcp.CallToolRequest) *extrahop.DetectionListParams {
	p := &extrahop.DetectionListParams{}

	if v := req.GetFloat("limit", 0); v > 0 {
		p.Limit = int(v)
	}
	if v := req.GetFloat("offset", 0); v > 0 {
		p.Offset = int(v)
	}
	if v := req.GetFloat("from", 0); v != 0 {
		p.From = int64(v)
	}
	if v := req.GetFloat("until", 0); v != 0 {
		p.Until = int64(v)
	}
	// statuses is an array; retrieve via the raw arguments map.
	if raw, ok := req.GetArguments()["statuses"].([]any); ok {
		for _, item := range raw {
			if s, ok := item.(string); ok {
				p.Statuses = append(p.Statuses, s)
			}
		}
	}

	return p
}
