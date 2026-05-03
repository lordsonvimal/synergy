# Tether File Explorer — Requirements

**Feature**: File Explorer Sidebar
**Phase**: 8
**Date**: 2026-05-03
**Author**: Lordson Vimal

---

## 1. Overview

A collapsible file explorer sidebar that shows the directory contents relative to the active terminal's current working directory. Enables browsing, navigating, previewing files, and performing file operations — all without typing shell commands.

### 1.1 Problem Statement

On mobile (and even desktop), navigating the filesystem via terminal commands (`ls`, `cd`, `cat`) is slow and error-prone. Users need a visual way to browse files, preview content, and perform common file operations while keeping the terminal available for commands.

### 1.2 Solution

A sidebar panel that:
1. Reads the active terminal's CWD via `tmux display-message -p '#{pane_current_path}'`
2. Serves directory listings and file content through new server endpoints
3. Renders a navigable folder tree with file preview, search, and context actions
4. Operates independently from the terminal — browsing doesn't pollute terminal history

---

## 2. Architecture

### 2.1 Server Endpoints

New Express routes on the relay server (port 5100):

| Method | Endpoint | Request | Response | Purpose |
|--------|----------|---------|----------|---------|
| GET | `/api/cwd/:tabId` | — | `{ path: string }` | Get CWD of a tmux session |
| GET | `/api/files` | `?dir=<path>` | `FileEntry[]` | List directory contents |
| GET | `/api/file` | `?path=<path>` | File content (streamed) | Serve file for preview/download |
| GET | `/api/files/search` | `?dir=<path>&q=<query>&depth=<n>` | `FileEntry[]` | Recursive file search |
| POST | `/api/files/rename` | `{ path, newName }` | `{ success, newPath }` | Rename file or folder |
| POST | `/api/files/delete` | `{ path }` | `{ success }` | Delete file or folder |
| POST | `/api/files/create` | `{ path, type: "file" \| "folder" }` | `{ success }` | Create new file or folder |
| GET | `/api/files/info` | `?path=<path>` | `FileInfo` | File/folder metadata |

#### FileEntry Type

```typescript
interface FileEntry {
  name: string;
  path: string;
  type: "file" | "folder" | "symlink";
  size: number;
  modified: string; // ISO date
  permissions: string; // e.g. "rwxr-xr-x"
}
```

#### FileInfo Type

```typescript
interface FileInfo {
  name: string;
  path: string;
  type: "file" | "folder" | "symlink";
  size: number;
  modified: string;
  created: string;
  permissions: string;
  owner: string;
  group: string;
  isHidden: boolean;
}
```

### 2.2 Security

- All file endpoints must validate that the requested path is within the user's home directory or a configurable allowed root
- Path traversal attacks must be prevented — resolve and validate absolute paths
- Symlinks must be resolved and validated before serving
- Delete operations require confirmation on the client side
- Auth token required on all endpoints (same shared secret as WebSocket)

---

## 3. Functional Requirements

### 3.1 Sidebar Panel

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-01 | Collapsible sidebar panel, slide-in from left (same animation pattern as settings panel) | Must |
| FE-02 | Toggle via a folder icon button in the StatusBar | Must |
| FE-03 | On open, fetch CWD of the active terminal's tmux session and list its contents | Must |
| FE-04 | Independent navigation — browsing folders in the sidebar does not `cd` in the terminal | Must |
| FE-05 | "Open in terminal" action — sends `cd <path>` to the active terminal when explicitly chosen | Must |
| FE-06 | Re-fetch CWD when the sidebar is opened (not polling) | Must |
| FE-07 | Re-fetch CWD when the active tab changes while the sidebar is open | Must |
| FE-08 | Breadcrumb bar at the top — shows current sidebar path, each segment is clickable to jump up | Must |
| FE-09 | Back button or swipe-right to navigate to parent directory | Must |

### 3.2 Folder Tree

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-10 | Display files and folders with appropriate icons (folder, file, symlink) | Must |
| FE-11 | Show file size and last modified date as subtle secondary text | Must |
| FE-12 | Folders listed first, then files (default sort) | Must |
| FE-13 | Sort options: by name (default), by modified date, by size | Should |
| FE-14 | Hidden files toggle — show/hide dotfiles via a toggle in the sidebar header | Must |
| FE-15 | Tap a folder to navigate into it (in the sidebar) | Must |
| FE-16 | Tap a file to open preview in a new tab | Must |
| FE-17 | Loading skeleton while directory contents are being fetched | Must |
| FE-18 | Empty state for empty directories | Must |
| FE-19 | Error state if directory read fails (permission denied, etc.) | Must |

