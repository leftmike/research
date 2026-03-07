# Enterprise Cybersecurity Products with Official Open Source MCP Servers

> Research conducted: March 2026
> Model Context Protocol (MCP) is an open standard introduced by Anthropic (November 2024),
> now governed by the Linux Foundation / Agentic AI Foundation (AAIF), co-founded by Anthropic,
> Block, and OpenAI.

---

## Overview

This document catalogs enterprise cybersecurity vendors that have published **official** open
source MCP servers. "Official" means the repository is owned/maintained by the vendor's own
GitHub organization, or explicitly attributed to the vendor in an official product announcement.
Community forks and third-party re-implementations are noted separately.

---

## 1. Google — Security Operations Suite

**GitHub:** [`google/mcp-security`](https://github.com/google/mcp-security)
**License:** Apache 2.0
**Status:** Generally Available
**Announcement:** [Google Cloud Security Community post](https://security.googlecloudcommunity.com/google-security-operations-2/build-with-google-cloud-security-mcp-servers-790)

### Covered Products
| Sub-server | Product |
|---|---|
| Google Security Operations (Chronicle) | SIEM — threat detection, investigation, hunting |
| Google Security Operations SOAR | Security orchestration, automation & response |
| Google Threat Intelligence (GTI) | Threat intelligence feeds & enrichment |
| Security Command Center (SCC) | Cloud security posture & risk management |

### Key Capabilities
- Each sub-server can be run independently
- Natural language queries against Chronicle security data lake
- SOAR playbook triggering via MCP tools
- GTI threat-actor and indicator lookups

---

## 2. CrowdStrike — Falcon Platform

### 2a. Falcon MCP
**GitHub:** [`CrowdStrike/falcon-mcp`](https://github.com/CrowdStrike/falcon-mcp)
**Status:** Public Preview (released August 2025)
**Deployable on:** AWS Bedrock AgentCore
**Install:** `uv tool install falcon-mcp` or `pip install falcon-mcp`
**Announcement:** [PulseMCP listing](https://www.pulsemcp.com/servers/crowdstrike-falcon); [Medium write-up](https://mikecybersec.medium.com/supercharged-secops-series-crowdstrike-falcon-mcp-pub-preview-efe5f11c31c2)

Connects AI agents to the CrowdStrike Falcon platform for automated security analysis. Modules are opt-in via `--modules` flag or `FALCON_MCP_MODULES` env var; all modules enabled by default.

#### Modules and Tools

| Module | Tools |
|---|---|
| **Core** | `falcon_check_connectivity`, `falcon_list_modules`, `falcon_list_enabled_modules` |
| **Detections** | `falcon_search_detections`, `falcon_get_detection_details` |
| **Incidents** | `falcon_search_incidents`, `falcon_get_incident_details`, `falcon_search_behaviors`, `falcon_get_behavior_details`, `falcon_show_crowd_score` |
| **Hosts** | `falcon_search_hosts`, `falcon_get_host_details` |
| **Intel** | `falcon_search_actors`, `falcon_search_indicators`, `falcon_search_reports`, `falcon_get_mitre_report` |
| **Identity Protection (IDP)** | `idp_investigate_entity` (entity timeline, relationship mapping, risk assessment) |
| **IOC** | `falcon_search_iocs`, `falcon_add_ioc`, `falcon_remove_iocs` |
| **Cloud Security** | `falcon_search_kubernetes_containers`, `falcon_count_kubernetes_containers`, `falcon_search_images_vulnerabilities` |
| **Discover** | `falcon_search_applications`, `falcon_search_unmanaged_assets` |
| **Spotlight** | `falcon_search_vulnerabilities` |
| **Serverless** | `falcon_search_serverless_vulnerabilities` |
| **NGSIEM** | `search_ngsiem` (CQL queries against Next-Gen SIEM) |
| **Scheduled Reports** | `falcon_search_scheduled_reports`, `falcon_launch_scheduled_report`, `falcon_search_report_executions`, `falcon_download_report_execution` |
| **Sensor Usage** | `falcon_search_sensor_usage` |

Each module exposes FQL (Falcon Query Language) guide resources alongside its tools, enabling LLMs to construct valid queries without prior FQL knowledge.

> ⚠️ Not recommended for production deployments yet; stable 1.0 release pending.

### 2b. AIDR MCP Server
**GitHub:** [`CrowdStrike/aidr-mcp-server`](https://github.com/CrowdStrike/aidr-mcp-server)
**Status:** Available

Provides integration with CrowdStrike's **AI-Driven Response (AIDR)** APIs.

---

## 3. Palo Alto Networks — Cortex & Prisma AIRS

### 3a. Cortex MCP Server
**Announcement:** [Palo Alto Networks Blog — Introducing the Cortex MCP Server](https://www.paloaltonetworks.com/blog/security-operations/introducing-the-cortex-mcp-server/)
**Status:** Open Beta

Allows any MCP-compatible AI client (Claude for Desktop, Cursor, etc.) to query Cortex XDR/XSIAM data in natural language. Out-of-the-box tools expose:
- Issues and cases
- Assets and endpoints
- Compliance results and tenant metadata

### 3b. Prisma AIRS MCP Relay (`pan-mcp-relay`)
**GitHub:** [`PaloAltoNetworks/pan-mcp-relay`](https://github.com/PaloAltoNetworks/pan-mcp-relay)
**Status:** Public Preview
**Announcement:** [Palo Alto Networks Blog — Securing AI Agent Innovation with Prisma AIRS](https://www.paloaltonetworks.com/blog/2025/06/securing-ai-agent-innovation-prisma-airs-mcp-server/)

A security-enhanced **MCP relay/proxy** rather than a product data connector. Sits between MCP clients and MCP servers to:
- Scan and block prompt injections
- Block malicious URLs in tool outputs
- Prevent sensitive data loss (DLP)
- Enforce AI agentic threat policies

---

## 4. SentinelOne — Purple AI

**GitHub:** [`Sentinel-One` GitHub organization](https://github.com/orgs/Sentinel-One/repositories)
**Status:** Generally Available
**Announcement:** [SentinelOne AI Vision press release](https://www.sentinelone.com/press/sentinelone-reveals-vision-for-securing-the-ai-powered-world/)

The **Purple AI MCP Server** exposes SentinelOne's Singularity Platform to any MCP-compatible AI framework. Capabilities include:
- Alert and detection queries
- Vulnerability and misconfiguration data
- Asset/inventory queries
- Integration with Purple AI natural language security analysis

SentinelOne also launched **Prompt Security for Agentic AI (Beta)**: real-time visibility, risk assessment, and governance for MCP-connected autonomous agents.

---

## 5. Microsoft — Microsoft Sentinel

**GitHub:** [`microsoft/sentinel-data-exploration-mcp`](https://github.com/microsoft/sentinel-data-exploration-mcp)
**Status:** Public Preview
**Remote MCP endpoint:** `https://sentinel.microsoft.com/mcp/data-exploration`
**Announcement:** [Microsoft Security Blog — Microsoft Sentinel for the agentic era](https://www.microsoft.com/en-us/security/blog/2025/09/30/empowering-defenders-in-the-era-of-agentic-ai-with-microsoft-sentinel/)
**Tech Community blog:** [Sentinel MCP + GitHub Copilot threat hunting](https://techcommunity.microsoft.com/blog/coreinfrastructureandsecurityblog/using-microsoft-sentinel-mcp-server-with-github-copilot-for-ai-powered-threat-hu/4464980)

Part of Microsoft's broader MCP catalog at [`microsoft/mcp`](https://github.com/microsoft/mcp). Provides:
- Natural language search across Sentinel data lake tables
- KQL-free threat hunting (password spray, impossible travel, etc.)
- Compatible with GitHub Copilot and Security Copilot agents
- Sentinel graph (public preview) for graph-based security context

---

## 6. Elastic — Elasticsearch / Security

**GitHub:** [`elastic/mcp-server-elasticsearch`](https://github.com/elastic/mcp-server-elasticsearch)
**Docker image:** `docker.elastic.co/mcp/elasticsearch`
**Status:** Deprecated as standalone (superseded by Elastic Agent Builder MCP endpoint in Elastic 9.2.0+)
**Docs:** [Elastic MCP documentation](https://www.elastic.co/docs/solutions/search/mcp)

Enables natural language queries against Elasticsearch indices. Supports stdio, SSE, and streamable-HTTP MCP transports. The successor is the **Elastic Agent Builder MCP endpoint** available in Elastic 9.2.0+ and Elasticsearch Serverless.

---

## 7. Splunk — SIEM / Observability

**Splunkbase:** [Splunk MCP Server (App 7931)](https://splunkbase.splunk.com/app/7931)
**Cisco Code Exchange:** [`CiscoDevNet/Splunk-MCP-Server-official`](https://developer.cisco.com/codeexchange/github/repo/CiscoDevNet/Splunk-MCP-Server-official/)
**Status:** Beta (v1.0.1, released February 7, 2026); 5,000+ downloads, ⭐ 5/5

Built by **Splunk LLC**, published via CiscoDevNet. Features:
- Execute SPL searches and extract insights for agentic LLM workflows
- Enterprise-grade RBAC — AI can only access data the user's Splunk role permits
- AI-assisted SPL generation from natural language (via Splunk AI Assistant)
- Supports both Splunk Enterprise and Splunk Cloud

> There is also an unofficial repo at `splunk/splunk-mcp-server2` (Python + TypeScript) in the Splunk GitHub org, with input sanitization guardrails and Docker support.

---

## 8. IBM — QRadar SIEM & Security Portfolio

**GitHub:** [`IBM/qradar-mcp-server`](https://github.com/IBM/qradar-mcp-server)
**IBM MCP collection:** [`IBM/mcp`](https://github.com/IBM/mcp)
**Status:** MVP / Demo (not production-ready)

IBM's official QRadar MCP server exposes **728+ QRadar REST API endpoints** through 4 MCP tools:
- Search offenses
- Run AQL queries
- Manage reference sets
- Investigate security incidents

The broader **`IBM/mcp`** collection also includes MCP servers for:
- **IBM Guardium Cryptography Manager** (292 API endpoints — encryption key management, discovery scans, crypto policy enforcement)
- **IBM Security Verify** (210 REST API endpoints — SSO configuration, user management, identity workflows)

IBM also maintains **[`IBM/mcp-context-forge`](https://github.com/IBM/mcp-context-forge)**: an AI Gateway and registry proxy that unifies MCP, A2A, and REST/gRPC APIs behind a single endpoint.

---

## 9. Wiz — Cloud Security Posture Management (CSPM)

**Product page:** [Wiz Blog — Introducing MCP Server for Wiz](https://www.wiz.io/blog/introducing-mcp-server-for-wiz)
**AWS Marketplace:** [Wiz MCP Server](https://aws.amazon.com/marketplace/pp/prodview-nlyysa5n2s7h6)
**Status:** Preview (available to Wiz customers)

The Wiz MCP Server translates plain-language queries into Wiz-specific operations:
- Query cloud resources and security findings
- Assess risk posture across the Wiz Security Graph
- Accelerate incident response and remediation
- Integration with Wiz Code for PR-linked findings

---

## 10. Orca Security — Cloud Security

**Blog:** [Orca — Innovating with GenAI and MCP](https://orca.security/resources/blog/innovating-genai-cloud-telemetry-model-context-protocol/)
**Status:** Available to Orca customers

Orca built an MCP server for their platform allowing AI clients (Claude, Cursor) to query Orca environment data using natural language in 50+ languages. Use cases include:
- Cloud security posture summaries
- Compliance data exploration
- Discovering buried risk stories without learning cloud-provider query languages

---

## 11. Snyk — Developer Security / SCA

**Official announcement:** [Snyk — Secure AI Coding with MCP](https://snyk.io/articles/secure-ai-coding-with-snyk-now-supporting-model-context-protocol-mcp/)
**CLI command:** `snyk mcp` (available from CLI v1.1296.2+)
**Status:** Generally Available

Snyk integrates directly into MCP-compatible IDEs and agentic tools (GitHub Copilot, Cursor, Continue, Windsurf, Qodo, etc.) via local MCP server. Embeds Snyk vulnerability scanning directly into agentic coding workflows.

---

## 12. Panther — Cloud-Native SIEM / Detection Engineering

**GitHub:** [`panther-labs/mcp-panther`](https://github.com/panther-labs/mcp-panther)
**Status:** Generally Available (joint open-source effort with Block)
**Blog:** [Panther — How MCP Helps Security Teams Scale SecOps](https://panther.com/blog/how-model-context-protocol-helps-security-teams-scale-secops)

The `mcp-panther` server provides:
- AI-powered alert triage with insights and recommendations
- Multi-alert pattern analysis (time-based aggregation)
- Detection listing with filtering by name, state, severity, tags, log types, and MITRE TTPs

---

## Summary Table

| Vendor | Product Category | GitHub Repo | Status |
|---|---|---|---|
| **Google** | SIEM / SOAR / Threat Intel / CSPM | [`google/mcp-security`](https://github.com/google/mcp-security) | GA |
| **CrowdStrike** | EDR / XDR / Threat Intel | [`CrowdStrike/falcon-mcp`](https://github.com/CrowdStrike/falcon-mcp) | Public Preview |
| **CrowdStrike** | AI-Driven Response | [`CrowdStrike/aidr-mcp-server`](https://github.com/CrowdStrike/aidr-mcp-server) | Available |
| **Palo Alto Networks** | XDR / XSIAM (Cortex) | Official (closed source endpoint) | Open Beta |
| **Palo Alto Networks** | AI Security (Prisma AIRS relay) | [`PaloAltoNetworks/pan-mcp-relay`](https://github.com/PaloAltoNetworks/pan-mcp-relay) | Public Preview |
| **SentinelOne** | EDR / Purple AI | [`Sentinel-One` org](https://github.com/orgs/Sentinel-One/repositories) | GA |
| **Microsoft** | SIEM (Sentinel) | [`microsoft/sentinel-data-exploration-mcp`](https://github.com/microsoft/sentinel-data-exploration-mcp) | Public Preview |
| **Elastic** | SIEM / Search | [`elastic/mcp-server-elasticsearch`](https://github.com/elastic/mcp-server-elasticsearch) | Deprecated → Agent Builder |
| **Splunk** | SIEM / Observability | [`CiscoDevNet/Splunk-MCP-Server-official`](https://developer.cisco.com/codeexchange/github/repo/CiscoDevNet/Splunk-MCP-Server-official/) | Beta |
| **IBM** | SIEM (QRadar) / IAM / Crypto | [`IBM/qradar-mcp-server`](https://github.com/IBM/qradar-mcp-server), [`IBM/mcp`](https://github.com/IBM/mcp) | MVP/Demo |
| **Wiz** | CSPM / Cloud Security | Customer-facing preview | Preview |
| **Orca Security** | CSPM / Cloud Security | Customer-facing | Available |
| **Snyk** | SCA / Developer Security | Via `snyk mcp` CLI | GA |
| **Panther** | Cloud-Native SIEM | [`panther-labs/mcp-panther`](https://github.com/panther-labs/mcp-panther) | GA |

---

## Notable Community / Research Projects (Non-Official)

| Project | Focus | GitHub |
|---|---|---|
| **OpenCTI MCP** | Threat intelligence (STIX 2.1, MITRE ATT&CK) | [`Spathodea-Network/opencti-mcp`](https://github.com/Spathodea-Network/opencti-mcp) |
| **Cloud Security Alliance MCP Security** | MCP security tools catalog | [`ModelContextProtocol-Security`](https://github.com/ModelContextProtocol-Security) |
| **Astrix MCP Secret Wrapper** | Secrets management for MCP servers | Open source (AWS Secrets Manager integration) |
| **MCP Security Hub** | Offensive security tools (Nmap, Nuclei, Ghidra, SQLMap) | [`FuzzingLabs/mcp-security-hub`](https://github.com/FuzzingLabs/mcp-security-hub) |
| **SlowMist MCP Security Checklist** | Security best practices | [`slowmist/MCP-Security-Checklist`](https://github.com/slowmist/MCP-Security-Checklist) |
| **Awesome MCP Security** | Curated resource list | [`Puliczek/awesome-mcp-security`](https://github.com/Puliczek/awesome-mcp-security) |

---

## Sources

- [Google Cloud Security MCP Servers announcement](https://security.googlecloudcommunity.com/google-security-operations-2/build-with-google-cloud-security-mcp-servers-790)
- [CrowdStrike falcon-mcp on GitHub](https://github.com/CrowdStrike/falcon-mcp)
- [CrowdStrike AIDR MCP Server on GitHub](https://github.com/CrowdStrike/aidr-mcp-server)
- [Palo Alto Networks — Introducing the Cortex MCP Server](https://www.paloaltonetworks.com/blog/security-operations/introducing-the-cortex-mcp-server/)
- [Palo Alto Networks — Securing AI Agent Innovation with Prisma AIRS MCP Server](https://www.paloaltonetworks.com/blog/2025/06/securing-ai-agent-innovation-prisma-airs-mcp-server/)
- [PaloAltoNetworks/pan-mcp-relay on GitHub](https://github.com/PaloAltoNetworks/pan-mcp-relay)
- [SentinelOne AI Vision press release](https://www.sentinelone.com/press/sentinelone-reveals-vision-for-securing-the-ai-powered-world/)
- [microsoft/sentinel-data-exploration-mcp on GitHub](https://github.com/microsoft/sentinel-data-exploration-mcp)
- [Microsoft Security Blog — Sentinel for the agentic era](https://www.microsoft.com/en-us/security/blog/2025/09/30/empowering-defenders-in-the-era-of-agentic-ai-with-microsoft-sentinel/)
- [Microsoft Tech Community — Sentinel MCP + GitHub Copilot](https://techcommunity.microsoft.com/blog/coreinfrastructureandsecurityblog/using-microsoft-sentinel-mcp-server-with-github-copilot-for-ai-powered-threat-hu/4464980)
- [elastic/mcp-server-elasticsearch on GitHub](https://github.com/elastic/mcp-server-elasticsearch)
- [Elastic MCP documentation](https://www.elastic.co/docs/solutions/search/mcp)
- [Splunk MCP Server on Splunkbase](https://splunkbase.splunk.com/app/7931)
- [CiscoDevNet/Splunk-MCP-Server-official](https://developer.cisco.com/codeexchange/github/repo/CiscoDevNet/Splunk-MCP-Server-official/)
- [Splunk blog — Securing AI agents with MCP](https://www.splunk.com/en_us/blog/security/securing-ai-agents-model-context-protocol.html)
- [IBM/qradar-mcp-server on GitHub](https://github.com/IBM/qradar-mcp-server)
- [IBM/mcp collection on GitHub](https://github.com/IBM/mcp)
- [IBM/mcp-context-forge on GitHub](https://github.com/IBM/mcp-context-forge)
- [Wiz Blog — Introducing MCP Server for Wiz](https://www.wiz.io/blog/introducing-mcp-server-for-wiz)
- [Orca Security — Innovating with GenAI and MCP](https://orca.security/resources/blog/innovating-genai-cloud-telemetry-model-context-protocol/)
- [Snyk — Secure AI Coding with MCP](https://snyk.io/articles/secure-ai-coding-with-snyk-now-supporting-model-context-protocol-mcp/)
- [panther-labs/mcp-panther on GitHub](https://github.com/panther-labs/mcp-panther)
- [Panther blog — How MCP Helps Security Teams Scale SecOps](https://panther.com/blog/how-model-context-protocol-helps-security-teams-scale-secops)
- [Google/mcp-security on GitHub](https://github.com/google/mcp-security)
- [Astrix — State of MCP Server Security 2025](https://astrix.security/learn/blog/state-of-mcp-server-security-2025/)
- [Palo Alto Networks — MCP Security Overview](https://www.paloaltonetworks.com/blog/cloud-security/model-context-protocol-mcp-a-security-overview/)
- [Top 10 Best MCP Servers in 2026 — CybersecurityNews](https://cybersecuritynews.com/best-model-context-protocol-mcp-servers/)
- [Puliczek/awesome-mcp-security on GitHub](https://github.com/Puliczek/awesome-mcp-security)
- [FuzzingLabs/mcp-security-hub on GitHub](https://github.com/FuzzingLabs/mcp-security-hub)
- [slowmist/MCP-Security-Checklist on GitHub](https://github.com/slowmist/MCP-Security-Checklist)
- [ModelContextProtocol-Security — Cloud Security Alliance](https://github.com/ModelContextProtocol-Security)
- [Spathodea-Network/opencti-mcp on GitHub](https://github.com/Spathodea-Network/opencti-mcp)
