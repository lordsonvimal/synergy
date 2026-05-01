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

## Brand & Design System — Everwise Crest

All frontend apps share the Everwise Crest design system. The personality is **trustworthy, professional, and clean** — inspired by Google Workspace, Microsoft 365, and Apple's enterprise products. No playful/whimsical aesthetics. Every app must feel like it belongs to the same product family.

### Shared Theme Package

Located at `packages/theme/`. Every app imports these tokens:

```css
@import "@synergy/theme/tokens.css";
@import "@synergy/theme/dark.css";
```

### Color Tokens (Tailwind v4 Extended)

**Only these tokens may be used. No raw hex values, no default Tailwind color palette (slate, gray, blue-500, etc.) in application code.**

#### Primary (Navy Blue) — brand backbone, buttons, links, focus rings

| Token | Light | Dark |
|-------|-------|------|
| `primary` | `#1D4ED8` | `#60A5FA` |
| `primary-hover` | `#1E40AF` | `#93C5FD` |
| `primary-subtle` | `#EFF6FF` | `#172554` |
| `primary-50` through `primary-950` | Full scale available |

#### Accent (Gold) — achievements, badges, secondary CTAs, highlights

| Token | Light | Dark |
|-------|-------|------|
| `accent` | `#B45309` | `#FBBF24` |
| `accent-hover` | `#92400E` | `#FCD34D` |
| `accent-subtle` | `#FFFBEB` | `#422006` |
| `accent-50` through `accent-950` | Full scale available |

#### Semantic Status Colors

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `success` | `#15803D` | `#4ADE80` | Confirmations, connected, saved |
| `warning` | `#A16207` | `#FACC15` | Pending, caution, long-running ops |
| `error` | `#B91C1C` | `#F87171` | Failures, destructive actions |
| `info` | `#0369A1` | `#38BDF8` | Tips, informational banners |

Each semantic color has `-hover`, `-subtle`, and full 50–950 scale variants.

#### Surfaces & Neutrals

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `canvas` | `#FAFBFD` | `#0B1120` | Page background |
| `surface` | `#FFFFFF` | `#141D2E` | Cards, panels, modals |
| `surface-raised` | `#FFFFFF` | `#1C2841` | Dropdowns, popovers |
| `muted` | `#F1F5F9` | `#1E293B` | Disabled, zebra rows |

#### Borders

| Token | Light | Dark | Utility Class | Usage |
|-------|-------|------|---------------|-------|
| `edge` | `#E2E8F0` | `#2D3B50` | `border-edge` | Dividers, card borders |
| `edge-strong` | `#CBD5E1` | `#475569` | `border-edge-strong` | Inputs, emphasized borders |

#### Text

| Token | Light | Dark | Utility Class | Usage |
|-------|-------|------|---------------|-------|
| `ink` | `#0F172A` | `#F1F5F9` | `text-ink` | Headlines, body |
| `ink-secondary` | `#475569` | `#94A3B8` | `text-ink-secondary` | Labels, descriptions |
| `ink-dim` | `#94A3B8` | `#64748B` | `text-ink-dim` | Placeholders, timestamps |
| `on-primary` | `#FFFFFF` | `#FFFFFF` | `text-on-primary` | Text on primary buttons |
| `on-accent` | `#FFFFFF` | `#422006` | `text-on-accent` | Text on gold backgrounds |

### Typography

| Token | Value |
|-------|-------|
| `font-sans` | Inter, system-ui, -apple-system, sans-serif |
| `font-mono` | JetBrains Mono, Fira Code, SF Mono, Consolas, monospace |
| `font-heading` | Inter, system-ui, -apple-system, sans-serif |

- Use Inter as the sole typeface (headings and body) — load weights 400, 500, 600, 700
- Font sizes via `clamp()` for fluid scaling: body `clamp(0.875rem, 2vw, 1rem)`, headings scale proportionally
- Line height: 1.5 for body, 1.2 for headings
- Letter spacing: -0.01em for headings, normal for body

### Border Radius — Element Mapping

Every UI element has a fixed radius assignment. Never mix radius sizes within the same visual level.

| Token | Value | Applies To |
|-------|-------|-----------|
| `radius-none` | `0` | Data tables, full-bleed sections, dividers |
| `radius-xs` | `0.125rem` (2px) | Badges, inline tags, status dots |
| `radius-sm` | `0.25rem` (4px) | Checkboxes, small chips, toggle tracks |
| `radius-md` | `0.375rem` (6px) | Buttons, inputs, selects, textareas, tabs |
| `radius-lg` | `0.5rem` (8px) | Cards, dropdowns, popovers, toasts, tooltips |
| `radius-xl` | `0.75rem` (12px) | Modals, dialogs, bottom sheets, side panels |
| `radius-2xl` | `1rem` (16px) | Large panels, page-level containers, hero sections |
| `radius-full` | `9999px` | Avatars, pills, circular icon buttons, progress bars |

