package ui

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ehsaniara/egressor/internal/audit"
	"github.com/ehsaniara/egressor/internal/config"
	"github.com/ehsaniara/egressor/internal/policy"
	"github.com/ehsaniara/egressor/internal/proxy"
	"github.com/ehsaniara/egressor/internal/tray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application struct.
// All exported methods are callable from the frontend.
type App struct {
	ctx     context.Context
	server  *proxy.Server
	store   *audit.SessionStore
	engine  *policy.Engine
	cfg     *config.Config
	cfgPath string

	pendingMu      sync.Mutex
	pendingPrompts map[string]chan policy.ContentPromptResponse
}

func NewApp(server *proxy.Server, store *audit.SessionStore, engine *policy.Engine, cfg *config.Config, cfgPath string) *App {
	return &App{
		server:         server,
		store:          store,
		engine:         engine,
		cfg:            cfg,
		cfgPath:        cfgPath,
		pendingPrompts: make(map[string]chan policy.ContentPromptResponse),
	}
}

// Startup is called by Wails when the app starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Push new sessions to the frontend in real-time
	a.store.OnSession(func(s *audit.Session) {
		wailsRuntime.EventsEmit(a.ctx, "session:new", s)
	})

	// Auto-start the proxy
	if err := a.server.Start(); err != nil {
		slog.Error("failed to start proxy", "err", err)
	}

	// Start system tray icon (macOS menu bar)
	if tray.Available() {
		go tray.Run(tray.Callbacks{
			OnPauseToggle: func(paused bool) {
				a.engine.SetBypassed(paused)
				if paused {
					slog.Info("policy paused via tray")
				} else {
					slog.Info("policy resumed via tray")
				}
			},
			OnQuit: func() {
				a.server.Stop()
				wailsRuntime.Quit(a.ctx)
			},
		})
	}
}

// Shutdown is called by Wails when the app closes.
func (a *App) Shutdown(ctx context.Context) {
	a.server.Stop()
}

// --- Session queries ---

func (a *App) GetRecentSessions(limit int) []*audit.Session {
	return a.store.Recent(limit)
}

func (a *App) GetSession(id string) *audit.Session {
	return a.store.GetByID(id)
}

func (a *App) GetStats() audit.StoreStats {
	return a.store.Stats()
}

// --- Policy management: deny patterns ---

func (a *App) GetDenyPatterns() []string {
	return a.engine.GetDenyPatterns()
}

func (a *App) SetDenyPatterns(patterns []string) {
	a.engine.SetDenyPatterns(patterns)
}

func (a *App) AddDenyPattern(pattern string) {
	a.engine.AddDenyPattern(pattern)
}

func (a *App) RemoveDenyPattern(pattern string) {
	a.engine.RemoveDenyPattern(pattern)
}

// --- Policy management: allowed directories ---

func (a *App) GetAllowedDirectories() []string {
	return a.engine.GetAllowedDirectories()
}

func (a *App) SetAllowedDirectories(dirs []string) {
	a.engine.SetAllowedDirectories(dirs)
}

func (a *App) AddAllowedDirectory(dir string) {
	dirs := a.engine.GetAllowedDirectories()
	dirs = append(dirs, dir)
	a.engine.SetAllowedDirectories(dirs)
}

func (a *App) RemoveAllowedDirectory(dir string) {
	dirs := a.engine.GetAllowedDirectories()
	filtered := dirs[:0]
	for _, d := range dirs {
		if d != dir {
			filtered = append(filtered, d)
		}
	}
	a.engine.SetAllowedDirectories(filtered)
}

// --- Policy management: content tags (hard block) ---

func (a *App) GetDenyContentTags() []string {
	return a.engine.GetDenyContentTags()
}

func (a *App) SetDenyContentTags(tags []string) {
	a.engine.SetDenyContentTags(tags)
}

func (a *App) AddDenyContentTag(tag string) {
	a.engine.AddDenyContentTag(tag)
}

func (a *App) RemoveDenyContentTag(tag string) {
	a.engine.RemoveDenyContentTag(tag)
}

// --- Policy management: content keywords (interactive) ---

func (a *App) GetDenyContentKeywords() []string {
	return a.engine.GetDenyContentKeywords()
}

func (a *App) SetDenyContentKeywords(keywords []string) {
	a.engine.SetDenyContentKeywords(keywords)
}

