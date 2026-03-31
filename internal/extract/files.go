package extract

import (
	"encoding/json"
	"regexp"
	"strings"
)

// FileRef represents a file reference detected in an API payload.
type FileRef struct {
	Path   string `json:"path"`
	Source string `json:"source"` // how it was detected: "json_field", "text_pattern"
}

// jsonPathKeys are JSON keys that typically hold file paths in LLM API payloads.
var jsonPathKeys = map[string]bool{
	"path":      true,
	"file":      true,
	"file_path": true,
	"filepath":  true,
	"filePath":  true,
	"filename":  true,
	"fileName":  true,
	"file_name": true,
	"source":    true,
	"uri":       true,
	"url":       true,
}

// textPatterns match file references embedded in text content sent to LLMs.
var textPatterns = []*regexp.Regexp{
	// ```lang:path/to/file or ```lang filepath
	regexp.MustCompile("```[a-zA-Z]*[: ]([\\w./-]+\\.[a-zA-Z0-9]+)"),
	// <file path="..."> or <file_path>...</file_path> or <source>...</source>
	regexp.MustCompile(`<(?:file|file_path|source)[^>]*?(?:path|name)?=?"([^"]+\.[a-zA-Z0-9]+)"`),
	regexp.MustCompile(`<(?:file_path|source|file)>([^<]+\.[a-zA-Z0-9]+)</`),
	// File: path/to/file or --- path/to/file ---
	regexp.MustCompile(`(?:^|\n)(?:File|PATH|file)[: ]+([^\s]+\.[a-zA-Z0-9]+)`),
	regexp.MustCompile(`(?:^|\n)---\s+([^\s]+\.[a-zA-Z0-9]+)\s+---`),
	// "from path/to/file" or "in path/to/file"
	regexp.MustCompile(`(?:from|in) ([^\s:,"]+/[^\s:,"]+\.[a-zA-Z0-9]+)`),
}

// FilesFromBody extracts file references from an HTTP request body.
// It handles JSON payloads (typical of LLM APIs) and falls back to text pattern matching.
func FilesFromBody(body string) []FileRef {
	if len(body) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var refs []FileRef

	add := func(path, source string) {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			return
		}
		if !looksLikeFilePath(path) {
			return
		}
		seen[path] = true
		refs = append(refs, FileRef{Path: path, Source: source})
	}

	// Try JSON parsing first
	body = strings.TrimSpace(body)
	if len(body) > 0 && (body[0] == '{' || body[0] == '[') {
		var parsed any
		if json.Unmarshal([]byte(body), &parsed) == nil {
			walkJSON(parsed, "", func(key, val string) {
				if jsonPathKeys[key] {
					add(val, "json_field")
				} else if looksLikeFilePath(val) && !looksLikeURL(val) && len(val) < 512 {
					add(val, "json_field")
				}
			})
			// Also scan string values for embedded text patterns
			walkJSONStrings(parsed, func(text string) {
				for _, re := range textPatterns {
					for _, match := range re.FindAllStringSubmatch(text, -1) {
						if len(match) > 1 {
							add(match[1], "text_pattern")
						}
					}
				}
			})
		}
	} else {
		// Plain text body — scan with patterns
		for _, re := range textPatterns {
			for _, match := range re.FindAllStringSubmatch(body, -1) {
				if len(match) > 1 {
					add(match[1], "text_pattern")
				}
			}
		}
	}

	return refs
}

// walkJSON recursively walks a parsed JSON value and calls fn for each string leaf
// with its parent key name and value.
func walkJSON(v any, parentKey string, fn func(key, val string)) {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			walkJSON(child, k, fn)
		}
	case []any:
		for _, child := range val {
			walkJSON(child, parentKey, fn)
		}
	case string:
		fn(parentKey, val)
	}
}

// walkJSONStrings calls fn for every string value in a parsed JSON tree,
// intended for scanning longer text content for embedded file references.
func walkJSONStrings(v any, fn func(string)) {
	switch val := v.(type) {
	case map[string]any:
		for _, child := range val {
			walkJSONStrings(child, fn)
		}
	case []any:
		for _, child := range val {
			walkJSONStrings(child, fn)
		}
	case string:
		if len(val) > 20 { // only scan substantial text blocks
			fn(val)
		}
	}
}

// looksLikeFilePath returns true if the string looks like a file path.
func looksLikeFilePath(s string) bool {
	if len(s) < 3 || len(s) > 512 {
		return false
	}
	// Must contain a dot for extension
	dotIdx := strings.LastIndex(s, ".")
	if dotIdx < 1 {
		return false
	}
	ext := s[dotIdx:]
	if len(ext) < 2 || len(ext) > 10 {
		return false
	}
	// Must contain a path separator or look like a relative file
	if strings.ContainsAny(s, "/\\") {
		return true
	}
	// Bare filename with known code extension
	return isCodeExtension(ext)
}

var codeExtensions = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
	".java": true, ".kt": true, ".rs": true, ".c": true, ".cpp": true, ".h": true,
	".cs": true, ".rb": true, ".php": true, ".swift": true, ".m": true,
	".yaml": true, ".yml": true, ".json": true, ".toml": true, ".xml": true,
	".html": true, ".css": true, ".scss": true, ".sql": true, ".sh": true,
	".md": true, ".txt": true, ".cfg": true, ".conf": true, ".ini": true,
	".proto": true, ".graphql": true, ".tf": true, ".vue": true, ".svelte": true,
}

func isCodeExtension(ext string) bool {
	return codeExtensions[strings.ToLower(ext)]
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
