# Synergy Monorepo

Polyglot Nx monorepo with Go and TypeScript apps.

## Monorepo Tools

- **Orchestration**: Nx 20.4.6
- **Package manager**: Yarn 4.7.0
- **Go workspace**: `go.work` at root

## Commands

```bash
yarn dev          # Run all apps in parallel (nx run-many --target=dev)
yarn serve        # Serve all apps in parallel
yarn lint         # Lint all apps
nx dev <app>      # Run a single app
```

## TypeScript Guidelines

These apply to all TypeScript apps in the monorepo.

### Strict Typing

- Strict mode enabled in all tsconfig files
- Never use `any` — use `unknown`, generics, or proper type narrowing instead
- Define explicit return types on exported functions

### Error Handling

- Handle runtime errors gracefully — no unhandled promise rejections or uncaught exceptions
- Use typed error handling (discriminated unions, Result patterns) over bare try/catch where appropriate
- Always provide meaningful error messages to the user

### Performance

- Performance is a feature — always target the most performant code with high readability
- Avoid unnecessary allocations, closures, and re-renders
- Prefer iterative approaches over recursive when dealing with large data
- Profile before optimizing — measure, don't guess

### Code Style

- Prefer early returns — avoid nested conditions
- Maximum cyclomatic complexity of 5 per function — break down into sub-functions
- Maximum file length: 500 lines — split into meaningful modules
- Reuse code — extract shared logic into utilities, avoid duplication
- Named exports over default exports
- File naming: lowercase kebab-case
- Variables/functions: camelCase
- Types/interfaces: PascalCase

### Formatting (Prettier)

- Double quotes
- 2-space indent
- Semicolons required
- No trailing commas
- 80 character line width
- Arrow parens avoided when possible

## App Structure

Each app lives in `apps/` with its own:
- `package.json` (dependencies scoped to the app)
- `project.json` (Nx targets: dev, serve, build, lint)
- `tsconfig.json` (app-specific TypeScript config, strict mode)

## Adding a New App

1. Create directory under `apps/`
2. Add `package.json` with scripts (`dev`, `build`, `serve`, `lint`)
3. Add `project.json` with Nx targets using `nx:run-commands`
4. Add `tsconfig.json` with `"strict": true`
5. For Go apps: add `go.mod` and register in root `go.work`
