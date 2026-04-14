# Contributing to Egressor

Thanks for your interest in contributing to Egressor! This document explains how to get started.

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 22+
- Xcode Command Line Tools (macOS, for desktop UI builds)

### Setup

```bash
git clone https://github.com/ehsaniara/egressor.git
cd egressor

# Build the frontend
cd internal/ui/frontend && npm install && npm run build && cd ../../..

# Build the binary (macOS with desktop UI)
CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags production -o egressor ./cmd/egressor

# Build headless only (any platform, no CGO)
go build -o egressor ./cmd/egressor
```

### Running Tests

```bash
go test ./internal/...
```

Run `go vet` before submitting:

```bash
go vet ./...
```

## How to Contribute

### Reporting Bugs

- Search [existing issues](https://github.com/ehsaniara/egressor/issues) first to avoid duplicates
- Use the **Bug Report** issue template
- Include your OS, Go version, and steps to reproduce

### Suggesting Features

- Open a **Feature Request** issue
- Explain the use case and why it would benefit Egressor users

### Submitting Code

1. Fork the repository
2. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/my-change
   ```
3. Make your changes
4. Add or update tests for any new functionality
5. Ensure all tests pass: `go test ./internal/...`
6. Ensure `go vet ./...` reports no issues
7. Commit with a clear message describing *what* and *why*
8. Push to your fork and open a pull request against `main`

### Pull Request Guidelines

- Keep PRs focused on a single change
- Include a clear description of what the PR does and why
- Add tests for new functionality
- Update documentation if behavior changes
- Make sure CI passes before requesting review

## Project Structure

```
cmd/egressor/          Entry point
internal/
  proxy/               TCP/HTTPS interception
  policy/              Policy engine (scope, patterns, tags, keywords)
  audit/               Session logging and storage
  ca/                  CA certificate generation
  extract/             File reference detection
  config/              YAML config loading
  tray/                macOS system tray
  ui/                  Wails desktop UI + React frontend
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep packages focused and well-scoped
- Use `slog` for structured logging
- Write table-driven tests where appropriate
- Handle errors explicitly with context

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).