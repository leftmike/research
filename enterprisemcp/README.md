# Enterprise Cybersecurity Products with Official Open Source MCP Servers

**Research Date:** March 7, 2026
**Protocol:** Model Context Protocol (MCP) — open standard by Anthropic, now stewarded by the Linux Foundation (Agentic AI Foundation / AAIF)

---

## What is MCP?

The **Model Context Protocol (MCP)** is an open standard introduced by Anthropic in November 2024 that standardizes how AI systems (LLMs) integrate with external tools, data sources, and services. In December 2025, Anthropic donated MCP to the **Agentic AI Foundation (AAIF)**, a directed fund under the Linux Foundation, co-founded by Anthropic, Block, and OpenAI. As of March 2025, OpenAI officially adopted MCP.

MCP eliminates the "N×M" custom integration problem, replacing it with a single standardized protocol that any MCP-compatible AI client can use.

---

## Enterprise Cybersecurity Vendors with Official Open Source MCP Servers

### 1. CrowdStrike — Falcon MCP Server

| Field | Detail |
|-------|--------|
| **Product** | CrowdStrike Falcon Platform |
| **Repository** | [github.com/CrowdStrike/falcon-mcp](https://github.com/CrowdStrike/falcon-mcp) |
| **Status** | GA — available on AWS Marketplace |
| **License** | Open source (community-driven, maintained by CrowdStrike) |
| **Availability** | AWS Marketplace (AI Agents and Tools category); Google Cloud |

**Description:**
`falcon-mcp` connects AI agents to the CrowdStrike Falcon platform, enabling intelligent security analysis in agentic workflows. Provides access to Falcon telemetry including detections, incidents, threat intelligence, and behavioral data. CrowdStrike has also partnered with Google Cloud to foster an open and interoperable AI security ecosystem through MCP.

**Key Capabilities:**
- Query Falcon detections, incidents, and alerts
- Access threat intelligence feeds
- Behavioral data analysis via natural language
- Integration with AWS Marketplace AI agent workflows

**References:**
- [CrowdStrike Falcon MCP Server — GitHub](https://github.com/CrowdStrike/falcon-mcp)
- [CrowdStrike and Google Cloud Advance AI-Native Integration with MCP](https://www.crowdstrike.com/en-us/blog/crowdstrike-google-cloud-ai-native-integration-mcp/)
- [CrowdStrike brings GenAI Security Tools to AWS Marketplace](https://www.crowdstrike.com/en-us/press-releases/crowdstrike-brings-genai-security-tools-to-aws-marketplace/)

---

### 2. Palo Alto Networks — Cortex MCP Server

| Field | Detail |
|-------|--------|
| **Product** | Cortex (XSIAM / XSOAR) |
| **Blog Announcement** | [Introducing the Cortex MCP Server](https://www.paloaltonetworks.com/blog/security-operations/introducing-the-cortex-mcp-server/) |
| **Status** | Open Beta |
| **Community PAN-OS Repo** | [github.com/cdot65/pan-os-mcp](https://github.com/cdot65/pan-os-mcp) |
| **License** | Open beta (Cortex); community open-source (PAN-OS) |

**Description:**
The Cortex MCP Server enables any MCP-compatible AI client to interact directly with the Cortex platform. Security analysts can use LLM-powered guidance to review, prioritize, and update security cases. A separate community-maintained MCP server exists for interfacing directly with Palo Alto Networks Next-Generation Firewalls (NGFW) via the XML API.

**Key Capabilities:**
- Natural language interaction with Cortex XSIAM cases
- Incident prioritization and triage
- Integration with Claude for Desktop and other LLM applications
- Community PAN-OS: full NGFW configuration and query via XML API

**References:**
- [Introducing the Cortex MCP Server — Palo Alto Networks Blog](https://www.paloaltonetworks.com/blog/security-operations/introducing-the-cortex-mcp-server/)
- [pan-os-mcp community project — GitHub](https://github.com/cdot65/pan-os-mcp)
- [Palo Alto Networks MCP server listing](https://mcp.so/server/paloalto-mcp-servers)

---

### 3. Splunk — MCP Server for Splunk Platform

| Field | Detail |
|-------|--------|
| **Product** | Splunk Cloud Platform / Splunk Enterprise |
| **Splunkbase App ID** | 7931 |
| **Splunkbase Link** | [Splunk MCP Server on Splunkbase](https://splunkbase.splunk.com/app/7931) |
| **Status** | GA (v1.0.1, released February 7, 2026) |
| **Support** | Splunk Supported |
| **Official Docs** | [help.splunk.com MCP Server docs](https://help.splunk.com/en/splunk-cloud-platform/mcp-server-for-splunk-platform/about-mcp-server-for-splunk-platform) |

**Description:**
Splunk's official MCP server provides a standardized, secure, and scalable interface to connect AI assistants and agents with data in the Splunk platform. Built by Splunk LLC and rated 5/5 stars with 5,000+ downloads on Splunkbase.

**Key Capabilities:**
- Universal connectivity via Streamable HTTP
- Enterprise-grade authentication and Role-Based Access Control (RBAC)
- Encrypted token security using public key encryption
- Natural language SPL query generation and execution
- Security operations and threat hunting integration

**References:**
- [Splunk MCP Server on Splunkbase](https://splunkbase.splunk.com/app/7931)
- [Unlock the Power of Splunk Cloud Platform with the MCP Server — Splunk Blog](https://www.splunk.com/en_us/blog/artificial-intelligence/unlock-the-power-of-splunk-cloud-platform-with-the-mcp-server.html)
- [Securing AI Agents: Model Context Protocol — Splunk Security Blog](https://www.splunk.com/en_us/blog/security/securing-ai-agents-model-context-protocol.html)

---

### 4. SentinelOne — Purple AI MCP Server

| Field | Detail |
|-------|--------|
| **Product** | Singularity Platform / Purple AI |
| **Status** | Available on GitHub |
| **License** | Open source |
| **Additional** | Prompt Security for Agentic AI (Beta) |

**Description:**
SentinelOne's Purple AI MCP Server provides secure, seamless integration between the Singularity Platform and any AI framework or LLM. Acts as a universal translator and intelligence hub, enabling developers and partners to build custom agentic AI experiences powered by SentinelOne's full security context and analytics.

**Key Capabilities:**
- Integration with any MCP-compatible LLM framework
- Access to Singularity Platform telemetry and analytics
- Support for custom agentic AI development
- Prompt Security for Agentic AI: real-time visibility, risk assessment, and governance for autonomous AI agents built on MCP

**References:**
- [SentinelOne Reveals Vision for Securing the AI-Powered World](https://www.sentinelone.com/press/sentinelone-reveals-vision-for-securing-the-ai-powered-world/)
- [Avoiding MCP Mania — SentinelOne Blog](https://www.sentinelone.com/blog/avoiding-mcp-mania-how-to-secure-the-next-frontier-of-ai/)

---

### 5. Microsoft — Sentinel Data Exploration MCP Server

| Field | Detail |
|-------|--------|
| **Product** | Microsoft Sentinel |
| **Repository** | [github.com/microsoft/sentinel-data-exploration-mcp](https://github.com/microsoft/sentinel-data-exploration-mcp) |
| **Remote Endpoint** | `https://sentinel.microsoft.com/mcp/data-exploration` |
| **Status** | Public Preview (Jan 2026) |
| **License** | Open source (Microsoft) |
| **Official Catalog** | [github.com/microsoft/mcp](https://github.com/microsoft/mcp) |

**Description:**
Microsoft's official Sentinel MCP Server enables natural language querying of Microsoft Sentinel's data lake. Any IDE, agent, or MCP-compatible tool can connect to the remote MCP endpoint. Microsoft has also released a broader catalog of official MCP servers covering Azure, Azure DevOps, and AKS.

**Key Capabilities:**
- Natural language KQL query generation
- Password-spray detection and investigation
- Impossible travel check automation
- Integration with GitHub Copilot and Security Copilot
- Access to Sentinel logs, incidents, and analytics

**Defender Advanced Hunting MCP:**
A separate official MCP server enables executing KQL queries against **Microsoft Defender Advanced Hunting** via natural language (released January–February 2026).

**References:**
- [Microsoft Sentinel Data Exploration MCP — GitHub](https://github.com/microsoft/sentinel-data-exploration-mcp)
- [Microsoft Official MCP Catalog — GitHub](https://github.com/microsoft/mcp)
- [Using Microsoft Sentinel MCP Server with GitHub Copilot — Microsoft Tech Community](https://techcommunity.microsoft.com/blog/coreinfrastructureandsecurityblog/using-microsoft-sentinel-mcp-server-with-github-copilot-for-ai-powered-threat-hu/4464980)
- [Empowering Defenders in the Era of Agentic AI — Microsoft Security Blog](https://www.microsoft.com/en-us/security/blog/2025/09/30/empowering-defenders-in-the-era-of-agentic-ai-with-microsoft-sentinel/)

---

### 6. IBM — Security Verify & OpenPages GRC MCP Servers

| Field | Detail |
|-------|--------|
| **Products** | IBM Security Verify, IBM OpenPages GRC, IBM Cloud VPC Security |
| **Repository** | [github.com/IBM/mcp](https://github.com/IBM/mcp) |
| **Status** | Production-ready and experimental |
| **License** | Open source (IBM) |

**Description:**
IBM's MCP collection includes multiple security-focused servers spanning identity management, governance, risk, and compliance (GRC), and cloud security.

**Key Security MCP Servers:**

| Server | Description |
|--------|-------------|
| **IBM Security Verify** | Access 210 IBM Security Verify REST API endpoints through 4 intelligent MCP tools — manage users, configure SSO, and orchestrate identity workflows |
| **IBM OpenPages GRC** | Experimental local MCP server enabling AI agents to securely interact with IBM OpenPages GRC platform through a standardized interface |
| **IBM Cloud VPC Security** | Provides access to IBM Cloud VPC resources and security analysis capabilities — interact with cloud infrastructure, backups, and security policies |

**References:**
- [IBM Official MCP Collection — GitHub](https://github.com/IBM/mcp)

---

### 7. HashiCorp — Vault MCP Server

| Field | Detail |
|-------|--------|
| **Product** | HashiCorp Vault (secrets management) |
| **Repository** | [github.com/hashicorp/vault-mcp-server](https://github.com/hashicorp/vault-mcp-server) |
| **Status** | Experimental (dev/eval only, not production) |
| **License** | Open source (HashiCorp) |
| **Docs** | [developer.hashicorp.com/vault/docs/mcp-server/overview](https://developer.hashicorp.com/vault/docs/mcp-server/overview) |

**Description:**
HashiCorp's official Vault MCP Server provides integration with Vault for secrets management and mount operations. Uses stdio and StreamableHTTP transports. A companion **Vault Radar MCP Server** enables natural language queries of complex risk datasets for secret leak identification. HashiCorp also has an official **Terraform MCP Server**.

**Key Capabilities:**
- Secrets management via natural language
- Vault mount and policy interaction
- Vault Radar: query critical/high severity leaked secrets
- Scoped APIs enforce least-privilege access
- Full audit trail of all operations

**Security Note:** Raw secrets are never directly exposed. Not recommended for production with untrusted MCP clients or LLMs.

**References:**
- [HashiCorp Vault MCP Server — GitHub](https://github.com/hashicorp/vault-mcp-server)
- [HashiCorp Introduces MCP Servers for Terraform and Vault — InfoQ](https://www.infoq.com/news/2025/08/hashicorp-mcp-servers-terraform-/)
- [Vault MCP Server Docs — HashiCorp Developer](https://developer.hashicorp.com/vault/docs/mcp-server/overview)

---

### 8. Snyk — Studio MCP Server

| Field | Detail |
|-------|--------|
| **Product** | Snyk (developer security / SCA / SAST) |
| **Repository** | [github.com/snyk/studio-mcp](https://github.com/snyk/studio-mcp) |
| **Status** | Official (via Snyk CLI `snyk mcp` command) |
| **License** | Open source (Snyk) |
| **MCP Registry** | Listed in official MCP registry |

**Description:**
Snyk is introducing an MCP server as part of the Snyk CLI, allowing MCP-enabled agentic tools to integrate Snyk security scanning directly. The `snyk mcp` CLI command invokes Snyk scans and retrieves security findings within any MCP-enabled environment.

**Key Capabilities:**
- Vulnerability scanning for code, dependencies, and configurations
- Seamless integration via `snyk mcp` CLI command
- Listed in the official MCP server registry
- Embeds Snyk vulnerability scanning directly into agentic workflows

**References:**
- [Snyk Studio MCP — GitHub](https://github.com/snyk/studio-mcp)
- [Official MCP Registry — registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io/)

---

### 9. Elastic — Elasticsearch MCP Server

| Field | Detail |
|-------|--------|
| **Product** | Elastic / Elasticsearch / Elastic Security |
| **Repository** | [github.com/elastic/mcp-server-elasticsearch](https://github.com/elastic/mcp-server-elasticsearch) |
| **Status** | Deprecated (superseded by Elastic Agent Builder MCP in Elastic 9.2.0+) |
| **License** | Open source (Elastic) |
| **Docker Image** | `docker.elastic.co/mcp/elasticsearch` |
| **Docs** | [elastic.co/docs/solutions/search/mcp](https://www.elastic.co/docs/solutions/search/mcp) |

**Description:**
Elastic's official MCP server connects agents to Elasticsearch data through natural language conversations. Supports Elasticsearch 8.x and 9.x. Now deprecated in favor of the newer **Elastic Agent Builder MCP endpoint** available in Elastic 9.2.0+.

**Key Capabilities:**
- List indices, get field mappings, execute search queries
- Supports ES|QL, stdio, SSE, and Streamable HTTP protocols
- Elastic Security AI Assistant: alert investigation, incident response, query generation
- Attack Discovery: AI-powered alert triage mapped to MITRE ATT&CK

**References:**
- [Elastic MCP Server — GitHub](https://github.com/elastic/mcp-server-elasticsearch)
- [Elastic MCP Docs](https://www.elastic.co/docs/solutions/search/mcp)
- [MCP Overview and Emerging Use Cases — Elastic Labs](https://www.elastic.co/search-labs/blog/mcp-current-state)

---

### 10. Datadog — Official Remote MCP Server

| Field | Detail |
|-------|--------|
| **Product** | Datadog (monitoring, SIEM, security) |
| **Official Docs** | [docs.datadoghq.com/bits_ai/mcp_server](https://docs.datadoghq.com/bits_ai/mcp_server/) |
| **Status** | Official (remote MCP server, under active development) |
| **License** | Proprietary / closed source (remote server); community GitHub repos exist |
| **Blog** | [Datadog Remote MCP Server announcement](https://www.datadoghq.com/blog/datadog-remote-mcp-server/) |

**Description:**
Datadog's official remote MCP Server acts as a bridge between Datadog and MCP-compatible AI agents (Claude Code, Codex, Goose, Cursor). Supports logs, metrics, traces, dashboards, monitors, incidents, and security signals.

**Key Security Capabilities:**
- **Security toolset:** Comprehensive security scan detecting SQL injection, XSS, path traversal, API keys, passwords, credentials
- Search and retrieve Cloud SIEM signals, App & API Protection signals, and Workload Protection signals
- SQL-based analysis of security findings from the last 24 hours
- OpenAI Codex CLI integration for on-call incident response

**Note:** Datadog's remote MCP server is hosted by Datadog (not self-hosted open source). Community open-source wrappers exist on GitHub.

**References:**
- [Datadog MCP Server Docs](https://docs.datadoghq.com/bits_ai/mcp_server/)
- [Datadog Remote MCP Server Blog](https://www.datadoghq.com/blog/datadog-remote-mcp-server/)

---

## AWS — Official MCP Servers (Security-Relevant)

| Field | Detail |
|-------|--------|
| **Repository** | [github.com/awslabs/mcp](https://github.com/awslabs/mcp) |
| **Status** | Official (AWS Labs) |
| **License** | Open source (Apache 2.0) |

**Description:**
AWS provides a suite of specialized MCP servers. Relevant to security: the Infrastructure as Code toolkit includes CloudFormation documentation access, CDK best practices guidance, **security validation**, and deployment troubleshooting.

**References:**
- [AWS Official MCP Servers — GitHub](https://github.com/awslabs/mcp)

---

## Summary Table

| Vendor | Product | GitHub Repository | Status | Official? |
|--------|---------|-------------------|--------|-----------|
| **CrowdStrike** | Falcon Platform | [CrowdStrike/falcon-mcp](https://github.com/CrowdStrike/falcon-mcp) | GA / AWS Marketplace | Yes |
| **Palo Alto Networks** | Cortex | *(blog announcement)* | Open Beta | Yes |
| **Palo Alto Networks** | PAN-OS NGFW | [cdot65/pan-os-mcp](https://github.com/cdot65/pan-os-mcp) | Community | Community |
| **Splunk** | Splunk Platform | [Splunkbase App 7931](https://splunkbase.splunk.com/app/7931) | GA v1.0.1 | Yes |
| **SentinelOne** | Purple AI / Singularity | *(GitHub — Purple AI MCP)* | Available | Yes |
| **Microsoft** | Microsoft Sentinel | [microsoft/sentinel-data-exploration-mcp](https://github.com/microsoft/sentinel-data-exploration-mcp) | Public Preview | Yes |
| **Microsoft** | Azure (full catalog) | [microsoft/mcp](https://github.com/microsoft/mcp) | GA | Yes |
| **IBM** | Security Verify / OpenPages / VPC | [IBM/mcp](https://github.com/IBM/mcp) | Prod + Experimental | Yes |
| **HashiCorp** | Vault | [hashicorp/vault-mcp-server](https://github.com/hashicorp/vault-mcp-server) | Experimental | Yes |
| **Snyk** | Snyk CLI / Studio | [snyk/studio-mcp](https://github.com/snyk/studio-mcp) | Official | Yes |
| **Elastic** | Elasticsearch / Security | [elastic/mcp-server-elasticsearch](https://github.com/elastic/mcp-server-elasticsearch) | Deprecated (use 9.2+) | Yes |
| **Datadog** | Monitoring / SIEM | *(remote server, docs only)* | Official remote | Yes |
| **AWS** | AWS Services | [awslabs/mcp](https://github.com/awslabs/mcp) | Official | Yes |

---

## Vendors Without Confirmed Official MCP Servers (as of March 2026)

The following major enterprise cybersecurity vendors were researched but do not appear to have published an official open-source MCP server:

| Vendor | Notes |
|--------|-------|
| **Tenable** | Published an MCP FAQ blog; no confirmed GitHub MCP server |
| **Qualys** | No official or community MCP server found |
| **Wiz** | No dedicated MCP server found (broad integrations page only) |
| **Rapid7** | Community-built InsightIDR MCP server exists but is not official |

---

## Key Resources

- [Official MCP Registry](https://registry.modelcontextprotocol.io/)
- [MCP GitHub Organization](https://github.com/modelcontextprotocol)
- [Awesome MCP Servers (curated list)](https://github.com/wong2/awesome-mcp-servers)
- [Awesome MCP Security](https://github.com/Puliczek/awesome-mcp-security)
- [MCP for Security Tools (pentest)](https://github.com/cyproxio/mcp-for-security)
- [State of MCP Server Security 2025 — Astrix Research](https://astrix.security/learn/blog/state-of-mcp-server-security-2025/)
- [Model Context Protocol — Wikipedia](https://en.wikipedia.org/wiki/Model_Context_Protocol)
