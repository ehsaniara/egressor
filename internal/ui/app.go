package ui

import (
	"context"
	"log/slog"

	"github.com/ehsaniara/egressor/internal/audit"
	"github.com/ehsaniara/egressor/internal/config"
	"github.com/ehsaniara/egressor/internal/policy"
	"github.com/ehsaniara/egressor/internal/proxy"
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
}

func NewApp(server *proxy.Server, store *audit.SessionStore, engine *policy.Engine, cfg *config.Config, cfgPath string) *App {
	return &App{
		server:  server,
		store:   store,
		engine:  engine,
		cfg:     cfg,
		cfgPath: cfgPath,
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

// --- Policy management ---

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

func (a *App) SaveConfig() error {
	a.cfg.Policy.DenyFilePatterns = a.engine.GetDenyPatterns()
	a.cfg.Policy.AllowedDirectories = a.engine.GetAllowedDirectories()
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
