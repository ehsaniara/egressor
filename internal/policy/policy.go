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
