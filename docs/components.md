# Components

A closer look at how each part of Egressor works.

## Proxy Server (`internal/proxy/proxy.go`)

The front door for all traffic. Binds to `127.0.0.1:8080` (configurable) and listens for HTTP CONNECT requests. When a client connects, the proxy:

1. Parses the target host and port from the CONNECT request
2. Dials a TCP connection to the real upstream server (5-second timeout)
3. Responds `200 Connection Established` to the client
4. Hands both connections to the TLS Interceptor

The server supports `Start()`, `Stop()`, and `IsRunning()` so the desktop UI can control it. In headless mode, it runs until interrupted with Ctrl+C.

## TLS Interceptor (`internal/proxy/intercept.go`)

The core of Egressor. Every HTTPS connection is intercepted -- there's no pass-through mode.

For each connection, the interceptor:

1. **TLS-terminates the client side** using a dynamically generated certificate signed by the local CA. The client thinks it's talking to the real server.
2. **TLS-connects to the real upstream server** using the server's actual certificate. The server thinks it's talking directly to the client.
3. **Relays HTTP requests** in a loop between the two decrypted streams.

For each request in the relay loop:
- The full body is buffered into memory before forwarding (capped at `max_body_size`)
- If the `Content-Type` matches `skip_content_types`, scanning is skipped
- Otherwise, file references are extracted and the four policy layers are checked
- If blocked: a 403 response is sent back over the TLS connection and the exchange is logged
- If allowed: the request is forwarded, the response is captured and relayed back

The interceptor holds a `PromptResolver` interface. In desktop mode, this is wired to the `App` struct which emits Wails events and blocks on a Go channel. In headless mode, it's a `HeadlessResolver` that blocks everything.

Key design decision: the body is fully buffered before forwarding. This adds latency, but it's the only way to inspect the payload before it leaves the machine. For typical LLM API requests (JSON payloads under 1MB), this isn't noticeable.

## File Extraction (`internal/extract/files.go`)

Scans request bodies for file references. LLM API payloads (from tools like Claude Code and Cursor) embed file contents in JSON. Egressor needs to figure out which files are being sent, even though every tool formats its payloads differently.

The extraction uses two approaches:

