package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// getDate returns the current date as a YYYY-MM-DD string.
func getDate() string {
	return time.Now().Format("2006-01-02")
}

// getTime returns the current time as a HH:MM:SS TZ string.
func getTime() string {
	return time.Now().Format("15:04:05 MST")
}

// dateTool returns the current date.
func dateTool(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: getDate()},
		},
	}, nil
}

// timeTool returns the current time with timezone.
func timeTool(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: getTime()},
		},
	}, nil
}

// readOSRelease parses /etc/os-release into a key=value map.
func readOSRelease() map[string]string {
	result := make(map[string]string)
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return result
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.IndexByte(line, '='); idx > 0 {
			result[line[:idx]] = strings.Trim(line[idx+1:], `"`)
		}
	}
	return result
}

// getOS returns the OS name and kernel version as a formatted string.
func getOS() string {
	release := readOSRelease()
	name := release["PRETTY_NAME"]
	if name == "" {
		name = release["NAME"]
	}
	if name == "" {
		name = runtime.GOOS
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "OS: %s\n", name)

	if data, err := os.ReadFile("/proc/version"); err == nil {
		fmt.Fprintf(&sb, "Kernel: %s\n", strings.TrimSpace(string(data)))
	}

	return strings.TrimRight(sb.String(), "\n")
}

// getHardware returns CPU model, core count, and total RAM as a formatted string.
func getHardware() string {
	var sb strings.Builder

	// CPU model from /proc/cpuinfo
	if f, err := os.Open("/proc/cpuinfo"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "model name") {
				if idx := strings.IndexByte(line, ':'); idx >= 0 {
					fmt.Fprintf(&sb, "CPU: %s\n", strings.TrimSpace(line[idx+1:]))
					break
				}
			}
		}
		f.Close()
	}

	fmt.Fprintf(&sb, "Cores: %d\n", runtime.NumCPU())

	// Total RAM from /proc/meminfo
	if f, err := os.Open("/proc/meminfo"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "MemTotal:") {
				var kb int64
				fmt.Sscanf(strings.TrimPrefix(line, "MemTotal:"), "%d", &kb)
				fmt.Fprintf(&sb, "RAM: %d MB\n", kb/1024)
				break
			}
		}
		f.Close()
	}

	return strings.TrimRight(sb.String(), "\n")
}

// osTool returns OS name and kernel version (Linux-specific).
func osTool(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: getOS()},
		},
	}, nil
}

// hardwareTool returns CPU model, core count, and total RAM (Linux-specific).
func hardwareTool(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: getHardware()},
		},
	}, nil
}

func newServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "sysmcp", Version: "0.1.0"}, nil)

	server.AddTool(&mcp.Tool{
		Name:        "date",
		Description: "Return the current date (YYYY-MM-DD)",
		InputSchema: &jsonschema.Schema{Type: "object"},
	}, dateTool)

	server.AddTool(&mcp.Tool{
		Name:        "time",
		Description: "Return the current time (HH:MM:SS timezone)",
		InputSchema: &jsonschema.Schema{Type: "object"},
	}, timeTool)

	server.AddTool(&mcp.Tool{
		Name:        "os",
		Description: "Return OS name and kernel version (Linux)",
		InputSchema: &jsonschema.Schema{Type: "object"},
	}, osTool)

	server.AddTool(&mcp.Tool{
		Name:        "hardware",
		Description: "Return CPU model, core count, and total RAM (Linux)",
		InputSchema: &jsonschema.Schema{Type: "object"},
	}, hardwareTool)

	return server
}

func main() {
	streaming := flag.Bool("streaming", false, "Use streaming HTTP instead of stdio")
	port := flag.Int("port", 8080, "Port to listen on (streaming mode only)")
	flag.Parse()

	if *streaming {
		server := newServer()
		addr := fmt.Sprintf(":%d", *port)
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("sysmcp listening on %s/mcp", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Fatalf("http server error: %v", err)
		}
	} else {
		server := newServer()
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}
}
