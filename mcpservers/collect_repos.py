#!/usr/bin/env python3
"""Collect source code repository URLs for MCP servers from three websites."""

import json
import re
import subprocess
import sys
import time
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed


def fetch_url(url, timeout=15):
    """Fetch a URL and return the response text."""
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode("utf-8", errors="replace")


def fetch_registry_repos():
    """Fetch all repos from registry.modelcontextprotocol.io API."""
    print("=== Fetching from registry.modelcontextprotocol.io ===", file=sys.stderr)
    all_repos = []
    cursor = None
    page = 0

    while True:
        page += 1
        url = "https://registry.modelcontextprotocol.io/v0.1/servers"
        if cursor:
            url += f"?cursor={cursor}"

        result = subprocess.run(
            ["curl", "-s", url], capture_output=True, text=True, timeout=30
        )
        if result.returncode != 0 or not result.stdout.strip():
            break

        try:
            data = json.loads(result.stdout)
        except json.JSONDecodeError:
            break

        entries = data.get("servers", [])
        if not entries:
            break

        for entry in entries:
            server = entry.get("server", {})
            name = server.get("name", "unknown")
            repo = server.get("repository", {})
            repo_url = repo.get("url", "")
            subfolder = repo.get("subfolder", "")

            if repo_url and ("github.com" in repo_url or "gitlab.com" in repo_url):
                all_repos.append({
                    "server_name": name,
                    "repo_url": normalize_repo_url(repo_url),
                    "subfolder": subfolder,
                    "source": "registry",
                })

        if page % 5 == 0:
            print(f"  Page {page}, {len(all_repos)} repos so far...", file=sys.stderr)

        # Pagination
        last = entries[-1]["server"]
        new_cursor = f"{last['name']}:{last['version']}"
        if new_cursor == cursor or len(entries) < 30:
            break
        cursor = new_cursor

    print(f"  Registry: found {len(all_repos)} repos", file=sys.stderr)
    return all_repos


def normalize_repo_url(url):
    """Normalize a GitHub/GitLab URL to a consistent format."""
    url = url.rstrip("/")
    # Remove .git suffix
    if url.endswith(".git"):
        url = url[:-4]
    # Remove tree/branch paths
    url = re.sub(r"/tree/[^/]+.*$", "", url)
    # Remove blob paths
    url = re.sub(r"/blob/[^/]+.*$", "", url)
    # Ensure https
    url = re.sub(r"^http://", "https://", url)
    return url


def fetch_pulsemcp_repo(slug):
    """Fetch a pulsemcp detail page and extract GitHub/GitLab repo links."""
    url = f"https://www.pulsemcp.com/servers/{slug}"
    try:
        content = fetch_url(url, timeout=15)
        # Look for GitHub/GitLab links
        links = re.findall(
            r'href="(https://(?:github|gitlab)\.com/[^"]+)"', content
        )
        # Filter to likely repo URLs (not issues, pulls, etc.)
        repos = []
        for link in links:
            normalized = normalize_repo_url(link)
            # Must have at least owner/repo pattern
            parts = normalized.replace("https://github.com/", "").replace(
                "https://gitlab.com/", ""
            ).split("/")
            if len(parts) >= 2 and parts[0] and parts[1]:
                # Skip issue/PR/wiki/releases links
                if not any(
                    x in normalized
                    for x in ["/issues", "/pull", "/wiki", "/releases",
                              "/actions", "/discussions", "/commits",
                              "/compare", "/stargazers", "/network"]
                ):
                    repos.append(normalized)
        # Deduplicate preserving order
        seen = set()
        unique = []
        for r in repos:
            if r not in seen:
                seen.add(r)
                unique.append(r)
        return slug, unique, None
    except Exception as e:
        return slug, [], str(e)


