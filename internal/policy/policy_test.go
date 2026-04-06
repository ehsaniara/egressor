package policy

import (
	"testing"

	"github.com/ehsaniara/egressor/internal/config"
)

func TestEvaluateFiles_DenyPatterns(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyFilePatterns: []string{
			"*.env",
			"*.pem",
			"*.key",
			"**/secrets/**",
			"**/credentials*",
			".aws/*",
		},
	})

	tests := []struct {
		name    string
		paths   []string
		allowed bool
	}{
		{"env file", []string{".env"}, false},
		{"nested env", []string{"config/.env"}, false},
		{"prod env", []string{".env.production"}, true},
		{"pem file", []string{"ca.pem"}, false},
		{"key file", []string{"private.key"}, false},
		{"secrets dir", []string{"config/secrets/db.yaml"}, false},
		{"credentials file", []string{"home/credentials.json"}, false},
		{"aws config", []string{".aws/config"}, false},
		{"aws creds", []string{".aws/credentials"}, false},
		{"normal go file", []string{"cmd/main.go"}, true},
		{"normal ts file", []string{"src/index.ts"}, true},
		{"empty list", []string{}, true},
		{"mix allowed and denied", []string{"src/app.go", ".env"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := engine.EvaluateFiles(tt.paths)
			if decision.Allowed != tt.allowed {
				t.Errorf("EvaluateFiles(%v) = allowed:%v, want allowed:%v (reason: %s)",
					tt.paths, decision.Allowed, tt.allowed, decision.Reason)
			}
		})
	}
}

func TestEvaluateFiles_NoPatterns(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{})
	decision := engine.EvaluateFiles([]string{".env", "secrets/key.pem"})
	if !decision.Allowed {
		t.Error("expected allowed when no patterns configured")
	}
}

func TestEvaluateFiles_Bypassed(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyFilePatterns: []string{"*.env"},
	})
	engine.SetBypassed(true)
	decision := engine.EvaluateFiles([]string{".env"})
	if !decision.Allowed {
		t.Error("expected allowed when policy bypassed")
	}
}

func TestEvaluateScope_InScope(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		AllowedDirectories: []string{"/home/user/project"},
	})

	tests := []struct {
		name    string
		paths   []string
		allowed bool
	}{
		{"file in project", []string{"/home/user/project/main.go"}, true},
		{"nested file in project", []string{"/home/user/project/src/app.go"}, true},
		{"file outside project", []string{"/etc/passwd"}, false},
		{"home dir file", []string{"/home/user/.ssh/id_rsa"}, false},
		{"sibling project", []string{"/home/user/other-project/main.go"}, false},
		{"parent traversal", []string{"/home/user/project/../.ssh/id_rsa"}, false},
		{"mix in and out of scope", []string{"/home/user/project/main.go", "/etc/passwd"}, false},
		{"empty list", []string{}, true},
		{"exact dir match", []string{"/home/user/project"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := engine.EvaluateScope(tt.paths)
			if decision.Allowed != tt.allowed {
				t.Errorf("EvaluateScope(%v) = allowed:%v, want allowed:%v (reason: %s)",
					tt.paths, decision.Allowed, tt.allowed, decision.Reason)
			}
		})
	}
}

func TestEvaluateScope_MultipleAllowedDirs(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		AllowedDirectories: []string{"/home/user/project-a", "/home/user/project-b"},
	})

	tests := []struct {
		name    string
		paths   []string
		allowed bool
	}{
		{"file in project-a", []string{"/home/user/project-a/main.go"}, true},
		{"file in project-b", []string{"/home/user/project-b/main.go"}, true},
		{"file in neither", []string{"/home/user/project-c/main.go"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := engine.EvaluateScope(tt.paths)
			if decision.Allowed != tt.allowed {
				t.Errorf("EvaluateScope(%v) = allowed:%v, want allowed:%v (reason: %s)",
					tt.paths, decision.Allowed, tt.allowed, decision.Reason)
			}
		})
	}
}

func TestEvaluateScope_NoDirectoriesConfigured(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{})
	decision := engine.EvaluateScope([]string{"/anywhere/file.go"})
	if !decision.Allowed {
		t.Error("expected allowed when no directories configured")
	}
}

func TestEvaluateScope_Bypassed(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		AllowedDirectories: []string{"/home/user/project"},
	})
	engine.SetBypassed(true)
	decision := engine.EvaluateScope([]string{"/etc/passwd"})
	if !decision.Allowed {
		t.Error("expected allowed when policy bypassed")
	}
}

func TestEvaluateScope_ParentTraversal(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		AllowedDirectories: []string{"/home/user/project"},
	})

	// ../.. traversal should be caught after path cleaning
	decision := engine.EvaluateScope([]string{"/home/user/project/../../etc/passwd"})
	if decision.Allowed {
		t.Error("expected blocked for parent traversal escaping allowed directory")
	}
}

func TestEvaluateContentKeywords_Match(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyContentKeywords: []string{"CONFIDENTIAL", "INTERNAL ONLY"},
	})

	result := engine.EvaluateContentKeywords("This document is CONFIDENTIAL and should not be shared", []string{"doc.txt"})
	if !result.HasMatch {
		t.Fatal("expected match")
	}
	if result.MatchedKeyword != "CONFIDENTIAL" {
		t.Errorf("expected keyword CONFIDENTIAL, got %q", result.MatchedKeyword)
	}
	if len(result.NeedPrompt) != 1 || result.NeedPrompt[0] != "doc.txt" {
		t.Errorf("expected doc.txt in NeedPrompt, got %v", result.NeedPrompt)
	}
}

