#!/usr/bin/env python3
"""Fetch remote MCP server endpoints from pulsemcp.com."""

import json
import re
import sys
import time
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE_URL = "https://www.pulsemcp.com"

# All remote server slugs from pulsemcp.com/servers?other[]=remote (pages 1-11)
REMOTE_SLUGS = [
    # Page 1
    "upstash-context7", "supabase", "github", "notion", "zapier", "webflow",
    "tavily-search", "firecrawl", "idosal-git-mcp", "stripe-agent-toolkit",
    "coinapi-realtime-exchange-rates", "exa", "sentry", "figma-dev-mode",
    "pinecone-assistant", "prisma-postgres", "generect", "dbt", "vercel",
    "linear", "shopify-storefront", "shopify-customer-accounts", "ovhcloud",
    "hubspot", "make", "atlassian", "ahrefs", "proofly-deepfake-detection",
    "khalidsaidi-a2abench", "timescale-pg-aiguide", "cryptoradi-schemaflow",
    "apify", "pipeboard-meta-ads", "google-bigquery", "monday-com",
    "miantiao-me-bm-md", "khromov-svelte-llm", "aws-knowledge", "asana",
    "findyourfivepm", "alphavantage", "supermemory",
    # Page 2
    "la-rebelion-labs-registry", "nymbo-tools", "plaid", "huggingface",
    "candiceai", "google-maps", "svelte", "deepwiki", "postman", "web-to-mcp",
    "gomarble", "unblocked", "pageindex", "servicebricks", "semrush", "crawleo",
    "fulcradynamics-fulcra-context", "knit", "pulumi", "continue-docs", "canva",
    "trayders-trayd", "lunarcrush", "mercadolibre-documentation", "adspirer",
    "pagerduty", "sideways", "todoist", "stackoverflow", "brightdata-web-scraping",
    "paypal-agent-toolkit", "runreveal", "tip4serv", "pulik-io-memory",
    "congress", "thoughtspot", "activepieces", "replicate",
    "cloudflare-workers-bindings", "cloudflare-docs", "norman-finance",
    "hostinger-seo-checker",
    # Page 3
    "bingowon-apple-rag", "nowledge-mem", "brutus-gr-osint-template", "gitlab",
    "lio1204-what2watch", "mcdonalds-china", "fireflies", "dot",
    "parallel-search", "predev-architect", "mixpanel", "miro", "mapbox-devkit",
    "newrelic", "pga", "clockwise", "gwiz", "noteit", "youcom", "uno-platform",
    "transcribe", "brick-directory", "sonatype", "mcpbundles-hub", "preloop",
    "microsoft-sentinel", "square", "contextual-ai", "reflag",
    "dhenenjay-axion-earth-engine", "peek-travel", "amplitude",
    "plusai-presentations", "perigon", "waystation-postgres",
    "google-workspace-developer-tools", "proxylink", "dmarcdkim", "apix420",
    "plexmcp", "basehub-forums", "semgrep",
    # Page 4
    "cloudflare-dns-analytics", "cloudflare-autorag", "cloudflare-radar",
    "cloudflare-browser-rendering", "cloudflare-workers-observability", "ref",
    "coresignal", "jimmcq-dice-rolling", "rostro", "equinix",
    "wildcard-deepcontext", "box", "lumetra-engram-memory", "mailerlite",
    "mailersend", "parallel-task", "vlei-wiki", "finfeed-historical-stock-data",
    "finfeed-sec-api", "scrapfly", "vonage-documentation", "microsoft-enterprise",
    "saidiibrahim-search-papers",
    "telerik-kendo-ui-angular-generator", "remote-mcp-registry", "close",
    "superloops-un-world-demographics", "earningsfeed", "ai-erd",
    "adrianmikula-jakarta-migration", "automem", "gossiper-shopify-admin",
    "neondatabase-mcp-server-neon", "axiomhq-mcp-server-axiom", "posthog",
    "cloudflare-one-casb", "cloudflare-dex-analysis", "cloudflare-auditlogs",
    "cloudflare-ai-gateway", "cloudflare-container-sandbox", "zekker6-helm",
    "timecamp-org-timecamp",
    # Page 5
    "wesbos-currency-conversion", "withastro-docs", "wtsolutions-excel-to-json",
    "stytch", "ilert", "king-of-the-grackles-reddit", "roadahead1-goweb3",
    "shutter-network-timelock", "siliconsociety-fortuna", "appwrite-docs",
    "devcycle", "saleor-commerce", "knowsync", "chainaware-behavioral-prediction",
    "coinapi-indexes", "coinapi-historical-exchange-rates",
    "finfeed-historical-data", "finfeed-realtime-financial-data", "galileo",
    "vulnebify", "lemonado", "waystation-airtable", "guru", "draup", "opsera",
    "hello-admin", "prefect", "vaadin", "statsig", "mintmcp-outlook-calendar",
    "mintmcp-gcal", "medusa", "jepto", "egnyte", "appsflyer", "ortto",
    "bekservice-famulor", "gh-virgen101-xportalx", "teckel-navigation-toolkit",
    "teckel-ethereum-toolkit", "wire-lennys-podcast", "ndkasndakn-refund-decide",
    # Page 6
    "ground", "gas-library-hub", "devopness", "koreal6803-finlab-ai",
    "janhms-needle", "mcpx", "financial-datasets", "yepcode-secure-execution",
    "llmtxt", "gralio", "kollektiv-document-management", "wix", "himalayas",
    "data-skunks-keywordspeopleuse", "buildkite", "pearl", "hub-tools",
    "freepik", "gumlet", "alby-bitcoin-payments", "listenetic",
    "bifrotek-supabase-http-stream-n8n", "hteek-aws-cost-explorer",
    "biocontext-ai-knowledgebase", "jameswlepage-wordpress-trac", "vapi",
    "iacomunia-coingecko", "liquidmetal-raindrop", "tubasasakunn-context-apps",
    "build-vault", "compose-and-dragons-dungeon-explorer", "tally", "customerio",
    "seolinkmap", "mcpcentral-io-langchain-hub", "opalstack", "shopify-catalog",
    "otto-google-ads", "boikot", "kleros-court", "aharvard-agentic-commerce",
    "exotel",
    # Page 7
    "matsjfunke-paperclip", "heyoncall", "poku-labs",
    "alt250-famxplor-family-travel", "laei-ro", "echo3d", "scalekit",
    "fctolabs-flight-search", "philschmid-gemini", "biel",
    "dam-butler-breville", "namesilo", "kismet-travel",
    "langgraph-nutrition-analyzer", "twelvelabs", "blooio-imessages",
    "aryaminus-h1b-job-search", "rotunda", "uidriver",
    "brokerchooser-broker-safety", "justcall", "bitrix24", "currenttimeutc",
    "mobile-text-alerts", "coinapi-finfeedapi",
    "swarm-corporation-medical-agents-aop", "semilattice", "howrisky",
    "tencent-prosearch", "subbu3012-kogna", "aws-managed", "mcp-analytics",
    "packmind", "shawndurrani-registry", "sophtron",
    "f-prompts-chat", "exoquery", "socialapis", "wire-chocolate-recipes",
    "arca", "tradeit", "mmorris35-devplan",
    # Page 8
    "clouatre-labs-math-learning", "cranot-agentskb", "secureprivacy", "linkly",
    "tweekit", "gander-tools-osm-tagging-schema", "isakskogstad-oecd",
    "isakskogstad-kolada", "aquaview", "trunk", "testiny", "scorecard",
    "navifare", "mia-platform-console", "mcpcentral-time", "mapbox",
    "isakskogstad-scb", "humanjesse-textarttools", "ax-platform",
    "augmnt-augments", "onlyoffice-docspace", "ksaklfszf921-skolverket",
    "github30-qiita", "github30-note-com", "catchmetrics", "zomato", "webforj",
    "teamwork-go", "predictleads", "mintmcp-outlook-email",
    "thinkchainai-agent-interviews", "enigma", "serkan-ozal-driflyte",
    "sideways-1", "blogcaster", "askman-agent-never-give-up", "mermaid-chart",
    "mintmcp-gmail", "redpanda-docs", "forex-gpt", "xtended",
    "sungminwoo0612-toolbartender",
    # Page 9
    "llaa33219-kakaotalk-emoticons", "lightdash", "esagu", "dock-ai", "dice",
    "bright-security", "enzyme", "profitelligence", "lilo-property", "okahu",
    "aistatusdashboard", "html-css-to-image", "scanmalware", "clickup",
    "apiiro-guardian", "twelve-data", "lapalma24", "google-mcp",
    "jakobwennberg-fortnox", "domainkits", "stayce-partd", "signal-relay",
    "oeradio", "bitte-protocol", "onecontext", "cloudflare-logpush", "intercom",
    "asrvd-flux-ao-arweave", "graphlit-mcpoogle", "jsdelivr-globalping",
    "simplescraper", "beatandraise-sec-filings", "mercado-pago", "windsor",
    "ticket-tailor", "job-stock-analysis-nse", "io-aerospace", "kiwi-flights",
    "youtube2text", "mxhero-mail2cloud-advanced", "themissinglinkhub",
    "companies-in-the-uk",
    # Page 10
    "particlefuture-1mcpserver", "dmontgomery40-faxbot", "hmr-docs", "qching",
    "twig-rag-agents", "ai-archive", "meminal", "florentine", "waystation-miro",
    "martinelli-jooq", "wire-christmas-carol", "katalon-testops",
    "gepuro-company-lens", "ohmyposh", "thisdot-docusign-navigator", "cycloid",
    "ventureforges-growth-forecast", "nomadstays",
    "courtneyr-dev-wordpress-trac", "well", "salrad22-code-sentinel",
    "balajsaleem-criterion", "jxbrowser", "mercurialsolo-counsel", "eansearch",
    "klaviyo", "searchapi", "outris-identity", "deeprecall",
    "akshayvkt-lenny-podcast", "unified-offer-protocol", "mercury",
    "civic-nexus", "leni", "waystation-jira", "waystation-office",
    "waystation-supabase", "waystation-wrike", "dialer", "payram-helper",
    "jinko-gojinko", "blaide-cookwith",
    # Page 11
    "jinko-mcp", "installmd-try", "snapcall", "ignission", "foqal",
    "windowsforum", "tableall", "syncline", "cryptorefills", "1stdibs",
    "outrun", "wishfinity", "vivideo", "billychl1-footballbin", "game-assets",
    "lexsocket-ted", "granola", "neglect-solana", "gopluto-ai", "alkemi-data",
    "paracetamol951-kash-click", "subconscious-ai", "riksdag-regering",
    "xcatcher", "balldontlie", "kaewz-manga-wordpress-nodeflow", "penfield-mcp",
    "cdata-connect-ai", "skybridge-capitals", "audio-intelligence", "agents",
    "n-3inc-e-stat", "thousandeyes", "amazon-eks", "web-agent", "idealift",
    "123elec", "llmse", "stayce-shortcut", "zeroheight", "planetscale",
]


