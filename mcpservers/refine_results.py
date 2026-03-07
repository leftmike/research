#!/usr/bin/env python3
"""Refine the raw analysis results to filter false positives and produce a clean report.

False positive patterns identified:
- createMessage used for generic message creation (Slack, Intercom, PostHog, WhatsApp, etc.)
- createMessageConnection from JSON-RPC library (LSP clients)
- createMessageBus for pub/sub patterns
- createMessageSignature for crypto signing
- .smithery/shttp/module.js bundled files (minified SDK code, not server implementation)
- openpgp.createMessage for PGP encryption
- Tetrad's create_message for MCP transport framing (not sampling)
- Generic createMessage in non-MCP contexts

True positive indicators for sampling:
- "sampling/createMessage" literal string
- Import of CreateMessageRequest/Result from MCP SDK
- session.create_message() calls
- server.createMessage() calls in MCP context
- SamplingCapability references

True positive indicators for elicitation:
- "elicitation/create" literal string
- Import of ElicitRequest/Result from MCP SDK
- ctx.elicit() calls
- ElicitationCapability references
- elicitInput() calls

False positive patterns for elicitation:
- "felicitations" (French word)
- Generic "elicitation" in non-MCP contexts
"""

import json
import re
import sys


def is_real_sampling(result):
    """Check if sampling matches are real MCP sampling usage."""
    matches = result.get("sampling_matches", [])
    if not matches:
        return False, []

    real_matches = []
    for m in matches:
        line = m["line"].lower()
        fname = m["file"].lower()

        # Skip minified/bundled files
        if ".smithery/" in fname or "module.js" in fname and len(line) > 150:
            continue

        # Skip JSON-RPC createMessageConnection (LSP, not MCP sampling)
        if "createmessageconnection" in line:
            continue

        # Skip createMessageBus (pub/sub, not MCP sampling)
        if "createmessagebus" in line or "message-bus" in line or "messagebus" in line:
            continue

        # Skip createMessageSignature (crypto, not MCP sampling)
        if "createmessagesignature" in line or "signature" in line:
            continue

        # Skip openpgp.createMessage (PGP, not MCP sampling)
        if "openpgp" in line:
            continue

        # Skip generic create_message that is clearly not MCP sampling
        # e.g., Slack/Intercom/WhatsApp/Gotify message creation
        if "create_message" in line or "createmessage" in line:
            # These are strong MCP sampling indicators
            if any(kw in line for kw in [
                "sampling/createmessage", "samplingmessage",
                "createmessagerequestparam", "createmessagerequest",
                "createmessageresult", "createmessagerequestschema",
                "createmessageresultschema",
                "samplingcapability", "sampling_capability",
                "session.create_message", ".createmessage(",
                "sampling", "sample",
            ]):
                real_matches.append(m)
                continue

            # Also strong: file is in a sampling-related path
            if any(kw in fname for kw in ["sampling", "sample"]):
                real_matches.append(m)
                continue

            # Skip: Slack, Intercom, WhatsApp, Gotify, PostHog, etc.
            if any(kw in fname for kw in [
                "slack", "intercom", "whatsapp", "gotify", "zulip",
                "session_replay", "session-replay", "message_template",
                "genie", "databricks",
            ]):
                continue

            # Skip: generic function definitions / API wrappers not related to MCP sampling
            if any(kw in line for kw in [
                "def create_message(", "func createmessage(",
                "function createmessage(", "createMessagePayload",
                "create_message_entity", "create_message_workflow",
                "_create_message",  # internal helper
                "handlers.message.createmessage",  # API wrapper (e.g. Intercom)
                "intercom_create_message",  # Intercom API
                "universalprovider.createmessage",  # generic LLM provider
                "handler.createmessage(",  # generic handler
                "this.createmessage(",  # generic method call
                "createMessage(providerid",  # provider dispatch
            ]):
                # But keep if it's in a sampling file
                if "sampling" not in fname:
                    continue

            # Skip: test helpers named createMessage
            if "function createmessage(" in line and "test" in fname:
                continue

        # For SamplingCapability - always real
        if "samplingcapability" in line:
            real_matches.append(m)
            continue

        # For create_message in sampling context
        if "sampling" in fname or "sampling" in line:
            real_matches.append(m)
            continue

        # For method definitions referencing sampling/createMessage
        if "sampling/createmessage" in line:
            real_matches.append(m)
            continue

        # If we get here, it's likely a borderline case - include if MCP-related
        if any(kw in line for kw in ["mcp", "protocol", "capability"]):
            real_matches.append(m)
            continue

    return len(real_matches) > 0, real_matches