func TestEvaluateContentKeywords_CaseInsensitive(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyContentKeywords: []string{"confidential"},
	})

	result := engine.EvaluateContentKeywords("This is CONFIDENTIAL data", []string{"file.go"})
	if !result.HasMatch {
		t.Error("expected case-insensitive match")
	}
}

func TestEvaluateContentKeywords_NoMatch(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyContentKeywords: []string{"CONFIDENTIAL"},
	})

	result := engine.EvaluateContentKeywords("This is a normal document", []string{"file.go"})
	if result.HasMatch {
		t.Error("expected no match")
	}
}

func TestEvaluateContentKeywords_NoKeywords(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{})

	result := engine.EvaluateContentKeywords("CONFIDENTIAL data", []string{"file.go"})
	if result.HasMatch {
		t.Error("expected no match when no keywords configured")
	}
}

func TestEvaluateContentKeywords_Bypassed(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyContentKeywords: []string{"CONFIDENTIAL"},
	})
	engine.SetBypassed(true)

	result := engine.EvaluateContentKeywords("CONFIDENTIAL data", []string{"file.go"})
	if result.HasMatch {
		t.Error("expected no match when policy bypassed")
	}
}

func TestEvaluateContentKeywords_WhitelistBypass(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyContentKeywords:     []string{"CONFIDENTIAL"},
		ContentKeywordWhitelist: []string{"trusted.go"},
	})

	result := engine.EvaluateContentKeywords("CONFIDENTIAL data", []string{"trusted.go", "untrusted.go"})
	if !result.HasMatch {
		t.Fatal("expected match")
	}
	if len(result.AutoAllowed) != 1 || result.AutoAllowed[0] != "trusted.go" {
		t.Errorf("expected trusted.go in AutoAllowed, got %v", result.AutoAllowed)
	}
	if len(result.NeedPrompt) != 1 || result.NeedPrompt[0] != "untrusted.go" {
		t.Errorf("expected untrusted.go in NeedPrompt, got %v", result.NeedPrompt)
	}
}

func TestEvaluateContentKeywords_BlacklistBlock(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyContentKeywords:     []string{"CONFIDENTIAL"},
		ContentKeywordBlacklist: []string{"blocked.go"},
	})

	result := engine.EvaluateContentKeywords("CONFIDENTIAL data", []string{"blocked.go", "other.go"})
	if !result.HasMatch {
		t.Fatal("expected match")
	}
	if len(result.AutoBlocked) != 1 || result.AutoBlocked[0] != "blocked.go" {
		t.Errorf("expected blocked.go in AutoBlocked, got %v", result.AutoBlocked)
	}
	if len(result.NeedPrompt) != 1 || result.NeedPrompt[0] != "other.go" {
		t.Errorf("expected other.go in NeedPrompt, got %v", result.NeedPrompt)
	}
}

func TestEvaluateContentKeywords_AllWhitelisted(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{
		DenyContentKeywords:     []string{"CONFIDENTIAL"},
		ContentKeywordWhitelist: []string{"a.go", "b.go"},
	})

	result := engine.EvaluateContentKeywords("CONFIDENTIAL data", []string{"a.go", "b.go"})
	if !result.HasMatch {
		t.Fatal("expected match")
	}
	if len(result.NeedPrompt) != 0 {
		t.Errorf("expected empty NeedPrompt when all whitelisted, got %v", result.NeedPrompt)
	}
	if len(result.AutoAllowed) != 2 {
		t.Errorf("expected 2 AutoAllowed, got %v", result.AutoAllowed)
	}
}

func TestContentKeywordWhitelistCRUD(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{})

	engine.AddToContentKeywordWhitelist("file.go")
	engine.AddToContentKeywordWhitelist("file.go") // duplicate
	wl := engine.GetContentKeywordWhitelist()
	if len(wl) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(wl))
	}

	engine.RemoveFromContentKeywordWhitelist("file.go")
	wl = engine.GetContentKeywordWhitelist()
	if len(wl) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(wl))
	}
}

func TestContentKeywordBlacklistCRUD(t *testing.T) {
	engine := NewEngine(config.PolicyConfig{})

	engine.AddToContentKeywordBlacklist("file.go")
	engine.AddToContentKeywordBlacklist("file.go") // duplicate
	bl := engine.GetContentKeywordBlacklist()
	if len(bl) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(bl))
	}

	engine.RemoveFromContentKeywordBlacklist("file.go")
	bl = engine.GetContentKeywordBlacklist()
	if len(bl) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(bl))
	}
}

func TestMatchFilePattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		match   bool
	}{
		{".env", "*.env", true},
		{"config/.env", "*.env", true},
		{".env.local", "*.env.*", true},
		{"ca.pem", "*.pem", true},
		{"path/to/ca.pem", "*.pem", true},
		{"main.go", "*.pem", false},
		{"config/secrets/db.yaml", "**/secrets/**", true},
		{"secrets/api.key", "**/secrets/**", true},
		{".aws/credentials", ".aws/*", true},
		{".aws/config", ".aws/*", true},
		{"src/main.go", "*.go", true},
		{"src/main.go", "**/main.go", true},
		{"deeply/nested/main.go", "**/main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			if got := matchFilePattern(tt.path, tt.pattern); got != tt.match {
				t.Errorf("matchFilePattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.match)
			}
		})
	}
}
