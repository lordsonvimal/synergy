import {
  Component,
  For,
  Show,
  createSignal,
  createEffect,
  onMount,
  onCleanup
} from "solid-js";
import { useTabs } from "../context/tabs.js";
import { useConnection } from "../context/connection.js";
import { destroyTabTerminal } from "./Terminal.js";

function highlightMatch(text: string, query: string): string {
  if (!query) return escapeHtml(text);
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const regex = new RegExp(`(${escaped})`, "gi");
  return escapeHtml(text).replace(
    regex,
    `<mark class="bg-warning-subtle text-warning rounded-xs px-0.5">$1</mark>`
  );
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export const TabBar: Component = () => {
  const { tabs, activeTabId, addTab, closeTab, setActiveTab, renameTab } =
    useTabs();
  const { send } = useConnection();
  const [editingId, setEditingId] = createSignal<string | null>(null);
  const [editValue, setEditValue] = createSignal("");
  const [overflowTabs, setOverflowTabs] = createSignal<string[]>([]);
  const [dropdownOpen, setDropdownOpen] = createSignal(false);
  const [contextMenu, setContextMenu] = createSignal<{
    tabId: string;
    x: number;
    y: number;
  } | null>(null);
  const [searchOpen, setSearchOpen] = createSignal(false);
  const [searchQuery, setSearchQuery] = createSignal("");

  let scrollRef: HTMLDivElement | undefined;

  const updateOverflow = (): void => {
    if (!scrollRef) return;
    const hidden: string[] = [];
    const buttons = scrollRef.querySelectorAll<HTMLElement>(
      "[data-tab-id]"
    );
    const containerRect = scrollRef.getBoundingClientRect();
    for (const btn of buttons) {
      const rect = btn.getBoundingClientRect();
      const fullyVisible =
        rect.left >= containerRect.left - 1 &&
        rect.right <= containerRect.right + 1;
      if (!fullyVisible) {
        const tabId = btn.getAttribute("data-tab-id");
        if (tabId) hidden.push(tabId);
      }
    }
    setOverflowTabs(hidden);
  };

  onMount(() => {
    const observer = new ResizeObserver(() => {
      updateOverflow();
    });
    if (scrollRef) {
      observer.observe(scrollRef);
    }
    onCleanup(() => observer.disconnect());
  });

  createEffect(() => {
    tabs();
    requestAnimationFrame(updateOverflow);
  });

  const handleAdd = (): void => {
    const id = addTab();
    send({ type: "create-tab", tabId: id });
    requestAnimationFrame(() => {
      if (scrollRef) {
        scrollRef.scrollLeft = scrollRef.scrollWidth;
      }
      updateOverflow();
    });
  };

  const destroyAndClose = (tabId: string): void => {
    send({ type: "close-tab", tabId });
    destroyTabTerminal(tabId);
    closeTab(tabId);
  };

  const handleClose = (e: MouseEvent, tabId: string): void => {
    e.stopPropagation();
    destroyAndClose(tabId);
  };

  const handleContextMenu = (e: MouseEvent, tabId: string): void => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({ tabId, x: e.clientX, y: e.clientY });
  };

  const closeOtherTabs = (tabId: string): void => {
    const toClose = tabs().filter((t) => t.id !== tabId);
    for (const t of toClose) {
      destroyAndClose(t.id);
    }
    setContextMenu(null);
  };

  const closeTabsToRight = (tabId: string): void => {
    const idx = tabs().findIndex((t) => t.id === tabId);
    const toClose = tabs().slice(idx + 1);
    for (const t of toClose) {
      destroyAndClose(t.id);
    }
    setContextMenu(null);
  };

  const closeTabsToLeft = (tabId: string): void => {
    const idx = tabs().findIndex((t) => t.id === tabId);
    const toClose = tabs().slice(0, idx);
    for (const t of toClose) {
      destroyAndClose(t.id);
    }
    setContextMenu(null);
  };

  const startRename = (tabId: string, currentLabel: string): void => {
    setEditingId(tabId);
    setEditValue(currentLabel);
  };

  const commitRename = (): void => {
    const id = editingId();
    if (id && editValue().trim()) {
      renameTab(id, editValue().trim());
    }
    setEditingId(null);
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter") {
      e.preventDefault();
      commitRename();
    } else if (e.key === "Escape") {
      setEditingId(null);
    }
  };

  const handleInputRef = (el: HTMLInputElement): void => {
    requestAnimationFrame(() => {
      el.focus();
      el.select();
    });
  };

  const selectFromDropdown = (tabId: string): void => {
    setActiveTab(tabId);
    setDropdownOpen(false);
    scrollToTab(tabId);
  };

  const scrollToTab = (tabId: string): void => {
    requestAnimationFrame(() => {
      const el = scrollRef?.querySelector(
        `[data-tab-id="${tabId}"]`
      ) as HTMLElement | null;
      el?.scrollIntoView({ inline: "center", behavior: "smooth" });
      updateOverflow();
    });
  };

  const getTabLabel = (tabId: string): string => {
    return tabs().find((t) => t.id === tabId)?.label ?? tabId;
  };

  const filteredTabs = (): { id: string; label: string }[] => {
    const q = searchQuery().toLowerCase();
    if (!q) return tabs();
    return tabs().filter((t) => t.label.toLowerCase().includes(q));
  };

  const selectFromSearch = (tabId: string): void => {
    setActiveTab(tabId);
    setSearchOpen(false);
    setSearchQuery("");
    scrollToTab(tabId);
  };

  const handleSearchKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Escape") {
      setSearchOpen(false);
      setSearchQuery("");
    } else if (e.key === "Enter") {
      const results = filteredTabs();
      if (results.length > 0) {
        selectFromSearch(results[0].id);
      }
    }
  };

  const handleSearchInputRef = (el: HTMLInputElement): void => {
    requestAnimationFrame(() => {
      el.focus();
    });
  };

  return (
    <nav
      class="flex items-center h-10 bg-surface border-b border-edge shrink-0"
      data-testid="tab-bar"
    >
      <div
        ref={scrollRef}
        class="flex-1 flex items-center h-full overflow-x-auto scrollbar-none"
        onScroll={updateOverflow}
      >
        <For each={tabs()}>
          {(tab) => (
            <button
              class={`flex items-center gap-1.5 h-full px-3 text-xs border-r border-edge cursor-pointer transition-colors whitespace-nowrap shrink-0 ${
                tab.id === activeTabId()
                  ? "bg-canvas text-primary font-semibold border-b-2 border-b-primary"
                  : "bg-surface text-ink-secondary hover:bg-muted hover:text-ink"
              }`}
              onClick={() => setActiveTab(tab.id)}
              onDblClick={() => startRename(tab.id, tab.label)}
              onContextMenu={(e) => handleContextMenu(e, tab.id)}
              data-tab-id={tab.id}
              data-testid={`tab-${tab.id}`}
            >
              <Show
                when={editingId() === tab.id}
                fallback={
                  <>
                    <span class="max-w-24 truncate">{tab.label}</span>
                    <Show when={tabs().length > 1}>
                      <span
                        class="text-ink-dim hover:text-ink hover:bg-muted rounded-xs px-0.5 transition-colors"
                        onClick={(e) => handleClose(e, tab.id)}
                        role="button"
                        aria-label={`Close ${tab.label}`}
                      >
                        ✕
                      </span>
                    </Show>
                  </>
                }
              >
                <input
                  ref={handleInputRef}
                  type="text"
                  class="w-20 bg-canvas border border-edge-strong rounded-xs text-xs text-ink px-1 py-0.5 outline-none focus:border-primary"
                  value={editValue()}
                  onInput={(e) => setEditValue(e.currentTarget.value)}
                  onBlur={commitRename}
                  onKeyDown={handleKeyDown}
                  data-testid={`tab-rename-${tab.id}`}
                />
              </Show>
            </button>
          )}
        </For>
      </div>

      <Show when={overflowTabs().length > 0}>
        <div class="relative shrink-0">
          <button
            class="flex items-center justify-center h-10 px-2 bg-surface border-l border-edge text-xs text-ink-secondary cursor-pointer hover:bg-muted hover:text-ink transition-colors"
            onClick={() => setDropdownOpen(!dropdownOpen())}
            aria-label={`${overflowTabs().length} more tabs`}
            data-testid="tab-overflow-toggle"
          >
            +{overflowTabs().length}
          </button>
          <Show when={dropdownOpen()}>
            <div
              class="fixed inset-0"
              onClick={() => setDropdownOpen(false)}
            />
            <div
              class="absolute top-full right-0 mt-1 bg-surface-raised border border-edge rounded-lg shadow-lg min-w-40 py-1 z-10"
              data-testid="tab-overflow-dropdown"
            >
              <For each={overflowTabs()}>
                {(tabId) => (
                  <button
                    class={`w-full text-left px-3 py-2 text-xs cursor-pointer transition-colors border-none ${
                      tabId === activeTabId()
                        ? "bg-primary-subtle text-primary font-medium"
                        : "bg-transparent text-ink hover:bg-muted"
                    }`}
                    onClick={() => selectFromDropdown(tabId)}
                  >
                    {getTabLabel(tabId)}
                  </button>
                )}
              </For>
            </div>
          </Show>
        </div>
      </Show>

      <button
        class="flex items-center justify-center w-8 h-10 bg-surface border-l border-edge text-ink-dim text-xs cursor-pointer hover:text-ink hover:bg-muted transition-colors shrink-0"
        onClick={() => setSearchOpen(true)}
        aria-label="Search tabs"
        data-testid="tab-search-toggle"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.3-4.3" />
        </svg>
      </button>

      <button
        class="flex items-center justify-center w-8 h-10 bg-surface border-l border-edge text-ink-dim text-sm cursor-pointer hover:text-ink hover:bg-muted transition-colors shrink-0"
        onClick={handleAdd}
        aria-label="New tab"
        data-testid="tab-add"
      >
        +
      </button>

      {/* Context menu */}
      <Show when={contextMenu()}>
        {(menu) => (
          <>
            <div
              class="fixed inset-0"
              onClick={() => setContextMenu(null)}
            />
            <div
              class="fixed bg-surface-raised border border-edge rounded-lg shadow-lg py-1 min-w-44 z-10"
              style={{
                left: `${menu().x}px`,
                top: `${menu().y}px`
              }}
              data-testid="tab-context-menu"
            >
              <button
                class="w-full text-left px-3 py-2 text-xs bg-transparent border-none text-ink cursor-pointer hover:bg-muted transition-colors"
                onClick={() => {
                  startRename(menu().tabId, getTabLabel(menu().tabId));
                  setContextMenu(null);
                }}
              >
                Rename
              </button>
              <Show when={tabs().length > 1}>
                <div class="h-px bg-edge mx-2 my-1" />
                <button
                  class="w-full text-left px-3 py-2 text-xs bg-transparent border-none text-ink cursor-pointer hover:bg-muted transition-colors"
                  onClick={() => closeOtherTabs(menu().tabId)}
                >
                  Close other tabs
                </button>
                <button
                  class="w-full text-left px-3 py-2 text-xs bg-transparent border-none text-ink cursor-pointer hover:bg-muted transition-colors"
                  onClick={() => closeTabsToLeft(menu().tabId)}
                >
                  Close tabs to the left
                </button>
                <button
                  class="w-full text-left px-3 py-2 text-xs bg-transparent border-none text-ink cursor-pointer hover:bg-muted transition-colors"
                  onClick={() => closeTabsToRight(menu().tabId)}
                >
                  Close tabs to the right
                </button>
              </Show>
            </div>
          </>
        )}
      </Show>

      {/* Search panel */}
      <Show when={searchOpen()}>
        <div
          class="fixed inset-0"
          onClick={() => {
            setSearchOpen(false);
            setSearchQuery("");
          }}
        />
        <div
          class="fixed top-12 left-4 right-4 max-w-80 bg-surface-raised border border-edge rounded-lg shadow-xl z-10 overflow-hidden"
          data-testid="tab-search-panel"
        >
          <div class="p-2 border-b border-edge">
            <input
              ref={handleSearchInputRef}
              type="text"
              class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim"
              placeholder="Search tabs..."
              value={searchQuery()}
              onInput={(e) => setSearchQuery(e.currentTarget.value)}
              onKeyDown={handleSearchKeyDown}
              data-testid="tab-search-input"
            />
          </div>
          <div class="max-h-48 overflow-y-auto py-1">
            <For
              each={filteredTabs()}
              fallback={
                <p class="px-3 py-2 text-xs text-ink-dim">No tabs found</p>
              }
            >
              {(tab) => (
                <button
                  class={`w-full text-left px-3 py-2 text-xs cursor-pointer transition-colors border-none ${
                    tab.id === activeTabId()
                      ? "bg-primary-subtle text-primary font-medium"
                      : "bg-transparent text-ink hover:bg-muted"
                  }`}
                  onClick={() => selectFromSearch(tab.id)}
                  innerHTML={highlightMatch(tab.label, searchQuery())}
                />
              )}
            </For>
          </div>
        </div>
      </Show>
    </nav>
  );
};
