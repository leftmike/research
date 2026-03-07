#!/usr/bin/env python3
"""Analyze MCP server repos for sampling and elicitation support.

Strategy: Download GitHub tarballs and grep for sampling/elicitation patterns.
This is much faster than git clone for large repos.
"""

import io
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
from concurrent.futures import ThreadPoolExecutor, as_completed

# Combined regex for quick first-pass filtering
QUICK_SAMPLING_RE = re.compile(
    r"createMessage|create_message|CreateMessageRequest|CreateMessageResult|"
    r"SamplingCapability|sampling_capability|samplingCapability|"
    r"sampling/createMessage",
    re.IGNORECASE,
)

QUICK_ELICITATION_RE = re.compile(
    r"elicitation/create|ElicitRequest|ElicitResult|"
    r"ElicitationCapability|elicitation_capability|elicitationCapability|"
    r"\.elicit\(|elicitation",
    re.IGNORECASE,
)

# Detailed patterns for match extraction
SAMPLING_PATTERNS = [
    r"sampling/createMessage",
    r"[Cc]reateMessage",
    r"create_message",
    r"CreateMessageRequest",
    r"CreateMessageResult",
    r"SamplingCapability",
    r"sampling_capability",
    r"samplingCapability",
]

ELICITATION_PATTERNS = [
    r"elicitation/create",
    r"ElicitRequest",
    r"ElicitResult",
    r"ElicitationCapability",
    r"elicitation_capability",
    r"elicitationCapability",
    r"\.elicit\(",
    r"elicitation",
]

# File extensions to search
SOURCE_EXTENSIONS = {
    ".py", ".ts", ".js", ".tsx", ".jsx", ".go", ".rs", ".java", ".kt",
    ".cs", ".rb", ".ex", ".exs", ".swift", ".php",
}

CLONE_DIR = "/tmp/mcp_repos"


def extract_owner_repo(repo_url):
    """Extract owner/repo from a GitHub URL."""
    m = re.match(r"https://github\.com/([^/]+)/([^/]+?)(?:\.git)?$", repo_url)
    if m:
        return m.group(1), m.group(2)
    return None, None