def fetch_pulsemcp_repos():
    """Fetch repos from pulsemcp.com detail pages."""
    print("=== Fetching from pulsemcp.com ===", file=sys.stderr)

    # Load slugs from the existing fetch script
    slugs_path = "/home/user/research/mcplist/fetch_pulsemcp.py"
    with open(slugs_path) as f:
        content = f.read()
    # Extract REMOTE_SLUGS list
    match = re.search(r"REMOTE_SLUGS\s*=\s*\[(.*?)\]", content, re.DOTALL)
    if not match:
        print("  ERROR: could not extract slugs", file=sys.stderr)
        return []
    slug_text = match.group(1)
    slugs = re.findall(r'"([^"]+)"', slug_text)

    all_repos = []
    errors = []

    with ThreadPoolExecutor(max_workers=20) as executor:
        futures = {
            executor.submit(fetch_pulsemcp_repo, slug): slug for slug in slugs
        }
        done = 0
        for future in as_completed(futures):
            done += 1
            slug, repos, error = future.result()
            for repo_url in repos:
                all_repos.append({
                    "server_name": slug,
                    "repo_url": repo_url,
                    "subfolder": "",
                    "source": "pulsemcp",
                })
            if error:
                errors.append((slug, error))
            if done % 50 == 0:
                print(
                    f"  Progress: {done}/{len(slugs)}, {len(all_repos)} repos...",
                    file=sys.stderr,
                )

    print(f"  PulseMCP: found {len(all_repos)} repos from {len(slugs)} slugs "
          f"({len(errors)} errors)", file=sys.stderr)
    return all_repos


def fetch_awesome_repos():
    """Fetch repos from awesome-remote-mcp-servers README."""
    print("=== Fetching from awesome-remote-mcp-servers ===", file=sys.stderr)

    url = "https://raw.githubusercontent.com/jaw9c/awesome-remote-mcp-servers/main/README.md"
    try:
        content = fetch_url(url, timeout=15)
    except Exception as e:
        print(f"  ERROR fetching README: {e}", file=sys.stderr)
        return []

    all_repos = []
    # Parse markdown links: - [Name](url) - description
    # or | [Name](url) | description |
    lines = content.split("\n")
    for line in lines:
        # Find all markdown links in the line
        links = re.findall(r'\[([^\]]+)\]\((https://(?:github|gitlab)\.com/[^)]+)\)', line)
        for name, link in links:
            normalized = normalize_repo_url(link)
            parts = normalized.replace("https://github.com/", "").replace(
                "https://gitlab.com/", ""
            ).split("/")
            if len(parts) >= 2 and parts[0] and parts[1]:
                all_repos.append({
                    "server_name": name,
                    "repo_url": normalized,
                    "subfolder": "",
                    "source": "awesome",
                })

    print(f"  Awesome list: found {len(all_repos)} repos", file=sys.stderr)
    return all_repos


def consolidate(registry_repos, pulsemcp_repos, awesome_repos):
    """Consolidate and deduplicate repos across sources."""
    # Group by repo_url
    by_url = {}
    for entry in registry_repos + pulsemcp_repos + awesome_repos:
        url = entry["repo_url"]
        if url not in by_url:
            by_url[url] = {
                "repo_url": url,
                "server_names": [],
                "sources": set(),
                "subfolder": entry.get("subfolder", ""),
            }
        by_url[url]["server_names"].append(entry["server_name"])
        by_url[url]["sources"].add(entry["source"])
        # Prefer non-empty subfolder
        if entry.get("subfolder") and not by_url[url]["subfolder"]:
            by_url[url]["subfolder"] = entry["subfolder"]

    # Convert to list and make JSON-serializable
    result = []
    for url, data in sorted(by_url.items()):
        # Deduplicate server names
        names = list(dict.fromkeys(data["server_names"]))
        result.append({
            "repo_url": data["repo_url"],
            "server_names": names,
            "sources": sorted(data["sources"]),
            "subfolder": data["subfolder"],
        })

    return result


def main():
    registry_repos = fetch_registry_repos()
    pulsemcp_repos = fetch_pulsemcp_repos()
    awesome_repos = fetch_awesome_repos()

    consolidated = consolidate(registry_repos, pulsemcp_repos, awesome_repos)

    output_path = "/home/user/research/mcpservers/repos.json"
    with open(output_path, "w") as f:
        json.dump(consolidated, f, indent=2)

    # Summary
    total = len(consolidated)
    by_source = {"registry": 0, "pulsemcp": 0, "awesome": 0}
    for entry in consolidated:
        for src in entry["sources"]:
            by_source[src] += 1

    print(f"\n=== Summary ===", file=sys.stderr)
    print(f"Total unique repos: {total}", file=sys.stderr)
    print(f"  From registry: {by_source['registry']}", file=sys.stderr)
    print(f"  From pulsemcp: {by_source['pulsemcp']}", file=sys.stderr)
    print(f"  From awesome list: {by_source['awesome']}", file=sys.stderr)
    print(f"Saved to {output_path}", file=sys.stderr)


if __name__ == "__main__":
    main()
