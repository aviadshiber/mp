# CLAUDE.md

## Project Overview

`mp` is a Go CLI tool that wraps the Mixpanel read API into a single command-line interface. It supports exporting events, running analytics queries (segmentation, funnels, retention, etc.), querying user/group profiles, and accessing project metadata. Output formats include JSON, CSV, JSONL, and human-readable tables.

**Repository:** `github.com/aviadshiber/mp`
**License:** MIT
**Go version:** 1.25.5

## Project Structure

```
cmd/                  # CLI commands (Cobra-based)
  mp/main.go          # Binary entry point
  root.go             # Root command, flag binding, subcommand registration
  config.go           # config set/get/list
  export.go           # Event export (JSONL streaming)
  profiles.go         # User profile queries
  profiles_groups.go  # Group profile queries
  query_*.go          # Analytics queries (segmentation, funnels, retention, etc.)
  schemas.go          # Schema list/get
  annotations.go      # Annotation list/get
  cohorts.go          # Cohort listing
  lookup_tables.go    # Lookup table listing
  pipelines.go        # Pipeline list/status
  activity.go         # Activity log
  version.go          # Version command
internal/
  client/client.go    # Mixpanel HTTP client (auth, rate limiting, gzip)
  config/config.go    # Config file + env var management (Viper)
  iostreams/          # I/O abstraction with TTY detection
  output/             # Formatters: JSON, CSV, JSONL, table
```

## Build & Development Commands

```bash
make build        # Build binary to bin/mp (injects version via ldflags)
make test         # Run go test -v ./...
make lint         # Run golangci-lint
make fmt          # Run go fmt ./...
make install      # Install to $GOPATH/bin
make clean        # Remove bin/ and dist/
make completions  # Generate shell completions (bash/zsh/fish)
make release      # Snapshot release via goreleaser
```

## Key Dependencies

- **cobra** — CLI framework for command structure
- **viper** — Configuration management (file + env vars)
- **gojq** — jq-compatible JSON filtering
- **tablewriter** — Human-readable table output
- **termenv** — Terminal color/styling

## Architecture & Conventions

### Command Pattern
Each command in `cmd/` follows the same pattern:
1. Define a `cobra.Command` with flags
2. In `RunE`, load config via `internal/config`, build an API client via `internal/client`
3. Call the Mixpanel API endpoint
4. Format and write output via `internal/output` formatters

### Configuration
- Config file: `~/.config/mp/config.yaml`
- Environment variable prefix: `MP_` (e.g., `MP_PROJECT_ID`, `MP_TOKEN`)
- Precedence: CLI flags > env vars > config file > defaults
- Sensitive values (`service_secret`) are masked in display

### Authentication
- Basic Auth with `service_account:service_secret` (Base64-encoded)
- Regional endpoints: `us` (default), `eu`, `in`
- Rate limiting with exponential backoff on 429 responses

### Output System
- TTY detection for automatic formatting (color, tables vs raw JSON)
- Global flags: `--json`, `--jq`, `--template`, `--quiet`
- `NO_COLOR` environment variable supported
- Templates use Go `text/template` syntax

### Code Style
- Standard `go fmt` formatting
- `golangci-lint` for linting
- Internal packages are under `internal/` (not importable externally)
- Follows the `gh` CLI pattern for I/O streams abstraction

## CI/CD

- **Release workflow** (`.github/workflows/release.yml`): triggers on `v*` tags, uses GoReleaser v2
- Builds for linux/darwin/windows on amd64/arm64
- Publishes to GitHub Releases and Homebrew tap (`aviadshiber/homebrew-tap`)

## Testing

No test files exist yet. Test infrastructure is set up via `make test` (`go test -v ./...`). New tests should follow standard Go conventions (`*_test.go` files alongside source).

## Common Tasks

- **Add a new API command:** Create a new file in `cmd/`, define a `cobra.Command`, register it in `root.go`'s `init()` function
- **Add a new output format:** Extend `internal/output/` with a new formatter
- **Change auth/HTTP behavior:** Modify `internal/client/client.go`
- **Update config options:** Modify `internal/config/config.go` and update relevant commands