def is_real_elicitation(result):
    """Check if elicitation matches are real MCP elicitation usage."""
    matches = result.get("elicitation_matches", [])
    if not matches:
        return False, []

    real_matches = []
    for m in matches:
        line = m["line"].lower()
        fname = m["file"].lower()

        # Skip minified/bundled files
        if ".smithery/" in fname or "module.js" in fname and len(line) > 150:
            continue

        # Skip "felicitations" (French word)
        if "felicitation" in line:
            continue

        # All other elicitation references are likely real (it's MCP-specific)
        real_matches.append(m)

    return len(real_matches) > 0, real_matches


def classify_refined(matches, feature):
    """Classify refined matches."""
    if not matches:
        return "none"

    files = set(m["file"] for m in matches)
    lines = [m["line"].lower() for m in matches]

    # Check if ALL files are tests
    all_test = all(
        any(x in f.lower() for x in ["test", "spec", "example", "mock", "__test__"])
        for f in files
    )

    # Check if ALL lines are imports/exports
    all_import = all(
        any(x in line for x in ["import ", "from ", "require(", "export ", "type ", "interface "])
        and not any(x in line for x in ["=", "()", "await ", "return "])
        for line in lines
    )

    # Check if it's docs-only
    all_docs = all(
        any(x in f.lower() for x in ["doc", "readme", "changelog", "2025-"])
        for f in files
    )

    if all_docs:
        return "docs_only"
    if all_test:
        return "test_only"
    if all_import:
        return "import_only"

    # Check for actual implementation (not just types/docs)
    has_impl = False
    for m in matches:
        line = m["line"].lower()
        fname = m["file"].lower()
        if "test" not in fname and "spec" not in fname and "doc" not in fname:
            if any(x in line for x in [
                "await ", "return ", "= await", ".createmessage(",
                "create_message(", ".elicit(", "elicitation/create",
                "sampling/createmessage", "method:",
            ]):
                has_impl = True
                break

    return "implementation" if has_impl else "reference"


