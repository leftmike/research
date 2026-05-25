# Claude Tools Reference

**Total: 34 tools** across four categories — 8 core tools always loaded, 20 deferred tools loaded on demand via `ToolSearch`, and 6 MCP tools for Google services (Gmail, Calendar, Drive).

---

## Core Tools (always available)

### 1. `Agent`
**Description:** Launch a new agent to handle complex, multi-step tasks. Each agent type has specific capabilities and tools available to it.

Available agent types:
- `claude-code-guide` — questions about Claude Code CLI, Agent SDK, Claude API
- `Explore` — fast codebase exploration by file patterns, keywords, or architecture questions
- `general-purpose` — researching complex questions, searching for code, multi-step tasks
- `Plan` — designing implementation plans
- `statusline-setup` — configure Claude Code status line setting

**Parameters:**
- `description` (string, required) — short 3-5 word description of the task
- `prompt` (string, required) — the task for the agent to perform
- `subagent_type` (string, optional) — one of the agent types listed above
- `isolation` (string, optional) — `"worktree"` to work on an isolated copy of the repo
- `model` (string, optional) — `"sonnet"`, `"opus"`, or `"haiku"`
- `run_in_background` (boolean, optional) — run agent in background

---

### 2. `Bash`
**Description:** Executes a given bash command and returns its output. The working directory persists between commands, but shell state does not. Shell environment is initialized from the user's profile.

**Parameters:**
- `command` (string, required) — the command to execute
- `description` (string, required) — clear, concise description of what the command does
- `run_in_background` (boolean, optional) — run in background
- `timeout` (integer, optional) — timeout in milliseconds (max 600000)
- `dangerouslyDisableSandbox` (boolean, optional) — override sandbox mode

---

### 3. `Edit`
**Description:** Performs exact string replacements in files. You must use `Read` at least once before editing. The edit will FAIL if `old_string` is not unique in the file.

**Parameters:**
- `file_path` (string, required) — absolute path to the file to modify
- `old_string` (string, required) — the text to replace
- `new_string` (string, required) — the text to replace it with (must differ from old_string)
- `replace_all` (boolean, optional, default false) — replace all occurrences

---

### 4. `Read`
**Description:** Reads a file from the local filesystem. Can read text files, images (PNG, JPG, etc.), PDFs (up to 20 pages per request), and Jupyter notebooks (.ipynb). Results use cat -n format with line numbers starting at 1.

**Parameters:**
- `file_path` (string, required) — absolute path to the file to read
- `limit` (integer, optional) — number of lines to read
- `offset` (integer, optional) — line number to start reading from
- `pages` (string, optional) — page range for PDFs (e.g. `"1-5"`)

---

### 5. `Write`
**Description:** Writes a file to the local filesystem. Overwrites existing files. You MUST use `Read` first if the file already exists. Prefer `Edit` for modifying existing files.

**Parameters:**
- `file_path` (string, required) — absolute path to the file to write
- `content` (string, required) — the content to write to the file

---

### 6. `Skill`
**Description:** Execute a skill within the main conversation. Skills provide specialized capabilities. When users reference a "slash command" or `/<something>`, they are referring to a skill.

Available skills: `update-config`, `keybindings-help`, `simplify`, `fewer-permission-prompts`, `loop`, `schedule`, `claude-api`, `init`, `review`, `security-review`

**Parameters:**
- `skill` (string, required) — exact name of the skill (no leading slash)
- `args` (string, optional) — optional arguments for the skill

---

### 7. `ScheduleWakeup`
**Description:** Schedule when to resume work in `/loop` dynamic mode. Pass the same `/loop` prompt back via `prompt` each turn so the next firing repeats the task. For autonomous loops pass `<<autonomous-loop-dynamic>>`. Delay is clamped to [60, 3600] seconds.

**Parameters:**
- `delaySeconds` (number, required) — seconds from now to wake up (clamped to [60, 3600])
- `reason` (string, required) — one short sentence explaining the chosen delay
- `prompt` (string, required) — the /loop input to fire on wake-up

---

