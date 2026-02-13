#!/bin/bash
# Test remote MCP servers from pulsemcp.com with gmcpt list
# Run servers in parallel batches

GMCPT="/home/user/research/mcplist/gmcpt"
SERVERS_JSON="/home/user/research/mcplist/pulsemcp_servers.json"
RESULTS_DIR="/home/user/research/mcplist/results_pulsemcp"
OUTPUT_FILE="/home/user/research/mcplist/pulsemcp.txt"

mkdir -p "$RESULTS_DIR"

# Extract server entries (skip URLs with template variables)
TOTAL=$(python3 -c "
import json
with open('$SERVERS_JSON') as f:
    servers = json.load(f)
servers = [s for s in servers if '{' not in s['url']]
print(len(servers))
")
echo "Testing $TOTAL server endpoints..."

test_server() {
    local idx="$1"
    local name="$2"
    local url="$3"
    local transport="$4"
    local result_file="$RESULTS_DIR/server_${idx}.txt"

    local sse_flag=""
    if [ "$transport" = "sse" ]; then
        sse_flag="-sse"
    fi

    # Run gmcpt list with a timeout
    local output
    output=$(timeout 15 "$GMCPT" list $sse_flag -tools -prompts -resources -json "$url" 2>&1)
    local exit_code=$?

    # Determine status
    local status="unknown"
    local auth_required="unknown"

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
}

export -f test_server
export GMCPT RESULTS_DIR

# Generate commands and run in parallel
python3 -c "
import json
with open('$SERVERS_JSON') as f:
    servers = json.load(f)
servers = [s for s in servers if '{' not in s['url']]
for i, s in enumerate(servers):
    name = s['name']
    url = s['url']
    transport = s['transport']
    # Escape tabs in name
    name = name.replace('\t', ' ')
    print(f'{i}\t{name}\t{url}\t{transport}')
" | while IFS=$'\t' read -r idx name url transport; do
    test_server "$idx" "$name" "$url" "$transport" &

    # Limit parallel jobs to 20
    if (( $(jobs -r | wc -l) >= 20 )); then
        wait -n 2>/dev/null || wait
    fi

    # Progress update every 50 servers
    if (( idx % 50 == 0 && idx > 0 )); then
        echo "  Started $idx / $TOTAL..."
    fi
done

wait
echo ""
echo "All servers tested. Compiling results..."

# Compile results into final output
python3 << 'PYEOF'
import json
import os

results_dir = "/home/user/research/mcplist/results_pulsemcp"
servers_json = "/home/user/research/mcplist/pulsemcp_servers.json"
output_file = "/home/user/research/mcplist/pulsemcp.txt"

with open(servers_json) as f:
    all_servers = json.load(f)

# Filter out template URLs
servers = [s for s in all_servers if '{' not in s['url']]

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
    f.write("MCP Servers (pulsemcp.com) - Authentication & Capabilities Report\n")
    f.write("Generated by gmcpt list against pulsemcp.com/servers?other[]=remote\n")
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
