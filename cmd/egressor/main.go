package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ehsaniara/egressor/internal/audit"
	"github.com/ehsaniara/egressor/internal/ca"
	"github.com/ehsaniara/egressor/internal/config"
	"github.com/ehsaniara/egressor/internal/policy"
	"github.com/ehsaniara/egressor/internal/proxy"
	"github.com/ehsaniara/egressor/internal/ui"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "path to config file (default: ./config.yaml, then ~/.egressor/config.yaml)")
	headless := flag.Bool("headless", false, "run without UI (terminal only)")
	showVersion := flag.Bool("version", false, "print version and exit")
	generateCA := flag.Bool("generate-ca", false, "generate CA certificate and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("egressor", version)
		return
	}

	resolvedConfig := resolveConfigPath(*configPath)
	cfg, err := config.Load(resolvedConfig)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	if *generateCA {
		authority, err := ca.GenerateToPath(cfg.Intercept.CACert, cfg.Intercept.CAKey)
		if err != nil {
			slog.Error("failed to generate CA", "err", err)
			os.Exit(1)
		}
		fmt.Printf("CA certificate written to %s\n", cfg.Intercept.CACert)
		fmt.Printf("CA key written to %s\n", cfg.Intercept.CAKey)
		fmt.Printf("CA subject: %s\n", authority.Cert.Subject.CommonName)
		fmt.Println()
		fmt.Println("To trust on macOS, run:")
		fmt.Printf("  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", cfg.Intercept.CACert)
		return
	}

	maxLogSize := int64(cfg.Logging.MaxSizeMB) * 1024 * 1024
	logger, err := audit.NewLogger(cfg.Logging.Format, cfg.Logging.File, maxLogSize)
	if err != nil {
		slog.Error("failed to create audit logger", "err", err)
		os.Exit(1)
	}
	defer logger.Close()

	caGenerated := !fileExists(cfg.Intercept.CACert) || !fileExists(cfg.Intercept.CAKey)
	authority, err := ca.LoadOrGenerate(cfg.Intercept.CACert, cfg.Intercept.CAKey)
	if err != nil {
		slog.Error("failed to load CA", "err", err)
		os.Exit(1)
	}
	if caGenerated {
		fmt.Println()
		fmt.Println("  CA certificate was not found and has been auto-generated:")
		fmt.Printf("    cert: %s\n", cfg.Intercept.CACert)
		fmt.Printf("    key:  %s\n", cfg.Intercept.CAKey)
		fmt.Println()
		fmt.Println("  To trust the CA on macOS (required for TLS interception), run:")
		fmt.Printf("    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", cfg.Intercept.CACert)
		fmt.Println()
		fmt.Println("  For Node.js tools (Claude Code, Kiro, Cursor), also set:")
		fmt.Printf("    export NODE_EXTRA_CA_CERTS=%s\n", cfg.Intercept.CACert)
		fmt.Println()
	}
	engine := policy.NewEngine(cfg.Policy)
	interceptor := proxy.NewInterceptor(authority, cfg.Intercept.LogBody, cfg.Intercept.MaxBodySize, engine)
	slog.Info("TLS interception enabled")

	if *headless {
		server := proxy.NewServer(cfg.ListenAddress, logger, interceptor)
		runHeadless(server, cfg)
		return
	}

	// Default: run with desktop UI
	store := audit.NewSessionStore(1000)
	sink := audit.NewMultiSink(logger, store)
	server := proxy.NewServer(cfg.ListenAddress, sink, interceptor)
	app := ui.NewApp(server, store, engine, cfg, resolvedConfig)

	slog.Info("egressor starting", "address", cfg.ListenAddress, "mode", "ui")
	if err := ui.RunUI(app); err != nil {
		slog.Error("ui error", "err", err)
		os.Exit(1)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func resolveConfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	// 1. ./config.yaml
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}
	// 2. ~/.egressor/config.yaml
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".egressor", "config.yaml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fall back to ./config.yaml (will produce a clear error on load)
	return "config.yaml"
}

func runHeadless(server *proxy.Server, cfg *config.Config) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("egressor starting", "address", cfg.ListenAddress, "mode", "headless")

	if err := server.ListenAndServe(ctx); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}

	slog.Info("egressor stopped")
}