**JSON field scanning:** Parses the body as JSON and walks the tree looking for:
- Keys that typically hold file paths: `path`, `file_path`, `filePath`, `filename`, `source`, `uri`
- String values that look like file paths (contain `/` or `\` and have a file extension)
- Filters out URLs (starting with `http://` or `https://`)

**Text pattern matching:** For file references embedded in longer text content (like code blocks or markdown), uses regex patterns:
- Markdown code fences: `` ```go:cmd/main.go ``
- XML-style tags: `<file path="config.yaml">`, `<source>lib/auth.rb</source>`
- Inline references: `File: src/main.py`, `from src/handler.ts`

Results are deduplicated. Each detected file gets a `source` label (`"json_field"` or `"text_pattern"`) for the audit log.

## Policy Engine (`internal/policy/policy.go`, `prompt.go`)

See [policy.md](policy.md) for the full breakdown of the four layers.

The engine is a single `Engine` struct that holds all policy state and provides thread-safe evaluation methods. All state is protected by a `sync.RWMutex` -- reads can happen concurrently, writes are exclusive.

Each policy layer has its own evaluation method that returns a `Decision` (allowed/denied with a reason). The interceptor calls them in sequence and stops at the first denial.

The engine also supports a bypass toggle (`atomic.Bool`). When bypassed, all evaluations return "allowed" without checking anything.

Runtime mutations (adding/removing patterns, directories, keywords, whitelist/blacklist entries) take effect immediately without restart. The UI calls these methods via Wails bindings.

## Certificate Authority (`internal/ca/`)

**`ca.go`** -- Generates and loads the root CA certificate.

On first run, Egressor creates a self-signed ECDSA P-256 root CA with 10-year validity. The certificate and key are written to `~/.egressor/ca.pem` and `~/.egressor/ca-key.pem`. The key is stored with `0600` permissions (owner read/write only).

On subsequent runs, the existing CA is loaded from disk.

**`cert.go`** -- Dynamic leaf certificate cache.

When a client connects to `api.anthropic.com:443`, the interceptor needs a certificate for that hostname. The `CertCache` generates one on the fly -- signed by the root CA -- and caches it in an LRU cache (1024 entries, 24-hour validity).

Implements Go's `tls.Config.GetCertificate` interface so it plugs directly into the TLS server config. Supports both DNS names and IP address SANs.

## Audit System (`internal/audit/`)

**`session.go`** -- Data models for `Session` (a single proxied connection) and `InterceptedExchange` (a request/response pair within a session). Sessions track timing, target host, dial status, and a list of exchanges. Each exchange records the method, URL, headers, body, detected files, and whether it was blocked.

**`logger.go`** -- Writes complete sessions as newline-delimited JSON to `~/.egressor/logs/audit.log`. When the file exceeds `max_size_mb`, it's rotated by renaming to `audit.log.<unix_timestamp>`. Rotated files accumulate -- there's no automatic cleanup.

**`store.go`** -- An in-memory ring buffer that holds the last 1000 sessions for the desktop UI. Supports observer callbacks: when a new session is added, all registered observers are notified. The UI uses this to push real-time updates to the frontend.

**`observer.go`** -- `SessionSink` interface (`Log(*Session)`) with a `MultiSink` implementation that fans out to multiple sinks. In practice, this means every session goes to both the file logger and the in-memory store. The proxy doesn't need to know about either -- it just calls `sink.Log(session)`.

## Desktop UI (`internal/ui/`, `internal/tray/`)

### Go layer

**`app.go`** -- The Wails-bound application struct. Every public method is callable from JavaScript. Handles:
- Session queries: `GetRecentSessions()`, `GetSession()`, `GetStats()`
- Policy management: CRUD methods for all rule types (patterns, directories, tags, keywords, whitelist, blacklist)
- Content keyword prompts: implements `PromptResolver` by emitting a Wails event and blocking on a Go channel with a 30-second timeout
- Config persistence: `SaveConfig()` writes the current policy state back to the YAML file

**`ui.go`** -- Configures and starts the Wails window. Frontend assets (the compiled React app) are embedded in the binary via `//go:embed`.

### System tray (`internal/tray/`)

**`tray.go`** (macOS only) -- Adds an icon to the macOS menu bar using `energye/systray`. The menu shows:
- Status: Running / Paused
- Pause / Resume toggle (syncs with the policy bypass toggle)
- Quit (stops the proxy and closes the app)

**`tray_stub.go`** -- No-op implementation for non-macOS platforms. The `Available()` function returns false so the app knows not to try.

### React frontend (`internal/ui/frontend/`)

Built with React 19, TypeScript, Tailwind CSS, and Vite. Communicates with the Go backend through Wails-generated bindings.

**Main views:**
- **Sessions tab** -- A live table of intercepted connections. Shows method, host, path, status, file count, and duration. Updates in real-time via the `session:new` event.
- **Detail panel** -- Click a session to see the full request and response side by side. Shows headers, body (formatted JSON), detected files, and block reasons.
- **Policy tab** -- Manage all policy rules in one place. Each rule type has its own section with add/remove controls.
- **Bottom bar** -- Proxy start/stop, policy pause/resume, and live stats (total sessions, blocked, files detected).

**Interactive prompt:**
- `ContentPromptModal.tsx` -- A full-screen modal that appears when a content keyword match needs user input. Shows the matched keyword, target URL, and affected files. Four action buttons: Allow Once, Allow Always, Block Once, Block Always. Has a 30-second countdown timer -- if the user doesn't respond, the request is blocked.
- `useContentPrompts.ts` -- React hook that queues incoming prompts and handles resolution.

## Configuration (`internal/config/config.go`)

YAML format with sensible defaults for everything. Supports `~` expansion for file paths. The `Load()` function applies defaults for missing values (like `max_body_size` and `skip_content_types`).

The `Save()` function writes the config back to disk -- used by the UI's "Save to config" button to persist policy changes.

Config resolution order:
1. `--config` flag (explicit path)
2. `./config.yaml` (current directory)
3. `~/.egressor/config.yaml` (home directory -- auto-created with defaults on first run)

See `config.yaml` in the project root for a fully commented example of every option.