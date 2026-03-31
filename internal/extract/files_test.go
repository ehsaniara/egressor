package extract

import (
	"encoding/json"
	"testing"
)

func TestFilesFromBody_AnthropicPayload(t *testing.T) {
	payload := `{
		"model": "claude-sonnet-4-20250514",
		"messages": [
			{
				"role": "user",
				"content": "Review this file:\n` + "```go:cmd/main.go" + `\npackage main\nfunc main() {}\n` + "```" + `"
			}
		]
	}`
	refs := FilesFromBody(payload)
	assertContainsPath(t, refs, "cmd/main.go")
}

func TestFilesFromBody_OpenAIPayload(t *testing.T) {
	payload := `{
		"model": "gpt-4",
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "Here is the code from src/handler.ts:\nconst x = 1;"}
				]
			}
		]
	}`
	refs := FilesFromBody(payload)
	assertContainsPath(t, refs, "src/handler.ts")
}

func TestFilesFromBody_JSONPathFields(t *testing.T) {
	payload := `{
		"file_path": "internal/config/config.go",
		"content": "package config"
	}`
	refs := FilesFromBody(payload)
	assertContainsPath(t, refs, "internal/config/config.go")
}

func TestFilesFromBody_NestedFileFields(t *testing.T) {
	payload := `{
		"files": [
			{"path": "src/index.ts", "content": "export default {}"},
			{"path": "src/utils.ts", "content": "export function foo() {}"}
		]
	}`
	refs := FilesFromBody(payload)
	assertContainsPath(t, refs, "src/index.ts")
	assertContainsPath(t, refs, "src/utils.ts")
}

func TestFilesFromBody_XMLTags(t *testing.T) {
	payload := `{
		"messages": [{"role": "user", "content": "<file path=\"config/settings.yaml\">key: value</file>"}]
	}`
	refs := FilesFromBody(payload)
	assertContainsPath(t, refs, "config/settings.yaml")
}

func TestFilesFromBody_SourceTag(t *testing.T) {
	payload := `{
		"messages": [{"role": "user", "content": "<source>lib/auth.rb</source>\ndef login; end"}]
	}`
	refs := FilesFromBody(payload)
	assertContainsPath(t, refs, "lib/auth.rb")
}

func TestFilesFromBody_PlainText(t *testing.T) {
	body := "File: src/main.py\nimport os\nprint('hello')"
	refs := FilesFromBody(body)
	assertContainsPath(t, refs, "src/main.py")
}

func TestFilesFromBody_DedupesFiles(t *testing.T) {
	payload := `{
		"files": [
			{"path": "main.go", "content": "a"},
			{"path": "main.go", "content": "b"}
		]
	}`
	refs := FilesFromBody(payload)
	count := 0
	for _, r := range refs {
		if r.Path == "main.go" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected main.go once, got %d times", count)
	}
}

func TestFilesFromBody_IgnoresURLs(t *testing.T) {
	payload := `{"url": "https://api.example.com/v1/chat"}`
	refs := FilesFromBody(payload)
	for _, r := range refs {
		if r.Path == "https://api.example.com/v1/chat" {
			t.Error("should not detect URLs as file paths")
		}
	}
}

func TestFilesFromBody_EmptyBody(t *testing.T) {
	refs := FilesFromBody("")
	if refs != nil {
		t.Errorf("expected nil, got %v", refs)
	}
}

func TestFilesFromBody_KiroStylePayload(t *testing.T) {
	// Kiro sends file contents to AWS Bedrock with file references
	payload, _ := json.Marshal(map[string]any{
		"messages": []map[string]any{
			{
				"role": "user",
				"content": "I need help with this project. Here are the relevant files:\n\n" +
					"```typescript:src/components/App.tsx\nimport React from 'react';\nexport default function App() { return <div/>; }\n```\n\n" +
					"```yaml:infrastructure/deploy.yaml\napiVersion: apps/v1\nkind: Deployment\n```",
			},
		},
	})
	refs := FilesFromBody(string(payload))
	assertContainsPath(t, refs, "src/components/App.tsx")
	assertContainsPath(t, refs, "infrastructure/deploy.yaml")
}

func assertContainsPath(t *testing.T, refs []FileRef, path string) {
	t.Helper()
	for _, r := range refs {
		if r.Path == path {
			return
		}
	}
	paths := make([]string, len(refs))
	for i, r := range refs {
		paths[i] = r.Path
	}
	t.Errorf("expected to find %q in refs, got %v", path, paths)
}
