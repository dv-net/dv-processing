<div align="center">

## 🤝 Contributing to DV Processing

*Guidelines for contributing to the project*

</div>

---

## 📋 Table of Contents

- [🚀 Getting Started](#-getting-started) — Setup development environment
- [🔄 Development Workflow](#-development-workflow) — Branch strategy and workflow
- [📐 Coding Standards](#-coding-standards) — Code style and conventions
- [🧪 Testing](#-testing) — Testing requirements and guidelines
- [💬 Commit Messages](#-commit-messages) — Commit message format
- [🔀 Pull Request Process](#-pull-request-process) — PR submission and review
- [🐛 Issue Reporting](#-issue-reporting) — How to report bugs
- [🔒 Security](#-security) — Security vulnerability reporting
- [👀 Code Review](#-code-review) — Review process and criteria
- [🏷️ Release Process](#-release-process) — Versioning and releases

---

## 🚀 Getting Started

### Prerequisites

- **Go 1.24.2+** — [Download](https://go.dev/dl/)
- **PostgreSQL** — Database operations
- **Make** — Build commands
- **Git** — Version control

### Setup

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/dv-processing.git
cd dv-processing

# 2. Add upstream remote
git remote add upstream https://github.com/dv-net/dv-processing.git

# 3. Install development tools
make install-dev-tools

# 4. Build and verify
make build
make fmt
make lint
```

> 💡 **Tip**: Run `go mod download` if dependencies are missing

---

## 🔄 Development Workflow

### Branch Strategy

- 🌿 **`main`** — Production-ready stable code
- 🔧 **`dev`** — Active development branch
- 🌱 **`feature/*`** — New features (target: `dev` or `main`)
- 🐛 **`fix/*`** — Bug fixes (target: `dev`)

### Workflow

```bash
# 1. Update main branch
git checkout main
git pull upstream main

# 2. Create feature branch
git checkout -b feature/your-feature-name

# 3. Make changes, then verify
make fmt
make lint
```

> ⚠️ **Important**: Always create PRs from feature branches, never from `main` or `dev`

---

## 📐 Coding Standards

### Style Guide

Follow [Effective Go](https://go.dev/doc/effective_go) and project conventions:

- **Formatting** — `gofumpt` (via `make fmt`)
- **Imports** — `goimports` for organization
- **Naming** — Go naming conventions
- **Errors** — Explicit handling required
- **Documentation** — Document all exported functions/types

### Linting

```bash
# Run linter
make lint
```

The project uses `golangci-lint` with strict rules. All linter errors must be resolved before submitting PRs.

### Architecture

```
cmd/                CLI entrypoints
internal/           Internal packages (not for external use)
  ├── handler/      HTTP/gRPC handlers
  ├── services/     Business logic
  ├── store/        Repositories and data access
  ├── config/       Configuration management
  ├── fsm/          Finite state machines for blockchains
  └── ...
pkg/                Shared libraries (for external use)
api/                Generated API code
schema/             Protocol buffer definitions
sql/                Database migrations and queries
```

### Key Rules

- 🚫 **Transactions** — Use proper transaction management via store layer
- ✅ **Structs** — Initialize all struct fields in constructors
- ✅ **Naming** — Use `snake_case` for JSON/YAML fields
- ✅ **Size** — Functions < 180 lines (handlers configurable)
- ✅ **Complexity** — Cyclomatic complexity < 60
- ✅ **Generated Code** — Never edit generated files directly (`.pb.go`, `.sql.go`, `.connect.go`)

---

## 🧪 Testing

### Requirements

- ✅ **New Features** — Must include tests
- ✅ **Bug Fixes** — Must include regression tests
- ✅ **Framework** — Use `testify` for assertions
- ✅ **Naming** — Test files: `*_test.go` in same package

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/service/package

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...
```

### Coverage

> 🎯 **Target**: **80%+** coverage for new code
> 
> Focus on testing business logic and edge cases

---

## 💬 Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Commit Types

- `feat` — New feature
- `fix` — Bug fix
- `docs` — Documentation changes
- `refactor` — Code refactoring
- `perf` — Performance improvements
- `test` — Adding or updating tests
- `chore` — Maintenance tasks
- `security` — Security fixes

### Example

```bash
feat(transfers): add Bitcoin Cash transfer support

Add support for Bitcoin Cash transfers with proper
error handling and retry logic.

Closes #123
```

---

## 🔀 Pull Request Process

### Before Submitting

```bash
# 1. Update your branch
git checkout main
git pull upstream main
git checkout your-branch
git rebase upstream/main

# 2. Run all checks
make fmt
make lint
```

### Creating PR

**Step 1**: Push your branch
```bash
git push origin your-branch
```

**Step 2**: Create PR on GitHub
- Target: `main` or `dev` branch
- Title: Clear and descriptive
- Description: Include what changed and why
- Issues: Link related issue numbers

**Step 3**: Verify requirements

- ✅ **Code style** — Follows project guidelines
- ✅ **Linting** — `make lint` passes
- ✅ **Documentation** — Updated if needed
- ✅ **Conflicts** — No merge conflicts
- ✅ **Commits** — Follow conventions
- ✅ **Generated Code** — Regenerated if schema/proto changed

### Review Process

- **Initial Review** — Within 48 hours
- **Follow-up** — Within 24 hours
- **CI Checks** — Must all pass
- **Branch Status** — Keep updated with target

> 💡 **Tip**: Address review comments promptly and keep your branch rebased

---

## 🐛 Issue Reporting

### Before Reporting

- 🔍 **Duplicates** — Check existing issues
- 🌿 **Branch** — Verify in latest `main` or `dev`
- 📦 **Version** — Ensure using latest version

### Issue Template

When creating an issue, include:

- **OS and Version** — Your environment details
- **Go Version** — `go version` output
- **Steps to Reproduce** — Clear, numbered steps
- **Expected Behavior** — What should happen
- **Actual Behavior** — What actually happens
- **Logs** — Relevant error logs
- **Configuration** — Relevant config (sanitized)

> 📝 **Note**: The more details you provide, the faster we can help

---

## 🔒 Security

### Security Issues

> ⚠️ **IMPORTANT**: **DO NOT** create public issues for security vulnerabilities.

- 📧 **Email** — [support@dv.net](mailto:support@dv.net)
- 📋 **Details** — Include detailed vulnerability information
- ⏱️ **Disclosure** — Allow time for fix before public disclosure

> 🔐 Security issues are handled privately to protect users

---

## 👀 Code Review

### Review Criteria

- ✅ **Code Quality** — Style and best practices
- ✅ **Test Coverage** — Adequate test coverage
- ✅ **Documentation** — Updated documentation
- ✅ **Security** — Security considerations
- ⚡ **Performance** — Performance impact
- 🔄 **Compatibility** — Backward compatibility

### Timeline

- **Initial Review** — **48 hours**
- **Follow-up Reviews** — **24 hours**
- **Merge Decision** — **1 week** (for approved PRs)

---

## 🏷️ Release Process

### Release Tags

- **Stable** — `vX.X.X` (production releases)
- **RC** — `vX.X.X-RC1` (release candidates)

### Process

```
1. Development in feature branches
2. Testing and stabilization
3. Merge to main
4. Tag stable release: vX.X.X
```

---

## 🛠️ Common Tasks

### Database Migrations

```bash
# Create new migration
make db-create-migration migration_name

# Apply migrations
make migrate up

# Rollback migrations
make migrate down
```

### Code Generation

```bash
# Generate SQL code
make gensql

# Generate Protocol Buffers
make genproto

# Generate ABI bindings
make genabi

# Generate all (SQL + Proto + ABI + Envs)
make gen

# Generate environment variables documentation
make genenvs
```

> ⚠️ **Warning**: Never edit generated files directly. Always update source files:
> - SQL queries: `sql/postgres/queries/*.sql`
> - Protocol definitions: `schema/**/*.proto`
> - ABI files: `pkg/walletsdk/**/*.abi`

### Running Server

```bash
# Build and run
make start

# Or run directly
go run ./cmd/app start -c config.yaml

# Run webhooks server
make webhooks
```

### Development Tools

```bash
# Install all development tools
make install-dev-tools

# Watch for changes (hot reload)
make watch

# Format code
make fmt

# Run linter
make lint

# Generate README from config
make genreadme
```

---

## 📚 Resources

- 📖 **Documentation** — [docs.dv.net](https://docs.dv.net)
- 🔌 **API Reference** — See `api/` directory
- 🧾 **Swagger** — See `api/api.swagger.json`
- 💬 **Support** — [dv.net/support](https://dv.net/#support)
- 📱 **Telegram** — [@dv_net_support_bot](https://t.me/dv_net_support_bot)

---

## 🔧 Project-Specific Notes

### Blockchain Support

The project supports multiple blockchains:
- **EVM-based**: Ethereum, BSC, Polygon, Arbitrum, Optimism, Linea
- **UTXO-based**: Bitcoin, Litecoin, Bitcoin Cash, Dogecoin
- **Tron**: Native Tron blockchain

When adding new blockchain support:
1. Add configuration in `internal/config/`
2. Implement FSM in `internal/fsm/`
3. Add wallet SDK support in `pkg/walletsdk/`
4. Update blockchain constants in `internal/constants/`

### State Machines

Blockchain processing uses finite state machines (FSM) located in `internal/fsm/`. Each blockchain has its own FSM implementation.

### Database

- Uses PostgreSQL with `pgx/v5` driver
- Migrations in `sql/postgres/migrations/`
- Queries in `sql/postgres/queries/`
- Code generation via `sqlc` and `pgxgen`

### Protocol Buffers

- Definitions in `schema/processing/`
- Generated code in `api/processing/`
- Use `buf` for linting and generation

---

<div align="center">

**Thank you for contributing to DV Processing!** 🙏

*Your contributions make this project better for everyone.*

</div>