Rules:
- A child element must never have a larger radius than its parent
- Nested elements reduce radius by one step (e.g., card `lg` → button inside card `md`)
- Adjacent elements at the same level use the same radius for visual consistency
- Interactive elements (buttons, inputs) always use `radius-md` regardless of context
- Never mix sharp corners and rounded corners on elements at the same hierarchy level

### Dark Mode Implementation

- All apps support dark mode from day one
- Use `[data-theme="dark"]` on `<html>` for user-toggled override
- Respect `prefers-color-scheme: dark` as system default
- Store user preference in `localStorage` key `theme`
- Apply theme class before first paint to prevent FOUC (inline script in `<head>`)
- Test every component in both themes — shadows, borders, and subtle backgrounds often break

### Utility Class Reference

Token names in `@theme` map directly to Tailwind utility classes. Here's how they resolve:

```html
<!-- Buttons -->
<button class="bg-primary text-on-primary rounded-md shadow-sm hover:bg-primary-hover">
  Save Changes
</button>
<button class="bg-surface text-ink border border-edge rounded-md hover:bg-muted">
  Cancel
</button>

<!-- Cards -->
<div class="bg-surface border border-edge rounded-lg shadow-md p-6">
  <h2 class="text-ink font-semibold">Card Title</h2>
  <p class="text-ink-secondary">Description text</p>
  <span class="text-ink-dim text-sm">Updated 2 hours ago</span>
</div>

<!-- Status badges -->
<span class="bg-success-subtle text-success rounded-xs px-2 py-0.5">Active</span>
<span class="bg-error-subtle text-error rounded-xs px-2 py-0.5">Failed</span>
<span class="bg-warning-subtle text-warning rounded-xs px-2 py-0.5">Pending</span>

<!-- Inputs -->
<input class="bg-surface border border-edge-strong rounded-md text-ink placeholder:text-ink-dim" />

<!-- Modals -->
<dialog class="bg-surface rounded-xl shadow-xl border border-edge">
  <div class="text-ink">Modal content</div>
</dialog>

<!-- Accent usage -->
<div class="bg-accent-subtle border border-accent-200 rounded-lg">
  <span class="text-accent font-medium">Achievement unlocked</span>
</div>
```

Full class mapping:

| Property | Classes available |
|----------|-----------------|
| Background | `bg-primary`, `bg-primary-hover`, `bg-primary-subtle`, `bg-primary-50`…`bg-primary-950`, `bg-accent`, `bg-success`, `bg-warning`, `bg-error`, `bg-info` (+ `-hover`, `-subtle`, `-50`…`-950`), `bg-canvas`, `bg-surface`, `bg-surface-raised`, `bg-muted` |
| Text | `text-ink`, `text-ink-secondary`, `text-ink-dim`, `text-on-primary`, `text-on-accent`, `text-primary`, `text-accent`, `text-success`, `text-warning`, `text-error`, `text-info` |
| Border | `border-edge`, `border-edge-strong`, `border-primary`, `border-accent`, `border-success`, `border-warning`, `border-error`, `border-info` |
| Ring | `ring-primary`, `ring-edge`, `ring-error` |
| Radius | `rounded-none`, `rounded-xs`, `rounded-sm`, `rounded-md`, `rounded-lg`, `rounded-xl`, `rounded-2xl`, `rounded-full` |
| Shadow | `shadow-sm`, `shadow-md`, `shadow-lg`, `shadow-xl` |

### Usage Rules

- **Never use raw hex/rgb values** in component code — always use token classes (`bg-primary`, `text-ink`, etc.)
- **Never use default Tailwind palette** (slate-500, gray-200, blue-600, etc.) — only use the defined token names
- **Never hardcode light/dark variants** in components — the token system handles theme switching automatically
- When a color need is not covered by existing tokens, propose a new token addition rather than using a one-off value

## UI/UX Rules

These apply to all frontend apps in the monorepo.

### Trustworthy SaaS Layout Patterns

Follow the layout conventions of Google Workspace, Microsoft 365, and Apple's enterprise products:

#### App Shell Structure

- **Desktop (≥1024px)**: Persistent left sidebar (240–280px) + top bar (56–64px) + main content area
- **Tablet (768–1023px)**: Collapsible sidebar (overlay or rail with icons only) + top bar + content
- **Mobile (<768px)**: Bottom navigation bar (max 5 items) + top bar (app title + actions) + full-width content
- The app shell (sidebar + top bar) remains stable during navigation — only the content area changes
- Top bar contains: logo/app name (left), global search (center on desktop), user avatar + notifications (right)

