# Egressor

[![CI](https://github.com/ehsaniara/egressor/actions/workflows/ci.yml/badge.svg)](https://github.com/ehsaniara/egressor/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ehsaniara/egressor)](https://goreportcard.com/report/github.com/ehsaniara/egressor)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Local-first egress monitoring and control for developer tools**

---

## What it does

Egressor sits between your AI coding tools and the LLM APIs they talk to. It intercepts every HTTPS request, shows you exactly what's being sent — including which files — and lets you block anything that shouldn't leave your machine.

It's built for developers who use tools like Claude Code, Kiro, or Cursor and want to know (and control) what data those tools send to external APIs.

- **TLS interception** — decrypts and inspects HTTPS payloads using a local CA
- **File detection** — identifies file paths and contents in API request bodies
- **Directory scope** — restricts which directories tools can access
- **File pattern blocking** — hard-blocks sensitive files like `.env`, `.pem`, and secrets
- **Content tags** — developers mark files with `// NO_LLM` to prevent them from being sent
- **Content keywords** — flags requests containing words like "CONFIDENTIAL" and asks what to do
- **Desktop UI** — real-time session inspector with request/response viewer
- **System tray** — menu bar icon on macOS with status and quick controls
- **Audit logging** — structured JSON logs with automatic rotation

```
Developer Tool  ──HTTPS──>  Egressor  ──HTTPS──>  LLM API
                            (inspect)
                            (detect files)
                            (block if denied)
                            (log everything)
```
![img.png](docs/img.png)

---

## Quick Start

### Install

**Homebrew (macOS):**
```bash
brew tap ehsaniara/tap
brew install egressor
```

**GitHub Releases (all platforms):**

Download the latest binary from [Releases](https://github.com/ehsaniara/egressor/releases), extract, and run.

**From source (macOS):**
```bash
git clone https://github.com/ehsaniara/egressor.git
cd egressor
cd internal/ui/frontend && npm install && npm run build && cd ../../..
CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags production -o egressor ./cmd/egressor
```

**From source (Windows — requires Go 1.24+ and Node.js 22+):**

```powershell
git clone https://github.com/ehsaniara/egressor.git
cd egressor
cd internal\ui\frontend; npm install; npm run build; cd ..\..\..
go build -tags production -o egressor.exe ./cmd/egressor
```

**From source (Linux — headless only):**

```bash
git clone https://github.com/ehsaniara/egressor.git
cd egressor
go build -o egressor ./cmd/egressor
```

### First run

Egressor auto-generates a CA certificate the first time you run it and prints setup instructions:

```bash
./egressor
```

You'll need to trust the CA so Egressor can intercept TLS traffic. This is a one-time step.

**macOS:**
```bash
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain ~/.egressor/ca.pem
```

**Linux:**

```bash
sudo cp ~/.egressor/ca.pem /usr/local/share/ca-certificates/egressor.crt
sudo update-ca-certificates
```

### Configure your tools (macOS / Linux)

Tell your LLM tools to route traffic through Egressor and trust its CA:

```bash
export NODE_EXTRA_CA_CERTS=~/.egressor/ca.pem
export HTTPS_PROXY=http://127.0.0.1:8080
```

Then launch your tool. All HTTPS traffic now flows through Egressor.

### Windows step-by-step

**1. Download and extract** Egressor from [GitHub Releases](https://github.com/ehsaniara/egressor/releases), or build from source.

**2. Run Egressor once** to auto-generate the CA certificate:
```powershell
.\egressor.exe
```

**3. Trust the CA** (run PowerShell as Administrator):
```powershell
Import-Certificate -FilePath "$env:USERPROFILE\.egressor\ca.pem" `
  -CertStoreLocation Cert:\LocalMachine\Root
```

**4. Set environment variables** so your LLM tools route through Egressor:
```powershell
$env:NODE_EXTRA_CA_CERTS = "$env:USERPROFILE\.egressor\ca.pem"
$env:HTTPS_PROXY = "http://127.0.0.1:8080"
```

To make these persist across sessions:
```powershell
[Environment]::SetEnvironmentVariable("NODE_EXTRA_CA_CERTS", "$env:USERPROFILE\.egressor\ca.pem", "User")
[Environment]::SetEnvironmentVariable("HTTPS_PROXY", "http://127.0.0.1:8080", "User")
```

**5. Start Egressor:**
```powershell
.\egressor.exe
```

**6. Launch your LLM tool** (Claude Code, Kiro, Cursor, etc.). All traffic to LLM APIs now flows through Egressor.

To stop intercepting, close Egressor and remove the proxy variable:
```powershell
[Environment]::SetEnvironmentVariable("HTTPS_PROXY", $null, "User")
```

---

## Usage

```bash
egressor                              # desktop UI (default)
egressor --headless                   # terminal only, no window
egressor --config /path/to/config.yaml  # custom config file
egressor --generate-ca                # generate CA certificate and exit
egressor --version                    # print version
```

Egressor looks for its config file in this order:

1. `--config` flag (if provided)
2. `./config.yaml` (current directory)
3. `~/.egressor/config.yaml` (created automatically on first run)

---

## How blocking works

Egressor checks every outbound request through four layers, in order. Each layer serves a different purpose:

### 1. Directory scope (`allowed_directories`)

Restricts which parts of the filesystem tools can access. If you set this to your project directory, any file reference outside that directory is blocked immediately. This catches tools trying to read `~/.ssh/id_rsa`, `/etc/passwd`, or files from other projects.

```yaml
allowed_directories:
  - "~/Projects/my-app"
```

Leave empty to allow all directories (default).

### 2. File patterns (`deny_file_patterns`)

Blocks requests that reference files matching glob patterns. This catches sensitive files regardless of where they are — even inside your allowed directories.

```yaml
deny_file_patterns:
  - "*.env"              # environment files
  - "*.pem"              # certificates
  - "*.key"              # private keys
  - "**/secrets/**"      # anything under a secrets/ directory
  - "**/credentials*"    # credential files
  - ".aws/*"             # AWS config
```

| Pattern              | What it matches                       |
|----------------------|---------------------------------------|
| `*.env`              | `.env`, `config/.env`                 |
| `*.pem`              | `ca.pem`, `path/to/cert.pem`          |
| `**/secrets/**`      | `config/secrets/db.yaml`              |
| `**/credentials*`    | `home/credentials.json`               |
| `.aws/*`             | `.aws/config`, `.aws/credentials`     |

### 3. Content tags (`deny_content_tags`) — hard block

Developers can mark individual files to prevent them from being sent to LLMs by adding a tag as a comment:

```go
// NO_LLM
package internal
```

```python
# NO_LLM
class TradeSecret:
```

```yaml
# NO_LLM
api_keys:
  production: sk-...
```

When Egressor detects a tag in the request body, it blocks immediately. No prompt, no whitelist bypass. The developer said no.

```yaml
deny_content_tags:
  - "NO_LLM"
```

### 4. Content keywords (`deny_content_keywords`) — interactive

For content that might be sensitive but needs human judgment. When a keyword is found, Egressor pauses the request and shows a prompt in the desktop UI:

- **Allow Once** — send this time, don't remember
- **Allow Always** — send and add the file to `content_whitelist` (won't ask again)
- **Block Once** — reject this time, don't remember
- **Block Always** — reject and add the file to `content_blacklist` (auto-block next time)

```yaml
deny_content_keywords:
  - "CONFIDENTIAL"
  - "INTERNAL ONLY"

# Auto-managed by the UI when users choose "Allow Always" or "Block Always"
content_whitelist: []
content_blacklist: []
```

In headless mode (no UI), keyword matches are blocked by default.

### Content type filtering

Binary and encoded content is skipped during scanning to avoid false positives. You can customize which content types to skip:

```yaml
intercept:
  skip_content_types:
    - "image/*"
    - "audio/*"
    - "video/*"
    - "application/octet-stream"
    - "application/zip"
    - "application/gzip"
    - "application/pdf"
```

---

## Desktop UI

The default mode opens a native desktop window (built with Wails + React):

- **Sessions tab** — live table of intercepted connections showing method, host, status, and file count
- **Detail panel** — click a session to inspect full request/response headers, body (formatted JSON), and detected files
- **Policy tab** — manage all policy rules: allowed directories, deny patterns, content tags, content keywords, whitelist, and blacklist
- **Bottom bar** — proxy start/stop, pause/resume policy, session stats
- **System tray** (macOS) — menu bar icon with status, pause/resume, and quit

Blocked requests show up in red with the reason displayed.

---

## How it works

Egressor performs a TLS man-in-the-middle on every HTTPS connection:

```
Client ──TLS(egressor cert)──> Egressor ──TLS(real cert)──> Server
```

1. Client sends `CONNECT api.anthropic.com:443`
2. Egressor opens a TCP connection to the real server
3. Egressor presents a dynamically generated certificate to the client (signed by its local CA)
4. Egressor opens its own TLS connection to the real server
5. Between two decrypted streams, it reads the plaintext HTTP request
6. Extracts file references from the JSON payload
7. Checks `allowed_directories` — blocks if any file is out of scope
8. Checks `deny_file_patterns` — blocks if any file path matches
9. Checks `deny_content_tags` — blocks if body contains a tag like `NO_LLM`
10. Checks `deny_content_keywords` — prompts user if body contains a keyword (checks whitelist/blacklist first)
11. If blocked: returns 403, logs the attempt, never forwards to the server
12. If allowed: forwards the request, captures the response, logs everything

### File detection

Egressor scans request bodies for file references using multiple strategies:

- **JSON field keys**: `path`, `file_path`, `filePath`, `filename`, `source`, `uri`
- **JSON string values** that look like file paths (contain `/` and a file extension)
- **Markdown code fences**: `` ```go:cmd/main.go ``
- **XML-style tags**: `<file path="config.yaml">`, `<source>lib/auth.rb</source>`
- **Text patterns**: `File: src/main.py`, `from src/handler.ts`

---

## Audit Logs

Every intercepted session is logged as newline-delimited JSON to `~/.egressor/logs/audit.log`. The log file is rotated automatically when it exceeds `max_size_mb` (default 2MB).

```json
{
  "session_id": "sess_a1b2c3d4",
  "target_host": "api.anthropic.com",
  "target_port": "443",
  "dial_status": "success",
  "exchanges": [
    {
      "method": "POST",
      "url": "https://api.anthropic.com/v1/messages",
      "detected_files": [
        {"path": "src/main.go", "source": "text_pattern"},
        {"path": ".env", "source": "json_field"}
      ],
      "blocked": true,
      "block_reason": "file \".env\" matches deny pattern \"*.env\"",
      "status_code": 403
    }
  ]
}
```

---

## Project Structure

```
cmd/egressor/main.go              Entry point, config resolution, mode selection
internal/
  proxy/
    proxy.go                      TCP listener, CONNECT handler, lifecycle
    intercept.go                  TLS MITM, HTTP relay, policy enforcement
  policy/
    policy.go                     Scope, patterns, tags, keywords engine
    prompt.go                     Interactive prompt types and resolver
  audit/
    session.go                    Session and exchange data models
    logger.go                     JSON logger with size-based rotation
    store.go                      In-memory ring buffer for UI
    observer.go                   SessionSink interface, MultiSink fan-out
  ca/
    ca.go                         CA generation and loading
    cert.go                       Leaf certificate LRU cache
  extract/
    files.go                      File reference extraction from payloads
  config/
    config.go                     YAML config with defaults and ~ expansion
  tray/
    tray.go                       macOS system tray (menu bar icon)
  ui/
    app.go                        Wails-bound app (sessions, policy, prompts)
    ui.go                         Wails window runner + embedded assets
    frontend/                     React + TypeScript + Tailwind CSS
config.yaml                       Default configuration (well-commented)
```

---

## Building

### Prerequisites

- Go 1.24+
- Node.js 22+
- Xcode Command Line Tools (macOS)

### Build

```bash
# Build frontend
cd internal/ui/frontend && npm install && npm run build && cd ../../..

# Build binary (macOS with desktop UI)
CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags production -o egressor ./cmd/egressor

# Build headless only (any platform, no CGO)
go build -o egressor ./cmd/egressor
```

### Run tests

```bash
go test ./internal/...
```

---

## Release

Pushing a version tag triggers the CI pipeline:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This builds binaries for:

| OS      | Arch         | Mode       |
|---------|--------------|------------|
| macOS   | amd64, arm64 | Desktop UI |
| Windows | amd64        | Desktop UI |
| Windows | arm64        | Headless   |
| Linux   | amd64, arm64 | Headless   |

Binaries are published to [GitHub Releases](https://github.com/ehsaniara/egressor/releases) and the Homebrew tap.

---

## License

MIT