def fetch_server_json(slug):
    """Fetch serverjson page and extract remote endpoints."""
    url = f"{BASE_URL}/servers/{slug}/serverjson"
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
        with urllib.request.urlopen(req, timeout=15) as resp:
            content = resp.read().decode("utf-8", errors="replace")

        # Find the JSON blob in a <script> tag containing versions and remotes
        scripts = re.findall(r'<script[^>]*>(.*?)</script>', content, re.DOTALL)
        data = None
        for script_content in scripts:
            stripped = script_content.strip()
            if '"versions"' not in stripped:
                continue
            try:
                data = json.loads(stripped)
                break
            except (json.JSONDecodeError, ValueError):
                continue
        if data is None:
            return slug, None, "no versions JSON found"
        versions = data.get("versions", [])
        if not versions:
            return slug, None, "empty versions"

        # Use the latest version (index 0)
        latest = versions[0].get("data", {})
        title = latest.get("title", latest.get("name", slug))
        remotes = latest.get("remotes", [])

        if not remotes:
            return slug, None, "no remotes"

        endpoints = []
        for r in remotes:
            ep_url = r.get("url", "")
            ep_type = r.get("type", "streamable-http")
            if ep_url:
                endpoints.append({
                    "name": title,
                    "slug": slug,
                    "url": ep_url,
                    "transport": ep_type,
                })

        return slug, endpoints, None

    except urllib.error.HTTPError as e:
        return slug, None, f"HTTP {e.code}"
    except urllib.error.URLError as e:
        return slug, None, f"URL error: {e.reason}"
    except Exception as e:
        return slug, None, str(e)


