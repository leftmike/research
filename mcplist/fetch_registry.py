#!/usr/bin/env python3
"""Fetch all MCP servers from the registry and extract remote endpoints."""

import json
import subprocess
import sys


def fetch_page(cursor=None):
    url = 'https://registry.modelcontextprotocol.io/v0.1/servers'
    if cursor:
        url += f'?cursor={cursor}'
    result = subprocess.run(['curl', '-s', url], capture_output=True, text=True, timeout=30)
    if result.returncode != 0:
        print(f"curl failed: {result.stderr}", file=sys.stderr)
        return None
    if not result.stdout.strip():
        return None
    return json.loads(result.stdout)


def extract_remote_servers(data):
    """Extract servers that have remote (SSE or streamable HTTP) endpoints."""
    servers = []
    for entry in data.get('servers', []):
        server = entry.get('server', {})
        name = server.get('name', 'unknown')
        version = server.get('version', 'unknown')
        description = server.get('description', '')

        remotes = server.get('remotes', [])
        for remote in remotes:
            rtype = remote.get('type', '')
            url = remote.get('url', '')

            # Check headers for auth requirements
            headers = remote.get('headers', [])
            auth_headers = []
            for h in headers:
                hname = h.get('name', '')
                if hname.lower() in ('authorization', 'x-api-key') or \
                   'auth' in hname.lower() or 'key' in hname.lower() or \
                   'token' in hname.lower():
                    auth_headers.append({
                        'name': hname,
                        'required': h.get('isRequired', False),
                        'secret': h.get('isSecret', False),
                        'description': h.get('description', ''),
                    })

            servers.append({
                'name': name,
                'version': version,
                'description': description,
                'url': url,
                'transport_type': rtype,
                'auth_headers': auth_headers,
                'all_headers': headers,
            })

        # Also check packages with remote transports
        for pkg in server.get('packages', []):
            transport = pkg.get('transport', {})
            ttype = transport.get('type', '')
            if ttype in ('sse', 'streamable-http'):
                url = transport.get('url', '')
                if url and not any(s['url'] == url for s in servers):
                    servers.append({
                        'name': name,
                        'version': version,
                        'description': description,
                        'url': url,
                        'transport_type': ttype,
                        'auth_headers': [],
                        'all_headers': [],
                    })

    return servers


def make_cursor(data):
    """Construct a cursor from the last server in the page."""
    servers = data.get('servers', [])
    if not servers:
        return None
    last = servers[-1]['server']
    return f"{last['name']}:{last['version']}"


def main():
    all_servers = []
    cursor = None
    page = 0
    seen_names = set()

    while True:
        page += 1
        print(f"Fetching page {page} (cursor: {cursor})...", file=sys.stderr)
        data = fetch_page(cursor)
        if data is None:
            break

        entries = data.get('servers', [])
        if not entries:
            break

        servers = extract_remote_servers(data)
        # Deduplicate - only keep latest version of each server+url combo
        new_servers = []
        for s in servers:
            key = f"{s['name']}|{s['url']}"
            if key not in seen_names:
                seen_names.add(key)
                new_servers.append(s)

        all_servers.extend(new_servers)
        print(f"  Found {len(new_servers)} new remote servers (total: {len(all_servers)})", file=sys.stderr)

        # Check for pagination
        new_cursor = make_cursor(data)
        if new_cursor == cursor or len(entries) < 30:
            break
        cursor = new_cursor

    # Write all servers to JSON for further processing
    with open('/home/user/research/mcplist/registry_servers.json', 'w') as f:
        json.dump(all_servers, f, indent=2)

    print(f"\nTotal remote servers found: {len(all_servers)}", file=sys.stderr)


if __name__ == '__main__':
    main()