### 8. `ToolSearch`
**Description:** Fetches full schema definitions for deferred tools so they can be called. Takes a query, matches against the deferred tool list, and returns matched tools' complete JSONSchema definitions. Query forms: `"select:Read,Edit,Grep"` (exact names) or keyword search.

**Parameters:**
- `query` (string, required) — query to find deferred tools
- `max_results` (integer, optional, default 5) — maximum number of results to return

---

## Deferred Tools (require ToolSearch to load schema before use)

### 9. `AskUserQuestion`
**Description:** Use when you need to ask the user questions during execution — to gather preferences, clarify ambiguous instructions, get decisions on implementation choices, or offer choices about direction. Users can always select "Other" for custom text input. In plan mode, use to clarify requirements BEFORE finalizing the plan; do NOT use to ask "Is my plan ready?" — use `ExitPlanMode` for that.

**Parameters:**
- `questions` (array, required, 1–4 items) — questions to ask the user, each with:
  - `question` (string, required) — the question text
  - `header` (string, required) — short chip/tag label (max 12 chars)
  - `options` (array, required, 2–4 items) — choices, each with:
    - `label` (string, required) — display text (1–5 words)
    - `description` (string, required) — explanation of this option
    - `preview` (string, optional) — markdown preview content for visual comparison
  - `multiSelect` (boolean, default false) — allow multiple selections
- `answers` (object, optional) — user answers collected by permission component
- `annotations` (object, optional) — per-question annotations from the user
- `metadata` (object, optional) — metadata for tracking/analytics (not displayed)

---

### 10. `CronCreate`
**Description:** Schedule a prompt to be enqueued at a future time. Works for both recurring schedules and one-shot reminders. Uses standard 5-field cron in the user's local timezone. Recurring tasks auto-expire after 7 days. Jobs live only in this Claude session.

**Parameters:**
- `cron` (string, required) — standard 5-field cron expression in local time (e.g. `"*/5 * * * *"`)
- `prompt` (string, required) — the prompt to enqueue at each fire time
- `recurring` (boolean, optional, default true) — true = fire on every match; false = fire once then auto-delete
- `durable` (boolean, optional, default false) — persist to `.claude/scheduled_tasks.json` and survive restarts

---

### 11. `CronDelete`
**Description:** Cancel a cron job previously scheduled with `CronCreate`. Removes it from the in-memory session store.

**Parameters:**
- `id` (string, required) — job ID returned by `CronCreate`

---

### 12. `CronList`
**Description:** List all cron jobs scheduled via `CronCreate` in this session.

**Parameters:** *(none)*

---

### 13. `EnterPlanMode`
**Description:** Use proactively when about to start a non-trivial implementation task. Transitions into plan mode where you can explore the codebase and design an implementation approach for user approval. Use for: new features, multiple valid approaches, code modifications, architectural decisions, multi-file changes, unclear requirements, or when user preferences matter.

**Parameters:** *(none)*

---

### 14. `ExitPlanMode`
**Description:** Use when in plan mode, after finishing writing the plan to the plan file, and ready for user approval. Signals done planning and ready for user review. Only use when the task requires planning implementation steps that require writing code.

**Parameters:**
- `allowedPrompts` (array, optional) — prompt-based permissions needed to implement the plan, each with:
  - `tool` (string, required) — must be `"Bash"`
  - `prompt` (string, required) — semantic description of the action (e.g. `"run tests"`)

---

### 15. `EnterWorktree`
**Description:** Use ONLY when explicitly instructed to work in a worktree. Creates an isolated git worktree inside `.claude/worktrees/` with a new branch based on HEAD. Use `ExitWorktree` to leave. Pass `path` to switch into an already-existing worktree.

**Parameters:**
- `name` (string, optional) — name for a new worktree (mutually exclusive with `path`)
- `path` (string, optional) — path to an existing worktree to enter (mutually exclusive with `name`)

---

### 16. `ExitWorktree`
**Description:** Exit a worktree session created by `EnterWorktree` and return to the original working directory. Only operates on worktrees created by `EnterWorktree` in this session. Is a no-op if called outside an `EnterWorktree` session.

