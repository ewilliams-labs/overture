# Overture Project Standards

## Shared Rules
- Use `github.com/ewilliams-labs/overture` as the root for all logic references.
- Prefer functional programming patterns over heavy class inheritance.
- **Atomic Implementation:** When generating logic, always generate the corresponding unit test file immediately after.

## Backend (Go)
- **Architecture:** Hexagonal / Ports & Adapters.
  - Domain: `backend/internal/core/domain` (Logic-only entities)
  - Ports: `backend/internal/core/ports` (Interfaces)
  - Services: `backend/internal/core/services` (Orchestration/Business logic)
  - Adapters: `backend/internal/adapters/` (HTTP, Spotify, Database)
- **Module Path:** `github.com/ewilliams-labs/overture/backend`
- **Testing:** Table-Driven tests only. Use `httptest` for REST adapters.
- **Context:** Always pass `ctx` (context.Context) as the first argument in IO-bound functions.
- **Errors:** Wrap errors with context: `fmt.Errorf("description: %w", err)`.
- **Naming:** Use short, idiomatic Go names (e.g., `ctx`, `err`, `svc`, `h` for handler).

## Frontend (React/TypeScript)
- **Architecture:** Feature-Sliced Design (FSD).
- **Styling:** Use Tailwind CSS.
- **Components:** Use Functional Components with Hooks (no Classes).
- **State:** Use TanStack Query (React Query) for server state.
- **Naming:** Use PascalCase for components and camelCase for hooks/utilities.
- **Props:** Use TypeScript `interface` for Prop definitions.