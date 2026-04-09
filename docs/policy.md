# Policy Layers

Egressor checks every outbound request through four policy layers, in order. Each layer has a different purpose and a different level of user interaction.

If any layer blocks the request, it stops there -- later layers aren't checked.

## Layer 1: Directory scope (`allowed_directories`)

**Type:** Hard block, no prompt

This is the broadest control. It restricts which parts of the filesystem LLM tools can access. If you set this to your project directory, any file reference outside that scope is blocked immediately.

This catches tools that try to read files they shouldn't -- like `~/.ssh/id_rsa`, `/etc/passwd`, `~/.aws/credentials`, or files from other projects on your machine.

```yaml
allowed_directories:
  - "~/Projects/my-app"
```

How it works:
- File paths detected in the request body are resolved to absolute paths
- Relative paths (like `../`) are cleaned before comparison
- Each path must fall within at least one allowed directory
- If no directories are configured, this layer is skipped (everything passes)

## Layer 2: File patterns (`deny_file_patterns`)

**Type:** Hard block, no prompt

Catches sensitive files by name, even if they're inside your allowed directories. Your project probably has an `.env` file -- you still don't want it sent to an LLM.

```yaml
deny_file_patterns:
  - "*.env"              # environment files (.env, .env.local)
  - "*.pem"              # certificates
  - "*.key"              # private keys
  - "**/secrets/**"      # anything under a secrets/ directory
  - "**/credentials*"    # credential files
  - ".aws/*"             # AWS config
```

How it works:
- Uses Go's `filepath.Match` for glob matching
- The `**/` prefix matches at any directory depth
- Also tries matching against just the filename (basename)
- Case-insensitive

## Layer 3: Content tags (`deny_content_tags`)

**Type:** Hard block, no prompt

This gives developers a way to opt individual files out of LLM processing. Add a tag as a comment at the top of any file:

```go
// NO_LLM
package internal
```

```python
# NO_LLM
class TradeSecret:
    ...
```

```yaml
# NO_LLM
api_keys:
  production: sk-...
```

When Egressor sees this tag in the request body, it blocks immediately. No prompt, no whitelist bypass. The developer explicitly marked the file.

```yaml
deny_content_tags:
  - "NO_LLM"
```

How it works:
- Case-insensitive substring search on the full request body
- The tag can be anywhere in the body, not just at the top (since the file content is embedded in a JSON payload, Egressor can't know where "the top" is)
- You can define multiple tags if your team uses different conventions

## Layer 4: Content keywords (`deny_content_keywords`)

**Type:** Interactive -- user is prompted

For content that might be sensitive but needs human judgment. Unlike the layers above, this one pauses the request and asks the user what to do.

```yaml
deny_content_keywords:
  - "CONFIDENTIAL"
  - "INTERNAL ONLY"
```

When a keyword is detected, Egressor shows a modal in the desktop UI with four options:

- **Allow Once** -- forward this request, don't remember the decision
- **Allow Always** -- forward and add the file to `content_whitelist` so it won't be asked again
- **Block Once** -- return 403, don't remember
- **Block Always** -- return 403 and add the file to `content_blacklist` so it's auto-blocked next time

The whitelist and blacklist are checked before prompting. If a file has already been approved or blocked permanently, the user isn't bothered again.

```yaml
# These are populated automatically by the UI.
# You can also edit them by hand.
content_whitelist: []
content_blacklist: []
```

In headless mode (no desktop UI), keyword matches are blocked by default since there's no way to prompt.

The prompt has a 30-second timeout. If the user doesn't respond, the request is blocked.

How it works:
- Case-insensitive substring search on the request body
- Files are partitioned into whitelist (auto-allow), blacklist (auto-block), and needs-prompt
- The interceptor goroutine blocks on a channel while waiting for the user's response
- The UI sends the decision back via a Wails-bound method

## Content type filtering

Before any of the content-based checks (layers 3 and 4) run, Egressor checks the request's `Content-Type` header. Binary and encoded content is skipped entirely -- Egressor has no OCR or binary decoding, so scanning these would only produce false positives.

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

Supports wildcards -- `image/*` matches `image/png`, `image/jpeg`, etc.

## Bypass

All four layers respect the policy bypass toggle. When bypassed (via the UI's "Pause Policy" button or the system tray), all checks are skipped and traffic flows through unmodified. This is useful for debugging or when you temporarily need unrestricted access.

Bypass state is not persisted -- it resets when Egressor restarts.