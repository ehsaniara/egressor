# Egressor -- Design Overview

Egressor is a local HTTPS proxy that sits between developer tools (Claude Code, Kiro, Cursor) and LLM APIs. It intercepts every outbound request, inspects the payload for file references and sensitive content, enforces configurable blocking rules, and logs everything.

The goal is simple: give developers visibility and control over what their AI tools send to external servers.

For details on specific topics, see:

- [Policy layers](policy.md) -- how blocking decisions are made
- [Data flow](data-flow.md) -- step-by-step request flows for each scenario
- [Components](components.md) -- how each part of the system works

## Architecture

```
┌─────────────────┐
│  Developer Tool  │  (Claude Code, Kiro, Cursor, etc.)
│  HTTPS_PROXY set │
└────────┬────────┘
         │ CONNECT host:port HTTP/1.1
         v
┌───────────────────────────────────────────────┐
│                   Egressor                     │
│                                                │
│  ┌──────────┐                                  │
│  │  Proxy   │  Accept CONNECT, dial upstream   │
│  │ Listener │                                  │
│  └────┬─────┘                                  │
│       │                                        │
│       v                                        │
│  ┌──────────────┐                              │
│  │    TLS       │  MITM with dynamic certs     │
│  │ Interceptor  │  HTTP/1.1 relay loop         │
│  └────┬─────────┘                              │
│       │                                        │
│       ├──> Skip binary content types           │
│       ├──> Extract file references             │
│       ├──> Check allowed_directories           │
│       │     └─ OUT OF SCOPE -> 403             │
│       ├──> Check deny_file_patterns            │
│       │     └─ PATTERN MATCH -> 403            │
│       ├──> Check deny_content_tags             │
│       │     └─ TAG FOUND (e.g. NO_LLM) -> 403 │
│       ├──> Check deny_content_keywords         │
│       │     ├─ WHITELIST -> auto-allow         │
│       │     ├─ BLACKLIST -> auto-block 403     │
│       │     └─ PROMPT USER -> allow/block      │
│       │                                        │
│       └──> ALLOWED -> forward upstream         │
│       │                                        │
│       v                                        │
│  ┌──────────┐  ┌───────────────┐               │
│  │  Audit   │  │  Session      │               │
│  │  Logger  │  │  Store (ring) │──> Wails UI   │
│  └──────────┘  └───────────────┘               │
│                                                │
│  ┌──────────┐  ┌──────────┐                    │
│  │ Desktop  │  │  System  │                    │
│  │   UI     │  │   Tray   │  (macOS menu bar)  │
│  └──────────┘  └──────────┘                    │
└───────────────────────────────────────────────┘
         │
         v
┌─────────────────┐
│ Remote Endpoint  │  (api.anthropic.com, etc.)
└─────────────────┘
```

## How it works, briefly

1. Developer configures their tool to use `HTTPS_PROXY=http://127.0.0.1:8080` and trusts the Egressor CA
2. The tool sends a `CONNECT` request for the LLM API host
3. Egressor dials the real server, then performs a TLS man-in-the-middle using a dynamically generated certificate
4. With both sides decrypted, Egressor reads the HTTP request, buffers the body, and runs it through four policy layers
5. If any layer blocks, the request never leaves the machine -- the client gets a 403
6. If everything passes, the request is forwarded normally and the response is relayed back
7. Every session is logged to disk and (in UI mode) pushed to the frontend in real-time

## Security considerations

- **CA key**: stored with `0600` permissions in `~/.egressor/`
- **Network scope**: binds to `127.0.0.1` only -- not accessible from other machines
- **Intercepted content**: full HTTP bodies can be logged -- treat audit logs as sensitive
- **CA trust**: must be explicitly added to the OS keychain by the user
- **No remote access**: there's no API server or remote management interface
- **Binary content**: Egressor has no OCR or binary decoding -- non-text content types are skipped entirely

## Project structure

```
cmd/egressor/main.go                  Entry point, config resolution, mode selection
internal/
  proxy/
    proxy.go                          TCP listener, CONNECT handler, lifecycle
    intercept.go                      TLS MITM, HTTP relay, policy enforcement
  policy/
    policy.go                         Scope, patterns, tags, keywords engine
    prompt.go                         PromptResolver interface, HeadlessResolver
  audit/
    session.go                        Session, InterceptedExchange, FileRef models
    logger.go                         JSON file logger with rotation
    store.go                          In-memory ring buffer for UI
    observer.go                       SessionSink interface, MultiSink
  ca/
    ca.go                             CA generation and loading
    cert.go                           Leaf certificate LRU cache
  extract/
    files.go                          File reference extraction from payloads
  config/
    config.go                         YAML config loader with defaults
  tray/
    tray.go                           macOS system tray (menu bar icon)
    tray_stub.go                      No-op for non-macOS platforms
  ui/
    app.go                            Wails-bound app (sessions, policy, prompts)
    ui.go                             Wails window runner + embedded assets
    frontend/                         React + TypeScript + Tailwind CSS
config.yaml                           Default configuration (fully commented)
docs/
  design.md                           This file -- architecture overview
  policy.md                           Policy layers and blocking rules
  data-flow.md                        Step-by-step request flow examples
  components.md                       Component details
```