**Parameters:**
- `action` (string, required) — `"keep"` (leave worktree on disk) or `"remove"` (delete worktree and branch)
- `discard_changes` (boolean, optional, default false) — required `true` when `action` is `"remove"` and there are uncommitted changes

---

### 17. `Monitor`
**Description:** Start a background monitor that streams events from a long-running script. Each stdout line is an event. Use for: one notification per occurrence indefinitely (unbounded command like `tail -f`), or one per occurrence until a known end (command that exits). For a single "tell me when done" notification, use `Bash` with `run_in_background` instead.

**Parameters:**
- `description` (string, required) — short human-readable description of what you are monitoring
- `timeout_ms` (integer, required, default 300000, max 3600000) — kill the monitor after this deadline
- `persistent` (boolean, required, default false) — run for the lifetime of the session (no timeout); stop with `TaskStop`
- `command` (string, required) — shell command or script; each stdout line is an event

---

### 18. `NotebookEdit`
**Description:** Completely replaces the contents of a specific cell in a Jupyter notebook (.ipynb file) with new source. The `cell_number` is 0-indexed. Use `edit_mode=insert` to add a new cell; `edit_mode=delete` to delete a cell.

**Parameters:**
- `notebook_path` (string, required) — absolute path to the Jupyter notebook file
- `new_source` (string, required) — the new source for the cell
- `cell_id` (string, optional) — ID of the cell to edit (new cell inserted after this ID, or at beginning if not specified)
- `cell_type` (string, optional) — `"code"` or `"markdown"` (required when inserting)
- `edit_mode` (string, optional) — `"replace"` (default), `"insert"`, or `"delete"`

---

### 19. `PushNotification`
**Description:** Sends a desktop notification in the user's terminal. If Remote Control is connected, also pushes to their phone. Pulls attention from whatever the user is doing — use only when there's a real chance they've walked away and something is worth coming back for, or when explicitly asked. Keep message under 200 characters, one line, no markdown.

**Parameters:**
- `message` (string, required) — notification body; under 200 characters
- `status` (string, required) — must be `"proactive"`

---

### 20. `RemoteTrigger`
**Description:** Call the claude.ai remote-trigger API. The OAuth token is added automatically. Actions: `list` (GET triggers), `get` (GET specific trigger), `create` (POST new trigger), `update` (POST update trigger), `run` (POST run trigger).

**Parameters:**
- `action` (string, required) — one of `"list"`, `"get"`, `"create"`, `"update"`, `"run"`
- `trigger_id` (string, optional) — required for `get`, `update`, and `run`
- `body` (object, optional) — required for `create` and `update`; optional for `run`

---

### 21. `TaskCreate`
**Description:** Create a structured task list for the current coding session. Use for complex multi-step tasks (3+ steps), non-trivial tasks, plan mode, or when the user explicitly requests a todo list. All tasks are created with status `pending`.

**Parameters:**
- `subject` (string, required) — brief, actionable title in imperative form
- `description` (string, required) — what needs to be done
- `activeForm` (string, optional) — present continuous form shown in spinner when in_progress
- `metadata` (object, optional) — arbitrary metadata to attach to the task

---

### 22. `TaskGet`
**Description:** Retrieve a task by its ID from the task list. Returns full task details including subject, description, status, blocks, and blockedBy.

**Parameters:**
- `taskId` (string, required) — the ID of the task to retrieve

---

### 23. `TaskList`
**Description:** List all tasks in the task list. Returns a summary of each task: id, subject, status, owner, and blockedBy.

**Parameters:** *(none)*

---

### 24. `TaskOutput`
**Description:** (DEPRECATED) Retrieves output from a running or completed background task (shell, agent, or remote session). For bash tasks prefer `Read` on the output file path. For local_agent tasks use the `Agent` tool result directly.

**Parameters:**
- `task_id` (string, required) — the task ID to get output from
- `block` (boolean, required, default true) — whether to wait for completion
- `timeout` (integer, required, default 30000, max 600000) — max wait time in ms

---

### 25. `TaskStop`
**Description:** Stops a running background task by its ID.

