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

interface SearchResult {
  paneId: string;
  tabId: string;
  label: string;
  paneIndex: number;
}

export const GlobalSearch: Component<GlobalSearchProps> = (props) => {
  const { getAllLeaves, setActivePane, setActiveTab } = usePanes();
  const [query, setQuery] = createSignal("");
  const [selectedIdx, setSelectedIdx] = createSignal(0);

  const allTabs = (): SearchResult[] => {
    const leaves = getAllLeaves();
    const results: SearchResult[] = [];
    for (let p = 0; p < leaves.length; p++) {
      const leaf = leaves[p];
      if (!leaf) continue;
      for (const tab of leaf.tabs) {
        results.push({
          paneId: leaf.id,
          tabId: tab.id,
          label: tab.label,
          paneIndex: p + 1
        });
      }
    }
    return results;
  };

  const filtered = (): SearchResult[] => {
    const q = query().toLowerCase();
    if (!q) return allTabs();
    return allTabs().filter(r => r.label.toLowerCase().includes(q));
  };

  createEffect(() => {
    filtered();
    setSelectedIdx(0);
  });

  const select = (result: SearchResult): void => {
    setActivePane(result.paneId);
    setActiveTab(result.paneId, result.tabId);
    props.onClose();
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    const results = filtered();
    if (e.key === "Escape") {
      props.onClose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIdx(i => Math.min(i + 1, results.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIdx(i => Math.max(i - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const result = results[selectedIdx()];
      if (result) select(result);
    }
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

  const mount = document.getElementById("search-layer");

  return (
    <Show when={props.open && mount}>
      <Portal mount={mount!}>
        <div class="fixed inset-0 flex justify-center pt-16 px-4">
          <div
            class="w-full max-w-md bg-surface-raised border border-edge rounded-xl shadow-xl overflow-hidden animate-fade-in"
            style={{ "max-height": "min(24rem, 60vh)" }}
            data-testid="global-search-panel"
          >
            <div class="p-3 border-b border-edge">
              <input
                ref={handleInputRef}
                type="text"
                class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim"
                placeholder="Search all tabs..."
                value={query()}
                onInput={e => setQuery(e.currentTarget.value)}
                onKeyDown={handleKeyDown}
                data-testid="global-search-input"
              />
            </div>
            <div class="overflow-y-auto" style={{ "max-height": "calc(min(24rem, 60vh) - 3.5rem)" }}>
              <For
                each={filtered()}
                fallback={
                  <p class="px-4 py-3 text-xs text-ink-dim">
                    No tabs found
                  </p>
                }
              >
                {(result, i) => (
                  <button
                    class={`w-full text-left px-4 py-2.5 text-sm cursor-pointer transition-colors border-none flex items-center justify-between gap-2 ${
                      i() === selectedIdx()
                        ? "bg-primary-subtle text-primary"
                        : "bg-transparent text-ink hover:bg-muted"
                    }`}
                    onClick={() => select(result)}
                    onMouseEnter={() => setSelectedIdx(i())}
                    data-testid={`global-search-result-${result.tabId}`}
                  >
                    <span
                      class="truncate"
                      innerHTML={highlightMatch(result.label, query())}
                    />
                    <Show when={getAllLeaves().length > 1}>
                      <span class="shrink-0 text-xs text-ink-dim">
                        Pane {result.paneIndex}
                      </span>
                    </Show>
                  </button>
                )}
              </For>
            </div>
          </div>
        </div>
      </Portal>
    </Show>
  );
};