#### Navigation

- **Primary navigation** lives in the sidebar (desktop) or bottom bar (mobile)
- **Secondary navigation** uses tabs within the content area or breadcrumbs
- Navigation items: icon + label (desktop), icon-only with tooltip (collapsed), icon + label (mobile bottom bar)
- Active state: subtle background fill + primary color indicator (left border on sidebar, bottom border on tabs)
- Maximum 7±2 primary navigation items — group overflow into a "More" menu
- Breadcrumbs for hierarchical content deeper than 2 levels

#### Content Layout

- **Maximum content width**: 1200px for reading content, 1440px for data-heavy dashboards — centered with auto margins
- **Consistent spacing rhythm**: use 8px grid (0.5rem increments) — all spacing must be multiples of 8
- **Card-based UI** for related content groups — consistent padding (16–24px), subtle border, rounded corners (8px)
- **Data tables**: sticky headers, horizontal scroll on mobile, row hover states, sortable columns with clear indicators
- **Empty states**: always provide illustration/icon + descriptive message + primary action CTA
- **Loading skeletons**: match the shape of content being loaded — never use a single generic spinner for page content

#### Visual Hierarchy

- Use font weight (500, 600, 700) and size to establish hierarchy — not color alone
- Maximum 3 levels of visual emphasis per screen: primary action, secondary content, tertiary/metadata
- One primary CTA per viewport — additional actions use secondary or ghost button styles
- White space is a feature — generous padding between sections (32–48px), between cards (16–24px)

#### Density

- **Default density**: comfortable spacing for education/consumer users (44px min row height in tables, 16px padding in inputs)
- **Compact density**: available as user preference for power users on desktop (36px row height, 12px padding)
- Never use compact density on mobile

#### Page Patterns

- **List pages**: filters/search at top, bulk actions contextually shown on selection, pagination or infinite scroll at bottom
- **Detail pages**: key info summary at top (hero card), related data in tabbed sections below
- **Form pages**: single-column on mobile, two-column on desktop for related field groups, sticky submit bar at bottom
- **Dashboard pages**: key metrics in cards at top, charts/graphs in grid below, activity feed in sidebar (desktop) or below (mobile)
- **Settings pages**: grouped sections with clear headings, toggle/select inputs, save confirmation per section or single page-level save

#### Interaction Patterns

- **Hover states**: subtle background change (muted token) — never change text color on hover alone
- **Active/pressed**: slightly darker than hover
- **Focus rings**: 2px solid ring using `primary` color with 2px offset — visible in both themes
- **Transitions**: 150ms for micro-interactions (hover, focus), 200ms for reveals (dropdowns), 300ms for page-level (modals, sidebars)
- **Disabled elements**: 50% opacity + `not-allowed` cursor — never remove from DOM to maintain layout stability

#### Responsive Behavior Summary

| Element | Mobile (<768px) | Tablet (768–1023px) | Desktop (≥1024px) |
|---------|-----------------|---------------------|-------------------|
| Navigation | Bottom bar | Collapsible rail | Persistent sidebar |
| Content width | Full bleed | Padded (16px) | Max-width centered |
| Cards | Stacked full-width | 2-column grid | 3–4 column grid |
| Tables | Card-list transform | Horizontal scroll | Full table |
| Modals | Full-screen sheet | Centered 600px | Centered 600px |
| Actions | FAB or bottom sheet | Inline buttons | Inline buttons |

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
- Follow the trustworthy SaaS layout patterns defined in "Brand & Design System" section above
- Use 8px spacing grid — all margins, padding, and gaps must be multiples of 0.5rem

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

### Responsive Design (Mobile First)

- Design for mobile viewports first, then progressively enhance for larger screens
- Use `min-width` media queries to add complexity at wider breakpoints — never start with desktop and strip down
- Breakpoints: 320px (small mobile), 375px (mobile), 768px (tablet), 1024px (desktop), 1440px (large desktop)
- Touch interactions are the default — hover states are enhancements, not requirements
- Navigation must be thumb-reachable on mobile — place primary actions in the bottom half of the screen
- Test layouts at every breakpoint — no horizontal scrolling, no content overflow, no truncated text without affordance
- Images and media must be responsive — use `srcset`, `picture`, or CSS `object-fit`
- Typography must scale fluidly — use `clamp()` for font sizes between mobile and desktop

### Theming

