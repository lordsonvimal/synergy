# Everwise Crest — Design Token Guidelines

## Why this system exists

Tokens solve two problems simultaneously: they prevent raw hex values from scattering across the codebase, and they make dark mode automatic. Every token in `tokens.css` has a paired dark value in `dark.css` — use the token and dark mode comes for free.

---

## Surfaces

Surfaces express **depth** — how nested something is in the layout. Elevation (whether something floats above the page) is expressed via shadow, not surface color.

| Token | Tailwind class | Use for |
|---|---|---|
| `surface-0` | `bg-surface-0` | Floating elements: modals, dropdowns, tooltips, command palettes. Always pair with a shadow (`shadow-md` or higher). |
| `surface-1` | `bg-surface-1` | Cards, panels, sidebars, headers, footers. The primary content surface. |
| `surface-2` | `bg-surface-2` | Nested sections within a card — participant rows, drill tiles, stat blocks inside a panel. |
| `surface-3` | `bg-surface-3` | Recessed / inset areas — code blocks, sunken inputs, read-only fields. |
| `surface-4` | `bg-surface-4` | Zebra rows, disabled backgrounds, hover states on flat list items. |
| `canvas` | `bg-canvas` | Page background only. Never use on any component. |

**Rules:**
- Never skip levels. A `surface-2` element must sit inside a `surface-1` container, not directly on `canvas`.
- Floating elements always get `surface-0` + a shadow token. Shadow communicates elevation; surface color communicates "this is a clean content surface."
- Hover states on flat buttons and list items use `hover:bg-surface-4` — not a custom color.

---

## Text

Text tokens are semantic roles, not visual intensity levels.

| Token | Tailwind class | Use for |
|---|---|---|
| `ink` | `text-ink` | Primary body text, headings, labels on interactive controls. |
| `ink-subtle` | `text-ink-subtle` | Secondary content — descriptions, metadata, helper text, nav items. |
| `ink-muted` | `text-ink-muted` | Tertiary content — timestamps, placeholders, captions, move numbers. |
| `ink-disabled` | `text-ink-disabled` | Disabled controls and non-interactive text. Never use on interactive elements. |

**Colored text rule:** For status-driven text (alerts, validation, availability) use the palette directly — `text-error`, `text-success`, `text-warning`, `text-info`, `text-primary`. No dedicated token needed.

---

## Borders

| Token | Tailwind class | Use for |
|---|---|---|
| `edge-subtle` | `border-edge-subtle` | Internal dividers within a card — hairline separators between rows. |
| `edge` | `border-edge` | Default card and panel borders. Most borders in the UI should use this. |
| `edge-strong` | `border-edge-strong` | Input borders (default state), focused ring bases, selected states. |
| `edge-emphasis` | `border-edge-emphasis` | High-contrast separators, active/pressed element outlines. |

**Tinted surface rule:** When a surface uses a palette background (e.g. `bg-primary-50`, `bg-teal-50`, `bg-error-subtle`), use `{color}-200` as the border — never override an edge token. Example:

```html
<!-- Correct: tinted callout card -->
<div class="bg-info-50 border border-info-200 rounded-md p-4">
  <p class="text-info">Your game is ready.</p>
</div>

<!-- Wrong: using edge on a tinted surface -->
<div class="bg-info-50 border border-edge rounded-md p-4">
```

**Dark mode note:** In dark mode, tinted surfaces should use the dark end of the palette (`bg-{color}-950` for background, `border-{color}-800` for border, `text-{color}-300` for text). The palette scales are inverted in `dark.css` so `bg-primary-50` will automatically render as a dark tint.

---

## Elevation

Elevation is shadow, not surface color. Use the existing Tailwind shadow utilities — the `--ec-shadow-*` custom properties mirror these values but should only be used in custom CSS when a Tailwind class is insufficient.

| Shadow | Use for |
|---|---|
| `shadow-sm` | Subtle lift — focused inputs, hovered cards |
| `shadow-md` | Dropdowns, date pickers, small popovers |
| `shadow-lg` | Modals, dialogs, drawer panels |
| `shadow-xl` | Command palettes, full overlays |

---

## Per-product overrides

Each product can override tokens in its own theme file (e.g. `apps/chess/theme.css`). The shared tokens are designed to work for all products without overrides — only reach for overrides when a product needs a meaningfully different personality.

```css
/* apps/chess/theme.css — example: subtle teal personality for ChessLeap */
@layer theme {
  /* Only override what genuinely differs for this product */
  --color-canvas: #E5EEF0;
}
```

Override the minimum. Any token can be overridden, but if you find yourself overriding more than 3–4 tokens, consider whether the design is using the palette (teal, violet, accent) correctly instead.

---

## Dark mode

Dark mode is automatic if you use tokens. Do not write any color that isn't a token — raw hex values will not adapt.

To force light or dark mode for testing: set `data-theme="light"` or `data-theme="dark"` on `<html>`. Without the attribute, the system follows `prefers-color-scheme`.

---

## Common mistakes

| Wrong | Right | Why |
|---|---|---|
| `bg-white` | `bg-surface-1` | White doesn't adapt to dark mode |
| `bg-gray-100` | `bg-surface-4` | Raw Tailwind palette bypasses the token system |
| `text-gray-500` | `text-ink-subtle` | Same reason |
| `border-gray-200` | `border-edge` | Same reason |
| `bg-surface-2` directly on `canvas` | `bg-surface-2` inside a `bg-surface-1` card | Depth levels must not be skipped |
| `bg-primary-50` with `border-edge` | `bg-primary-50` with `border-primary-200` | Tinted surfaces need palette borders |
