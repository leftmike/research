#!/usr/bin/env python3
"""Fetch remote MCP servers from smithery.ai registry API."""

import json
import time
import urllib.request
import urllib.error

BASE_URL = "https://registry.smithery.ai/servers"
PAGE_SIZE = 100
OUTPUT_FILE = "/home/user/research/mcplist/smithery_servers.json"


def fetch_page(page: int) -> dict:
    url = f"{BASE_URL}?q=&pageSize={PAGE_SIZE}&page={page}"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read())


def get_server_connection(qualified_name: str) -> dict | None:
    """Fetch individual server details to get connection URL."""
    url = f"https://registry.smithery.ai/servers/{qualified_name}"
    try:
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=15) as resp:
            return json.loads(resp.read())
    except Exception as e:
        return None


def main():
    all_servers = []
    page = 1

    print("Fetching server list from smithery.ai registry...")
    while True:
        try:
            data = fetch_page(page)
        except Exception as e:
            print(f"  Error fetching page {page}: {e}")
            break

        servers = data.get("servers", [])
        if not servers:
            break

        # Only keep remote/deployed servers
        remote_servers = [s for s in servers if s.get("remote") and s.get("isDeployed")]
        all_servers.extend(remote_servers)

        total_count = data.get("totalCount", 0)
        print(f"  Page {page}: {len(servers)} servers ({len(remote_servers)} remote), total so far: {len(all_servers)} / {total_count}")

        if len(all_servers) >= total_count or len(servers) < PAGE_SIZE:
            break
        page += 1
        time.sleep(0.2)

    print(f"\nFound {len(all_servers)} remote/deployed servers total")

    # Now fetch connection details for each server
    print("\nFetching connection details for each server...")
    servers_with_urls = []

    for i, server in enumerate(all_servers):
        qname = server.get("qualifiedName", "")
        display_name = server.get("displayName", qname)
        print(f"  [{i+1}/{len(all_servers)}] {display_name} ({qname})")

        details = get_server_connection(qname)
        if details:
            connections = details.get("connections", [])
            for conn in connections:
                conn_type = conn.get("type", "")
                config_schema = conn.get("configSchema", {})
                url = conn.get("url", "")

                # Skip if no URL or has required config (means auth/apikey required for URL generation)
                if not url:
                    continue

                # Determine transport
                if conn_type == "sse" or url.endswith("/sse") or url.endswith("/mcpSse"):
                    transport = "sse"
                else:
                    transport = "streamable-http"

                servers_with_urls.append({
                    "name": display_name,
                    "qualifiedName": qname,
                    "url": url,
                    "transport": transport,
                    "conn_type": conn_type,
                })
        else:
            # Use homepage URL pattern as fallback
            pass

        time.sleep(0.1)

    print(f"\nFound {len(servers_with_urls)} endpoints with URLs")

    with open(OUTPUT_FILE, "w") as f:
        json.dump(servers_with_urls, f, indent=2)
    print(f"Saved to {OUTPUT_FILE}")


if __name__ == "__main__":
    main()
