#!/bin/bash
# Test each MCP server with gmcpt list and record results
# Run servers in parallel batches

GMCPT="/home/user/research/mcplist/gmcpt"
SERVERS_JSON="/home/user/research/mcplist/registry_servers.json"
RESULTS_DIR="/home/user/research/mcplist/results"
FINAL_OUTPUT="/home/user/research/mcplist/mcp_servers.txt"

mkdir -p "$RESULTS_DIR"

# Extract server entries
TOTAL=$(python3 -c "import json; data=json.load(open('$SERVERS_JSON')); print(len(data))")
echo "Testing $TOTAL servers..."

test_server() {
    local idx="$1"
    local name="$2"
    local url="$3"
    local transport="$4"
    local auth_info="$5"
    local result_file="$RESULTS_DIR/server_${idx}.txt"

    local sse_flag=""
    if [ "$transport" = "sse" ]; then
        sse_flag="-sse"
    fi

    # Run gmcpt list with a timeout
    local output
    output=$(timeout 10 "$GMCPT" list $sse_flag -tools -prompts -resources -json "$url" 2>&1)
    local exit_code=$?

    # Determine status
    local status="unknown"
    local auth_required="unknown"

    if echo "$output" | grep -q "Forbidden"; then
        status="connection_blocked"
    elif echo "$output" | grep -q "Unauthorized\|401\|unauthorized\|auth"; then
        status="auth_required"
        auth_required="yes"
    elif echo "$output" | grep -q "connection refused\|no such host\|dial tcp"; then
        status="connection_failed"
    elif echo "$output" | grep -q "timeout\|deadline exceeded"; then
        status="timeout"
    elif [ $exit_code -eq 0 ]; then
        status="success"
        auth_required="no"
    else
        status="error"
    fi

    # If registry metadata indicates auth headers
    if [ -n "$auth_info" ] && [ "$auth_info" != "none" ]; then
        auth_required="yes (registry metadata: $auth_info)"
    elif [ "$auth_required" = "unknown" ] && [ -z "$auth_info" -o "$auth_info" = "none" ]; then
        auth_required="unknown (no registry auth metadata)"
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
import json, shlex
with open('$SERVERS_JSON') as f:
    servers = json.load(f)
for i, s in enumerate(servers):
    name = s['name']
    url = s['url']
    transport = s['transport_type']
    auth_headers = s.get('auth_headers', [])
    if auth_headers:
        auth_info = '; '.join(h['name'] for h in auth_headers)
    else:
        auth_info = 'none'
    print(f'{i}\t{name}\t{url}\t{transport}\t{auth_info}')
" | while IFS=$'\t' read -r idx name url transport auth_info; do
    test_server "$idx" "$name" "$url" "$transport" "$auth_info" &

    # Limit parallel jobs to 20
    if (( $(jobs -r | wc -l) >= 20 )); then
        wait -n 2>/dev/null || wait
    fi

    # Progress update every 50 servers
    if (( idx % 50 == 0 )); then
        echo "  Started $idx / $TOTAL..."
    fi
done

wait
echo "All servers tested. Compiling results..."

# Compile results into final output
python3 << 'PYEOF'
import json
import os
import re

results_dir = "/home/user/research/mcplist/results"
servers_json = "/home/user/research/mcplist/registry_servers.json"
output_file = "/home/user/research/mcplist/mcp_servers.txt"

with open(servers_json) as f:
    servers = json.load(f)

results = []
for i in range(len(servers)):
    result_file = os.path.join(results_dir, f"server_{i}.txt")
    if not os.path.exists(result_file):
        continue
    with open(result_file) as f:
        content = f.read()

    # Parse result fields
    fields = {}
    for line in content.split('\n'):
        if ':' in line and not line.startswith('OUTPUT:') and not line.startswith('---'):
            key, _, val = line.partition(': ')
            fields[key.strip()] = val.strip()

    # Extract output section
    output_start = content.find('OUTPUT:')
    output_end = content.find('\n---')
    output = content[output_start+8:output_end].strip() if output_start >= 0 else ''

    # Parse tools, prompts, resources from JSON output if successful
    tools = []
    prompts = []
    resources = []

    if fields.get('STATUS') == 'success':
        try:
            data = json.loads(output)
            tools = [t.get('name', '') for t in data.get('tools', [])]
            prompts = [p.get('name', '') for p in data.get('prompts', [])]
            resources = [r.get('name', r.get('uri', '')) for r in data.get('resources', [])]
        except (json.JSONDecodeError, TypeError):
            pass

    results.append({
        'name': fields.get('NAME', servers[i]['name']),
        'url': fields.get('URL', servers[i]['url']),
        'transport': fields.get('TRANSPORT', servers[i]['transport_type']),
        'auth_required': fields.get('AUTH_REQUIRED', 'unknown'),
        'status': fields.get('STATUS', 'unknown'),
        'tools': tools,
        'prompts': prompts,
        'resources': resources,
        'raw_output': output[:500],
    })

# Write final output
with open(output_file, 'w') as f:
    f.write("MCP Server Registry - Auth Check and Capabilities Report\n")
    f.write("=" * 70 + "\n")
    f.write(f"Total servers tested: {len(results)}\n\n")

    # Summary stats
    statuses = {}
    auth_yes = 0
    auth_no = 0
    for r in results:
        statuses[r['status']] = statuses.get(r['status'], 0) + 1
        if 'yes' in r['auth_required'].lower():
            auth_yes += 1
        elif r['auth_required'] == 'no':
            auth_no += 1

    f.write("Status Summary:\n")
    for s, c in sorted(statuses.items()):
        f.write(f"  {s}: {c}\n")
    f.write(f"\nAuth Required (from registry metadata): {auth_yes}\n")
    f.write(f"No Auth Required (from successful connection): {auth_no}\n")
    f.write("\n" + "=" * 70 + "\n\n")

    for r in results:
        f.write(f"Server: {r['name']}\n")
        f.write(f"  URL: {r['url']}\n")
        f.write(f"  Transport: {r['transport']}\n")
        f.write(f"  Auth Required: {r['auth_required']}\n")
        f.write(f"  Connection Status: {r['status']}\n")
        if r['tools']:
            f.write(f"  Tools: {', '.join(r['tools'])}\n")
        if r['prompts']:
            f.write(f"  Prompts: {', '.join(r['prompts'])}\n")
        if r['resources']:
            f.write(f"  Resources: {', '.join(r['resources'])}\n")
        if r['status'] not in ('success', 'connection_blocked') and r['raw_output']:
            f.write(f"  Error: {r['raw_output'][:200]}\n")
        f.write("\n")

print(f"Results written to {output_file}")
print(f"Total: {len(results)} servers")
PYEOF

echo "Done!"
