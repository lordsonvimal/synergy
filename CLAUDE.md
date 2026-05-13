# Synergy Monorepo

## Brand & Design System — Everwise Crest

All frontend apps share the Everwise Crest design system. The personality is **trustworthy, professional, and clean** — inspired by Google Workspace, Microsoft 365, and Apple's enterprise products. No playful/whimsical aesthetics.

**Before writing any UI code, read `packages/theme/tokens.css` for the full token definitions.** Never use raw hex values or default Tailwind palette (slate, gray, blue-500, etc.) — only use the token classes defined in the theme package.

## UI/UX Rules

- **Spacing rhythm**: 8px grid (0.5rem increments) — all spacing must be multiples of 8
- **Z-Index**: layers 1–10, each a direct child of `<body>` using a semantic element — elements within a layer never need their own z-index. Order: 1 canvas, 2 main, 3 headers/footers, 4 side panels, 5 overlays, 6 modals, 7 toasts, 8 tooltips, 9 loading, 10 critical alerts.
- **Clickable elements**: every button, link, and interactive element must have `cursor-pointer` and a visible hover state (background, border, or color change). No exceptions — includes tab switchers, icon buttons, and form submit buttons.

## Architecture & Design Patterns

### Local-First Development

- Apps must run fully offline or on local network without external service dependencies
- Use browser APIs (IndexedDB, Cache API, Web Crypto) before reaching for cloud services
- Design for sync-later when network features are needed
- Development setup must work without internet after initial dependency install

### Leak Prevention

- Every `setInterval`, `setTimeout`, `addEventListener`, `observe()`, or subscription must have a corresponding cleanup in `onCleanup` (SolidJS), `useEffect` return (React), or equivalent lifecycle hook
- Buffers and accumulators that grow over time must be bounded — define a max size and trim oldest entries when exceeded
- WebSocket/SSE connections must be explicitly closed on component unmount
- Observers (ResizeObserver, MutationObserver, IntersectionObserver) must call `disconnect()` in cleanup
- Timers that fire repeatedly must be cleared even if the component is expected to live for the app's lifetime

### Regression Prevention

- If no established pattern exists for an app, propose an architecture with rationale before implementing