func (a *App) AddDenyContentKeyword(keyword string) {
	a.engine.AddDenyContentKeyword(keyword)
}

func (a *App) RemoveDenyContentKeyword(keyword string) {
	a.engine.RemoveDenyContentKeyword(keyword)
}

func (a *App) GetContentWhitelist() []string {
	return a.engine.GetContentWhitelist()
}

func (a *App) RemoveFromContentWhitelist(path string) {
	a.engine.RemoveFromContentWhitelist(path)
}

func (a *App) GetContentBlacklist() []string {
	return a.engine.GetContentBlacklist()
}

func (a *App) RemoveFromContentBlacklist(path string) {
	a.engine.RemoveFromContentBlacklist(path)
}

// --- Policy bypass ---

func (a *App) IsPolicyBypassed() bool {
	return a.engine.IsBypassed()
}

func (a *App) SetPolicyBypassed(bypassed bool) {
	a.engine.SetBypassed(bypassed)
	if bypassed {
		slog.Info("policy paused — all traffic allowed")
	} else {
		slog.Info("policy resumed")
	}
}

// --- Content keyword prompt resolution ---

// PromptUser implements policy.PromptResolver. It emits a Wails event and blocks
// until the frontend calls ResolveContentPrompt or the 30s timeout expires.
func (a *App) PromptUser(prompt policy.ContentPrompt) policy.ContentPromptResponse {
	ch := make(chan policy.ContentPromptResponse, 1)

	a.pendingMu.Lock()
	a.pendingPrompts[prompt.ID] = ch
	a.pendingMu.Unlock()

	wailsRuntime.EventsEmit(a.ctx, "content:prompt", prompt)

	select {
	case resp := <-ch:
		return resp
	case <-time.After(30 * time.Second):
		a.pendingMu.Lock()
		delete(a.pendingPrompts, prompt.ID)
		a.pendingMu.Unlock()
		slog.Warn("content prompt timed out, blocking",
			"prompt_id", prompt.ID,
			"keyword", prompt.MatchedKeyword,
		)
		return policy.ContentPromptResponse{Action: policy.PromptBlockOnce}
	}
}

// ResolveContentPrompt is called by the frontend to respond to a content keyword prompt.
func (a *App) ResolveContentPrompt(promptID string, action string) {
	a.pendingMu.Lock()
	ch, ok := a.pendingPrompts[promptID]
	if ok {
		delete(a.pendingPrompts, promptID)
	}
	a.pendingMu.Unlock()

	if !ok {
		return
	}

	promptAction := policy.PromptAction(action)

	// Handle persistent decisions
	// For allow_always/block_always, we apply to all files from the original prompt.
	// The interceptor already has the file paths; the whitelist/blacklist stores them.
	// We retrieve pending prompt info from the channel interaction.

	ch <- policy.ContentPromptResponse{Action: promptAction}
}

// ResolveContentPromptForFile is called by the frontend with a specific file path
// for whitelist/blacklist persistence, separate from the blocking resolution.
func (a *App) ResolveContentPromptForFile(action string, filePath string) {
	switch policy.PromptAction(action) {
	case policy.PromptAllowAlways:
		a.engine.AddToContentWhitelist(filePath)
		slog.Info("file added to content keyword whitelist", "path", filePath)
	case policy.PromptBlockAlways:
		a.engine.AddToContentBlacklist(filePath)
		slog.Info("file added to content keyword blacklist", "path", filePath)
	}
}

// --- Config persistence ---

func (a *App) SaveConfig() error {
	a.cfg.Policy.DenyFilePatterns = a.engine.GetDenyPatterns()
	a.cfg.Policy.AllowedDirectories = a.engine.GetAllowedDirectories()
	a.cfg.Policy.DenyContentTags = a.engine.GetDenyContentTags()
	a.cfg.Policy.DenyContentKeywords = a.engine.GetDenyContentKeywords()
	a.cfg.Policy.ContentWhitelist = a.engine.GetContentWhitelist()
	a.cfg.Policy.ContentBlacklist = a.engine.GetContentBlacklist()
	return config.Save(a.cfgPath, a.cfg)
}

// --- Proxy control ---

func (a *App) StartProxy() error {
	return a.server.Start()
}

func (a *App) StopProxy() {
	a.server.Stop()
}

func (a *App) IsProxyRunning() bool {
	return a.server.IsRunning()
}

func (a *App) GetListenAddress() string {
	return a.server.Address()
}