- All apps must use the shared Everwise Crest design tokens from `packages/theme/`
- Import `@synergy/theme/tokens.css` and `@synergy/theme/dark.css` in every app's root CSS
- Only use token classes — never raw hex values or default Tailwind palette
- Ensure all theme combinations meet WCAG 2.1 AA contrast ratios (4.5:1 for text, 3:1 for UI elements)
- Theme switching must be instantaneous — no flash of unstyled content (FOUC)
- Respect `prefers-color-scheme` system preference as the default, with user override stored locally
- Test every component in both light and dark themes — shadows, borders, and subtle backgrounds often break in dark mode
- See "Brand & Design System" section above for complete token reference

### Nielsen Norman UX Principles

- **Visibility of System Status**: Always keep users informed about what is happening through appropriate feedback within reasonable time
- **Match Between System and Real World**: Use language, concepts, and conventions familiar to the user — no developer jargon in UI
- **User Control and Freedom**: Provide clear "emergency exits" — undo, cancel, back navigation — so users never feel trapped
- **Consistency and Standards**: Follow platform conventions and maintain internal consistency across all apps in the monorepo
- **Error Prevention**: Design to prevent errors before they occur — use constraints, confirmations for destructive actions, and smart defaults
- **Recognition Rather Than Recall**: Make options, actions, and information visible — minimize memory load on the user
- **Flexibility and Efficiency of Use**: Support both novice and expert users — provide shortcuts and accelerators without cluttering the interface
- **Aesthetic and Minimalist Design**: Every element must earn its place — remove visual noise, prioritize content hierarchy
- **Help Users Recognize, Diagnose, and Recover from Errors**: Error messages in plain language, indicate the problem precisely, and suggest a constructive solution
- **Help and Documentation**: Provide contextual help where needed — tooltips, inline hints, and searchable documentation

### User Notifications and Feedback

- Notify users for all long-running operations — show progress indicators (determinate when duration is known, indeterminate otherwise)
- Operations exceeding 1 second must show a loading state; operations exceeding 5 seconds must show progress or estimated time
- Success confirmations for user-initiated actions that have no immediate visible result (e.g., saving, sending, background sync)
- All failures and errors must be communicated in user-friendly language — never expose stack traces, error codes, or technical details raw
- Error notifications must include: (1) what went wrong in plain language, (2) a suggested action to resolve the issue
- Provide actionable recovery options directly in the error UI — retry buttons, alternative paths, or links to relevant settings
- Use appropriate notification patterns: inline for field-level errors, toast for transient feedback, banner for persistent system states, modal for blocking errors requiring immediate attention
- Notifications must be accessible — use ARIA live regions, ensure screen readers announce them, and never rely solely on color

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

### Leak Prevention

Proactively prevent resource leaks at authoring time — do not rely on testing or user reports to catch them.

- Every `setInterval`, `setTimeout`, `addEventListener`, `observe()`, or subscription must have a corresponding cleanup in `onCleanup` (SolidJS), `useEffect` return (React), or equivalent lifecycle hook
- Event handler arrays and callback registries must support removal — if a consumer registers a handler, provide an unregister mechanism tied to component lifecycle
- Buffers and accumulators (strings, arrays) that grow over time must be bounded — define a max size and trim/discard oldest entries when exceeded
- WebSocket/SSE connections must be explicitly closed on component unmount and on navigation away
- Observers (ResizeObserver, MutationObserver, IntersectionObserver) must call `disconnect()` in cleanup
- Timers that fire repeatedly must be cleared even if the component is expected to live for the app's lifetime — the app may remount (HMR, navigation)

### Silent Failure Handling

Anticipate and handle failures that would otherwise go unnoticed by the user.

- Before implementing, identify operations that can silently fail: network calls that return no response, APIs that degrade (e.g., Web Speech API stopping without error), background processes that exit, WebSocket messages that never arrive
- If a silent failure is possible, implement detection (timeouts, health checks, heartbeats) and surface it to the user via the appropriate notification pattern (toast, inline status, banner)
- When a silent failure requires a design decision to resolve (e.g., auto-retry vs. manual retry, fallback behavior, degraded mode), clarify with the user which approach to take — present the options with trade-offs and a recommendation
- Never swallow errors in catch blocks without either logging or notifying — if an error is truly ignorable, add a comment explaining why

### Documenting Changes

- Add inline comments for complex logic, non-obvious side effects, workarounds, and performance trade-offs — one line explaining WHY
- Architectural or structural changes (new layers, new patterns, new data flows) must be documented in the relevant `CLAUDE.md` or project-level docs so future work follows the same pattern
- When introducing a new pattern (e.g., DOM layer architecture, portal-based overlays, server-sent state), document: what the pattern is, when to use it, and how to add a new instance

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
