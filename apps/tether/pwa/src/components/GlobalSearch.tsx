import {
  Component,
  For,
  Show,
  createSignal,
  createEffect,
  onCleanup
} from "solid-js";
import { Portal } from "solid-js/web";
import { usePanes } from "../context/panes.js";
import { useSettings, Shortcut } from "../context/settings.js";
import { useConnection } from "../context/connection.js";

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function highlightMatch(text: string, query: string): string {
  if (!query) return escapeHtml(text);
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const regex = new RegExp(`(${escaped})`, "gi");
  return escapeHtml(text).replace(
    regex,
    `<mark class="bg-warning-subtle text-warning rounded-xs px-0.5">$1</mark>`
  );
}

interface GlobalSearchProps {
  open: boolean;
  onClose: () => void;
}

interface TabResult {
  kind: "tab";
  paneId: string;
  tabId: string;
  label: string;
  paneIndex: number;
}

interface CommandResult {
  kind: "command";
  shortcut: Shortcut;
}

type SearchResult = TabResult | CommandResult;

const SECTION_LABELS: Record<string, string> = { tab: "Tabs", command: "Commands" };

function isSectionStart(results: SearchResult[], idx: number): string | null {
  const item = results[idx];
  if (!item) return null;
  const prev = results[idx - 1];
  const isNewSection = !prev || prev.kind !== item.kind;
  return isNewSection ? SECTION_LABELS[item.kind] ?? null : null;
}