### 3.3 File Preview

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-20 | Open file preview in a new pane tab (non-terminal tab type); preview tabs are persistent across reloads (saved in localStorage with terminal tabs) | Must |
| FE-21 | Preview tab shows filename in the tab bar, distinguishable from terminal tabs (file icon prefix) | Must |
| FE-22 | Code/text files: syntax-highlighted view using highlight.js (already a dependency) | Must |
| FE-23 | Images (jpg, png, gif, svg, webp): rendered via `<img>` tag | Must |
| FE-24 | Markdown files: rendered to HTML using `marked` (already a dependency) | Must |
| FE-25 | PDF files: rendered via `<iframe>` on desktop/Android; download link fallback on iOS | Should |
| FE-26 | Video (mp4, webm): rendered via `<video>` tag | Should |
| FE-27 | Audio (mp3, wav, ogg): rendered via `<audio>` tag | Should |
| FE-28 | Binary/unknown files: show file info card (name, size, type, modified) + download button | Must |
| FE-29 | Files > 500 MB: fall back to info card + download link (no preview). Text files > 1 MB: show first 5000 lines with a "File truncated" notice | Must |
| FE-30 | Line numbers in code preview | Should |
| FE-31 | Word wrap toggle in code preview | Should |

#### File Type Detection

| Category | Extensions | Render Method |
|----------|-----------|---------------|
| Code | .ts, .tsx, .js, .jsx, .py, .go, .rs, .java, .c, .cpp, .h, .css, .scss, .html, .xml, .json, .yaml, .yml, .toml, .sh, .bash, .zsh, .sql, .graphql, .proto | highlight.js `<pre><code>` |
| Markdown | .md, .mdx | `marked` → HTML |
| Image | .jpg, .jpeg, .png, .gif, .svg, .webp, .ico, .bmp | `<img>` |
| PDF | .pdf | `<iframe>` / download |
| Video | .mp4, .webm, .mov | `<video>` |
| Audio | .mp3, .wav, .ogg, .m4a | `<audio>` |
| Plain text | .txt, .log, .env, .gitignore, .editorconfig, LICENSE, Makefile, Dockerfile | `<pre>` (no highlighting) |
| Binary | everything else | Info card + download |

#### Mobile Compatibility

| Format | iOS Safari | Android Chrome | Desktop |
|--------|-----------|---------------|---------|
| Code/text | Yes | Yes | Yes |
| Images | Yes | Yes | Yes |
| Markdown | Yes | Yes | Yes |
| PDF | Download link (iframe unreliable) | Yes (iframe) | Yes (iframe) |
| Video | Yes (native player) | Yes (native player) | Yes |
| Audio | Yes | Yes | Yes |

### 3.4 File Search

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-40 | Search input in the sidebar header — filters visible directory contents in real-time | Must |
| FE-41 | Deep search — recursive search from current sidebar directory, depth-limited (default 5 levels) | Must |
| FE-42 | Search results show relative path from the search root | Must |
| FE-43 | Tap a search result to open preview (file) or navigate (folder) | Must |
| FE-44 | Integrate with GlobalSearch — add a "Files" section to results; `/` prefix triggers files-only mode | Must |
| FE-45 | GlobalSearch file results use the active terminal's CWD as the search root | Must |
| FE-46 | Search debounced (300ms) to avoid excessive server requests | Must |
| FE-47 | Search respects hidden files toggle | Must |
| FE-48 | File results paginated — first 20 results shown, "Show more" loads next page | Must |
| FE-49 | GlobalSearch mode switcher — segmented pills (Tabs / Files / Commands) or prefix-based (`>` commands, `/` files, no prefix = all) | Must |

### 3.5 Context Actions

Right-click (desktop) or long-press (mobile) on a file/folder to show a context menu.

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-50 | Context menu appears as a floating card near the touch/click point | Must |
| FE-51 | Copy full path to clipboard | Must |
| FE-52 | Copy relative path (relative to CWD) to clipboard | Must |
| FE-53 | Rename — inline edit of the filename | Must |
| FE-54 | Delete — confirmation dialog before deleting | Must |
| FE-55 | File/folder info — show size, modified date, permissions, owner in a detail view | Must |
| FE-56 | Open in terminal — sends `cd <path>` to active terminal (folders only) | Must |
| FE-57 | New file — create a new empty file in the current directory, open inline rename | Should |
| FE-58 | New folder — create a new empty folder, open inline rename | Should |
| FE-59 | Download — download file via the file-serve endpoint (files only) | Should |
| FE-60 | Paste path into terminal — inserts the file/folder path at the terminal cursor | Should |

