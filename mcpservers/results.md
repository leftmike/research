# MCP Server Source Code Analysis: Sampling & Elicitation Support


## METHODOLOGY
  Repository URLs were collected from three sources:
    1. registry.modelcontextprotocol.io - API provides repo URLs directly
       (primary source: 2424 unique repos)
    2. pulsemcp.com - detail pages blocked by egress proxy (0 repos)
    3. github.com/jaw9c/awesome-remote-mcp-servers - README parsed
       (4 additional unique repos)

  Each repository was downloaded (GitHub tarball) and searched for
  patterns indicating MCP sampling or elicitation support.
  Results were filtered to remove false positives (generic
  createMessage functions, minified SDK bundles, etc.) and classified
  as implementation, test-only, docs-only, import-only, or reference.


## SUMMARY
  Total unique repositories: **2427**
  Successfully analyzed: **2059**
  Not found (404/deleted): **338**
  Other errors: **30**

  SAMPLING (after filtering false positives):
    Total repos with real MCP sampling indicators: 32
      Implementation: 27
      Reference (types/config): 4
      Test only: 1
      Docs only: 0
      Import only: 0

  ELICITATION (after filtering false positives):
    Total repos with real MCP elicitation indicators: 56
      Implementation: 31
      Reference (types/config): 17
      Test only: 5
      Docs only: 3
      Import only: 0

  Repos implementing BOTH sampling AND elicitation: **8**



## SERVERS WITH SAMPLING IMPLEMENTATION