export const GlobalSearch: Component<GlobalSearchProps> = (props) => {
  const { getAllLeaves, setActivePane, setActiveTab, activeTabId } = usePanes();
  const { settings } = useSettings();
  const { send } = useConnection();
  const [query, setQuery] = createSignal("");
  const [selectedIdx, setSelectedIdx] = createSignal(0);

  const isCommandMode = (): boolean => query().startsWith(">");

  const effectiveQuery = (): string => {
    const q = query();
    if (q.startsWith(">")) return q.slice(1).trim();
    return q.trim();
  };

  const matchingTabs = (): TabResult[] => {
    if (isCommandMode()) return [];
    const q = effectiveQuery().toLowerCase();
    return getAllLeaves().flatMap((leaf, p) =>
      leaf.tabs
        .filter(tab => !q || tab.label.toLowerCase().includes(q))
        .map(tab => ({
          kind: "tab" as const,
          paneId: leaf.id,
          tabId: tab.id,
          label: tab.label,
          paneIndex: p + 1
        }))
    );
  };

  const matchingCommands = (): CommandResult[] => {
    const q = effectiveQuery().toLowerCase();
    return settings()
      .shortcuts.filter(
        s =>
          !q ||
          s.label.toLowerCase().includes(q) ||
          s.command.toLowerCase().includes(q)
      )
      .map(s => ({ kind: "command" as const, shortcut: s }));
  };

  const filtered = (): SearchResult[] => [
    ...matchingTabs(),
    ...matchingCommands()
  ];

  createEffect(() => {
    filtered();
    setSelectedIdx(0);
  });

  const selectResult = (result: SearchResult): void => {
    if (result.kind === "tab") {
      setActivePane(result.paneId);
      setActiveTab(result.paneId, result.tabId);
    } else {
      send({
        type: "text",
        tabId: activeTabId(),
        data: result.shortcut.command
      });
    }
    props.onClose();
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    const results = filtered();
    const actions: Record<string, () => void> = {
      Escape: () => props.onClose(),
      ArrowDown: () => setSelectedIdx(i => Math.min(i + 1, results.length - 1)),
      ArrowUp: () => setSelectedIdx(i => Math.max(i - 1, 0)),
      Enter: () => { const r = results[selectedIdx()]; if (r) selectResult(r); }
    };
    const action = actions[e.key];
    if (action) { e.preventDefault(); action(); }
  };

  const handleInputRef = (el: HTMLInputElement): void => {
    requestAnimationFrame(() => el.focus());
  };

  createEffect(() => {
    if (!props.open) {
      setQuery("");
      setSelectedIdx(0);
    }
  });

  createEffect(() => {
    if (!props.open) return;
    const handler = (e: MouseEvent): void => {
      const target = e.target as HTMLElement;
      if (!target.closest("[data-testid='global-search-panel']")) {
        props.onClose();
      }
    };
    document.addEventListener("pointerdown", handler);
    onCleanup(() => {
      document.removeEventListener("pointerdown", handler);
    });
  });

  const mount = document.getElementById("search-layer") as HTMLElement | null;

  return (
    <Show when={props.open && mount}>
      <Portal mount={mount as HTMLElement}>
        <div class="fixed inset-0 flex justify-center pt-16 px-4">
          <div
            class="w-full max-w-md bg-surface-raised border border-edge rounded-xl shadow-xl overflow-hidden animate-fade-in"
            style={{ "max-height": "min(28rem, 65vh)" }}
            data-testid="global-search-panel"
          >
            <div class="p-3 border-b border-edge">
              <input
                ref={handleInputRef}
                type="text"
                class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim"
                placeholder="Search tabs and commands... (> for commands)"
                value={query()}
                onInput={e => setQuery(e.currentTarget.value)}
                onKeyDown={handleKeyDown}
                data-testid="global-search-input"
              />
            </div>
            <div
              class="overflow-y-auto"
              style={{ "max-height": "calc(min(28rem, 65vh) - 3.5rem)" }}
            >
              <Show
                when={filtered().length > 0}
                fallback={
                  <p class="px-4 py-3 text-xs text-ink-dim">
                    No results found
                  </p>
                }
              >
                <For each={filtered()}>
                  {(result, i) => {
                    const sectionLabel = (): string | null =>
                      isSectionStart(filtered(), i());
                    return (
                      <>
                        <Show when={sectionLabel()}>
                          <Show when={i() > 0}>
                            <div class="border-t border-edge" />
                          </Show>
                          <div class="px-4 pt-2.5 pb-1">
                            <span class="text-[11px] font-medium text-ink-dim uppercase tracking-wider">
                              {sectionLabel()}
                            </span>
                          </div>
                        </Show>
                        <Show
                          when={result.kind === "tab"}
                          fallback={
                            <button
                              class={`w-full text-left px-4 py-2.5 text-sm cursor-pointer transition-colors border-none flex items-center justify-between gap-2 ${
                                i() === selectedIdx()
                                  ? "bg-primary-subtle text-primary"
                                  : "bg-transparent text-ink hover:bg-muted"
                              }`}
                              onClick={() => selectResult(result)}
                              onMouseEnter={() => setSelectedIdx(i())}
                              data-testid={`global-search-command-${(result as CommandResult).shortcut.id}`}
                            >
                              <span class="flex items-center gap-2 truncate">
                                <svg
                                  xmlns="http://www.w3.org/2000/svg"
                                  width="14"
                                  height="14"
                                  viewBox="0 0 24 24"
                                  fill="none"
                                  stroke="currentColor"
                                  stroke-width="2"
                                  stroke-linecap="round"
                                  stroke-linejoin="round"
                                  class="shrink-0 opacity-50"
                                >
                                  <path d="M13 2 L3 14h9l-1 8 10-12h-9l1-8z" />
                                </svg>
                                <span
                                  class="truncate"
                                  innerHTML={highlightMatch(
                                    (result as CommandResult).shortcut.label,
                                    effectiveQuery()
                                  )}
                                />
                              </span>
                              <span class="shrink-0 text-xs text-ink-dim font-mono truncate max-w-32">
                                {(result as CommandResult).shortcut.command}
                              </span>
                            </button>
                          }
                        >
                          <button
                            class={`w-full text-left px-4 py-2.5 text-sm cursor-pointer transition-colors border-none flex items-center justify-between gap-2 ${
                              i() === selectedIdx()
                                ? "bg-primary-subtle text-primary"
                                : "bg-transparent text-ink hover:bg-muted"
                            }`}
                            onClick={() => selectResult(result)}
                            onMouseEnter={() => setSelectedIdx(i())}
                            data-testid={`global-search-result-${(result as TabResult).tabId}`}
                          >
                            <span class="flex items-center gap-2 truncate">
                              <svg
                                xmlns="http://www.w3.org/2000/svg"
                                width="14"
                                height="14"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                class="shrink-0 opacity-50"
                              >
                                <polyline points="4 17 10 11 4 5" />
                                <line x1="12" y1="19" x2="20" y2="19" />
                              </svg>
                              <span
                                class="truncate"
                                innerHTML={highlightMatch(
                                  (result as TabResult).label,
                                  effectiveQuery()
                                )}
                              />
                            </span>
                            <Show when={getAllLeaves().length > 1}>
                              <span class="shrink-0 text-xs text-ink-dim">
                                Pane {(result as TabResult).paneIndex}
                              </span>
                            </Show>
                          </button>
                        </Show>
                      </>
                    );
                  }}
                </For>
              </Show>
            </div>
          </div>
        </div>
      </Portal>
    </Show>
  );
};
