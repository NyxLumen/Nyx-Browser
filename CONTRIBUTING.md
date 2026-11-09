# Contributing to Nyx Browser

Thank you for your interest in contributing to Nyx Browser! We welcome contributions, bug fixes, documentation improvements, and feature proposals.

## Project Structure

Nyx Browser is organized into two primary components:
- **Backend**: Written in [Go](https://go.dev/) (Go 1.27+), located in the root repository and subpackages under module path `github.com/nyxlumen/nyx-browser/v2`.
- **Frontend**: Written in [Vue 3](https://vuejs.org/) and TypeScript with Vite and Pinia, located in the [`frontend`](frontend/) directory.

To clone the repository:

```bash
git clone https://github.com/NyxLumen/Nyx-Browser.git
cd Nyx-Browser
```

We use [Taskfile](https://taskfile.dev/) to automate development tasks, builds, and releases.

## Building from Source

You can build the full project (frontend assets + Go binary) using Taskfile:

```bash
task build
```

Or step-by-step:

### 1. Frontend

We use Node.js (v22/v24) and [pnpm](https://pnpm.io/) to manage the frontend build process:

```bash
cd frontend

# Install dependencies
pnpm install

# Build static assets (compiles to frontend/dist)
pnpm run build

# Start development server with hot module reloading
pnpm run dev

# Run frontend test suite
pnpm test
```

### 2. Backend

Ensure Go dependencies are downloaded:

```bash
go mod download
```

Build or run Nyx Browser:

```bash
# Compile binary
go build -o nyx .

# Run directly
go run . -r /path/to/files -p 8080 -d ./nyx.db

# Run backend tests
go test -v ./...
```

## Documentation & CLI Reference

Documentation is maintained in the [`docs`](docs/) directory as Markdown.

The command-line interface reference under [`docs/cli`](docs/cli/) is automatically generated from Cobra commands:

```bash
go run . docs --out docs/cli
```

Alternatively, you can run:

```bash
task docs:cli:generate
```

Always ensure the CLI documentation is regenerated if you introduce or alter CLI flags, commands, or defaults.

## Translations

Locale files live in [`frontend/src/i18n`](frontend/src/i18n) and can be edited directly. Pull requests adding or improving translations are welcomed.

## Authentication Providers

To build a new authentication provider, implement the `Auther` interface in [`auth/auth.go`](auth/auth.go):

```go
// Auther is the authentication interface.
type Auther interface {
    // Auth is called to authenticate a request.
    Auth(r *http.Request, s *users.Storage, root string) (*users.User, error)
}
```

After implementing the interface:
1. Add the implementation under the [`auth/`](auth/) directory.
2. Register configuration flags in [`cmd/config.go`](cmd/config.go).
3. Connect the backend retrieval in [`storage/bolt/auth.go`](storage/bolt/auth.go).

## Pull Request Guidelines

1. Ensure all Go unit and integration tests pass: `go test -count=1 ./...`
2. Ensure Go code is formatted and passes vet checks: `go vet ./...`
3. Ensure frontend tests and typechecks pass: `pnpm --dir frontend test` and `pnpm --dir frontend build`
4. If modifying CLI flags or commands, run `go run . docs --out docs/cli` to keep documentation in sync.
