# Egressor — Application Design

## Architecture Overview

Egressor is a local HTTPS intercepting proxy that monitors and controls outbound traffic from developer tools. Every HTTPS connection is TLS-terminated, inspected for file references, checked against deny policies, and logged.

```
┌─────────────────┐
│  Developer Tool  │  (Claude Code, Kiro, Cursor, etc.)
│  HTTPS_PROXY set │
└────────┬────────┘
         │ CONNECT host:port HTTP/1.1
         ▼
┌─────────────────────────────────────────────┐
│                 Egressor                     │
│                                              │
│  ┌──────────┐                                │
│  │  Proxy   │  Accept CONNECT, dial upstream │
│  │ Listener │                                │
│  └────┬─────┘                                │
│       │                                      │
│       ▼                                      │
│  ┌──────────────┐                            │
│  │    TLS       │  MITM: dynamic certs       │
│  │ Interceptor  │  HTTP/1.1 relay loop       │
│  └────┬─────────┘                            │
│       │                                      │
│       ├──▶ Extract file references           │
│       ├──▶ Check allowed_directories         │
│       │     └─ OUT OF SCOPE → 403            │
│       ├──▶ Check deny_file_patterns          │
│       │     ├─ BLOCKED → 403 to client       │
│       │     └─ ALLOWED → forward upstream    │
│       │                                      │
│       ▼                                      │
│  ┌──────────┐  ┌───────────────┐             │
│  │  Audit   │  │  Session      │             │
│  │  Logger  │  │  Store (ring) │──▶ Wails UI │
│  └──────────┘  └───────────────┘             │
│                                              │
│  ┌──────────┐                                │
│  │ Desktop  │  Wails + React                 │
│  │   UI     │  Sessions / Policy / Controls  │
│  └──────────┘                                │
└─────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Remote Endpoint  │  (api.anthropic.com, etc.)
└─────────────────┘
```

## Components

### Proxy Server (`internal/proxy/proxy.go`)

- Binds to `127.0.0.1:8080` (configurable)
- Accepts HTTP CONNECT requests
- Dials upstream TCP connection (5s timeout)
- Passes both connections to the TLS Interceptor
- Supports `Start()` / `Stop()` / `IsRunning()` for UI-driven lifecycle

### TLS Interceptor (`internal/proxy/intercept.go`)

All connections are intercepted — there is no pass-through tunnel mode.

For each connection:
1. TLS-terminate the client side with a dynamic certificate (from cert cache)
2. TLS-connect to the real upstream server
3. HTTP/1.1 relay loop:
   - Read full request body into buffer
   - Extract file references from the body
   - Evaluate file paths against `allowed_directories` — block if out of scope
   - Evaluate file paths against `deny_file_patterns` — block if matched
   - If blocked: send 403 back to client, log, stop
   - If allowed: forward request to upstream, relay response back
4. Record exchange in session

Key design: the body is fully buffered before forwarding to enable file detection and policy enforcement before the request reaches the LLM.

### File Extraction (`internal/extract/files.go`)

Scans intercepted request bodies for file references. Handles:

- **JSON fields**: walks parsed JSON looking for keys like `path`, `file_path`, `filename`, `source`, `uri` and string values that look like file paths
- **Text patterns**: regex matching for markdown code fences (`` ```lang:path ``), XML tags (`<file path="...">`), and text references (`File: path`, `from path/to/file`)
- Deduplicates results, filters out URLs, validates file extensions

Returns `[]FileRef{Path, Source}` where Source is `"json_field"` or `"text_pattern"`.

### Policy Engine (`internal/policy/policy.go`)

Two-layer policy enforcement:

**Directory scope** — `EvaluateScope(paths []string) Decision`:
- Checks if file paths fall within `allowed_directories`
- Resolves relative paths against cwd, cleans `../` traversals
- If no directories configured, all paths are allowed (default)
- Runtime mutation: `GetAllowedDirectories()`, `SetAllowedDirectories()`

**File pattern deny** — `EvaluateFiles(paths []string) Decision`:
- Checks paths against `deny_file_patterns`
- Pattern matching: `filepath.Match` for globs, `**/` prefix for recursive matching, basename fallback
- Runtime mutation: `GetDenyPatterns()`, `SetDenyPatterns()`, `AddDenyPattern()`, `RemoveDenyPattern()`

Both layers:
- Pause/bypass via atomic bool (for UI toggle)
- Thread-safe with `sync.RWMutex`

### Certificate Authority (`internal/ca/`)

**`ca.go`** — Self-signed ECDSA P-256 root CA:
- `LoadOrGenerate()`: loads from `~/.egressor/ca.pem` or auto-generates
- 10-year validity, `KeyUsageCertSign`
- Key stored with `0600` permissions

**`cert.go`** — Dynamic leaf certificate cache:
- `GetCertificate(hello)`: implements `tls.Config.GetCertificate`
- LRU cache (1024 entries), 24-hour cert validity
- Generates per-hostname leaf certs signed by the CA
- Supports both DNS names and IP SANs

### Audit Logger (`internal/audit/logger.go`)

- Newline-delimited JSON to file (`~/.egressor/logs/audit.log`)
- Size-based rotation: when file exceeds `max_size_mb`, renames to `audit.log.<unix_epoch>`
- Rotated files accumulate indefinitely
- Mutex-protected for concurrent writes

### Session Store (`internal/audit/store.go`)

- In-memory ring buffer (1000 sessions) for the desktop UI
- `OnSession(fn)` observer callback — pushes new sessions to Wails frontend via events
- `Recent(limit)`, `GetByID(id)`, `Stats()` for UI queries
- Thread-safe with `sync.RWMutex`

### Session Sink (`internal/audit/observer.go`)

- `SessionSink` interface: `Log(*Session)`
- `MultiSink` fans out to both Logger (file) and SessionStore (UI)
- The proxy server accepts any `SessionSink`, keeping it decoupled from specific consumers

### Desktop UI (`internal/ui/`)

**Go layer:**
- `app.go` — Wails-bound struct, all public methods callable from frontend
- `ui.go` — Wails window configuration and runner
- Frontend assets embedded via `//go:embed all:frontend/dist`

**React frontend** (`internal/ui/frontend/`):
- Sessions tab: live table with real-time updates via `EventsOn("session:new")`
- Detail panel: request/response inspector with JSON viewer, detected files, blocked indicator
- Policy tab: allowed directories and deny patterns with save-to-config
- Bottom bar: proxy controls, policy pause/resume, stats

### Configuration (`internal/config/config.go`)

- YAML format with sensible defaults
- `~` expansion for paths
- `Save()` for persisting UI changes back to file
- Config resolution: `--config` flag → `./config.yaml` → `~/.egressor/config.yaml`

## Data Flow

### Allowed request

```
1. Client → CONNECT api.anthropic.com:443
2. Proxy: dial TCP to api.anthropic.com:443
3. Proxy → Client: 200 Connection Established
4. Interceptor: TLS handshake with client (dynamic cert)
5. Interceptor: TLS handshake with upstream (real cert)
6. Interceptor: read HTTP request, buffer body
7. Extract: scan body → detected_files: ["src/main.go"]
8. Policy: EvaluateScope(["src/main.go"]) → in scope
9. Policy: EvaluateFiles(["src/main.go"]) → allowed
10. Interceptor: forward request to upstream
11. Interceptor: read response, forward to client
12. Logger: write session JSON to audit.log
13. Store: add session, emit "session:new" event → UI
```

### Blocked request

```
1-6. Same as above
7. Extract: scan body → detected_files: [".env"]
8. Policy: EvaluateScope([".env"]) → in scope (or blocked if outside allowed dirs)
9. Policy: EvaluateFiles([".env"]) → denied (matches "*.env")
10. Interceptor: send 403 back to client over TLS
11. Logger: write session with blocked=true, block_reason
12. Store: add session → UI shows red row
```

## Security Considerations

- **CA key**: `0600` permissions, stored in `~/.egressor/`
- **Network scope**: binds to `127.0.0.1` only — not remotely accessible
- **Intercepted content**: full HTTP bodies logged — treat audit logs as sensitive
- **CA trust**: must be explicitly added to OS keychain by the user
- **Node.js tools**: require `NODE_EXTRA_CA_CERTS` pointing to the CA cert

## Project Structure

```
cmd/egressor/main.go                  Entry point, config resolution, mode selection
internal/
  proxy/
    proxy.go                          TCP listener, CONNECT handler, lifecycle
    intercept.go                      TLS MITM, HTTP relay, file extraction, blocking
  policy/
    policy.go                         Directory scope + deny pattern engine
  audit/
    session.go                        Session, InterceptedExchange, FileRef models
    logger.go                         JSON file logger with rotation
    store.go                          In-memory ring buffer for UI
    observer.go                       SessionSink interface, MultiSink
    auditfakes/                       Counterfeiter-generated test fakes
  ca/
    ca.go                             CA generation and loading
    cert.go                           Leaf certificate LRU cache
  extract/
    files.go                          File reference extraction from payloads
  config/
    config.go                         YAML config loader with defaults
  ui/
    app.go                            Wails-bound application struct
    ui.go                             Wails window runner + embedded assets
    frontend/                         React + TypeScript + Tailwind CSS
      src/
        App.tsx                       Two-tab layout (Sessions / Policy)
        components/
          SessionTable.tsx            Live session list
          SessionDetail.tsx           Exchange inspector
          RequestPane.tsx             Request headers + body + files
          ResponsePane.tsx            Response headers + body
          PolicyEditor.tsx            Allowed dirs + deny pattern CRUD
          ProxyControls.tsx           Start/stop/pause + stats
          JsonViewer.tsx              Formatted JSON display
        hooks/
          useSessions.ts              Real-time session state + Wails events
          usePolicy.ts                Policy management
config.yaml                           Default configuration
.goreleaser-macos.yaml                macOS build (Wails UI, amd64 + arm64)
.goreleaser-linux.yaml                Linux build (headless, amd64 + arm64)
.goreleaser-windows.yaml              Windows build (headless, amd64 + arm64)
.github/workflows/release.yml        CI: tag → test → build → GitHub Release
```