def download_and_search(entry, idx, total):
    """Download a repo tarball and search for patterns."""
    repo_url = entry["repo_url"]
    subfolder = entry.get("subfolder", "")

    result = {
        "repo_url": repo_url,
        "server_names": entry["server_names"],
        "sources": entry["sources"],
        "subfolder": subfolder,
        "status": "unknown",
        "sampling_matches": [],
        "elicitation_matches": [],
    }

    owner, repo = extract_owner_repo(repo_url)
    if not owner:
        # Try gitlab or non-standard URL — fall back to git clone
        return clone_and_search(entry, idx, result)

    extract_path = os.path.join(CLONE_DIR, f"repo_{idx}")

    try:
        # Clean up
        if os.path.exists(extract_path):
            shutil.rmtree(extract_path)
        os.makedirs(extract_path, exist_ok=True)

        # Download tarball using curl
        tarball_url = f"https://github.com/{owner}/{repo}/archive/refs/heads/main.tar.gz"
        proc = subprocess.run(
            ["curl", "-sL", "--max-time", "20", "-o", "-", tarball_url],
            capture_output=True, timeout=25,
        )

        if proc.returncode != 0 or len(proc.stdout) < 100:
            # Try master branch
            tarball_url = f"https://github.com/{owner}/{repo}/archive/refs/heads/master.tar.gz"
            proc = subprocess.run(
                ["curl", "-sL", "--max-time", "20", "-o", "-", tarball_url],
                capture_output=True, timeout=25,
            )

        if proc.returncode != 0 or len(proc.stdout) < 100:
            stderr = proc.stderr.decode("utf-8", errors="replace")[:200]
            if "404" in stderr or len(proc.stdout) < 100:
                result["status"] = "not_found"
            elif "403" in stderr:
                result["status"] = "blocked"
            else:
                result["status"] = "download_error"
                result["error"] = stderr
            return result

        # Extract tarball
        try:
            tar_data = io.BytesIO(proc.stdout)
            with tarfile.open(fileobj=tar_data, mode="r:gz") as tar:
                # Security: limit extraction size and filter paths
                members = []
                total_size = 0
                for member in tar.getmembers():
                    if member.isfile():
                        ext = os.path.splitext(member.name)[1].lower()
                        if ext in SOURCE_EXTENSIONS:
                            if member.size < 500000:  # Skip files > 500KB
                                total_size += member.size
                                if total_size < 50_000_000:  # 50MB total limit
                                    members.append(member)
                tar.extractall(extract_path, members=members)
        except (tarfile.TarError, Exception) as e:
            result["status"] = "extract_error"
            result["error"] = str(e)[:200]
            return result

        # Search extracted files
        search_path = extract_path
        if subfolder:
            # Find the extracted directory (GitHub tarballs have a top-level dir)
            entries = os.listdir(extract_path)
            if len(entries) == 1 and os.path.isdir(os.path.join(extract_path, entries[0])):
                base = os.path.join(extract_path, entries[0])
                sub = os.path.join(base, subfolder)
                if os.path.isdir(sub):
                    search_path = sub
                else:
                    search_path = base
            # else search everything

        sampling_matches = []
        elicitation_matches = []

        for root, dirs, files in os.walk(search_path):
            # Skip node_modules etc. (shouldn't be in tarball but just in case)
            dirs[:] = [d for d in dirs if d not in {
                "node_modules", "vendor", "dist", "build", "__pycache__",
                ".next", "target", "venv", ".venv",
            }]

            for fname in files:
                ext = os.path.splitext(fname)[1].lower()
                if ext not in SOURCE_EXTENSIONS:
                    continue

                fpath = os.path.join(root, fname)
                rel_path = os.path.relpath(fpath, extract_path)

                try:
                    with open(fpath, "r", errors="replace") as f:
                        content = f.read()
                except Exception:
                    continue

                # Quick check before detailed pattern matching
                has_sampling = QUICK_SAMPLING_RE.search(content)
                has_elicitation = QUICK_ELICITATION_RE.search(content)

                if not has_sampling and not has_elicitation:
                    continue

                if has_sampling:
                    for pattern in SAMPLING_PATTERNS:
                        for m in re.finditer(f".*{pattern}.*", content):
                            line = m.group().strip()[:200]
                            if is_likely_real_usage(line, "sampling"):
                                sampling_matches.append({
                                    "file": rel_path,
                                    "pattern": pattern,
                                    "line": line,
                                })
                                if len(sampling_matches) > 30:
                                    break
                        if len(sampling_matches) > 30:
                            break

                if has_elicitation:
                    for pattern in ELICITATION_PATTERNS:
                        for m in re.finditer(f".*{pattern}.*", content):
                            line = m.group().strip()[:200]
                            if is_likely_real_usage(line, "elicitation"):
                                elicitation_matches.append({
                                    "file": rel_path,
                                    "pattern": pattern,
                                    "line": line,
                                })
                                if len(elicitation_matches) > 30:
                                    break
                        if len(elicitation_matches) > 30:
                            break

        result["sampling_matches"] = dedupe_matches(sampling_matches)
        result["elicitation_matches"] = dedupe_matches(elicitation_matches)
        result["status"] = "analyzed"

    except subprocess.TimeoutExpired:
        result["status"] = "timeout"
    except Exception as e:
        result["status"] = "error"
        result["error"] = str(e)[:200]
    finally:
        if os.path.exists(extract_path):
            shutil.rmtree(extract_path, ignore_errors=True)

    return result


def clone_and_search(entry, idx, result):
    """Fallback: shallow clone for non-GitHub repos."""
    repo_url = entry["repo_url"]
    clone_path = os.path.join(CLONE_DIR, f"clone_{idx}")

    try:
        if os.path.exists(clone_path):
            shutil.rmtree(clone_path)

        clone_url = repo_url + ".git" if not repo_url.endswith(".git") else repo_url
        proc = subprocess.run(
            ["git", "clone", "--depth", "1", "--single-branch", "-q",
             "--filter=blob:limit=500k", clone_url, clone_path],
            capture_output=True, text=True, timeout=20,
        )
        if proc.returncode != 0:
            err = proc.stderr.strip()[:200]
            result["status"] = "clone_error"
            result["error"] = err
            return result

        # Use git grep for speed
        for feature, patterns in [("sampling", SAMPLING_PATTERNS), ("elicitation", ELICITATION_PATTERNS)]:
            matches = []
            for pattern in patterns:
                try:
                    proc = subprocess.run(
                        ["git", "-C", clone_path, "grep", "-l", "-i", pattern],
                        capture_output=True, text=True, timeout=5,
                    )
                    if proc.returncode == 0:
                        for fpath in proc.stdout.strip().split("\n")[:5]:
                            if fpath and any(fpath.endswith(ext) for ext in SOURCE_EXTENSIONS):
                                matches.append({
                                    "file": fpath,
                                    "pattern": pattern,
                                    "line": f"(matched in {fpath})",
                                })
                except Exception:
                    pass
            result[f"{feature}_matches"] = dedupe_matches(matches)

        result["status"] = "analyzed"
    except subprocess.TimeoutExpired:
        result["status"] = "timeout"
    except Exception as e:
        result["status"] = "error"
        result["error"] = str(e)[:200]
    finally:
        if os.path.exists(clone_path):
            shutil.rmtree(clone_path, ignore_errors=True)

    return result