def main():
    results_path = "/home/user/research/mcpservers/results.json"
    with open(results_path) as f:
        results = json.load(f)

    sampling_real = []
    elicitation_real = []

    for r in results:
        is_samp, samp_matches = is_real_sampling(r)
        is_elic, elic_matches = is_real_elicitation(r)

        if is_samp:
            entry = {
                "repo_url": r["repo_url"],
                "server_names": r["server_names"],
                "sources": r["sources"],
                "subfolder": r.get("subfolder", ""),
                "matches": samp_matches,
                "classification": classify_refined(samp_matches, "sampling"),
            }
            sampling_real.append(entry)

        if is_elic:
            entry = {
                "repo_url": r["repo_url"],
                "server_names": r["server_names"],
                "sources": r["sources"],
                "subfolder": r.get("subfolder", ""),
                "matches": elic_matches,
                "classification": classify_refined(elic_matches, "elicitation"),
            }
            elicitation_real.append(entry)

    # Count analyzed
    analyzed = sum(1 for r in results if r["status"] == "analyzed")
    not_found = sum(1 for r in results if r["status"] == "not_found")

    # Generate report
    report_path = "/home/user/research/mcpservers/results.txt"
    with open(report_path, "w") as f:
        f.write("MCP Server Source Code Analysis: Sampling & Elicitation Support\n")
        f.write("=" * 78 + "\n\n")

        f.write("METHODOLOGY\n")
        f.write("-" * 78 + "\n")
        f.write("  Repository URLs were collected from three sources:\n")
        f.write("    1. registry.modelcontextprotocol.io - API provides repo URLs directly\n")
        f.write("       (primary source: 2424 unique repos)\n")
        f.write("    2. pulsemcp.com - detail pages blocked by egress proxy (0 repos)\n")
        f.write("    3. github.com/jaw9c/awesome-remote-mcp-servers - README parsed\n")
        f.write("       (4 additional unique repos)\n\n")
        f.write("  Each repository was downloaded (GitHub tarball) and searched for\n")
        f.write("  patterns indicating MCP sampling or elicitation support.\n")
        f.write("  Results were filtered to remove false positives (generic\n")
        f.write("  createMessage functions, minified SDK bundles, etc.) and classified\n")
        f.write("  as implementation, test-only, docs-only, import-only, or reference.\n\n")

        f.write("SUMMARY\n")
        f.write("-" * 78 + "\n")
        f.write(f"  Total unique repositories: {len(results)}\n")
        f.write(f"  Successfully analyzed: {analyzed}\n")
        f.write(f"  Not found (404/deleted): {not_found}\n")
        f.write(f"  Other errors: {len(results) - analyzed - not_found}\n\n")

        # Classify
        samp_impl = [r for r in sampling_real if r["classification"] == "implementation"]
        samp_ref = [r for r in sampling_real if r["classification"] == "reference"]
        samp_test = [r for r in sampling_real if r["classification"] == "test_only"]
        samp_docs = [r for r in sampling_real if r["classification"] == "docs_only"]
        samp_import = [r for r in sampling_real if r["classification"] == "import_only"]

        elic_impl = [r for r in elicitation_real if r["classification"] == "implementation"]
        elic_ref = [r for r in elicitation_real if r["classification"] == "reference"]
        elic_test = [r for r in elicitation_real if r["classification"] == "test_only"]
        elic_docs = [r for r in elicitation_real if r["classification"] == "docs_only"]
        elic_import = [r for r in elicitation_real if r["classification"] == "import_only"]

        f.write(f"  SAMPLING (after filtering false positives):\n")
        f.write(f"    Total repos with real MCP sampling indicators: {len(sampling_real)}\n")
        f.write(f"      Implementation: {len(samp_impl)}\n")
        f.write(f"      Reference (types/config): {len(samp_ref)}\n")
        f.write(f"      Test only: {len(samp_test)}\n")
        f.write(f"      Docs only: {len(samp_docs)}\n")
        f.write(f"      Import only: {len(samp_import)}\n\n")

        f.write(f"  ELICITATION (after filtering false positives):\n")
        f.write(f"    Total repos with real MCP elicitation indicators: {len(elicitation_real)}\n")
        f.write(f"      Implementation: {len(elic_impl)}\n")
        f.write(f"      Reference (types/config): {len(elic_ref)}\n")
        f.write(f"      Test only: {len(elic_test)}\n")
        f.write(f"      Docs only: {len(elic_docs)}\n")
        f.write(f"      Import only: {len(elic_import)}\n\n")

        # Both
        both_impl = set(r["repo_url"] for r in samp_impl) & set(r["repo_url"] for r in elic_impl)
        f.write(f"  Repos implementing BOTH sampling AND elicitation: {len(both_impl)}\n\n")

        f.write("=" * 78 + "\n\n")

        sections = [
            ("SERVERS WITH SAMPLING IMPLEMENTATION", samp_impl),
            ("SERVERS WITH ELICITATION IMPLEMENTATION", elic_impl),
            ("SERVERS WITH SAMPLING REFERENCES (test/docs/import/config)", samp_ref + samp_test + samp_docs + samp_import),
            ("SERVERS WITH ELICITATION REFERENCES (test/docs/import/config)", elic_ref + elic_test + elic_docs + elic_import),
        ]

        for title, items in sections:
            f.write(f"{title}\n")
            f.write("-" * 78 + "\n\n")
            if items:
                for r in sorted(items, key=lambda x: x["repo_url"]):
                    f.write(f"  Repository: {r['repo_url']}\n")
                    f.write(f"  Server names: {', '.join(r['server_names'][:3])}")
                    if len(r['server_names']) > 3:
                        f.write(f" (+{len(r['server_names'])-3} more)")
                    f.write("\n")
                    f.write(f"  Sources: {', '.join(r['sources'])}\n")
                    if r.get("subfolder"):
                        f.write(f"  Subfolder: {r['subfolder']}\n")
                    f.write(f"  Classification: {r['classification']}\n")
                    f.write(f"  Evidence ({len(r['matches'])} matches):\n")
                    for m in r["matches"][:5]:
                        f.write(f"    {m['file']}:\n")
                        f.write(f"      {m['line'][:150]}\n")
                    if len(r["matches"]) > 5:
                        f.write(f"    ... and {len(r['matches']) - 5} more matches\n")
                    f.write("\n")
            else:
                f.write("  (none found)\n\n")

    print(f"Report saved to {report_path}", file=sys.stderr)
    print(f"\nRefined results:", file=sys.stderr)
    print(f"  Sampling implementation: {len(samp_impl)}", file=sys.stderr)
    print(f"  Sampling other: {len(sampling_real) - len(samp_impl)}", file=sys.stderr)
    print(f"  Elicitation implementation: {len(elic_impl)}", file=sys.stderr)
    print(f"  Elicitation other: {len(elicitation_real) - len(elic_impl)}", file=sys.stderr)

    # Save refined JSON
    refined = {
        "summary": {
            "total_repos": len(results),
            "analyzed": analyzed,
            "not_found": not_found,
            "sampling_implementation": len(samp_impl),
            "sampling_reference": len(sampling_real) - len(samp_impl),
            "elicitation_implementation": len(elic_impl),
            "elicitation_reference": len(elicitation_real) - len(elic_impl),
            "both_implementation": len(both_impl),
        },
        "sampling_implementation": samp_impl,
        "sampling_other": samp_ref + samp_test + samp_docs + samp_import,
        "elicitation_implementation": elic_impl,
        "elicitation_other": elic_ref + elic_test + elic_docs + elic_import,
    }
    refined_path = "/home/user/research/mcpservers/results_refined.json"
    with open(refined_path, "w") as f:
        json.dump(refined, f, indent=2)
    print(f"Refined JSON saved to {refined_path}", file=sys.stderr)


if __name__ == "__main__":
    main()
