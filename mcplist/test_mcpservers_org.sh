#!/bin/bash
# Test remote MCP servers from mcpservers.org with gmcpt list

GMCPT="/home/user/research/mcplist/gmcpt"
RESULTS_DIR="/home/user/research/mcplist/results_mcpservers_org"
OUTPUT_FILE="/home/user/research/mcplist/mcpservers_org.txt"

mkdir -p "$RESULTS_DIR"

# Define servers: name|url|transport (sse or streamable-http)
SERVERS=(
    "GitHub|https://api.githubcopilot.com/mcp/|streamable-http"
    "Notion|https://mcp.notion.com/mcp|streamable-http"
    "Sentry|https://mcp.sentry.dev/sse|sse"
    "Linear|https://mcp.linear.app/sse|sse"
    "Figma|https://mcp.figma.com/mcp|streamable-http"
    "DeepWiki|https://mcp.deepwiki.com/mcp|streamable-http"
    "DeepWiki (SSE)|https://mcp.deepwiki.com/sse|sse"
    "Intercom|https://mcp.intercom.com/sse|sse"
    "Neon|https://mcp.neon.tech/sse|sse"
    "Supabase|https://mcp.supabase.com/mcp|streamable-http"
    "PayPal|https://mcp.paypal.com/sse|sse"
    "Square|https://mcp.squareup.com/sse|sse"
    "CoinGecko|https://mcp.api.coingecko.com/sse|sse"
    "Ahrefs|https://api.ahrefs.com/mcp/mcp|streamable-http"
    "Ahrefs (SSE)|https://api.ahrefs.com/mcp/mcpSse|sse"
    "Asana|https://mcp.asana.com/sse|sse"
    "Atlassian|https://mcp.atlassian.com/v1/sse|sse"
    "Wix|https://mcp.wix.com/sse|sse"
    "Webflow|https://mcp.webflow.com/sse|sse"
    "Globalping|https://mcp.globalping.dev/sse|sse"
    "Semgrep|https://mcp.semgrep.ai/sse|sse"
    "Fetch|https://remote.mcpservers.org/fetch/mcp|streamable-http"
    "Sequential Thinking|https://remote.mcpservers.org/sequentialthinking/mcp|streamable-http"
    "EdgeOne Pages|https://remote.mcpservers.org/edgeone-pages/mcp|streamable-http"
)

