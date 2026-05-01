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

## UI/UX Rules

These apply to all frontend apps in the monorepo.

### Z-Index Layering

- Use a layered architecture with z-index values 1 through 10
- Each layer is a direct child of `<body>` using a semantic sectioning element (`<main>`, `<header>`, `<aside>`, `<dialog>`, etc.)
- Elements within a layer must never need their own z-index — position them using layout alone
- Layer order:
  1. Background/canvas
  2. Main content (`<main>`)
  3. Fixed headers/footers (`<header>`, `<footer>`)
  4. Side panels (`<aside>`)
  5. Overlays/backdrops
  6. Modals (`<dialog>`)
  7. Toasts/notifications
  8. Tooltips/popovers
  9. Loading screens
  10. Critical system alerts

### Layout

- Use flexbox or CSS grid for all layouts — never use floats, tables-for-layout, or absolute positioning for layout purposes
- Prefer CSS grid for 2D layouts, flexbox for 1D alignment
- Avoid fixed pixel dimensions — use relative units, min/max constraints, and intrinsic sizing

### Semantic HTML

- Use appropriate HTML elements for their intended purpose: `<nav>`, `<article>`, `<section>`, `<aside>`, `<figure>`, `<time>`, `<mark>`, `<details>`, etc.
- Headings must follow hierarchy (`h1` > `h2` > `h3`) — never skip levels
- Use `<button>` for actions, `<a>` for navigation — never the reverse
- Lists of items use `<ul>`/`<ol>`/`<dl>`
- Forms use `<fieldset>`, `<legend>`, and `<label>` elements

### Accessibility

- All interactive elements must be keyboard accessible — focusable, activatable, and have visible focus indicators
- Use ARIA attributes only when native HTML semantics are insufficient
- All images/icons must have `alt` text or `aria-label` (decorative images use `alt=""`)
- Color must never be the sole indicator of state — pair with icons, text, or patterns
- Touch targets: minimum 44x44px
- Focus order must follow visual reading order
- Support `prefers-reduced-motion` for animations
- All form inputs must have associated labels

### DOM Attributes for Testing

- Add `data-testid` to interactive elements and key content containers
- Add `id` to landmark elements, form controls, and anchor targets
- Use consistent naming: `data-testid="component-name-element"` (e.g., `data-testid="chat-area-message-list"`)

### Documentation in Code

- Add comments when the code is not self-explanatory: complex algorithms, non-obvious side effects, browser workarounds, and performance trade-offs
- For complex UI interactions, document the expected behavior and edge cases
- Keep comments concise — one line explaining WHY, not WHAT

### DOM Structure

- Avoid nesting beyond 4-5 levels deep — flatten with grid/flex or extract into components
- Prefer composition over deep nesting
- Each component should have a single wrapping element with a clear role

## Architecture & Design Patterns

These principles apply to all apps in the monorepo.

### Scalability

- Separate concerns into distinct layers: data, logic, and presentation
- Use feature-based directory structure — group by capability, not by file type
- Design interfaces/contracts between modules so teams/features can grow independently
- Prefer horizontal scaling (more modules) over vertical (deeper hierarchies)

### Replaceability

- Depend on abstractions, not implementations — wrap third-party libraries behind app-owned interfaces
- Use dependency injection or factory patterns so implementations can be swapped without touching consumers
- Isolate side effects (network, storage, timers) behind service boundaries
- Never import third-party types into core business logic — map at the boundary

### Performance

- Measure first — use browser DevTools, Lighthouse, and profiling before optimizing
- Lazy-load non-critical paths (dynamic imports, code-splitting by route)
- Minimize main-thread work — offload heavy computation to workers when appropriate
- Batch DOM operations, avoid layout thrashing
- Use efficient data structures — Map/Set over Object/Array for lookups
- Debounce/throttle high-frequency events (scroll, resize, input)

### Maintainability

- Write code for the reader, not the writer — favour clarity over cleverness
- One concern per file, one responsibility per function
- Keep the dependency graph acyclic and shallow
- Refactor toward patterns only when duplication becomes a maintenance burden (rule of three)
- Tests must be fast, deterministic, and independent

### SOLID Principles

- **Single Responsibility**: Each module/class/function does one thing. If you can't name it without "and", split it.
- **Open/Closed**: Extend behavior through composition and hooks, not by modifying existing code
- **Liskov Substitution**: Subtypes must be usable wherever their base type is expected without surprises
- **Interface Segregation**: Prefer multiple small, focused interfaces over one large one
- **Dependency Inversion**: High-level modules depend on abstractions. Low-level modules implement them.

### Requirement Clarity

- When requirements are ambiguous, ask clarifying questions before implementing
- Document assumptions made during implementation
- If a requirement has multiple valid interpretations, propose the options with trade-offs

### Security

- Never trust user input — validate and sanitize at every system boundary
- No secrets in code, config files, or client bundles — use environment variables
- Apply principle of least privilege for file access, network calls, and permissions
- Escape output contextually (HTML, SQL, shell, URL)
- Use parameterized queries — never string-concatenate user input into queries
- Set security headers: CSP, X-Frame-Options, X-Content-Type-Options
- All forms must include CSRF protection
- Authentication tokens must be stored securely (httpOnly cookies preferred over localStorage)

### OWASP Top 10

- Injection: parameterized queries, input validation, contextual output encoding
- Broken Authentication: strong session management, no credentials in URLs
- Sensitive Data Exposure: encrypt at rest and in transit, minimize data retention
- XML External Entities: disable DTD processing, use JSON
- Broken Access Control: deny by default, validate permissions server-side
- Security Misconfiguration: minimal installs, no default credentials, disable debug in production
- XSS: encode output, use CSP, avoid innerHTML with user data
- Insecure Deserialization: validate and type-check all deserialized data
- Using Components with Known Vulnerabilities: audit dependencies, keep them updated
- Insufficient Logging: log security events, never log secrets

### Local-First Development

- Apps must run fully offline or on local network without external service dependencies
- Use browser APIs (IndexedDB, Cache API, Web Crypto) before reaching for cloud services
- Design for sync-later when network features are needed
- Development setup must work without internet after initial dependency install

### Dependency Management

- Prefer native browser APIs and platform capabilities over third-party libraries
- Before adding a dependency, evaluate: can this be done in <50 lines of app code?
- When a dependency is needed, prefer small, focused packages over large frameworks
- Audit bundle size impact before adding any client-side dependency
- Pin dependency versions — no floating ranges in production

### Regression Prevention

- Follow existing patterns in the codebase — consistency trumps personal preference
- New code must not break existing tests or functionality
- When changing shared code, verify all consumers still work
- If no established pattern exists for an app, propose an architecture with rationale before implementing
- Feature additions must include tests covering the happy path and key edge cases

### Additional Best Practices

- **Idempotency**: Operations that can be retried (API handlers, event processors) must produce the same result on repeated execution
- **Graceful Degradation**: Features must degrade gracefully when optional dependencies (speech API, battery API, network) are unavailable
- **Configuration over Code**: Behaviour that varies by environment belongs in config, not conditionals
- **Immutability by Default**: Prefer immutable data structures and pure functions; mutate only when performance requires it
- **Observability**: Emit structured logs with context (request ID, user action, timing) at meaningful boundaries
- **Fail Fast**: Validate preconditions at function entry and throw early with descriptive messages rather than propagating invalid state

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