def is_likely_real_usage(line, feature_type):
    """Filter out false positives."""
    lower = line.lower()

    if feature_type == "sampling":
        # "createMessage" is very specific to MCP sampling
        if "createmessage" in lower:
            return True
        if "create_message" in lower:
            return True
        # "sampling" alone can be a false positive (data sampling, etc.)
        if "sampling" in lower and "createmessage" not in lower and "create_message" not in lower:
            if any(kw in lower for kw in [
                "mcp", "capability", "capabilities", "server", "client",
                "protocol", "context", "session",
            ]):
                return True
            return False
        return True

    if feature_type == "elicitation":
        return True

    return True


def dedupe_matches(matches):
    """Deduplicate matches by file + line."""
    seen = set()
    result = []
    for m in matches:
        key = (m["file"], m["line"])
        if key not in seen:
            seen.add(key)
            result.append(m)
    return result


def classify_usage(matches, feature):
    """Classify whether matches indicate real implementation."""
    if not matches:
        return "none"

    files = set(m["file"] for m in matches)
    lines = [m["line"].lower() for m in matches]

    test_only = all(
        any(x in f.lower() for x in ["test", "spec", "example", "mock", "__test__"])
        for f in files
    )

    import_only = all(
        any(x in line for x in ["import ", "from ", "require(", "export ", "type ", "interface "])
        for line in lines
    )

    sdk_indicators = ["sdk", "lib/", "packages/sdk", "client/"]
    is_sdk = any(any(ind in f.lower() for ind in sdk_indicators) for f in files)

    if test_only:
        return "test_only"
    if import_only:
        return "import_only"
    if is_sdk:
        return "sdk_code"
    return "implementation"


def main():
    repos_path = "/home/user/research/mcpservers/repos.json"
    with open(repos_path) as f:
        repos = json.load(f)

    print(f"Analyzing {len(repos)} repositories...", file=sys.stderr)

    os.makedirs(CLONE_DIR, exist_ok=True)

    all_results = []
    stats = {
        "total": len(repos),
        "analyzed": 0,
        "not_found": 0,
        "blocked": 0,
        "download_error": 0,
        "extract_error": 0,
        "clone_error": 0,
        "timeout": 0,
        "error": 0,
        "has_sampling": 0,
        "has_elicitation": 0,
    }

    with ThreadPoolExecutor(max_workers=15) as executor:
        futures = {
            executor.submit(download_and_search, entry, idx, len(repos)): idx
            for idx, entry in enumerate(repos)
        }

        done = 0
        for future in as_completed(futures):
            done += 1
            result = future.result()
            all_results.append(result)

            status = result["status"]
            if status in stats:
                stats[status] += 1

            if result["sampling_matches"]:
                stats["has_sampling"] += 1
            if result["elicitation_matches"]:
                stats["has_elicitation"] += 1

            if done % 100 == 0:
                print(
                    f"  Progress: {done}/{len(repos)} "
                    f"(analyzed={stats['analyzed']}, not_found={stats['not_found']}, "
                    f"sampling={stats['has_sampling']}, "
                    f"elicitation={stats['has_elicitation']})",
                    file=sys.stderr,
                )

    all_results.sort(key=lambda r: r["repo_url"])

    output_path = "/home/user/research/mcpservers/results.json"
    with open(output_path, "w") as f:
        json.dump(all_results, f, indent=2)

    print(f"\n=== Analysis Summary ===", file=sys.stderr)
    for k, v in stats.items():
        print(f"  {k}: {v}", file=sys.stderr)
    print(f"Results saved to {output_path}", file=sys.stderr)

    generate_report(all_results, stats)