**Repository:** [IamNishant51/atlas-mcp-server](https://github.com/IamNishant51/atlas-mcp-server)
**Server names:** io.github.IamNishant51/atlas-pipeline
**Sources:** registry
**Classification:** `implementation`
**Evidence** (5 matches):
- `atlas-mcp-server-main/src/providers/llm-provider.ts`
  - `// Using 'as any' cast here because the SDK types for 'sampling/createMessage'`
- `atlas-mcp-server-main/src/providers/llm-provider.ts`
  - `method: 'sampling/createMessage',`
- `atlas-mcp-server-main/src/providers/llm-provider.ts`
  - `import { CreateMessageRequestSchema } from '@modelcontextprotocol/sdk/types.js';`
- `atlas-mcp-server-main/src/providers/llm-provider.ts`
  - `// Note: The MCP SDK might not expose CreateMessageRequestSchema directly in all versions,`
- `atlas-mcp-server-main/src/providers/llm-provider.ts`
  - `CreateMessageRequestSchema`


**Repository:** [ImRonAI/mcp-server-browserbase](https://github.com/ImRonAI/mcp-server-browserbase)
**Server names:** ai.smithery/ImRonAI-mcp-server-browserbase
**Sources:** registry
**Classification:** `implementation`
**Evidence** (2 matches):
- `mcp-server-browserbase-main/src/mcp/sampling.ts`
  - `* The server sends sampling/createMessage requests to ask the client`
- `mcp-server-browserbase-main/src/mcp/sampling.ts`
  - `method: "sampling/createMessage",`


**Repository:** [Kastalien-Research/thoughtbox](https://github.com/Kastalien-Research/thoughtbox)
**Server names:** io.github.Kastalien-Research/thoughtbox
**Sources:** registry
**Classification:** `implementation`
**Evidence** (9 matches):
- `thoughtbox-main/docs/2025-11-25.ts`
  - `* Whether the client supports task-augmented sampling/createMessage requests.`
- `thoughtbox-main/docs/2025-11-25.ts`
  - `* Parameters for a `sampling/createMessage` request.`
- `thoughtbox-main/docs/2025-11-25.ts`
  - `* @category `sampling/createMessage``
- `thoughtbox-main/docs/2025-11-25.ts`
  - `method: "sampling/createMessage";`
- `thoughtbox-main/docs/2025-11-25.ts`
  - `* The client's response to a sampling/createMessage request from the server.`
  - *... and 4 more matches*


**Repository:** [Klavis-AI/klavis](https://github.com/Klavis-AI/klavis)
**Server names:** ai.klavis/strata
**Sources:** registry
**Classification:** `implementation`
**Evidence** (1 matches):
- `klavis-main/mcp_servers/intercom/src/server.ts`
  - `const result = await handlers.message.createMessage({`


**Repository:** [MervinPraison/PraisonAI](https://github.com/MervinPraison/PraisonAI)
**Server names:** io.github.MervinPraison/praisonai
**Sources:** registry
**Classification:** `implementation`
**Evidence** (5 matches):
- `PraisonAI-main/src/praisonai/tests/unit/mcp_server/test_sampling.py`
  - `response = asyncio.run(handler.create_message(request))`
- `PraisonAI-main/src/praisonai/praisonai/mcp_server/sampling.py`
  - `async def create_message(`
- `PraisonAI-main/src/praisonai/praisonai/mcp_server/auth/scopes.py`
  - `"sampling/createMessage": ScopeRequirement(["sampling:create"]),`
- `PraisonAI-main/examples/mcp_server/mcp_sampling_example.py`
  - `response = await handler.create_message(basic_request)`
- `PraisonAI-main/examples/mcp_server/mcp_sampling_example.py`
  - `response = await handler.create_message(tool_request)`


**Repository:** [OtherVibes/mcp-as-a-judge](https://github.com/OtherVibes/mcp-as-a-judge)
**Server names:** io.github.OtherVibes/mcp-as-a-judge
**Sources:** registry
**Classification:** `implementation`
**Evidence** (16 matches):
- `mcp-as-a-judge-main/tests/test_task_sizing.py`
  - `"mcp_as_a_judge.messaging.factory.MessagingProviderFactory.check_sampling_capability"`
- `mcp-as-a-judge-main/tests/conftest.py`
  - `mock_context.session.create_message = AsyncMock()`
- `mcp-as-a-judge-main/tests/conftest.py`
  - `mock_context.session.create_message.return_value = MagicMock(`
- `mcp-as-a-judge-main/tests/test_messaging_layer.py`
  - `ctx.session.create_message = AsyncMock()`
- `mcp-as-a-judge-main/tests/test_messaging_layer.py`
  - `ctx.session.create_message = AsyncMock(return_value=mock_result)`
  - *... and 11 more matches*


**Repository:** [SamMorrowDrums/remarkable-mcp](https://github.com/SamMorrowDrums/remarkable-mcp)
**Server names:** io.github.SamMorrowDrums/remarkable
**Sources:** registry
**Classification:** `implementation`
**Evidence** (3 matches):
- `remarkable-mcp-main/test_server.py`
  - `from mcp.types import ClientCapabilities, SamplingCapability`
- `remarkable-mcp-main/test_server.py`
  - `mock_caps = ClientCapabilities(sampling=SamplingCapability())`
- `remarkable-mcp-main/remarkable_mcp/sampling.py`
  - `result = await session.create_message(`


**Repository:** [SmartBear/smartbear-mcp](https://github.com/SmartBear/smartbear-mcp)
**Server names:** com.smartbear/smartbear-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (1 matches):
- `smartbear-mcp-main/src/common/pollyfills.ts`
  - `const response = await server.server.createMessage({`


**Repository:** [askman-dev/agent-never-give-up-mcp](https://github.com/askman-dev/agent-never-give-up-mcp)
**Server names:** io.github.askman-dev/agent-never-give-up
**Sources:** registry
**Classification:** `implementation`
**Evidence** (3 matches):
- `agent-never-give-up-mcp-main/src/mcpServer.ts`
  - `createMessage: (params: unknown) => Promise<SamplingResult>;`
- `agent-never-give-up-mcp-main/src/mcpServer.ts`
  - `typeof serverWithSampling.sampling?.createMessage === "function"`
- `agent-never-give-up-mcp-main/src/mcpServer.ts`
  - `await serverWithSampling.sampling.createMessage(samplingParams);`


**Repository:** [browserbase/mcp-server-browserbase](https://github.com/browserbase/mcp-server-browserbase)
**Server names:** ai.smithery/browserbasehq-mcp-browserbase, io.github.browserbase/mcp-server-browserbase
**Sources:** registry
**Classification:** `implementation`
**Evidence** (1 matches):
- `mcp-server-browserbase-main/src/mcp/sampling.ts`
  - `* The server sends sampling/createMessage requests to ask the client`


**Repository:** [codefuturist/email-mcp](https://github.com/codefuturist/email-mcp)
**Server names:** io.github.codefuturist/email-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (2 matches):
- `email-mcp-main/src/services/hooks.service.ts`
  - `* - Falls through to AI triage via `sampling/createMessage` if no rule matched`
- `email-mcp-main/src/services/hooks.service.ts`
  - `const result = await srv.createMessage({`


**Repository:** [cyanheads/mcp-ts-template](https://github.com/cyanheads/mcp-ts-template)
**Server names:** io.github.cyanheads/mcp-ts-template
**Sources:** registry
**Classification:** `implementation`
**Evidence** (5 matches):
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-code-review-sampling.tool.ts`
  - `createMessage: (args: {`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-code-review-sampling.tool.ts`
  - `return typeof (ctx as SamplingSdkContext)?.createMessage === 'function';`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-code-review-sampling.tool.ts`
  - `const samplingResult = await sdkContext.createMessage({`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-code-review-sampling.tool.ts`
  - `function hasSamplingCapability(ctx: SdkContext): ctx is SamplingSdkContext {`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-code-review-sampling.tool.ts`
  - `if (!hasSamplingCapability(sdkContext)) {`


**Repository:** [himorishige/hatago-mcp-hub](https://github.com/himorishige/hatago-mcp-hub)
**Server names:** io.github.himorishige/hatago-mcp-hub
**Sources:** registry
**Classification:** `implementation`
**Evidence** (6 matches):
- `hatago-mcp-hub-main/packages/core/src/types/rpc.ts`
  - `| 'sampling/createMessage';`
- `hatago-mcp-hub-main/packages/core/src/rpc/methods.ts`
  - `sampling_createMessage: 'sampling/createMessage'`
- `hatago-mcp-hub-main/packages/hub/src/rpc/handlers.ts`
  - `sampling_createMessage: 'sampling/createMessage'`
- `hatago-mcp-hub-main/packages/hub/src/rpc/dispatch.test.ts`
  - `'sampling/createMessage'`
- `hatago-mcp-hub-main/packages/hub/src/rpc/dispatch.ts`
  - `sampling_createMessage: 'sampling/createMessage'`
  - *... and 1 more matches*


**Repository:** [jfarcand/iphone-mirroir-mcp](https://github.com/jfarcand/iphone-mirroir-mcp)
**Server names:** io.github.jfarcand/iphone-mirroir-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (5 matches):
    mirroir-mcp-main/Sources/mirroir-mcp/MCPServer.swift:
      /// Send a sampling/createMessage request to the MCP client and wait for the response.
    mirroir-mcp-main/Sources/mirroir-mcp/MCPServer.swift:
      // Build JSON-RPC request for sampling/createMessage
    mirroir-mcp-main/Sources/mirroir-mcp/MCPServer.swift:
      "method": "sampling/createMessage",
    mirroir-mcp-main/Sources/HelperLib/MCPProtocol.swift:
      /// Parameters for a sampling/createMessage server-to-client request.
    mirroir-mcp-main/Sources/HelperLib/MCPProtocol.swift:
      /// Response from a sampling/createMessage request.


**Repository:** [karashiiro/my-cool-proxy](https://github.com/karashiiro/my-cool-proxy)
**Server names:** io.github.karashiiro/my-cool-proxy
**Sources:** registry
**Classification:** `implementation`
**Evidence** (12 matches):
- `my-cool-proxy-main/packages/mcp-client/src/client-session.ts`
  - `* (e.g., sampling/createMessage, elicitation/create).`
- `my-cool-proxy-main/apps/gateway/src/handlers/proxy-handlers.ts`
  - `CreateMessageRequestSchema,`
- `my-cool-proxy-main/apps/gateway/src/mcp/gateway-server.ts`
  - `* This is called when an upstream MCP server sends a sampling/createMessage request.`
- `my-cool-proxy-main/apps/gateway/src/mcp/gateway-server.ts`
  - `CreateMessageRequest,`
- `my-cool-proxy-main/apps/gateway/src/mcp/gateway-server.ts`
  - `CreateMessageResult,`
  - *... and 7 more matches*


**Repository:** [paiml/rust-mcp-sdk](https://github.com/paiml/rust-mcp-sdk)
**Server names:** io.github.paiml/pmcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (17 matches):
- `rust-mcp-sdk-main/tests/new_features.rs`
  - `use pmcp::{Client, Content, CreateMessageRequest, Role, SamplingMessage, StdioTransport};`
- `rust-mcp-sdk-main/tests/new_features.rs`
  - `let request = CreateMessageRequest {`
- `rust-mcp-sdk-main/tests/new_features.rs`
  - `Content, CreateMessageParams, CreateMessageResult, SamplingHandler, Server, TokenUsage,`
- `rust-mcp-sdk-main/tests/new_features.rs`
  - `) -> pmcp::Result<CreateMessageResult> {`
- `rust-mcp-sdk-main/tests/new_features.rs`
  - `Ok(CreateMessageResult {`
  - *... and 12 more matches*


**Repository:** [paulbreuler/limps](https://github.com/paulbreuler/limps)
**Server names:** io.github.paulbreuler/limps
**Sources:** registry
**Classification:** `implementation`
**Evidence** (8 matches):
- `limps-main/packages/limps/tests/rlm/sampling.test.ts`
  - `const response1 = await mockClient.createMessage({`
- `limps-main/packages/limps/tests/rlm/sampling.test.ts`
  - `const response2 = await mockClient.createMessage({`
- `limps-main/packages/limps/tests/rlm/sampling.test.ts`
  - `const response = await mockClient.createMessage({`
- `limps-main/packages/limps/src/rlm/sampling.ts`
  - `* Create a sampling message (MCP createMessage request).`
- `limps-main/packages/limps/src/rlm/sampling.ts`
  - `createMessage(request: SamplingRequest): Promise<SamplingResponse>;`
  - *... and 3 more matches*


**Repository:** [payclaw/badge-server](https://github.com/payclaw/badge-server)
**Server names:** io.github.payclaw/badge
**Sources:** registry
**Classification:** `implementation`
**Evidence** (1 matches):
- `badge-server-main/src/sampling.ts`
  - `serverRef.createMessage({`


**Repository:** [payclaw/mcp-server](https://github.com/payclaw/mcp-server)
**Server names:** io.github.payclaw/payclaw, io.github.payclaw/spend
**Sources:** registry
**Classification:** `implementation`
**Evidence** (6 matches):
- `mcp-server-main/src/sampling.ts`
  - `serverRef.createMessage({`
- `mcp-server-main/src/sampling.test.ts`
  - `it("createMessage not supported -> no_sampling, no crash", async () => {`
- `mcp-server-main/src/sampling.test.ts`
  - `createMessage: vi.fn().mockRejectedValue(new Error("Method not found")),`
- `mcp-server-main/src/sampling.test.ts`
  - `it("createMessage times out -> inconclusive, trip evicted", async () => {`
- `mcp-server-main/src/sampling.test.ts`
  - `createMessage: vi.fn().mockImplementation(`
  - *... and 1 more matches*


**Repository:** [polydev-ai/polydev](https://github.com/polydev-ai/polydev)
**Server names:** io.github.polydev-ai/polydev
**Sources:** registry
**Classification:** `implementation`
**Evidence** (20 matches):
- `polydev-main/src/lib/api/index.ts`
  - `return handler.createMessage(options)`
- `polydev-main/src/lib/api/index.ts`
  - `return universalProvider.createMessage(providerId, options)`
- `polydev-main/src/lib/api/providers/complete-provider-system.ts`
  - `const response = await this.createMessage(providerId, streamOptions)`
- `polydev-main/src/lib/api/providers/complete-provider-system.ts`
  - `const response = await this.createMessage(providerId, testOptions)`
- `polydev-main/src/lib/api/providers/enhanced-handlers.ts`
  - `const response = await this.createMessage(testOptions)`
  - *... and 15 more matches*


**Repository:** [ruvnet/claude-flow](https://github.com/ruvnet/claude-flow)
**Server names:** io.github.ruvnet/claude-flow
**Sources:** registry
**Classification:** `implementation`
**Evidence** (14 matches):
- `ruflo-main/v3/@claude-flow/mcp/src/types.ts`
  - `export interface CreateMessageRequest {`
- `ruflo-main/v3/@claude-flow/mcp/src/types.ts`
  - `export interface CreateMessageResult {`
- `ruflo-main/v3/@claude-flow/mcp/src/sampling.ts`
  - `* Create a message (sampling/createMessage)`
- `ruflo-main/v3/@claude-flow/mcp/src/sampling.ts`
  - `CreateMessageRequest,`
- `ruflo-main/v3/@claude-flow/mcp/src/sampling.ts`
  - `CreateMessageResult,`
  - *... and 9 more matches*


**Repository:** [securecoders/opengraph-io-mcp](https://github.com/securecoders/opengraph-io-mcp)
**Server names:** io.github.securecoders/opengraph-io-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (5 matches):
- `opengraph-io-mcp-main/src/mcp.ts`
  - `method: "sampling/createMessage",`
- `opengraph-io-mcp-main/src/mcp.ts`
  - `CreateMessageRequest,`
- `opengraph-io-mcp-main/src/mcp.ts`
  - `CreateMessageResultSchema,`
- `opengraph-io-mcp-main/src/mcp.ts`
  - `const request: CreateMessageRequest = {`
- `opengraph-io-mcp-main/src/mcp.ts`
  - `return await server.request(request, CreateMessageResultSchema);`


**Repository:** [toby/mirror-mcp](https://github.com/toby/mirror-mcp)
**Server names:** io.github.toby/mirror-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (3 matches):
- `mirror-mcp-main/src/reflection-engine.ts`
  - `method: 'sampling/createMessage',`
- `mirror-mcp-main/src/reflection-engine.ts`
  - `import { CreateMessageRequestSchema } from '@modelcontextprotocol/sdk/types.js';`
- `mirror-mcp-main/src/reflection-engine.ts`
  - `CreateMessageRequestSchema`


**Repository:** [tuananh/hyper-mcp](https://github.com/tuananh/hyper-mcp)
**Server names:** io.github.tuananh/hyper-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (18 matches):
- `hyper-mcp-main/src/service.rs`
  - `host_fn!(create_message(ctx: PluginServiceContext; sampling_msg: Json<CreateMessageRequestParam>) -> Json<CreateMessageResult> {`
- `hyper-mcp-main/src/service.rs`
  - `ctx.handle.block_on(peer.create_message(sampling_msg)).map(Json).map_err(Error::from)`
- `hyper-mcp-main/templates/plugins/go/imports.go`
  - `// CreateMessage Request message creation through the client's sampling interface.`
- `hyper-mcp-main/templates/plugins/go/imports.go`
  - `// It takes input of CreateMessageRequestParam ()`
- `hyper-mcp-main/templates/plugins/go/imports.go`
  - `// And it returns an output *CreateMessageResult ()`
  - *... and 13 more matches*


**Repository:** [wavyrai/rm-mcp](https://github.com/wavyrai/rm-mcp)
**Server names:** io.github.wavyrai/rm-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (3 matches):
- `rm-mcp-main/test_server.py`
  - `from mcp.types import ClientCapabilities, SamplingCapability`
- `rm-mcp-main/test_server.py`
  - `mock_caps = ClientCapabilities(sampling=SamplingCapability())`
- `rm-mcp-main/rm_mcp/ocr/sampling.py`
  - `result = await session.create_message(`


**Repository:** [xieyuschen/gopls-mcp](https://github.com/xieyuschen/gopls-mcp)
**Server names:** io.github.xieyuschen/gopls-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (14 matches):
- `gopls-mcp-master/tools/internal/mcp/generate.go`
  - `"CreateMessageRequest": {`
- `gopls-mcp-master/tools/internal/mcp/generate.go`
  - `"CreateMessageResult": {},`
- `gopls-mcp-master/tools/internal/mcp/mcp_test.go`
  - `CreateMessageHandler: func(context.Context, *ClientSession, *CreateMessageParams) (*CreateMessageResult, error) {`
- `gopls-mcp-master/tools/internal/mcp/mcp_test.go`
  - `return &CreateMessageResult{Model: "aModel"}, nil`
- `gopls-mcp-master/tools/internal/mcp/mcp_test.go`
  - `res, err := ss.CreateMessage(ctx, &CreateMessageParams{})`
  - *... and 9 more matches*


**Repository:** [yarnabrina/learn-model-context-protocol](https://github.com/yarnabrina/learn-model-context-protocol)
**Server names:** io.github.yarnabrina/mcp-learning
**Sources:** registry
**Classification:** `implementation`
**Evidence** (8 matches):
- `learn-model-context-protocol-main/src/mcp_learning/mcp_server/simplification.py`
  - `response = await context.session.create_message(`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_client/client.py`
  - `CreateMessageRequestParams,`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_client/client.py`
  - `CreateMessageResult,`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_client/client.py`
  - `parameters: CreateMessageRequestParams,`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_client/client.py`
  - `) -> CreateMessageResult | ErrorData:`
  - *... and 3 more matches*


## SERVERS WITH ELICITATION IMPLEMENTATION


**Repository:** [Algiras/debugium](https://github.com/Algiras/debugium)
**Server names:** io.github.Algiras/debugium
**Sources:** registry
**Classification:** `implementation`
**Evidence** (18 matches):
- `debugium-main/crates/debugium-server/src/mcp/mod.rs`
  - `"method": "elicitation/create",`
- `debugium-main/crates/debugium-server/src/mcp/mod.rs`
  - `"method": "elicitation/createUrl",`
- `debugium-main/crates/debugium-server/src/mcp/mod.rs`
  - `/// Supports optional elicitation (form/url) when the connecting client`
- `debugium-main/crates/debugium-server/src/mcp/mod.rs`
  - `elicitation_form: bool,`
- `debugium-main/crates/debugium-server/src/mcp/mod.rs`
  - `elicitation_url: bool,`
  - *... and 13 more matches*


**Repository:** [ChiR24/Unreal_mcp](https://github.com/ChiR24/Unreal_mcp)
**Server names:** ai.smithery/ChiR24-unreal_mcp, ai.smithery/ChiR24-unreal_mcp_server, io.github.ChiR24/unreal-engine-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (16 matches):
- `Unreal_mcp-main/src/server/tool-registry.ts`
  - `import { createElicitationHelper, PrimitiveSchema } from '../utils/elicitation.js';`
- `Unreal_mcp-main/src/server/tool-registry.ts`
  - `const elicitation = createElicitationHelper(this.server, this.logger);`
- `Unreal_mcp-main/src/server/tool-registry.ts`
  - `elicit: elicitation.elicit,`
- `Unreal_mcp-main/src/server/tool-registry.ts`
  - `supportsElicitation: elicitation.supports,`
- `Unreal_mcp-main/src/server/tool-registry.ts`
  - `elicitationTimeoutMs: this.defaultElicitationTimeoutMs,`
  - *... and 11 more matches*


**Repository:** [Defenter-AI/defenter-proxy](https://github.com/Defenter-AI/defenter-proxy)
**Server names:** io.github.Defenter-AI/defenter-proxy
**Sources:** registry
**Classification:** `implementation`
**Evidence** (9 matches):
- `defenter-proxy-main/src/wrapper/server.py`
  - `elicitation_handler=security_middleware.secure_elicitation_handler,`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `async def secure_elicitation_handler(self, message, response_type, params, context):`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `# FIXME: elicitation message, params, and context should be redacted before logging`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `self.logger.info(f"secure_elicitation_handler: "`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `method='elicitation/request',`
  - *... and 4 more matches*


**Repository:** [KuudoAI/amazon_ads_mcp](https://github.com/KuudoAI/amazon_ads_mcp)
**Server names:** io.github.KuudoAI/amazon_ads_mcp, io.github.tspicer/amazon_ads_mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (5 matches):
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/builtin_tools.py`
  - `result = await ctx.elicit(`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/builtin_tools.py`
  - `This tool uses MCP elicitation to present available profiles to the user`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/builtin_tools.py`
  - `2. Present them to the user via elicitation`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/builtin_tools.py`
  - `# Define the selection structure for elicitation`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/builtin_tools.py`
  - `# Use elicitation to let user select`


**Repository:** [MCPower-Security/mcpower-proxy](https://github.com/MCPower-Security/mcpower-proxy)
**Server names:** io.github.MCPower-Security/mcpower-proxy, io.github.ai-mcpower/mcpower-proxy
**Sources:** registry
**Classification:** `implementation`
**Evidence** (9 matches):
- `defenter-proxy-main/src/wrapper/server.py`
  - `elicitation_handler=security_middleware.secure_elicitation_handler,`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `async def secure_elicitation_handler(self, message, response_type, params, context):`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `# FIXME: elicitation message, params, and context should be redacted before logging`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `self.logger.info(f"secure_elicitation_handler: "`
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `method='elicitation/request',`
  - *... and 4 more matches*


**Repository:** [MervinPraison/PraisonAI](https://github.com/MervinPraison/PraisonAI)
**Server names:** io.github.MervinPraison/praisonai
**Sources:** registry
**Classification:** `implementation`
**Evidence** (21 matches):
- `PraisonAI-main/src/praisonai/tests/unit/mcp_server/test_elicitation.py`
  - `result = asyncio.run(handler.elicit(request))`
- `PraisonAI-main/src/praisonai/tests/unit/mcp_server/test_elicitation.py`
  - `def test_elicitation_modes(self):`
- `PraisonAI-main/src/praisonai/tests/unit/mcp_server/test_elicitation.py`
  - `"""Test elicitation mode values."""`
- `PraisonAI-main/src/praisonai/tests/unit/mcp_server/test_elicitation.py`
  - `from praisonai.mcp_server.elicitation import ElicitationMode`
- `PraisonAI-main/src/praisonai/tests/unit/mcp_server/test_elicitation.py`
  - `def test_elicitation_statuses(self):`
  - *... and 16 more matches*


**Repository:** [Oortonaut/mcacp](https://github.com/Oortonaut/mcacp)
**Server names:** io.github.Oortonaut/mcacp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (11 matches):
- `mcacp-master/tests/permissions.test.ts`
  - `it('falls back to operator mode when no elicitation sender is set', async () => {`
- `mcacp-master/tests/permissions.test.ts`
  - `it('sends elicitation and returns selected option', async () => {`
- `mcacp-master/src/server/index.ts`
  - `import type { ElicitRequestFormParams } from '@modelcontextprotocol/sdk/types.js';`
- `mcacp-master/src/server/index.ts`
  - `requestedSchema: schema as ElicitRequestFormParams['requestedSchema'],`
- `mcacp-master/src/server/index.ts`
  - `} as ElicitRequestFormParams['requestedSchema'],`
  - *... and 6 more matches*


**Repository:** [OtherVibes/mcp-as-a-judge](https://github.com/OtherVibes/mcp-as-a-judge)
**Server names:** io.github.OtherVibes/mcp-as-a-judge
**Sources:** registry
**Classification:** `implementation`
**Evidence** (26 matches):
- `mcp-as-a-judge-main/test_real_scenario.py`
  - `# Create a mock context that doesn't support elicitation`
- `mcp-as-a-judge-main/test_real_scenario.py`
  - `elif "MCP elicitation failed" in result:`
- `mcp-as-a-judge-main/tests/test_enhanced_features.py`
  - `elicitation functionality.`
- `mcp-as-a-judge-main/tests/test_enhanced_features.py`
  - `# With elicitation provider, we expect either success or fallback message`
- `mcp-as-a-judge-main/src/mcp_as_a_judge/models.py`
  - `# with dynamic model generation in _generate_dynamic_elicitation_model()`
  - *... and 21 more matches*


**Repository:** [SamMorrowDrums/remarkable-mcp](https://github.com/SamMorrowDrums/remarkable-mcp)
**Server names:** io.github.SamMorrowDrums/remarkable
**Sources:** registry
**Classification:** `implementation`
**Evidence** (19 matches):
- `remarkable-mcp-main/test_server.py`
  - `from mcp.types import ClientCapabilities, ElicitationCapability`
- `remarkable-mcp-main/test_server.py`
  - `mock_caps = ClientCapabilities(elicitation=ElicitationCapability())`
- `remarkable-mcp-main/test_server.py`
  - `def test_client_supports_elicitation(self):`
- `remarkable-mcp-main/test_server.py`
  - `"""Test client_supports_elicitation."""`
- `remarkable-mcp-main/test_server.py`
  - `from remarkable_mcp.capabilities import client_supports_elicitation`
  - *... and 14 more matches*


**Repository:** [SmartBear/smartbear-mcp](https://github.com/SmartBear/smartbear-mcp)
**Server names:** com.smartbear/smartbear-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (24 matches):
- `smartbear-mcp-main/src/tests/unit/common/server.test.ts`
  - `// Mock elicitation support`
- `smartbear-mcp-main/src/tests/unit/common/server.test.ts`
  - `// Since elicitation is supported, the wrapper should call elicitInput`
- `smartbear-mcp-main/src/tests/unit/common/pollyfills.test.ts`
  - `it("should return polyfill result when elicitation is not supported", async () => {`
- `smartbear-mcp-main/src/tests/unit/common/pollyfills.test.ts`
  - `it("should return elicit result when elicitation succeeds", async () => {`
- `smartbear-mcp-main/src/tests/unit/common/pollyfills.test.ts`
  - `it("should return polyfill result when elicitation throws error", async () => {`
  - *... and 19 more matches*


**Repository:** [YuliiaKovalova/dotnet-template-mcp](https://github.com/YuliiaKovalova/dotnet-template-mcp)
**Server names:** io.github.YuliiaKovalova/dotnet-template-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (24 matches):
    dotnet-template-mcp-main/src/Microsoft.TemplateEngine.MCP/McpFeatureFlags.cs:
      /// Environment variable to enable/disable elicitation for interactive parameter collection.
    dotnet-template-mcp-main/src/Microsoft.TemplateEngine.MCP/McpFeatureFlags.cs:
      /// Whether elicitation is enabled for interactive parameter collection.
    dotnet-template-mcp-main/src/Microsoft.TemplateEngine.MCP/Host/ElicitationHelper.cs:
      var result = await server.ElicitAsync(new ElicitRequestParams
    dotnet-template-mcp-main/src/Microsoft.TemplateEngine.MCP/Host/ElicitationHelper.cs:
      internal static ElicitRequestParams.RequestSchema BuildSchemaFromParameters(
    dotnet-template-mcp-main/src/Microsoft.TemplateEngine.MCP/Host/ElicitationHelper.cs:
      var properties = new Dictionary<string, ElicitRequestParams.PrimitiveSchemaDefinition>();
    ... and 19 more matches


**Repository:** [anirbanbasu/pymcp](https://github.com/anirbanbasu/pymcp)
**Server names:** ai.smithery/anirbanbasu-pymcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (10 matches):
- `pymcp-master/tests/test_server.py`
  - `from fastmcp.client.elicitation import ElicitRequestParams, ElicitResult`
- `pymcp-master/tests/test_server.py`
  - `params: ElicitRequestParams,`
- `pymcp-master/tests/test_server.py`
  - `) -> ElicitResult:`
- `pymcp-master/tests/test_server.py`
  - `async def random_elicitation_handler(`
- `pymcp-master/tests/test_server.py`
  - `logger.info(f"Received elicitation request: {message}")`
  - *... and 5 more matches*


**Repository:** [apiarya/wemo-mcp-server](https://github.com/apiarya/wemo-mcp-server)
**Server names:** io.github.apiarya/wemo
**Sources:** registry
**Classification:** `implementation`
**Evidence** (8 matches):
- `wemo-mcp-server-main/tests/test_server.py`
  - `async def test_explicit_subnet_skips_elicitation(self):`
- `wemo-mcp-server-main/tests/test_server.py`
  - `# Either completes or errors — main check is no elicitation called`
- `wemo-mcp-server-main/tests/test_server.py`
  - `# scan_network elicitation paths (covers L403-418)`
- `wemo-mcp-server-main/tests/test_server.py`
  - `async def test_elicitation_accept_with_subnet(self):`
- `wemo-mcp-server-main/tests/test_server.py`
  - `async def test_elicitation_cancel_returns_error(self):`
  - *... and 3 more matches*


**Repository:** [cashfree/cashfree-mcp](https://github.com/cashfree/cashfree-mcp)
**Server names:** io.github.cashfree/cashfree-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (31 matches):
- `cashfree-mcp-main/src/types.ts`
  - `elicitation?: ElicitationConfiguration;`
- `cashfree-mcp-main/src/openapi/helpers.ts`
  - `method: "elicitation/create",`
- `cashfree-mcp-main/src/openapi/helpers.ts`
  - `ElicitRequest,`
- `cashfree-mcp-main/src/openapi/helpers.ts`
  - `): ElicitRequest {`
- `cashfree-mcp-main/src/openapi/helpers.ts`
  - `// If endpoint has elicitation config, make fields optional`
  - *... and 26 more matches*


**Repository:** [containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server)
**Server names:** io.github.containers/kubernetes-mcp-server
**Sources:** registry
**Classification:** `implementation`
**Evidence** (21 matches):
- `kubernetes-mcp-server-main/pkg/mcp/elicit.go`
  - `func (s *sessionElicitor) Elicit(ctx context.Context, params *api.ElicitParams) (*api.ElicitResult, error) {`
- `kubernetes-mcp-server-main/pkg/mcp/elicit.go`
  - `return &api.ElicitResult{Action: result.Action, Content: result.Content}, nil`
- `kubernetes-mcp-server-main/pkg/mcp/elicit.go`
  - `// ErrElicitationNotSupported is returned when the MCP client does not support elicitation.`
- `kubernetes-mcp-server-main/pkg/mcp/elicit.go`
  - `var ErrElicitationNotSupported = errors.New("client does not support elicitation")`
- `kubernetes-mcp-server-main/pkg/mcp/elicit.go`
  - `// The go-sdk does not export a typed error for unsupported elicitation.`
  - *... and 16 more matches*


**Repository:** [cycloidio/cycloid-mcp-server](https://github.com/cycloidio/cycloid-mcp-server)
**Server names:** io.cycloid.mcp/server
**Sources:** registry
**Classification:** `implementation`
**Evidence** (16 matches):
- `cycloid-mcp-server-master/src/components/stacks.py`
  - `stack_name_result = await ctx.elicit(stack_name_prompt, response_type=str)`
- `cycloid-mcp-server-master/src/components/stacks.py`
  - `use_case_result = await ctx.elicit(use_case_prompt, response_type=available_use_cases)`
- `cycloid-mcp-server-master/src/components/stacks.py`
  - `service_catalog_result = await ctx.elicit(`
- `cycloid-mcp-server-master/src/components/stacks.py`
  - `confirmation_result = await ctx.elicit(summary, response_type=["confirm"])`
- `cycloid-mcp-server-master/src/components/stacks.py`
  - `f"Use case elicitation result: action={use_case_result.action}, "  # noqa: E501`
  - *... and 11 more matches*


**Repository:** [dynatrace-oss/Dynatrace-mcp](https://github.com/dynatrace-oss/Dynatrace-mcp)
**Server names:** io.github.dynatrace-oss/Dynatrace-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (1 matches):
- `dynatrace-mcp-main/src/index.ts`
  - `return false; // Default to deny if elicitation fails`


**Repository:** [grupo-avispa/dsr_mcp_server](https://github.com/grupo-avispa/dsr_mcp_server)
**Server names:** io.github.grupo-avispa/dsr_mcp_server
**Sources:** registry
**Classification:** `implementation`
**Evidence** (1 matches):
- `dsr_mcp_server-master/src/dsr_mcp_server/server.py`
  - `result = await ctx.elicit(`


**Repository:** [hashicorp/terraform-mcp-server](https://github.com/hashicorp/terraform-mcp-server)
**Server names:** io.github.hashicorp/terraform-mcp-server
**Sources:** registry
**Classification:** `implementation`
**Evidence** (18 matches):
- `terraform-mcp-server-main/pkg/tools/dynamic_tool.go`
  - `// createDynamicTFEToolWithElicitation creates a TFE tool with dynamic availability checking that also needs MCPServer for elicitation`
- `terraform-mcp-server-main/pkg/tools/tfe/create_no_code_workspace_test.go`
  - `// Verify the tool has elicitation capabilities through its configuration`
- `terraform-mcp-server-main/pkg/tools/tfe/create_no_code_workspace.go`
  - `mcp.WithDescription(`Creates a new Terraform No Code module workspace. The tool uses the MCP elicitation feature to automatically discover and collect`
- `terraform-mcp-server-main/pkg/tools/tfe/create_no_code_workspace.go`
  - `elicitationProperties, requestedVars := buildElicitationSchema(moduleMetadata, noCodeModule)`
- `terraform-mcp-server-main/pkg/tools/tfe/create_no_code_workspace.go`
  - `result, err := requestVariableValues(ctx, mcpServer, params.noCodeModuleID, elicitationProperties, requestedVars)`
  - *... and 13 more matches*


**Repository:** [iowarp/clio-kit](https://github.com/iowarp/clio-kit)
**Server names:** io.github.iowarp/adios-mcp, io.github.iowarp/arxiv-mcp, io.github.iowarp/chronolog-mcp (+13 more)
**Sources:** registry
**Classification:** `implementation`
**Evidence** (5 matches):
- `clio-kit-main/clio-kit-mcp-servers/hdf5/src/hdf5_mcp/server.py`
  - `format_result = await ctx.elicit(`
- `clio-kit-main/clio-kit-mcp-servers/hdf5/src/hdf5_mcp/server.py`
  - `- Context-aware operations (progress, LLM sampling, elicitation)`
- `clio-kit-main/clio-kit-mcp-servers/hdf5/src/hdf5_mcp/server.py`
  - `ctx: Context for elicitation`
- `clio-kit-main/clio-kit-mcp-servers/hdf5/src/hdf5_mcp/server.py`
  - `# Ask user for format (if ctx available and client supports elicitation)`
- `clio-kit-main/clio-kit-mcp-servers/hdf5/src/hdf5_mcp/server.py`
  - `logger.warning(f"Error during elicitation: {e}")`


**Repository:** [jongalloway/dotnet-mcp](https://github.com/jongalloway/dotnet-mcp)
**Server names:** io.github.jongalloway/dotnet-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (26 matches):
    dotnet-mcp-main/DotNetMcp/Tools/Cli/DotNetCliTools.Project.Consolidated.cs:
      var elicitResult = await server.ElicitAsync(new ElicitRequestParams
    dotnet-mcp-main/DotNetMcp/Tools/Cli/DotNetCliTools.Project.Consolidated.cs:
      RequestedSchema = new ElicitRequestParams.RequestSchema
    dotnet-mcp-main/DotNetMcp/Tools/Cli/DotNetCliTools.Project.Consolidated.cs:
      Properties = new Dictionary<string, ElicitRequestParams.PrimitiveSchemaDefinition>
    dotnet-mcp-main/DotNetMcp/Tools/Cli/DotNetCliTools.Project.Consolidated.cs:
      ["confirmed"] = new ElicitRequestParams.BooleanSchema
    dotnet-mcp-main/DotNetMcp/Tools/Cli/DotNetCliTools.Project.Consolidated.cs:
      // Request confirmation via elicitation when client supports it
    ... and 21 more matches


**Repository:** [justfsl50/expense-mcp](https://github.com/justfsl50/expense-mcp)
**Server names:** io.github.justfsl50/expense-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (2 matches):
- `expense-mcp-main/server.py`
  - `result = await ctx.elicit(`
- `expense-mcp-main/server.py`
  - `"""Schema for expense deletion confirmation elicitation."""`


**Repository:** [karashiiro/my-cool-proxy](https://github.com/karashiiro/my-cool-proxy)
**Server names:** io.github.karashiiro/my-cool-proxy
**Sources:** registry
**Classification:** `implementation`
**Evidence** (28 matches):
- `my-cool-proxy-main/packages/mcp-client/src/client-manager.ts`
  - `// servers know they can send sampling/elicitation requests through us`
- `my-cool-proxy-main/packages/mcp-client/src/client-manager.ts`
  - `* know they can send sampling/elicitation requests through the proxy.`
- `my-cool-proxy-main/packages/mcp-client/src/client-manager.ts`
  - `// Forward elicitation capability if downstream supports it`
- `my-cool-proxy-main/packages/mcp-client/src/client-manager.ts`
  - `if (downstreamCaps.elicitation) {`
- `my-cool-proxy-main/packages/mcp-client/src/client-manager.ts`
  - `caps.elicitation = downstreamCaps.elicitation;`
  - *... and 23 more matches*


**Repository:** [portel-dev/ncp](https://github.com/portel-dev/ncp)
**Server names:** io.github.portel-dev/ncp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (31 matches):
- `ncp-main/scripts/test-http-auth.js`
  - `import { detectHTTPCredentials } from '../dist/utils/elicitation-helper.js';`
- `ncp-main/tests/test-server-notifications.ts`
  - `console.log('3. If elicitation times out, notification will be queued');`
- `ncp-main/tests/manual/test-network-permissions.js`
  - `* Run this to see how the elicitation adapter works.`
- `ncp-main/tests/manual/test-network-permissions.js`
  - `// Create a mock elicitation function (simulates user input)`
- `ncp-main/tests/manual/test-network-permissions.js`
  - `// Create NetworkPolicyManager with elicitation support`
  - *... and 26 more matches*


**Repository:** [refined-element/lightning-enable-mcp](https://github.com/refined-element/lightning-enable-mcp)
**Server names:** io.github.refined-element/lightning-enable-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (27 matches):
    lightning-enable-mcp-main/dotnet/src/LightningEnable.Mcp/Tools/AccessL402ResourceTool.cs:
      var schema = new ElicitRequestParams.RequestSchema
    lightning-enable-mcp-main/dotnet/src/LightningEnable.Mcp/Tools/AccessL402ResourceTool.cs:
      ["approved"] = new ElicitRequestParams.BooleanSchema
    lightning-enable-mcp-main/dotnet/src/LightningEnable.Mcp/Tools/AccessL402ResourceTool.cs:
      var response = await server.ElicitAsync(new ElicitRequestParams
    lightning-enable-mcp-main/dotnet/src/LightningEnable.Mcp/Tools/AccessL402ResourceTool.cs:
      ["confirmAmount"] = new ElicitRequestParams.StringSchema
    lightning-enable-mcp-main/dotnet/src/LightningEnable.Mcp/Tools/AccessL402ResourceTool.cs:
      /// <param name="server">MCP server for elicitation.</param>
    ... and 22 more matches


**Repository:** [romainsantoli-web/mcp-openclaw](https://github.com/romainsantoli-web/mcp-openclaw)
**Server names:** io.github.romainsantoli-web/mcp-openclaw
**Sources:** registry
**Classification:** `implementation`
**Evidence** (25 matches):
- `mcp-openclaw-main/tests/test_cov_100f.py`
  - `proxy), spec_compliance (elicitation types, audio, JSON schema, SSE, icons), platform_audit`
- `mcp-openclaw-main/tests/test_cov_100f.py`
  - `# spec_compliance — elicitation, audio, JSON schema, SSE, icons`
- `mcp-openclaw-main/tests/test_cov_100f.py`
  - `def test_elicitation_unsupported_type(self, tmp_path):`
- `mcp-openclaw-main/tests/test_cov_100f.py`
  - `from src.spec_compliance import elicitation_audit`
- `mcp-openclaw-main/tests/test_cov_100f.py`
  - `cfg = _write(tmp_path, {"mcp": {"elicitation": {`
  - *... and 20 more matches*


**Repository:** [thehesiod/psquare-mcp](https://github.com/thehesiod/psquare-mcp)
**Server names:** io.github.thehesiod/psquare
**Sources:** registry
**Classification:** `implementation`
**Evidence** (7 matches):
- `psquare-mcp-main/src/parentsquare_mcp/server.py`
  - `result = await ctx.elicit(`
- `psquare-mcp-main/src/parentsquare_mcp/server.py`
  - `"""Schema for MFA code elicitation."""`
- `psquare-mcp-main/src/parentsquare_mcp/server.py`
  - `"""Try inline elicitation for MFA code; fall back to text message if unsupported."""`
- `psquare-mcp-main/src/parentsquare_mcp/server.py`
  - `# Try elicitation — prompt user for MFA code inline`
- `psquare-mcp-main/src/parentsquare_mcp/server.py`
  - `# Client doesn't support elicitation — fall back to text message`
  - *... and 2 more matches*


**Repository:** [tuananh/hyper-mcp](https://github.com/tuananh/hyper-mcp)
**Server names:** io.github.tuananh/hyper-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (30 matches):
- `hyper-mcp-main/src/service.rs`
  - `host_fn!(create_elicitation(ctx: PluginServiceContext; elicitation_msg: Json<CreateElicitationRequestParamWithTimeout>) -> Json<CreateElicitationResul`
- `hyper-mcp-main/src/service.rs`
  - `let elicitation_msg = elicitation_msg.into_inner();`
- `hyper-mcp-main/src/service.rs`
  - `if peer.supports_elicitation() {`
- `hyper-mcp-main/src/service.rs`
  - `if let Some(timeout) = elicitation_msg.timeout {`
- `hyper-mcp-main/src/service.rs`
  - `tracing::info!("Creating elicitation from {} with timeout {:?}", ctx.plugin_name, timeout);`
  - *... and 25 more matches*


**Repository:** [wavyrai/rm-mcp](https://github.com/wavyrai/rm-mcp)
**Server names:** io.github.wavyrai/rm-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (19 matches):
- `rm-mcp-main/test_server.py`
  - `from mcp.types import ClientCapabilities, ElicitationCapability`
- `rm-mcp-main/test_server.py`
  - `mock_caps = ClientCapabilities(elicitation=ElicitationCapability())`
- `rm-mcp-main/test_server.py`
  - `def test_client_supports_elicitation(self):`
- `rm-mcp-main/test_server.py`
  - `"""Test client_supports_elicitation."""`
- `rm-mcp-main/test_server.py`
  - `from rm_mcp.capabilities import client_supports_elicitation`
  - *... and 14 more matches*


**Repository:** [xorrkaz/cml-mcp](https://github.com/xorrkaz/cml-mcp)
**Server names:** io.github.xorrkaz/cml-mcp
**Sources:** registry
**Classification:** `implementation`
**Evidence** (7 matches):
- `cml-mcp-main/src/cml_mcp/tools/labs.py`
  - `result = await ctx.elicit("Are you sure you want to wipe the lab?", response_type=None)`
- `cml-mcp-main/src/cml_mcp/tools/labs.py`
  - `result = await ctx.elicit("Are you sure you want to delete the lab?", response_type=None)`
- `cml-mcp-main/src/cml_mcp/tools/annotations.py`
  - `result = await ctx.elicit("Are you sure you want to delete the annotation?", response_type=None)`
- `cml-mcp-main/src/cml_mcp/tools/users_groups.py`
  - `result = await ctx.elicit("Are you sure you want to delete this user?", response_type=None)`
- `cml-mcp-main/src/cml_mcp/tools/users_groups.py`
  - `result = await ctx.elicit("Are you sure you want to delete this group?", response_type=None)`
  - *... and 2 more matches*


**Repository:** [yarnabrina/learn-model-context-protocol](https://github.com/yarnabrina/learn-model-context-protocol)
**Server names:** io.github.yarnabrina/mcp-learning
**Sources:** registry
**Classification:** `implementation`
**Evidence** (29 matches):
- `learn-model-context-protocol-main/src/mcp_learning/mcp_server/exponentiation.py`
  - `elicitation_result = await context.elicit(`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_server/exponentiation.py`
  - `await context.report_progress(1, total=2, message="Starting MCP elicitation.")`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_server/exponentiation.py`
  - `await context.report_progress(1, total=2, message="Finished MCP elicitation.")`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_server/exponentiation.py`
  - `match elicitation_result.action:`
- `learn-model-context-protocol-main/src/mcp_learning/mcp_server/exponentiation.py`
  - `corrected_exponent = elicitation_result.data.corrected_exponent`
  - *... and 24 more matches*


## SERVERS WITH SAMPLING REFERENCES (test/docs/import/config)


**Repository:** [Defenter-AI/defenter-proxy](https://github.com/Defenter-AI/defenter-proxy)
**Server names:** io.github.Defenter-AI/defenter-proxy
**Sources:** registry
**Classification:** `reference`
**Evidence** (1 matches):
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `method='sampling/create_message',`


**Repository:** [KuudoAI/amazon_ads_mcp](https://github.com/KuudoAI/amazon_ads_mcp)
**Server names:** io.github.KuudoAI/amazon_ads_mcp, io.github.tspicer/amazon_ads_mcp
**Sources:** registry
**Classification:** `reference`
**Evidence** (7 matches):
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/builtin_tools.py`
  - `the MCP client to support sampling (createMessage capability).`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/sampling_handler.py`
  - `CreateMessageRequestParams,`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/server/sampling_handler.py`
  - `params: CreateMessageRequestParams,`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/utils/sampling_helpers.py`
  - `CreateMessageRequestParams,`
- `amazon_ads_mcp-main/src/amazon_ads_mcp/utils/sampling_helpers.py`
  - `params = CreateMessageRequestParams(`
  - *... and 2 more matches*


**Repository:** [MCPower-Security/mcpower-proxy](https://github.com/MCPower-Security/mcpower-proxy)
**Server names:** io.github.MCPower-Security/mcpower-proxy, io.github.ai-mcpower/mcpower-proxy
**Sources:** registry
**Classification:** `reference`
**Evidence** (1 matches):
- `defenter-proxy-main/src/wrapper/middleware.py`
  - `method='sampling/create_message',`


**Repository:** [composable-delivery/snowfakery-mcp](https://github.com/composable-delivery/snowfakery-mcp)
**Server names:** io.github.composable-delivery/snowfakery-mcp
**Sources:** registry
**Classification:** `reference`
**Evidence** (5 matches):
- `snowfakery-mcp-main/tests/test_agentic.py`
  - `from mcp.types import CreateMessageResult, TextContent`
- `snowfakery-mcp-main/tests/test_agentic.py`
  - `mock_ctx.sample.return_value = CreateMessageResult(`
- `snowfakery-mcp-main/tests/test_agentic.py`
  - `CreateMessageResult(`
- `snowfakery-mcp-main/snowfakery_mcp/tools/agentic.py`
  - `# Note: This assumes the client supports sampling (CreateMessageRequest)`
- `snowfakery-mcp-main/snowfakery_mcp/tools/agentic.py`
  - `# result is likely CreateMessageResult`


**Repository:** [getmockd/mockd](https://github.com/getmockd/mockd)
**Server names:** io.mockd/mockd
**Sources:** registry
**Classification:** `test_only`
**Evidence** (3 matches):
- `mockd-main/pkg/mcp/types.go`
  - `Sampling    *SamplingCapability    `json:"sampling,omitempty"``
- `mockd-main/pkg/mcp/types.go`
  - `// SamplingCapability describes client LLM sampling capability.`
- `mockd-main/pkg/mcp/types.go`
  - `type SamplingCapability struct{}`


## SERVERS WITH ELICITATION REFERENCES (test/docs/import/config)


**Repository:** [Kastalien-Research/thoughtbox](https://github.com/Kastalien-Research/thoughtbox)
**Server names:** io.github.Kastalien-Research/thoughtbox
**Sources:** registry
**Classification:** `docs_only`
**Evidence** (15 matches):
- `thoughtbox-main/docs/2025-11-25.ts`
  - `* Whether the client supports task-augmented elicitation/create requests.`
- `thoughtbox-main/docs/2025-11-25.ts`
  - `* @category `elicitation/create``
- `thoughtbox-main/docs/2025-11-25.ts`
  - `method: "elicitation/create";`
- `thoughtbox-main/docs/2025-11-25.ts`
  - `elicitations: ElicitRequestURLParams[];`
- `thoughtbox-main/docs/2025-11-25.ts`
  - `export interface ElicitRequestFormParams extends TaskAugmentedRequestParams {`
  - *... and 10 more matches*


**Repository:** [LinuxSuRen/atest-mcp-server](https://github.com/LinuxSuRen/atest-mcp-server)
**Server names:** io.github.LinuxSuRen/atest-mcp-server
**Sources:** registry
**Classification:** `test_only`
**Evidence** (1 matches):
- `atest-mcp-server-master/pkg/atest.go`
  - `var elicitResult *mcp.ElicitResult`


**Repository:** [SajmustafaKe/frappe-dev-mcp-server](https://github.com/SajmustafaKe/frappe-dev-mcp-server)
**Server names:** io.github.SajmustafaKe/frappe-dev-mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (1 matches):
- `frappe-dev-mcp-server-main/frappe_mcp/server/types.py`
  - `elicitation: dict[str, Any] | None = None`


**Repository:** [SonarSource/sonarqube-mcp-server](https://github.com/SonarSource/sonarqube-mcp-server)
**Server names:** io.github.SonarSource/sonarqube-mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (1 matches):
- `sonarqube-mcp-server-master/src/main/java/org/sonarsource/sonarqube/mcp/client/McpClientManager.java`
  - `.elicitation()`


**Repository:** [akougkas/zulipchat-mcp](https://github.com/akougkas/zulipchat-mcp)
**Server names:** io.github.akougkas/zulipchat
**Sources:** registry
**Classification:** `reference`
**Evidence** (2 matches):
- `zulipchat-mcp-main/src/zulipchat_mcp/tools/ai_analytics.py`
  - `High-level analytical tools that use LLM elicitation for sophisticated insights.`
- `zulipchat-mcp-main/src/zulipchat_mcp/tools/search.py`
  - `Analytics moved to ai_analytics.py for LLM elicitation.`


**Repository:** [alisaitteke/docker-mcp](https://github.com/alisaitteke/docker-mcp)
**Server names:** io.github.alisaitteke/docker-mcp
**Sources:** registry
**Classification:** `docs_only`
**Evidence** (4 matches):
- `docker-mcp-master/src/index.ts`
  - `// Enable elicitation capability - client will negotiate if it supports it`
- `docker-mcp-master/src/index.ts`
  - `// If client doesn't support elicitation, we fall back to confirm parameter`
- `docker-mcp-master/src/tools/index.ts`
  - `// Enable elicitation capability if client supports it`
- `docker-mcp-master/src/tools/volumes.ts`
  - `// Fallback to confirm parameter if elicitation fails`


**Repository:** [aws/mcp-proxy-for-aws](https://github.com/aws/mcp-proxy-for-aws)
**Server names:** io.github.aws/mcp-proxy-for-aws
**Sources:** registry
**Classification:** `test_only`
**Evidence** (14 matches):
- `mcp-proxy-for-aws-main/tests/integ/test_proxy_simple_mcp_server.py`
  - `async def test_handle_elicitation_when_accepting(`
- `mcp-proxy-for-aws-main/tests/integ/test_proxy_simple_mcp_server.py`
  - `"""Test calling tool which supports elicitation and accepting it."""`
- `mcp-proxy-for-aws-main/tests/integ/test_proxy_simple_mcp_server.py`
  - `tool_input = {'elicitation_expected': 'Accept'}`
- `mcp-proxy-for-aws-main/tests/integ/test_proxy_simple_mcp_server.py`
  - `async def test_handle_elicitation_when_declining(`
- `mcp-proxy-for-aws-main/tests/integ/test_proxy_simple_mcp_server.py`
  - `"""Test calling tool which supports elicitation and declining it."""`
  - *... and 9 more matches*


**Repository:** [bekirdag/docdex](https://github.com/bekirdag/docdex)
**Server names:** io.github.bekirdag/docdex
**Sources:** registry
**Classification:** `docs_only`
**Evidence** (2 matches):
- `docdex-main/src/mcp_server.rs`
  - `if let Some(elicitation) = req_caps.get("elicitation") {`
- `docdex-main/src/mcp_server.rs`
  - `obj.insert("elicitation".to_string(), elicitation.clone());`


**Repository:** [cloudflare/mcp-server-cloudflare](https://github.com/cloudflare/mcp-server-cloudflare)
**Server names:** com.cloudflare.mcp/mcp
**Sources:** registry
**Classification:** `reference`
**Evidence** (1 matches):
- `mcp-server-cloudflare-main/packages/mcp-common/src/tools/hyperdrive.tools.ts`
  - `// TODO: Once elicitation is available in MCP as a way to securely pass parameters, re-enable this tool. See: https://github.com/modelcontextprotocol/`


**Repository:** [cyanheads/clinicaltrialsgov-mcp-server](https://github.com/cyanheads/clinicaltrialsgov-mcp-server)
**Server names:** io.github.cyanheads/clinicaltrialsgov-mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (1 matches):
- `clinicaltrialsgov-mcp-server-main/src/mcp-server/tools/utils/toolDefinition.ts`
  - `* - `sendRequest`: Send a new request to the client (e.g., for elicitation).`


**Repository:** [cyanheads/mcp-ts-template](https://github.com/cyanheads/mcp-ts-template)
**Server names:** io.github.cyanheads/mcp-ts-template
**Sources:** registry
**Classification:** `reference`
**Evidence** (8 matches):
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-madlibs-elicitation.tool.ts`
  - `* @fileoverview Complete, declarative definition for the 'template_madlibs_elicitation' tool.`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-madlibs-elicitation.tool.ts`
  - `* @module src/mcp-server/tools/definitions/template-madlibs-elicitation.tool`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-madlibs-elicitation.tool.ts`
  - `const TOOL_NAME = 'template_madlibs_elicitation';`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-madlibs-elicitation.tool.ts`
  - `'Plays a game of Mad Libs. If any parts of speech (noun, verb, adjective) are missing, it will use elicitation to ask the user for them.';`
- `mcp-ts-template-main/src/mcp-server/tools/definitions/template-madlibs-elicitation.tool.ts`
  - `// Check the elicitation action before parsing the value`
  - *... and 3 more matches*


**Repository:** [cyanheads/protein-mcp-server](https://github.com/cyanheads/protein-mcp-server)
**Server names:** io.github.cyanheads/protein-mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (4 matches):
- `protein-mcp-server-main/src/mcp-server/server.ts`
  - `elicitation: {},`
- `protein-mcp-server-main/src/mcp-server/tools/utils/toolDefinition.ts`
  - `* - `sendRequest`: Send a new request to the client (e.g., for elicitation).`
- `protein-mcp-server-main/src/mcp-server/tools/utils/toolHandlerFactory.ts`
  - `// Define a type for a context that may have elicitation capabilities.`
- `protein-mcp-server-main/src/mcp-server/tools/utils/toolHandlerFactory.ts`
  - `// If the SDK context supports elicitation, add it to our app context.`


**Repository:** [cyanheads/pubmed-mcp-server](https://github.com/cyanheads/pubmed-mcp-server)
**Server names:** io.github.cyanheads/pubmed-mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (1 matches):
- `pubmed-mcp-server-main/src/mcp-server/tools/utils/toolDefinition.ts`
  - `* - `sendRequest`: Send a new request to the client (e.g., for elicitation).`


**Repository:** [cyanheads/survey-mcp-server](https://github.com/cyanheads/survey-mcp-server)
**Server names:** io.github.cyanheads/survey-mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (4 matches):
- `survey-mcp-server-main/src/mcp-server/server.ts`
  - `elicitation: {},`
- `survey-mcp-server-main/src/mcp-server/tools/utils/toolDefinition.ts`
  - `* - `sendRequest`: Send a new request to the client (e.g., for elicitation).`
- `survey-mcp-server-main/src/mcp-server/tools/utils/toolHandlerFactory.ts`
  - `// Define a type for a context that may have elicitation capabilities.`
- `survey-mcp-server-main/src/mcp-server/tools/utils/toolHandlerFactory.ts`
  - `// If the SDK context supports elicitation, add it to our app context.`


**Repository:** [futuresearch/everyrow-sdk](https://github.com/futuresearch/everyrow-sdk)
**Server names:** io.github.futuresearch/everyrow-mcp
**Sources:** registry
**Subfolder:** everyrow-mcp
**Classification:** `reference`
**Evidence** (2 matches):
- `everyrow-sdk-main/everyrow-mcp/src/everyrow_mcp/tool_helpers.py`
  - `"[%s] client=%s/%s sampling=%s elicitation=%s roots=%s ui=%s",`
- `everyrow-sdk-main/everyrow-mcp/src/everyrow_mcp/tool_helpers.py`
  - `caps.elicitation is not None if caps else False,`


**Repository:** [gander-tools/osm-tagging-schema-mcp](https://github.com/gander-tools/osm-tagging-schema-mcp)
**Server names:** io.github.gander-tools/osm-tagging-schema-mcp
**Sources:** registry
**Classification:** `reference`
**Evidence** (6 matches):
- `osm-tagging-schema-mcp-master/src/metadata.ts`
  - `* 1. Form elicitation - Structured forms with validation`
- `osm-tagging-schema-mcp-master/src/metadata.ts`
  - `* 2. URL elicitation - Redirect user to URL for input`
- `osm-tagging-schema-mcp-master/src/metadata.ts`
  - `* - OAuth/authentication flows (URL elicitation)`
- `osm-tagging-schema-mcp-master/src/metadata.ts`
  - `* TODO: Implement elicitation when interactive features are needed`
- `osm-tagging-schema-mcp-master/src/metadata.ts`
  - `export const elicitationMetadata: Record<string, ElicitationMetadata> = {`
  - *... and 1 more matches*


**Repository:** [getmockd/mockd](https://github.com/getmockd/mockd)
**Server names:** io.mockd/mockd
**Sources:** registry
**Classification:** `test_only`
**Evidence** (3 matches):
- `mockd-main/pkg/mcp/types.go`
  - `Elicitation *ElicitationCapability `json:"elicitation,omitempty"``
- `mockd-main/pkg/mcp/types.go`
  - `// ElicitationCapability describes client user info request capability.`
- `mockd-main/pkg/mcp/types.go`
  - `type ElicitationCapability struct{}`


**Repository:** [janwilmake/install-this-mcp](https://github.com/janwilmake/install-this-mcp)
**Server names:** Install This MCP
**Sources:** awesome
**Classification:** `reference`
**Evidence** (1 matches):
- `install-this-mcp-main/server-card.ts`
  - `elicitation?: object;`


**Repository:** [mapbox/mcp-devkit-server](https://github.com/mapbox/mcp-devkit-server)
**Server names:** io.github.mapbox/mcp-devkit-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (26 matches):
- `mcp-devkit-server-main/src/index.ts`
  - `const elicitationTools = getElicitationTools();`
- `mcp-devkit-server-main/src/index.ts`
  - `const enabledElicitationTools = filterTools(elicitationTools, config);`
- `mcp-devkit-server-main/src/index.ts`
  - `// Register elicitation tools if client supports elicitation`
- `mcp-devkit-server-main/src/index.ts`
  - `if (clientCapabilities?.elicitation && enabledElicitationTools.length > 0) {`
- `mcp-devkit-server-main/src/index.ts`
  - `data: `Client supports elicitation. Registering ${enabledElicitationTools.length} elicitation-dependent tools``
  - *... and 21 more matches*


**Repository:** [mapbox/mcp-server](https://github.com/mapbox/mcp-server)
**Server names:** io.github.mapbox/mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (25 matches):
- `mcp-server-main/src/index.ts`
  - `const elicitationTools = getElicitationTools();`
- `mcp-server-main/src/index.ts`
  - `const enabledElicitationTools = filterTools(elicitationTools, config);`
- `mcp-server-main/src/index.ts`
  - `// Register elicitation tools if client supports elicitation`
- `mcp-server-main/src/index.ts`
  - `if (clientCapabilities?.elicitation && enabledElicitationTools.length > 0) {`
- `mcp-server-main/src/index.ts`
  - `data: `Client supports elicitation. Registering ${enabledElicitationTools.length} elicitation-dependent tools``
  - *... and 20 more matches*


**Repository:** [mongodb-js/mongodb-mcp-server](https://github.com/mongodb-js/mongodb-mcp-server)
**Server names:** io.github.mongodb-js/mongodb-mcp-server
**Sources:** registry
**Classification:** `reference`
**Evidence** (17 matches):
- `mongodb-mcp-server-main/scripts/generate/generateToolDocumentation.ts`
  - `elicitation: {`
- `mongodb-mcp-server-main/tests/unit/toolBase.test.ts`
  - `import type { Elicitation } from "../../src/elicitation.js";`
- `mongodb-mcp-server-main/tests/unit/toolBase.test.ts`
  - `elicitation: mockElicitation,`
- `mongodb-mcp-server-main/tests/unit/toolContext.test.ts`
  - `import type { Elicitation } from "../../src/elicitation.js";`
- `mongodb-mcp-server-main/tests/unit/toolContext.test.ts`
  - `const elicitation = {`
  - *... and 12 more matches*


**Repository:** [paiml/rust-mcp-sdk](https://github.com/paiml/rust-mcp-sdk)
**Server names:** io.github.paiml/pmcp
**Sources:** registry
**Classification:** `reference`
**Evidence** (30 matches):
- `rust-mcp-sdk-main/crates/mcp-tester/src/tester.rs`
  - `elicitation: Some(Default::default()),`
- `rust-mcp-sdk-main/crates/mcp-tester/src/tester.rs`
  - `/// - Sends spec-compliant client capabilities (sampling, elicitation, roots)`
- `rust-mcp-sdk-main/crates/mcp-tester/src/tester.rs`
  - `"elicitation": {},   // Client can provide user input`
- `rust-mcp-sdk-main/crates/mcp-tester/src/tester.rs`
  - `instead of the correct ones (sampling, elicitation, roots). \`
- `rust-mcp-sdk-main/tests/state_machine_properties.rs`
  - `// Note: Client capabilities are what the CLIENT supports (sampling, elicitation, roots)`
  - *... and 25 more matches*


**Repository:** [rosch100/mcp-encrypted-sqlite](https://github.com/rosch100/mcp-encrypted-sqlite)
**Server names:** io.github.rosch100/mcp-encrypted-sqlite
**Sources:** registry
**Classification:** `test_only`
**Evidence** (3 matches):
- `mcp-encrypted-sqlite-main/src/main/java/com/example/mcp/sqlite/McpServer.java`
  - `JsonObject elicitationCap = new JsonObject();`
- `mcp-encrypted-sqlite-main/src/main/java/com/example/mcp/sqlite/McpServer.java`
  - `elicitationCap.addProperty("listChanged", false);`
- `mcp-encrypted-sqlite-main/src/main/java/com/example/mcp/sqlite/McpServer.java`
  - `capabilities.add("elicitation", elicitationCap);`


**Repository:** [rosch100/mcp-sqlite](https://github.com/rosch100/mcp-sqlite)
**Server names:** io.github.rosch100/mcp-sqlite
**Sources:** registry
**Classification:** `test_only`
**Evidence** (3 matches):
- `mcp-encrypted-sqlite-main/src/main/java/com/example/mcp/sqlite/McpServer.java`
  - `JsonObject elicitationCap = new JsonObject();`
- `mcp-encrypted-sqlite-main/src/main/java/com/example/mcp/sqlite/McpServer.java`
  - `elicitationCap.addProperty("listChanged", false);`
- `mcp-encrypted-sqlite-main/src/main/java/com/example/mcp/sqlite/McpServer.java`
  - `capabilities.add("elicitation", elicitationCap);`


**Repository:** [signadot/cli](https://github.com/signadot/cli)
**Server names:** io.github.signadot/cli
**Sources:** registry
**Classification:** `reference`
**Evidence** (12 matches):
- `cli-main/internal/mcp/remote/remote.go`
  - `func (r *Remote) proxyElicitation(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {`
- `cli-main/internal/mcp/remote/remote.go`
  - `// If the local client supports elicitation and elicitation is not`
- `cli-main/internal/mcp/remote/remote.go`
  - `// disabled, set up a handler to proxy elicitation requests`
- `cli-main/internal/mcp/remote/remote.go`
  - `r.log.Debug("elicitation handler configured for remote client")`
- `cli-main/internal/mcp/remote/remote.go`
  - `r.log.Debug("elicitation disabled via --disable-elicitation flag")`
  - *... and 7 more matches*

