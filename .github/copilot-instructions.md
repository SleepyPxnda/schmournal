# Copilot Instructions — Schmournal (Post-Refactor)

## Purpose
Schmournal is a terminal work journal written in Go with Bubble Tea.

The codebase has been refactored to a layered architecture. When making changes, preserve strict layer boundaries and keep business logic out of UI and infrastructure.

## Architecture Overview

### Layer mapping
- `internal/ui/tui/`:
  Presentation layer (view state, key handling, rendering, user interaction).
- `internal/application/usecase/` and `internal/application/dto/`:
  Application layer (orchestration, input validation, workflow coordination, DTO mapping).
- `internal/domain/model/`, `internal/domain/service/`, `internal/domain/repository/`:
  Domain layer (entities, business rules, domain services, repository interfaces).
- `internal/infrastructure/`:
  Infrastructure layer (filesystem persistence, config/state storage, clock/time provider implementations).
- `main.go`:
  Composition root (dependency wiring only).

### Dependency rule (must hold)
- Presentation -> Application
- Application -> Domain
- Infrastructure -> Domain (implements interfaces)
- Domain -> (nothing outward)

Do not introduce reverse dependencies.

## Current Data and Persistence Model

- Day records: one JSON file per day (`YYYY-MM-DD.json`) in the workspace storage directory.
- Workspace todos: `todos.json` per workspace (active + archived).
- App config: TOML in `~/.config/schmournal.config`.
- App runtime state: JSON in `~/.config/schmournal.state`.
- Exports: Markdown files in `~/.journal/exports/`.
- List view shortcuts include both weekly summary (`week_view`, default `v`) and stats overview (`stats_view`, default `s`).

Important: Active TODOs are workspace-global, not day-specific. Day records may persist a `today_done` snapshot of TODO trees completed and archived on that day for day-level review.

## Layer-Specific Coding Rules

### Presentation (`internal/ui/tui/`)
- Use application use cases for reads/writes; avoid direct repository calls.
- Keep business rules out of handlers/views.
- UI model remains a value type; handlers return `(tea.Model, tea.Cmd)`.
- Use UI adapter helpers for mapping between UI structs and application DTOs.
- Prefer key-to-action mapping helpers and routing seams for screen handlers to keep input interpretation separate from state mutations.
- Preserve existing UX behaviors (status auto-clear, todo interaction model, viewport behaviors).

### Application (`internal/application/usecase`)
- Orchestrate use cases; do not perform filesystem or framework-specific work.
- Validate inputs and return explicit errors.
- Use domain repository interfaces and domain services only.
- Keep DTO mappers centralized (`dto.go`, `state_dto.go`) and reused.
- Work-form submission orchestration (entry split/merge/edit persistence) belongs in `SubmitWorkFormUseCase`, not UI handlers.
- TODO archive and archive-clear flows belong in `ManageTodosUseCase`; UI should trigger commands and consume returned DTO state.

### Domain (`internal/domain`)
- Pure business logic only.
- No dependency on UI, config package, or infrastructure package.
- Repository contracts live in `internal/domain/repository`.
- Domain services (duration parsing/formatting, clock conversion, export generation, todo ops) are the source of business behavior.

### Infrastructure (`internal/infrastructure`)
- Implement domain repository interfaces and provider interfaces.
- Keep serialization/file/path concerns here.
- Avoid leaking infrastructure types across layer boundaries.

## Key Conventions to Keep

- `main.go` is the only composition root.
- Workspace switching must rebuild use case set via `UseCaseSetFactory`.
- Active workspace persistence goes through `StateRepository`.
- UI should avoid using legacy `journal.*` service helpers when equivalent domain/application service exists.
- Path handling uses `StorageManager` (`internal/infrastructure/persistence/json/storage.go`).

## Tests and Verification

Always run:
- `go test ./...`
- `go build -o schmournal.exe .`

Infrastructure changes should include integration-style repository tests using temp directories.

## Refactoring Guardrails

- Prefer small, surgical changes.
- Reuse existing helpers/mappers before introducing new ones.
- Avoid broad fallbacks or silent error handling.
- Do not bypass use cases by calling infrastructure directly from UI/application.
- If a change crosses layers, verify contracts and mappings in all touched layers.

## Mandatory Documentation Maintenance

When you change architecture, layering contracts, data flow, key conventions, or persistence behavior, you must update this file in the same change set.

Treat `.github/copilot-instructions.md` as a living architecture contract:
- Update outdated sections immediately.
- Add new conventions when introducing them.
- Remove rules that are no longer valid.
