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

export const TabBar: Component = () => {
  const { tabs, activeTabId, addTab, closeTab, setActiveTab, renameTab } =
    useTabs();
  const { send } = useConnection();
  const [editingId, setEditingId] = createSignal<string | null>(null);
  const [editValue, setEditValue] = createSignal("");
  const [overflowTabs, setOverflowTabs] = createSignal<string[]>([]);
  const [dropdownOpen, setDropdownOpen] = createSignal(false);

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

  const handleClose = (e: MouseEvent, tabId: string): void => {
    e.stopPropagation();
    send({ type: "close-tab", tabId });
    destroyTabTerminal(tabId);
    closeTab(tabId);
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
                  ? "bg-canvas text-ink font-medium"
                  : "bg-surface text-ink-secondary hover:bg-muted hover:text-ink"
              }`}
              onClick={() => setActiveTab(tab.id)}
              onDblClick={() => startRename(tab.id, tab.label)}
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
        class="flex items-center justify-center w-8 h-10 bg-surface border-l border-edge text-ink-dim text-sm cursor-pointer hover:text-ink hover:bg-muted transition-colors shrink-0"
        onClick={handleAdd}
        aria-label="New tab"
        data-testid="tab-add"
      >
        +
      </button>
    </nav>
  );
};
