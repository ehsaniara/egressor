package policy

import (
	"fmt"
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

// EvaluateFiles checks if any detected file paths match deny_file_patterns.
func (e *Engine) EvaluateFiles(paths []string) Decision {
	if e.bypassed.Load() {
		return Decision{Allowed: true, Reason: "policy bypassed (paused)"}
	}

	e.mu.RLock()
	patterns := e.cfg.DenyFilePatterns
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
