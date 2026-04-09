# Data Flow

Step-by-step examples of how requests flow through Egressor in different scenarios.

## Allowed request

Everything checks out -- the request is forwarded to the LLM API.

```
1.  Client sends CONNECT api.anthropic.com:443
2.  Proxy dials TCP to api.anthropic.com:443
3.  Proxy responds 200 Connection Established
4.  Interceptor does TLS handshake with client (using a dynamic cert)
5.  Interceptor does TLS handshake with upstream (using the real cert)
6.  Interceptor reads the HTTP request and buffers the body
7.  Content-Type is application/json -- not skipped
8.  File extraction finds ["src/main.go"] in the JSON payload
9.  EvaluateScope: src/main.go is inside ~/Projects/my-app -- passes
10. EvaluateFiles: no deny pattern matches src/main.go -- passes
11. EvaluateContentTags: body doesn't contain NO_LLM -- passes
12. EvaluateContentKeywords: body doesn't contain CONFIDENTIAL -- passes
13. Request forwarded to api.anthropic.com
14. Response received, forwarded back to client
15. Session logged to audit.log as JSON
16. Session pushed to UI via "session:new" event
```

## Blocked by directory scope

A tool tries to read a file outside the allowed project directory.

```
1-8. Same as above, but file extraction finds ["/etc/passwd"]
9.   EvaluateScope: /etc/passwd is outside ~/Projects/my-app
10.  Interceptor sends 403 to client: "file is outside allowed directories"
11.  Session logged with blocked=true
12.  UI shows the session in red
```

## Blocked by file pattern

A request references a `.env` file that matches a deny pattern.

```
1-8. Same as above, but file extraction finds [".env"]
9.   EvaluateScope: .env is inside the allowed directory -- passes
10.  EvaluateFiles: ".env" matches pattern "*.env"
11.  Interceptor sends 403 to client
12.  Session logged with blocked=true, reason: matches "*.env"
```

## Blocked by content tag

A developer added `// NO_LLM` to a file, and a tool tried to send it.

```
1-8.  Same as above, file extraction finds ["internal/secrets.go"]
9.    EvaluateScope: file is in scope -- passes
10.   EvaluateFiles: no pattern match -- passes
11.   EvaluateContentTags: body contains "NO_LLM"
12.   Interceptor sends 403 to client
13.   Session logged with blocked=true
```

The developer said no, and Egressor enforced it. No prompt, no override.

## Interactive keyword prompt

A request body contains the word "CONFIDENTIAL" and the files haven't been whitelisted or blacklisted.

```
1-8.  Same as above, file extraction finds ["report.md"]
9-11. Scope, patterns, tags all pass
12.   EvaluateContentKeywords: body contains "CONFIDENTIAL"
      - report.md is not in whitelist or blacklist -> needs prompt
13.   Interceptor calls resolver.PromptUser() and blocks
14.   App emits "content:prompt" event to the React frontend
15.   Frontend shows a modal:
      "Request contains keyword CONFIDENTIAL"
      Files: report.md
      [Allow Once] [Allow Always] [Block Once] [Block Always]
16.   User clicks "Block Always"
17.   Frontend calls ResolveContentPrompt() to unblock the interceptor
      Frontend calls ResolveContentPromptForFile() to add report.md to blacklist
18.   App adds report.md to blacklist, sends decision to channel
19.   Interceptor receives the block decision, sends 403 to client
20.   Session logged with blocked=true
```

Next time a request references `report.md` and contains a keyword, it's auto-blocked without prompting.

## Keyword prompt with whitelist hit

Same as above, but the file was previously approved.

```
1-8.  Same as above, file extraction finds ["report.md"]
9-11. Scope, patterns, tags all pass
12.   EvaluateContentKeywords: body contains "CONFIDENTIAL"
      - report.md is in whitelist -> auto-allowed
      - No files need prompting
13.   Request forwarded to upstream normally
```

No prompt, no delay. The user already said this file is fine.

## Binary content skipped

A request contains an image payload that shouldn't be scanned.

```
1-6.  Same as above
7.    Content-Type is image/png -- matches skip_content_types
8.    File extraction and all policy checks are skipped
9.    Request forwarded to upstream directly
10.   Session logged (body not scanned)
```