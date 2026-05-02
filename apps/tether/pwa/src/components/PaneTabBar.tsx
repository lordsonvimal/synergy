import {
  Component,
  For,
  Show,
  createSignal,
  createEffect,
  onMount,
  onCleanup
} from "solid-js";
import { usePanes, LeafNode } from "../context/panes.js";
import { useConnection } from "../context/connection.js";
import { destroyInstance } from "../lib/terminal-instances.js";

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

interface PaneTabBarProps {
  paneId: string;
  pane: LeafNode;
}

export const PaneTabBar: Component<PaneTabBarProps> = (props) => {
  const {
    addTab,
    closeTab,
    setActiveTab,
    renameTab,
    splitPane,
    mergePane,
    confirmMerge,
    moveTab,
    activePaneId,
    getAllLeaves
  } = usePanes();
  const { send } = useConnection();
  const [editingId, setEditingId] = createSignal<string | null>(null);
  const [mergeConfirm, setMergeConfirm] = createSignal(false);
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
  const [dragOverIndex, setDragOverIndex] = createSignal<number | null>(null);

  let scrollRef: HTMLDivElement | undefined;

  const tabs = () => props.pane.tabs;
  const activeTabId = () => props.pane.activeTabId;

  const dismissAll = (): void => {
    setContextMenu(null);
    setDropdownOpen(false);
    setMergeConfirm(false);
    if (searchOpen()) {
      setSearchOpen(false);
      setSearchQuery("");
    }
  };

  onMount(() => {
    const handler = (e: MouseEvent): void => {
      const target = e.target as HTMLElement;
      if (!target.closest(`[data-testid="pane-tab-bar-${props.paneId}"]`)) {
        dismissAll();
      }
    };
    document.addEventListener("pointerdown", handler);
    onCleanup(() => document.removeEventListener("pointerdown", handler));
  });

  const updateOverflow = (): void => {
    if (!scrollRef) return;
    const hidden: string[] = [];
    const buttons = scrollRef.querySelectorAll<HTMLElement>("[data-tab-id]");
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
    const observer = new ResizeObserver(() => updateOverflow());
    if (scrollRef) observer.observe(scrollRef);
    onCleanup(() => observer.disconnect());
  });

  createEffect(() => {
    tabs();
    requestAnimationFrame(updateOverflow);
  });

  const handleAdd = (): void => {
    const id = addTab(props.paneId);
    send({ type: "create-tab", tabId: id });
    requestAnimationFrame(() => {
      if (scrollRef) scrollRef.scrollLeft = scrollRef.scrollWidth;
      updateOverflow();
    });
  };

  const destroyAndClose = (tabId: string): void => {
    send({ type: "close-tab", tabId });
    destroyInstance(tabId);
    closeTab(props.paneId, tabId);
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
    for (const t of toClose) destroyAndClose(t.id);
    setContextMenu(null);
  };

  const closeTabsToRight = (tabId: string): void => {
    const idx = tabs().findIndex((t) => t.id === tabId);
    const toClose = tabs().slice(idx + 1);
    for (const t of toClose) destroyAndClose(t.id);
    setContextMenu(null);
  };

  const closeTabsToLeft = (tabId: string): void => {
    const idx = tabs().findIndex((t) => t.id === tabId);
    const toClose = tabs().slice(0, idx);
    for (const t of toClose) destroyAndClose(t.id);
    setContextMenu(null);
  };

  const handleSplitRight = (): void => {
    const menu = contextMenu();
    if (!menu) return;
    const newTabId = splitPane(props.paneId, "horizontal");
    send({ type: "create-tab", tabId: newTabId });
    setContextMenu(null);
  };

  const handleSplitDown = (): void => {
    const menu = contextMenu();
    if (!menu) return;
    const newTabId = splitPane(props.paneId, "vertical");
    send({ type: "create-tab", tabId: newTabId });
    setContextMenu(null);
  };

  const startRename = (tabId: string, currentLabel: string): void => {
    setEditingId(tabId);
    setEditValue(currentLabel);
  };

  const commitRename = (): void => {
    const id = editingId();
    if (id && editValue().trim()) {
      renameTab(props.paneId, id, editValue().trim());
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
    setActiveTab(props.paneId, tabId);
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
    setActiveTab(props.paneId, tabId);
    setSearchOpen(false);
    setSearchQuery("");
    scrollToTab(tabId);
  };

  const handleSearchKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Escape") {
      setSearchOpen(false);
      setSearchQuery("");
    } else if (e.key === "Enter") {
      const first = filteredTabs()[0];
      if (first) selectFromSearch(first.id);
    }
  };

  const handleSearchInputRef = (el: HTMLInputElement): void => {
    requestAnimationFrame(() => el.focus());
  };

  const handleDragStart = (e: DragEvent, tabId: string): void => {
    if (!e.dataTransfer) return;
    e.dataTransfer.setData(
      "application/x-tether-tab",
      JSON.stringify({ sourcePaneId: props.paneId, tabId })
    );
    e.dataTransfer.effectAllowed = "move";
  };

  const handleDragOver = (e: DragEvent, index: number): void => {
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move";
    setDragOverIndex(index);
  };

  const handleDragLeave = (): void => {
    setDragOverIndex(null);
  };

  const handleDrop = (e: DragEvent, index: number): void => {
    e.preventDefault();
    setDragOverIndex(null);
    if (!e.dataTransfer) return;
    const raw = e.dataTransfer.getData("application/x-tether-tab");
    if (!raw) return;
    const { sourcePaneId, tabId } = JSON.parse(raw);
    if (sourcePaneId === props.paneId) {
      // Reorder within same pane — skip for now (just move between panes)
      return;
    }
    moveTab(sourcePaneId, tabId, props.paneId, index);
  };

  const isActive = () => activePaneId() === props.paneId;

  return (
    <nav
      class={`flex items-center h-9 border-b shrink-0 ${
        isActive() ? "bg-surface border-primary" : "bg-surface border-edge"
      }`}
      data-testid={`pane-tab-bar-${props.paneId}`}
    >
      <div
        ref={scrollRef}
        class="flex-1 flex items-center h-full overflow-x-auto scrollbar-none"
        onScroll={updateOverflow}
        onDragOver={(e) => handleDragOver(e, tabs().length)}
        onDragLeave={handleDragLeave}
        onDrop={(e) => handleDrop(e, tabs().length)}
      >
        <For each={tabs()}>
          {(tab, i) => (
            <button
              class={`flex items-center gap-1.5 h-full px-3 text-xs border-r border-edge cursor-pointer transition-colors whitespace-nowrap shrink-0 ${
                tab.id === activeTabId()
                  ? "bg-canvas text-primary font-semibold border-b-2 border-b-primary"
                  : "bg-surface text-ink-secondary hover:bg-muted hover:text-ink"
              } ${dragOverIndex() === i() ? "border-l-2 border-l-primary" : ""}`}
              onClick={() => setActiveTab(props.paneId, tab.id)}
              onDblClick={() => startRename(tab.id, tab.label)}
              onContextMenu={(e) => handleContextMenu(e, tab.id)}
              draggable={editingId() !== tab.id}
              onDragStart={(e) => handleDragStart(e, tab.id)}
              onDragOver={(e) => handleDragOver(e, i())}
              onDragLeave={handleDragLeave}
              onDrop={(e) => handleDrop(e, i())}
              data-tab-id={tab.id}
              data-testid={`tab-${tab.id}`}
            >
              <Show
                when={editingId() === tab.id}
                fallback={
                  <>
                    <span class="max-w-24 truncate">{tab.label}</span>
                    <span
                      class="text-ink-dim hover:text-ink hover:bg-muted rounded-xs px-0.5 transition-colors"
                      onClick={(e) => handleClose(e, tab.id)}
                      role="button"
                      aria-label={`Close ${tab.label}`}
                    >
                      ✕
                    </span>
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
            class="flex items-center justify-center h-9 px-2 bg-surface border-l border-edge text-xs text-ink-secondary cursor-pointer hover:bg-muted hover:text-ink transition-colors"
            onClick={() => setDropdownOpen(!dropdownOpen())}
            aria-label={`${overflowTabs().length} more tabs`}
          >
            +{overflowTabs().length}
          </button>
          <Show when={dropdownOpen()}>
            <div class="absolute top-full right-0 mt-1 bg-surface-raised border border-edge rounded-lg shadow-lg min-w-40 py-1 z-10">
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

      <Show when={tabs().length > 3}>
        <button
          class="flex items-center justify-center w-7 h-9 bg-surface border-l border-edge text-ink-dim text-xs cursor-pointer hover:text-ink hover:bg-muted transition-colors shrink-0"
          onClick={() => setSearchOpen(true)}
          aria-label="Search tabs"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
        </button>
      </Show>

      <button
        class="flex items-center justify-center w-7 h-9 bg-surface border-l border-edge text-ink-dim text-sm cursor-pointer hover:text-ink hover:bg-muted transition-colors shrink-0"
        onClick={handleAdd}
        aria-label="New tab"
      >
        +
      </button>

      {/* Context menu */}
      <Show when={contextMenu()}>
        {(menu) => (
          <>
            <div
              class="fixed bg-surface-raised border border-edge rounded-lg shadow-lg py-1 min-w-44 z-10"
              style={{
                left: `${menu().x}px`,
                top: `${menu().y}px`
              }}
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
              <div class="h-px bg-edge mx-2 my-1" />
              <button
                class="w-full text-left px-3 py-2 text-xs bg-transparent border-none text-ink cursor-pointer hover:bg-muted transition-colors"
                onClick={handleSplitRight}
              >
                Split Right
              </button>
              <button
                class="w-full text-left px-3 py-2 text-xs bg-transparent border-none text-ink cursor-pointer hover:bg-muted transition-colors"
                onClick={handleSplitDown}
              >
                Split Down
              </button>
              <Show when={getAllLeaves().length > 1}>
                <button
                  class="w-full text-left px-3 py-2 text-xs bg-transparent border-none text-ink cursor-pointer hover:bg-muted transition-colors"
                  onClick={() => {
                    setMergeConfirm(true);
                    setContextMenu(null);
                  }}
                >
                  Merge Pane
                </button>
              </Show>
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
        <div class="fixed top-12 left-4 right-4 max-w-80 bg-surface-raised border border-edge rounded-lg shadow-xl z-10 overflow-hidden">
          <div class="p-2 border-b border-edge">
            <input
              ref={handleSearchInputRef}
              type="text"
              class="w-full bg-canvas border border-edge-strong rounded-md text-sm text-ink px-3 py-2 outline-none focus:border-primary focus:ring-2 focus:ring-primary/25 placeholder:text-ink-dim"
              placeholder="Search tabs..."
              value={searchQuery()}
              onInput={(e) => setSearchQuery(e.currentTarget.value)}
              onKeyDown={handleSearchKeyDown}
            />
          </div>
          <div class="max-h-48 overflow-y-auto py-1">
            <For
              each={filteredTabs()}
              fallback={<p class="px-3 py-2 text-xs text-ink-dim">No tabs found</p>}
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

      {/* Merge confirmation */}
      <Show when={mergeConfirm()}>
        {(() => {
          const info = mergePane(props.paneId);
          if (!info) {
            setMergeConfirm(false);
            return null;
          }
          return (
            <div class="fixed inset-0 flex items-center justify-center z-10">
              <div
                class="absolute inset-0 bg-canvas/60"
                onClick={() => setMergeConfirm(false)}
              />
              <div class="relative bg-surface-raised border border-edge rounded-xl shadow-xl p-5 max-w-72">
                <p class="text-sm text-ink mb-1 font-medium">Merge pane?</p>
                <p class="text-xs text-ink-secondary mb-4">
                  All tabs in this pane will move into the adjacent pane
                  ({info.targetLabel}).
                </p>
                <div class="flex gap-2 justify-end">
                  <button
                    class="px-3 py-1.5 text-xs bg-surface border border-edge rounded-md text-ink cursor-pointer hover:bg-muted transition-colors"
                    onClick={() => setMergeConfirm(false)}
                  >
                    Cancel
                  </button>
                  <button
                    class="px-3 py-1.5 text-xs bg-primary border-none rounded-md text-on-primary cursor-pointer hover:bg-primary-hover transition-colors"
                    onClick={() => {
                      confirmMerge(props.paneId);
                      setMergeConfirm(false);
                    }}
                  >
                    Merge
                  </button>
                </div>
              </div>
            </div>
          );
        })()}
      </Show>
    </nav>
  );
};