TOTAL=${#SERVERS[@]}
echo "Testing $TOTAL server endpoints..."

for idx in "${!SERVERS[@]}"; do
    IFS='|' read -r name url transport <<< "${SERVERS[$idx]}"
    result_file="$RESULTS_DIR/server_${idx}.txt"

    sse_flag=""
    if [ "$transport" = "sse" ]; then
        sse_flag="-sse"
    fi

    echo "  [$((idx+1))/$TOTAL] Testing: $name ($url)..."

    # Run gmcpt list with a timeout
    output=$(timeout 15 "$GMCPT" list $sse_flag -tools -prompts -resources -json "$url" 2>&1)
    exit_code=$?

    # Determine status
    status="unknown"
    auth_required="unknown"

    if echo "$output" | grep -qi "Forbidden"; then
        status="connection_blocked"
    elif echo "$output" | grep -qi "Unauthorized\|401\|unauthorized"; then
        status="auth_required"
        auth_required="yes (confirmed by gmcpt)"
    elif echo "$output" | grep -qi "connection refused\|no such host\|dial tcp.*connect"; then
        status="connection_failed"
    elif echo "$output" | grep -qi "timeout\|deadline exceeded\|context deadline"; then
        status="timeout"
    elif [ $exit_code -eq 124 ]; then
        status="timeout"
    elif [ $exit_code -eq 0 ]; then
        status="success"
        auth_required="no"
    else
        status="error"
    fi

    # Write result
    echo "NAME: $name" > "$result_file"
    echo "URL: $url" >> "$result_file"
    echo "TRANSPORT: $transport" >> "$result_file"
    echo "AUTH_REQUIRED: $auth_required" >> "$result_file"
    echo "STATUS: $status" >> "$result_file"
    echo "EXIT_CODE: $exit_code" >> "$result_file"
    echo "OUTPUT:" >> "$result_file"
    echo "$output" >> "$result_file"
    echo "---" >> "$result_file"
done

echo ""
echo "All servers tested. Compiling results..."

# Compile results
python3 << 'PYEOF'
import json
import os

results_dir = "/home/user/research/mcplist/results_mcpservers_org"
output_file = "/home/user/research/mcplist/mcpservers_org.txt"

# Get all result files in order
result_files = sorted(
    [f for f in os.listdir(results_dir) if f.startswith('server_') and f.endswith('.txt')],
    key=lambda x: int(x.split('_')[1].split('.')[0])
)

parsed_results = []
for rf in result_files:
    with open(os.path.join(results_dir, rf)) as f:
        content = f.read()

    fields = {}
    for line in content.split('\n'):
        if line.startswith('OUTPUT:') or line.startswith('---'):
            break
        if ': ' in line:
            key, _, val = line.partition(': ')
            fields[key.strip()] = val.strip()

    output_start = content.find('OUTPUT:')
    output_end = content.find('\n---')
    raw_output = content[output_start + 8:output_end].strip() if output_start >= 0 else ''

    tools = []
    prompts = []
    resources = []
    status = fields.get('STATUS', 'unknown')

    if status == 'success':
        try:
            data = json.loads(raw_output)
            tools = [t.get('name', '') for t in data.get('tools', [])]
            prompts = [p.get('name', '') for p in data.get('prompts', [])]
            resources = [r.get('name', r.get('uri', '')) for r in data.get('resources', [])]
        except (json.JSONDecodeError, TypeError):
            pass

    is_sse = fields.get('TRANSPORT', '') == 'sse'

    parsed_results.append({
        'name': fields.get('NAME', ''),
        'url': fields.get('URL', ''),
        'transport': fields.get('TRANSPORT', ''),
        'is_sse': is_sse,
        'auth_required': fields.get('AUTH_REQUIRED', 'unknown'),
        'status': status,
        'tools': tools,
        'prompts': prompts,
        'resources': resources,
        'error_snippet': raw_output[:300] if status not in ('success', 'connection_blocked') else '',
    })

# Write final output
with open(output_file, 'w') as f:
    f.write("MCP Servers (mcpservers.org) - Authentication & Capabilities Report\n")
    f.write("Generated by gmcpt list against mcpservers.org/remote-mcp-servers\n")
    f.write("=" * 78 + "\n\n")

    total = len(parsed_results)
    sse_count = sum(1 for r in parsed_results if r['is_sse'])
    streamable_count = sum(1 for r in parsed_results if not r['is_sse'])

    gmcpt_success = sum(1 for r in parsed_results if r['status'] == 'success')
    gmcpt_auth = sum(1 for r in parsed_results if r['status'] == 'auth_required')
    gmcpt_blocked = sum(1 for r in parsed_results if r['status'] == 'connection_blocked')
    gmcpt_failed = sum(1 for r in parsed_results if r['status'] == 'connection_failed')
    gmcpt_timeout = sum(1 for r in parsed_results if r['status'] == 'timeout')
    gmcpt_error = sum(1 for r in parsed_results if r['status'] == 'error')

    f.write("SUMMARY\n")
    f.write(f"  Total remote MCP server endpoints: {total}\n")
    f.write(f"  SSE transport: {sse_count}\n")
    f.write(f"  Streamable HTTP transport: {streamable_count}\n")
    f.write(f"\n")
    f.write(f"  gmcpt list results:\n")
    f.write(f"    Successful: {gmcpt_success}\n")
    f.write(f"    Auth required (confirmed by server): {gmcpt_auth}\n")
    f.write(f"    Connection blocked (egress proxy): {gmcpt_blocked}\n")
    f.write(f"    Connection failed: {gmcpt_failed}\n")
    f.write(f"    Timeout: {gmcpt_timeout}\n")
    f.write(f"    Other error: {gmcpt_error}\n")
    f.write("\n" + "=" * 78 + "\n\n")

    for r in parsed_results:
        f.write(f"Server: {r['name']}\n")
        f.write(f"  URL: {r['url']}\n")
        f.write(f"  SSE: {'yes' if r['is_sse'] else 'no'}\n")
        f.write(f"  Transport: {r['transport']}\n")
        f.write(f"  Auth Required: {r['auth_required']}\n")
        f.write(f"  gmcpt Status: {r['status']}\n")

        if r['tools']:
            f.write(f"  Tools: {', '.join(r['tools'])}\n")
        if r['prompts']:
            f.write(f"  Prompts: {', '.join(r['prompts'])}\n")
        if r['resources']:
            f.write(f"  Resources: {', '.join(r['resources'])}\n")

        if r['error_snippet']:
            f.write(f"  Error: {r['error_snippet']}\n")

        f.write("\n")

print(f"Wrote {output_file} with {total} server endpoints")
PYEOF

echo "Done!"
