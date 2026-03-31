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