#### Context Menu Items by Type

| Action | Files | Folders |
|--------|-------|---------|
| Copy full path | Yes | Yes |
| Copy relative path | Yes | Yes |
| Rename | Yes | Yes |
| Delete | Yes | Yes |
| Info | Yes | Yes |
| Open in terminal (cd) | — | Yes |
| Open preview | Yes | — |
| Download | Yes | — |
| Paste path into terminal | Yes | Yes |
| New file | — | Yes |
| New folder | — | Yes |

### 3.6 Mobile-Specific Interactions

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-70 | Swipe left on a file/folder row to reveal quick actions (delete, rename) — iOS Mail pattern | Should |
| FE-71 | Long-press to open context menu (500ms, same as shortcut long-press) | Must |
| FE-72 | Pull-to-refresh to reload current directory | Should |
| FE-73 | Sidebar opens as a full-screen overlay on mobile (<768px) | Must |
| FE-74 | Sidebar opens as a side panel (280px) on desktop (>=1024px) | Must |
| FE-75 | Sidebar opens as an overlay panel on tablet (768-1023px) | Should |

### 3.7 Desktop-Specific Interactions

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-80 | Right-click to open context menu | Must |
| FE-81 | Drag a file from the sidebar and drop on the terminal to paste its path | Should |
| FE-82 | Keyboard shortcut to toggle sidebar (e.g., Cmd+B or Cmd+.) | Should |
| FE-83 | Keyboard navigation within the sidebar — arrow keys to move, Enter to open, Backspace to go up | Should |
| FE-84 | Preview on hover — thumbnail tooltip for image files when hovering | Could |

### 3.8 Additional UX Features

| ID | Requirement | Priority |
|----|-------------|----------|
| FE-90 | Pinned/favorite directories — bookmark frequently-accessed paths, shown at the top of the sidebar | Could |
| FE-91 | Multi-select mode — long-press to enter selection mode (mobile), Shift+click (desktop) for bulk delete/copy | Could |
| FE-92 | Sort indicator — show current sort direction with an arrow icon in the header | Should |
| FE-93 | File icons by extension — different icons for code, image, markdown, config, folder, etc. | Should |
| FE-94 | Folder badge — show item count next to folder name | Could |

---

## 4. UI Design

### 4.1 Sidebar Layout

```
Desktop (>=1024px):
┌────────────────────────┬──────────────────────────────────┐
│ File Explorer (280px)  │  Terminal panes                  │
│                        │                                  │
│ [< breadcrumb/path]    │  $ claude                        │
│ [Search...] [. toggle] │                                  │
│                        │                                  │
│ 📁 src/          12 items │                               │
│ 📁 tests/         3 items │                               │
│ 📁 node_modules/       │                                  │
│ 📄 package.json   2.1KB│                                  │
│ 📄 tsconfig.json  482B │                                  │
│ 📄 README.md     1.4KB │                                  │
│                        │                                  │
│                        │                                  │
└────────────────────────┴──────────────────────────────────┘

Mobile (<768px):
┌──────────────────────────────────┐
│ File Explorer          ✕ Close   │
│                                  │
│ [< breadcrumb/path]              │
│ [Search...]         [. toggle]   │
│                                  │
│ 📁 src/              12 items    │
│ 📁 tests/             3 items    │
│ 📄 package.json       2.1KB     │
│ 📄 README.md          1.4KB     │
│                                  │
│                                  │
└──────────────────────────────────┘
```

### 4.2 Breadcrumb Bar

```
[ ~ ] / [ projects ] / [ synergy ] / [ apps ] / [ tether ]

- Each segment is a clickable button
- Tapping a segment navigates the sidebar to that directory
- Overflow: horizontal scroll with fade edges on mobile
- Current (last) segment is bold / primary color
```

### 4.3 File/Folder Row

```
┌──────────────────────────────────────────────────┐
│ 📁  src/                              12 items   │
│     Modified 2 hours ago                         │
├──────────────────────────────────────────────────┤
│ 📄  package.json                        2.1 KB   │
│     Modified 3 days ago                          │
└──────────────────────────────────────────────────┘

- Icon (folder/file type) + name on the first line
- Size (files) or item count (folders) right-aligned
- Modified date as secondary text below
- Hover: bg-muted
- Active/selected: bg-primary-subtle
```

### 4.4 Context Menu

