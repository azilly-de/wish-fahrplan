# AGENTS.md

## Project Overview

`wish-fahrplan` is a Go terminal application using the [charm](https://charm.land) ecosystem.
Users connect via SSH and see real-time train schedules for configured routes (e.g. "Heim -> Büro").
Built with `wish` (SSH server) and `bubbletea` (TUI framework).

## Project Structure

```
.
├── main.go              # Entry point, SSH server setup
├── model.go             # Bubbletea model (state, Init, Update, View)
├── fahrplan.go          # Schedule fetching and parsing logic
├── connection.go        # Connection/route configuration types
├── PROJECT.md           # Project description (German)
└── AGENTS.md            # This file
```

Keep the project flat — one package (`main`) unless it grows significantly.

## Build / Lint / Test Commands

```bash
# Initialize the module (once)
go mod init github.com/<user>/wish-fahrplan

# Download dependencies
go mod tidy

# Build the binary
go build -o wish-fahrplan .

# Run locally (listens on SSH port)
go run .

# Run all tests
go test ./...

# Run a single test by name
go test -run TestFunctionName ./...

# Run tests with verbose output
go test -v ./...

# Run tests in a specific package
go test ./internal/...

# Lint (requires golangci-lint)
golangci-lint run ./...

# Format code
gofmt -w .
# or
go fmt ./...

# Vet for common issues
go vet ./...
```

Before committing, always run: `go fmt ./... && go vet ./... && go test ./...`

## Dependencies

Core charm libraries to use:

- `github.com/charmbracelet/wish` — SSH server framework
- `github.com/charmbracelet/bubbletea` — TUI framework (Elm architecture)
- `github.com/charmbracelet/lipgloss` — Terminal styling
- `github.com/charmbracelet/log` — Structured logging (optional)

Use the standard library (`net/http`, `encoding/json`, `html`) for HTTP fetching and HTML parsing.
Add new dependencies with `go get <module>` then `go tidy`.

## Code Style Guidelines

### Imports

Group imports in three blocks separated by blank lines:

```go
import (
    // Standard library
    "fmt"
    "net/http"
    "time"

    // External dependencies
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/wish"

    // Internal (if using packages)
    // "github.com/<user>/wish-fahrplan/internal/parser"
)
```

### Types and Naming

- Use camelCase for unexported identifiers, PascalCase for exported.
- Prefer short, descriptive receiver names (e.g., `m` for model, `s` for server).
- Define types for domain concepts early:

```go
type Connection struct {
    Name        string
    OriginID    string
    DestID      string
}

type Departure struct {
    Time      time.Time
    Line      string
    Stop      string
    ArrivalAt time.Time
}
```

### Functions

- Keep functions short and focused. If a function exceeds ~40 lines, split it.
- Return errors as the last return value. Wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("fetching schedule for %s: %w", conn.Name, err)
}
```

### Error Handling

- Always check and handle errors. Do not use `_ = err`.
- Use `fmt.Errorf` with `%w` for wrapping; `errors.Is` / `errors.As` for checking.
- For the TUI, surface errors in the view rather than logging and exiting silently.

### Bubbletea Conventions

- The model is a single struct. Keep related state together.
- Messages (`Msg` types) should be small, purpose-specific structs.
- Use `tea.Cmd` for side effects (HTTP calls, timers). Never do I/O in `Update`.
- Use `lipgloss.Style` for layout, not raw ANSI codes.

```go
type model struct {
    connections []Connection
    departures  map[string][]Departure
    err         error
    loading     bool
}

func (m model) Init() tea.Cmd { return fetchAllDepartures(m.connections) }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m model) View() string { ... }
```

### Configuration

Hard-code route definitions as Go slices in the source. The project spec calls for
adding connections via code changes, not config files:

```go
var connections = []Connection{
    {Name: "Heim -> Büro", OriginID: "de:07111:010240", DestID: "de:07111:010001"},
    {Name: "Büro -> Heim", OriginID: "de:07111:010001", DestID: "de:07111:010240"},
}
```

## Testing

- Place tests in `*_test.go` files in the same package.
- Name tests `TestFunctionName` or `TestTypeName_MethodName`.
- Use table-driven tests for parsing and data transformation:

```go
func TestParseDeparture(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Departure
        wantErr bool
    }{
        {"valid row", "<td>...</td>", Departure{...}, false},
        {"empty row", "", Departure{}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := parseDeparture(tt.input)
            // assertions...
        })
    }
}
```

- Use `net/http/httptest` for testing HTTP scraping logic.
- Do not test Bubbletea rendering output in unit tests; test model state transitions.

## Commits and PRs

- Write imperative commit messages: "Add departure parsing", not "Added" or "Adds".
- Keep commits atomic — one logical change per commit.
- Run `go fmt`, `go vet`, and `go test` before every commit.

## General Conventions

- Go version: use the latest stable (1.22+).
- Line length: no hard limit, but keep lines under ~120 chars where practical.
- Comments: use doc comments on exported types/functions. No comments on obvious code.
- No `panic` in production code. Return errors instead.
- Use `context.Context` as the first parameter for functions that do I/O or may be cancelled.
