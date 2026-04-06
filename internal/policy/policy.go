// Package policy implements directory scope and file pattern enforcement
// for intercepted requests.
package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ehsaniara/egressor/internal/config"
)

type Decision struct {
	Allowed bool
	Reason  string
}

type Engine struct {
	mu       sync.RWMutex
	cfg      config.PolicyConfig
	bypassed atomic.Bool
}

func (e *Engine) SetBypassed(b bool) {
	e.bypassed.Store(b)
}

func (e *Engine) IsBypassed() bool {
	return e.bypassed.Load()
}

func NewEngine(cfg config.PolicyConfig) *Engine {
	// Clean and resolve allowed directories at construction time
	dirs := make([]string, 0, len(cfg.AllowedDirectories))
	for _, d := range cfg.AllowedDirectories {
		cleaned := filepath.Clean(d)
		dirs = append(dirs, cleaned)
	}
	cfg.AllowedDirectories = dirs
	return &Engine{cfg: cfg}
}

// GetDenyPatterns returns the current deny file patterns.
func (e *Engine) GetDenyPatterns() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, len(e.cfg.DenyFilePatterns))
	copy(out, e.cfg.DenyFilePatterns)
	return out
}

// SetDenyPatterns replaces all deny file patterns.
func (e *Engine) SetDenyPatterns(patterns []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.DenyFilePatterns = make([]string, len(patterns))
	copy(e.cfg.DenyFilePatterns, patterns)
}

// AddDenyPattern appends a single deny pattern.
func (e *Engine) AddDenyPattern(pattern string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.DenyFilePatterns = append(e.cfg.DenyFilePatterns, pattern)
}

// RemoveDenyPattern removes a single deny pattern.
func (e *Engine) RemoveDenyPattern(pattern string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	filtered := e.cfg.DenyFilePatterns[:0]
	for _, p := range e.cfg.DenyFilePatterns {
		if p != pattern {
			filtered = append(filtered, p)
		}
	}
	e.cfg.DenyFilePatterns = filtered
}

// GetAllowedDirectories returns the current allowed directories.
func (e *Engine) GetAllowedDirectories() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, len(e.cfg.AllowedDirectories))
	copy(out, e.cfg.AllowedDirectories)
	return out
}

// SetAllowedDirectories replaces the allowed directories list.
func (e *Engine) SetAllowedDirectories(dirs []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cleaned := make([]string, len(dirs))
	for i, d := range dirs {
		cleaned[i] = filepath.Clean(d)
	}
	e.cfg.AllowedDirectories = cleaned
}

// EvaluateScope checks if any detected file paths fall outside the allowed directories.
// If no allowed directories are configured, all paths are allowed.
func (e *Engine) EvaluateScope(paths []string) Decision {
	if e.bypassed.Load() {
		return Decision{Allowed: true, Reason: "policy bypassed (paused)"}
	}

	e.mu.RLock()
	allowedDirs := make([]string, len(e.cfg.AllowedDirectories))
	copy(allowedDirs, e.cfg.AllowedDirectories)
	e.mu.RUnlock()

	if len(allowedDirs) == 0 {
		return Decision{Allowed: true, Reason: "no directory scope configured"}
	}

	for _, p := range paths {
		if !isInScope(p, allowedDirs) {
			return Decision{
				Allowed: false,
				Reason:  fmt.Sprintf("file %q is outside allowed directories", p),
			}
		}
	}
	return Decision{Allowed: true, Reason: "all files within allowed directories"}
}

// isInScope checks whether a file path falls within any of the allowed directories.
func isInScope(filePath string, allowedDirs []string) bool {
	// Resolve the path to absolute for comparison
	resolved := resolvePath(filePath)

	for _, dir := range allowedDirs {
		// A file is in scope if its resolved path starts with the allowed dir + separator
		dirWithSep := dir + string(filepath.Separator)
		if resolved == dir || strings.HasPrefix(resolved, dirWithSep) {
			return true
		}
	}
	return false
}

// resolvePath attempts to resolve a file path to an absolute, cleaned path.
// For relative paths, it resolves against the current working directory.
func resolvePath(p string) string {
	p = filepath.Clean(p)
	if filepath.IsAbs(p) {
		return p
	}
	// Resolve relative paths against cwd
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, p)
	}
	return p
}

// --- Content keyword methods ---

// GetDenyContentKeywords returns the current deny content keywords.
func (e *Engine) GetDenyContentKeywords() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, len(e.cfg.DenyContentKeywords))
	copy(out, e.cfg.DenyContentKeywords)
	return out
}

// SetDenyContentKeywords replaces all deny content keywords.
func (e *Engine) SetDenyContentKeywords(keywords []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.DenyContentKeywords = make([]string, len(keywords))
	copy(e.cfg.DenyContentKeywords, keywords)
}

// AddDenyContentKeyword appends a single deny content keyword.
func (e *Engine) AddDenyContentKeyword(keyword string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.DenyContentKeywords = append(e.cfg.DenyContentKeywords, keyword)
}

// RemoveDenyContentKeyword removes a single deny content keyword.
func (e *Engine) RemoveDenyContentKeyword(keyword string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	filtered := e.cfg.DenyContentKeywords[:0]
	for _, k := range e.cfg.DenyContentKeywords {
		if k != keyword {
			filtered = append(filtered, k)
		}
	}
	e.cfg.DenyContentKeywords = filtered
}

// GetContentKeywordWhitelist returns file paths that bypass content keyword checks.
func (e *Engine) GetContentKeywordWhitelist() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, len(e.cfg.ContentKeywordWhitelist))
	copy(out, e.cfg.ContentKeywordWhitelist)
	return out
}