**Parameters:**
- `task_id` (string, optional) — the ID of the background task to stop
- `shell_id` (string, optional) — deprecated; use `task_id` instead

---

### 26. `TaskUpdate`
**Description:** Update a task in the task list. Can mark tasks resolved/deleted, update details, or establish dependencies. Only mark completed when FULLY accomplished.

**Parameters:**
- `taskId` (string, required) — the ID of the task to update
- `status` (string, optional) — `"pending"`, `"in_progress"`, `"completed"`, or `"deleted"`
- `subject` (string, optional) — new task title
- `description` (string, optional) — new description
- `activeForm` (string, optional) — present continuous form shown in spinner
- `owner` (string, optional) — new task owner
- `metadata` (object, optional) — metadata keys to merge (set key to null to delete it)
- `addBlocks` (array of strings, optional) — task IDs that this task blocks
- `addBlockedBy` (array of strings, optional) — task IDs that must complete before this one

---

### 27. `WebFetch`
**Description:** Fetches content from a specified URL and processes it using an AI model. Converts HTML to markdown, then processes it with a prompt using a small, fast model. Will fail for authenticated or private URLs — use a specialized MCP tool instead for those. Includes a 15-minute self-cleaning cache. For GitHub URLs, prefer `gh` CLI via `Bash`.

**Parameters:**
- `url` (string, required) — fully-formed valid URL (HTTP auto-upgraded to HTTPS)
- `prompt` (string, required) — describes what information to extract from the page

---

### 28. `WebSearch`
**Description:** Search the web and use results to inform responses. Provides up-to-date information for current events and recent data. Returns search results with links as markdown hyperlinks. Only available in the US. After answering, MUST include a "Sources:" section listing all relevant URLs.

**Parameters:**
- `query` (string, required, min length 2) — the search query
- `allowed_domains` (array of strings, optional) — only include results from these domains
- `blocked_domains` (array of strings, optional) — never include results from these domains

---

## MCP Tools (Google integrations — require OAuth authentication)

### 29. `mcp__claude_ai_Gmail__authenticate`
**Description:** The `claude.ai Gmail` MCP server is installed but requires authentication. Call this to start the OAuth flow — you'll receive an authorization URL to share with the user. Once the user completes authorization, the server's real tools become available automatically.

**Parameters:** *(none)*

---

### 30. `mcp__claude_ai_Gmail__complete_authentication`
**Description:** Complete an in-progress OAuth flow for the `claude.ai Gmail` MCP server by submitting the callback URL. Call `mcp__claude_ai_Gmail__authenticate` first. After the user authorizes, pass the full `http://localhost:<port>/callback?code=...&state=...` URL here.

**Parameters:**
- `callback_url` (string, required) — the full callback URL from the browser address bar after authorizing

---

### 31. `mcp__claude_ai_Google_Calendar__authenticate`
**Description:** The `claude.ai Google Calendar` MCP server is installed but requires authentication. Call this to start the OAuth flow — you'll receive an authorization URL to share with the user.

**Parameters:** *(none)*

---

### 32. `mcp__claude_ai_Google_Calendar__complete_authentication`
**Description:** Complete an in-progress OAuth flow for the `claude.ai Google Calendar` MCP server by submitting the callback URL. Call `mcp__claude_ai_Google_Calendar__authenticate` first.

**Parameters:**
- `callback_url` (string, required) — the full callback URL from the browser address bar after authorizing

---

### 33. `mcp__claude_ai_Google_Drive__authenticate`
**Description:** The `claude.ai Google Drive` MCP server is installed but requires authentication. Call this to start the OAuth flow — you'll receive an authorization URL to share with the user.

**Parameters:** *(none)*

---

### 34. `mcp__claude_ai_Google_Drive__complete_authentication`
**Description:** Complete an in-progress OAuth flow for the `claude.ai Google Drive` MCP server by submitting the callback URL. Call `mcp__claude_ai_Google_Drive__authenticate` first.

**Parameters:**
- `callback_url` (string, required) — the full callback URL from the browser address bar after authorizing