def generate_report(results, stats):
    """Generate a human-readable report."""
    report_path = "/home/user/research/mcpservers/results.txt"

    sampling_results = [r for r in results if r["sampling_matches"]]
    elicitation_results = [r for r in results if r["elicitation_matches"]]

    with open(report_path, "w") as f:
        f.write("MCP Server Source Code Analysis: Sampling & Elicitation Support\n")
        f.write("=" * 78 + "\n\n")

        f.write("METHODOLOGY\n")
        f.write("-" * 78 + "\n")
        f.write("  Repository URLs were collected from three websites:\n")
        f.write("    1. registry.modelcontextprotocol.io (API provides repo URLs)\n")
        f.write("    2. pulsemcp.com (detail pages scraped for GitHub links)\n")
        f.write("    3. github.com/jaw9c/awesome-remote-mcp-servers (README parsed)\n\n")
        f.write("  Each repository was downloaded and searched for patterns indicating\n")
        f.write("  MCP sampling (sampling/createMessage) or elicitation (elicitation/create)\n")
        f.write("  support. Matches were classified as implementation, test-only,\n")
        f.write("  import-only, or SDK code.\n\n")

        f.write("SUMMARY\n")
        f.write("-" * 78 + "\n")
        f.write(f"  Total unique repositories: {stats['total']}\n")
        f.write(f"  Successfully analyzed: {stats['analyzed']}\n")
        f.write(f"  Not found (404/deleted): {stats['not_found']}\n")
        f.write(f"  Blocked (egress proxy): {stats.get('blocked', 0)}\n")
        f.write(f"  Download error: {stats.get('download_error', 0)}\n")
        f.write(f"  Extract error: {stats.get('extract_error', 0)}\n")
        f.write(f"  Clone error: {stats.get('clone_error', 0)}\n")
        f.write(f"  Timeout: {stats['timeout']}\n")
        f.write(f"  Other error: {stats['error']}\n\n")

        f.write(f"  Repos with sampling indicators: {len(sampling_results)}\n")
        f.write(f"  Repos with elicitation indicators: {len(elicitation_results)}\n\n")

        # Classify
        sampling_impl = []
        sampling_other = []
        for r in sampling_results:
            c = classify_usage(r["sampling_matches"], "sampling")
            r["sampling_classification"] = c
            (sampling_impl if c == "implementation" else sampling_other).append(r)

        elicitation_impl = []
        elicitation_other = []
        for r in elicitation_results:
            c = classify_usage(r["elicitation_matches"], "elicitation")
            r["elicitation_classification"] = c
            (elicitation_impl if c == "implementation" else elicitation_other).append(r)

        f.write(f"  Sampling - likely implementation: {len(sampling_impl)}\n")
        f.write(f"  Sampling - other (test/import/SDK): {len(sampling_other)}\n")
        f.write(f"  Elicitation - likely implementation: {len(elicitation_impl)}\n")
        f.write(f"  Elicitation - other (test/import/SDK): {len(elicitation_other)}\n\n")
        f.write("=" * 78 + "\n\n")

        sections = [
            ("SERVERS WITH SAMPLING SUPPORT (likely implementation)", sampling_impl, "sampling"),
            ("SERVERS WITH ELICITATION SUPPORT (likely implementation)", elicitation_impl, "elicitation"),
            ("SERVERS WITH SAMPLING REFERENCES (test/import/SDK only)", sampling_other, "sampling"),
            ("SERVERS WITH ELICITATION REFERENCES (test/import/SDK only)", elicitation_other, "elicitation"),
        ]

        for title, items, feature in sections:
            f.write(f"{title}\n")
            f.write("=" * 78 + "\n\n")
            if items:
                for r in items:
                    write_server_detail(f, r, feature)
            else:
                f.write("  (none found)\n\n")

    print(f"Report saved to {report_path}", file=sys.stderr)

    # Update JSON with classifications
    output_path = "/home/user/research/mcpservers/results.json"
    with open(output_path, "w") as f:
        json.dump(results, f, indent=2)


def write_server_detail(f, result, feature):
    """Write detailed info for one server."""
    f.write(f"  Repository: {result['repo_url']}\n")
    f.write(f"  Server names: {', '.join(result['server_names'])}\n")
    f.write(f"  Sources: {', '.join(result['sources'])}\n")
    if result.get("subfolder"):
        f.write(f"  Subfolder: {result['subfolder']}\n")

    matches = result.get(f"{feature}_matches", [])
    classification = result.get(f"{feature}_classification", "unknown")
    f.write(f"  Classification: {classification}\n")
    f.write(f"  Matches ({len(matches)}):\n")
    for m in matches[:10]:
        f.write(f"    {m['file']}: {m['line']}\n")
    if len(matches) > 10:
        f.write(f"    ... and {len(matches) - 10} more\n")
    f.write("\n")


if __name__ == "__main__":
    main()