def main():
    print(f"Fetching server.json for {len(REMOTE_SLUGS)} remote servers...")

    all_endpoints = []
    errors = []

    with ThreadPoolExecutor(max_workers=20) as executor:
        futures = {executor.submit(fetch_server_json, slug): slug for slug in REMOTE_SLUGS}
        done = 0
        for future in as_completed(futures):
            done += 1
            slug, endpoints, error = future.result()
            if endpoints:
                all_endpoints.extend(endpoints)
            if error:
                errors.append((slug, error))
            if done % 50 == 0:
                print(f"  Progress: {done}/{len(REMOTE_SLUGS)} done, "
                      f"{len(all_endpoints)} endpoints found...")

    print(f"\nDone. Found {len(all_endpoints)} remote endpoints from "
          f"{len(REMOTE_SLUGS)} servers.")
    print(f"Errors/skipped: {len(errors)}")

    # Save results
    output_path = "/home/user/research/mcplist/pulsemcp_servers.json"
    with open(output_path, "w") as f:
        json.dump(all_endpoints, f, indent=2)
    print(f"Saved to {output_path}")

    # Print summary of errors
    if errors:
        print("\nErrors:")
        for slug, err in sorted(errors):
            print(f"  {slug}: {err}")


if __name__ == "__main__":
    main()