// AddToContentKeywordWhitelist adds a file path to the content keyword whitelist.
func (e *Engine) AddToContentKeywordWhitelist(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, p := range e.cfg.ContentKeywordWhitelist {
		if p == path {
			return
		}
	}
	e.cfg.ContentKeywordWhitelist = append(e.cfg.ContentKeywordWhitelist, path)
}

// RemoveFromContentKeywordWhitelist removes a file path from the whitelist.
func (e *Engine) RemoveFromContentKeywordWhitelist(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	filtered := e.cfg.ContentKeywordWhitelist[:0]
	for _, p := range e.cfg.ContentKeywordWhitelist {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	e.cfg.ContentKeywordWhitelist = filtered
}

// GetContentKeywordBlacklist returns file paths that are always blocked by content keyword checks.
func (e *Engine) GetContentKeywordBlacklist() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, len(e.cfg.ContentKeywordBlacklist))
	copy(out, e.cfg.ContentKeywordBlacklist)
	return out
}

// AddToContentKeywordBlacklist adds a file path to the content keyword blacklist.
func (e *Engine) AddToContentKeywordBlacklist(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, p := range e.cfg.ContentKeywordBlacklist {
		if p == path {
			return
		}
	}
	e.cfg.ContentKeywordBlacklist = append(e.cfg.ContentKeywordBlacklist, path)
}

// RemoveFromContentKeywordBlacklist removes a file path from the blacklist.
func (e *Engine) RemoveFromContentKeywordBlacklist(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	filtered := e.cfg.ContentKeywordBlacklist[:0]
	for _, p := range e.cfg.ContentKeywordBlacklist {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	e.cfg.ContentKeywordBlacklist = filtered
}

// ContentKeywordResult holds the outcome of a content keyword evaluation.
type ContentKeywordResult struct {
	HasMatch       bool
	MatchedKeyword string
	AutoAllowed    []string // file paths resolved by whitelist
	AutoBlocked    []string // file paths resolved by blacklist
	NeedPrompt     []string // file paths needing user decision
}

// EvaluateContentKeywords scans the body for deny_content_keywords and partitions
// detected file paths into whitelist-allowed, blacklist-blocked, and needs-prompt.
func (e *Engine) EvaluateContentKeywords(body string, filePaths []string) ContentKeywordResult {
	if e.bypassed.Load() {
		return ContentKeywordResult{}
	}

	e.mu.RLock()
	keywords := make([]string, len(e.cfg.DenyContentKeywords))
	copy(keywords, e.cfg.DenyContentKeywords)
	whitelist := make(map[string]bool, len(e.cfg.ContentKeywordWhitelist))
	for _, p := range e.cfg.ContentKeywordWhitelist {
		whitelist[p] = true
	}
	blacklist := make(map[string]bool, len(e.cfg.ContentKeywordBlacklist))
	for _, p := range e.cfg.ContentKeywordBlacklist {
		blacklist[p] = true
	}
	e.mu.RUnlock()

	if len(keywords) == 0 {
		return ContentKeywordResult{}
	}

	// Case-insensitive keyword scan
	bodyLower := strings.ToLower(body)
	var matchedKeyword string
	for _, kw := range keywords {
		if strings.Contains(bodyLower, strings.ToLower(kw)) {
			matchedKeyword = kw
			break
		}
	}
	if matchedKeyword == "" {
		return ContentKeywordResult{}
	}

	// Partition file paths by whitelist/blacklist
	result := ContentKeywordResult{
		HasMatch:       true,
		MatchedKeyword: matchedKeyword,
	}
	for _, fp := range filePaths {
		switch {
		case whitelist[fp]:
			result.AutoAllowed = append(result.AutoAllowed, fp)
		case blacklist[fp]:
			result.AutoBlocked = append(result.AutoBlocked, fp)
		default:
			result.NeedPrompt = append(result.NeedPrompt, fp)
		}
	}
	return result
}

// EvaluateFiles checks if any detected file paths match deny_file_patterns.
func (e *Engine) EvaluateFiles(paths []string) Decision {
	if e.bypassed.Load() {
		return Decision{Allowed: true, Reason: "policy bypassed (paused)"}
	}

	e.mu.RLock()
	patterns := make([]string, len(e.cfg.DenyFilePatterns))
	copy(patterns, e.cfg.DenyFilePatterns)
	e.mu.RUnlock()

	if len(patterns) == 0 {
		return Decision{Allowed: true, Reason: "no file patterns configured"}
	}

	for _, path := range paths {
		for _, pattern := range patterns {
			if matchFilePattern(path, pattern) {
				return Decision{
					Allowed: false,
					Reason:  fmt.Sprintf("file %q matches deny pattern %q", path, pattern),
				}
			}
		}
	}
	return Decision{Allowed: true, Reason: "no file patterns matched"}
}

func matchFilePattern(path, pattern string) bool {
	path = strings.ToLower(path)
	pattern = strings.ToLower(pattern)

	// Handle ** prefix: match against any suffix of the path
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		// Check every subpath
		for i := 0; i < len(path); i++ {
			if i == 0 || path[i-1] == '/' {
				if matched, _ := filepath.Match(suffix, path[i:]); matched {
					return true
				}
			}
		}
		return false
	}

	// Try matching against the full path
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}

	// Also try matching against just the filename
	base := filepath.Base(path)
	if matched, _ := filepath.Match(pattern, base); matched {
		return true
	}

	return false
}
