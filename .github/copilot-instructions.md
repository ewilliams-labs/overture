# Overture Project Standards

## Shared Rules
- Use `github.com/ewilliams-labs/overture` as the root for all logic references.
- Prefer functional programming patterns over heavy class inheritance.

## Backend (Go)
- **Architecture:** Hexagonal / Ports & Adapters.
- **Testing:** Table-Driven tests only.
- **Context:** Always pass `ctx` as the first argument in IO-bound functions.

## Frontend (React/TypeScript)
- **Styling:** Use Tailwind CSS.
- **Components:** Use Functional Components with Hooks (no Classes).
- **State:** Use TanStack Query (React Query) for server state.
- **Naming:** Use PascalCase for components and camelCase for hooks/utilities.
- **Props:** Use TypeScript `interface` for Prop definitions.