```
┌──────────────────────┐
│ 📋 Copy path         │
│ 📋 Copy relative path│
│ ───────────────────  │
│ ✏️  Rename            │
│ 📥 Download          │
│ ───────────────────  │
│ ℹ️  Info              │
│ 📁 Open in terminal  │
│ ───────────────────  │
│ 🗑️  Delete            │
└──────────────────────┘

- Floating card, positioned near click/touch point
- Close on outside click or Escape
- Dividers group related actions
- Destructive actions (delete) at the bottom, styled with text-error
```

### 4.5 File Preview Tab

```
┌──────────────────────────────────────────────────┐
│ [Tab: terminal] [Tab: 📄 package.json ✕]         │
├──────────────────────────────────────────────────┤
│  1 │ {                                           │
│  2 │   "name": "tether",                         │
│  3 │   "version": "1.0.0",                       │
│  4 │   "scripts": {                              │
│  5 │     "dev": "...",                            │
│  ...                                             │
└──────────────────────────────────────────────────┘

- Non-terminal tab with file icon in tab bar
- Code view: line numbers + syntax highlighting
- Image view: centered, fit-to-container, pinch-to-zoom on mobile
- Markdown view: rendered HTML with prose styling
- Header bar with filename, size, and action buttons (download, copy path)
```

---

## 5. WebSocket Protocol Additions

```
Phone→Server:  { type: "get-cwd", tabId: string }
Server→Phone:  { type: "cwd", tabId: string, path: string }
```

Note: Most file operations use REST endpoints rather than WebSocket, since they are request-response (not streaming). The WebSocket `get-cwd` message is a convenience since we already have a WS connection per tab.

---

## 6. Implementation Phases

### Phase 8a: Server Endpoints
1. `GET /api/cwd/:tabId` — read CWD from tmux
2. `GET /api/files?dir=<path>` — list directory
3. `GET /api/file?path=<path>` — serve file content
4. `GET /api/files/info?path=<path>` — file metadata
5. Path validation and security (traversal prevention, auth)

### Phase 8b: Sidebar UI
1. Sidebar panel component (slide-in, collapsible)
2. Breadcrumb navigation bar
3. File/folder list with icons, size, modified date
4. Folder navigation (tap to enter, breadcrumb to go up)
5. Hidden files toggle
6. Sort options
7. Loading/empty/error states

### Phase 8c: File Preview
1. Non-terminal tab type in the pane system
2. Code preview with highlight.js + line numbers
3. Image preview
4. Markdown preview with `marked`
5. PDF/video/audio preview
6. Binary file info card + download

### Phase 8d: Context Actions
1. Context menu component (right-click / long-press)
2. Copy path / Copy relative path
3. Rename (inline edit)
4. Delete with confirmation
5. File info detail view
6. Open in terminal (cd)
7. New file / New folder

### Phase 8e: File Search
1. Sidebar search input with debounced filtering
2. Server endpoint for recursive search
3. Search results display with relative paths
4. GlobalSearch integration (files section)

### Phase 8f: Advanced Interactions
1. Swipe-to-reveal actions (mobile)
2. Drag file to terminal to paste path (desktop)
3. Pull-to-refresh
4. Keyboard navigation in sidebar
5. Pinned directories

---

## 7. Dependencies

No new dependencies required. Leverages existing:
- `highlight.js` — syntax highlighting for code preview
- `marked` — markdown rendering
- `express` — REST endpoints

---

## 8. Design Decisions

| Question | Decision |
|----------|----------|
| File preview tab persistence | **Persistent** — preview tabs survive page reload, stored in localStorage alongside terminal tabs |
| Inline file editing in preview | **No** — preview is read-only. Users edit via terminal/editor commands |
| Maximum file size for preview | **500 MB** — files larger than 500 MB fall back to info card + download link |
| GlobalSearch file results limits | **Paginated by file count** — return first N results (e.g., 20), with a "Show more" action. Depth-limited to 5 levels |

### 8.1 GlobalSearch Modes

GlobalSearch should support switching between result types via prefix or mode toggle:

| Mode | Trigger | Results shown |
|------|---------|---------------|
| All (default) | No prefix | Tabs + Files + Commands (mixed, grouped by section) |
| Commands only | `>` prefix | Only shortcut commands |
| Files only | `/` prefix | Only file search results |

Each mode is paginated independently:
- **Tabs**: show all (typically small count)
- **Files**: show first 20 results, "Show more" loads next page
- **Commands**: show all (typically small count)

A segmented control or subtle mode pills (Tabs / Files / Commands) at the top of the search results allows switching without typing a prefix.
