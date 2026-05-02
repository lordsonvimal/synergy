import {
  Component,
  For,
  Show,
  createSignal,
  createEffect,
  onMount,
  onCleanup
} from "solid-js";
import { Portal } from "solid-js/web";
import { usePanes, LeafNode } from "../context/panes.js";
import { useConnection } from "../context/connection.js";
import { destroyInstance } from "../lib/terminal-instances.js";

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
    reorderTab,
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
  const [dragOverIndex, setDragOverIndex] = createSignal<number | null>(null);

  let scrollRef: HTMLDivElement | undefined;

  const tabs = () => props.pane.tabs;
  const activeTabId = () => props.pane.activeTabId;

  const dismissAll = (): void => {
    setContextMenu(null);
    setDropdownOpen(false);
    setMergeConfirm(false);
  };

  onMount(() => {
    const handler = (e: MouseEvent): void => {
      const target = e.target as HTMLElement;
      if (mergeConfirm() && target.closest("#modal-layer")) return;
      const inTabBar = target.closest(
        `[data-testid="pane-tab-bar-${props.paneId}"]`
      );
      if (!inTabBar) {
        dismissAll();
        return;
      }
      if (contextMenu() && !target.closest("[data-context-menu]")) {
        setContextMenu(null);
      }
      if (dropdownOpen() && !target.closest("[data-dropdown-menu]")) {
        setDropdownOpen(false);
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


  const [dragSourceTabId, setDragSourceTabId] = createSignal<string | null>(
    null
  );

  const handleDragStart = (e: DragEvent, tabId: string): void => {
    if (!e.dataTransfer) return;
    e.dataTransfer.setData(
      "application/x-tether-tab",
      JSON.stringify({ sourcePaneId: props.paneId, tabId })
    );
    e.dataTransfer.effectAllowed = "move";
    setDragSourceTabId(tabId);
  };

  const handleDragEnd = (): void => {
    setDragSourceTabId(null);
    setDragOverIndex(null);
  };

  const computeInsertIndex = (e: DragEvent): number => {
    if (!scrollRef) return tabs().length;
    const tabEls = scrollRef.querySelectorAll<HTMLElement>("[data-tab-id]");
    for (let i = 0; i < tabEls.length; i++) {
      const el = tabEls[i];
      if (!el) continue;
      const rect = el.getBoundingClientRect();
      const midX = rect.left + rect.width / 2;
      if (e.clientX < midX) return i;
    }
    return tabs().length;
  };

  const handleScrollDragOver = (e: DragEvent): void => {
    if (!e.dataTransfer) return;
    if (!e.dataTransfer.types.includes("application/x-tether-tab")) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDragOverIndex(computeInsertIndex(e));
  };

  const handleScrollDragLeave = (e: DragEvent): void => {
    if (
      scrollRef &&
      e.relatedTarget instanceof Node &&
      scrollRef.contains(e.relatedTarget)
    ) {
      return;
    }
    setDragOverIndex(null);
  };

  const handleScrollDrop = (e: DragEvent): void => {
    e.preventDefault();
    e.stopPropagation();
    const insertIdx = dragOverIndex();
    setDragOverIndex(null);
    setDragSourceTabId(null);
    if (insertIdx === null || !e.dataTransfer) return;
    const raw = e.dataTransfer.getData("application/x-tether-tab");
    if (!raw) return;
    const { sourcePaneId, tabId } = JSON.parse(raw);
    if (sourcePaneId === props.paneId) {
      reorderTab(props.paneId, tabId, insertIdx);
    } else {
      moveTab(sourcePaneId, tabId, props.paneId, insertIdx);
    }
  };

  const indicatorLeft = (): number | null => {
    const idx = dragOverIndex();
    if (idx === null || !scrollRef) return null;
    const tabEls = scrollRef.querySelectorAll<HTMLElement>("[data-tab-id]");
    const scrollLeft = scrollRef.scrollLeft;
    const containerLeft = scrollRef.getBoundingClientRect().left;
    if (idx < tabEls.length) {
      const el = tabEls[idx];
      if (!el) return null;
      return el.getBoundingClientRect().left - containerLeft + scrollLeft;
    }
    const lastEl = tabEls[tabEls.length - 1];
    if (!lastEl) return 0;
    return lastEl.getBoundingClientRect().right - containerLeft + scrollLeft;
  };

  const isActive = () => activePaneId() === props.paneId;

  return (
    <nav
      class={`flex items-center h-9 shrink-0 ${
        isActive()
          ? "bg-surface border-t-2 border-t-primary border-b border-b-edge"
          : "bg-muted border-b border-b-edge"
      }`}
      data-testid={`pane-tab-bar-${props.paneId}`}
    >
      <div
        ref={scrollRef}
        class="relative flex-1 flex items-center h-full overflow-x-auto scrollbar-none"
        onScroll={updateOverflow}
        onDragOver={handleScrollDragOver}
        onDragLeave={handleScrollDragLeave}
        onDrop={handleScrollDrop}
      >
        <For each={tabs()}>
          {(tab) => (
            <button
              class={`flex items-center gap-1.5 h-full px-3 text-xs border-r border-edge cursor-pointer whitespace-nowrap shrink-0 ${
                tab.id === activeTabId()
                  ? "bg-canvas text-ink font-medium"
                  : "bg-transparent text-ink-secondary hover:bg-surface hover:text-ink"
              } ${
                dragSourceTabId() === tab.id ? "opacity-40" : ""
              } transition-colors`}
              onClick={() => setActiveTab(props.paneId, tab.id)}
              onDblClick={() => startRename(tab.id, tab.label)}
              onContextMenu={(e) => handleContextMenu(e, tab.id)}
              draggable={editingId() !== tab.id}
              onDragStart={(e) => handleDragStart(e, tab.id)}
              onDragEnd={handleDragEnd}
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
        <Show when={indicatorLeft() !== null}>
          <div
            class="absolute top-1 bottom-1 w-0.5 bg-primary rounded-full pointer-events-none"
            style={{ left: `${indicatorLeft()}px` }}
          />
        </Show>
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
            <div data-dropdown-menu class="absolute top-full right-0 mt-1 bg-surface-raised border border-edge rounded-lg shadow-lg min-w-40 py-1 z-10">
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
              data-context-menu
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

      {/* Merge confirmation — portaled to modal layer so dividers can't overlap */}
      <Show when={mergeConfirm()}>
        {(() => {
          const info = mergePane(props.paneId);
          if (!info) {
            setMergeConfirm(false);
            return null;
          }
          return (
            <Portal mount={document.getElementById("modal-layer")!}>
              <div class="fixed inset-0 flex items-center justify-center">
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
            </Portal>
          );
        })()}
      </Show>
    </nav>
  );
};
