import {
  Component,
  JSX,
  createContext,
  createEffect,
  createSignal,
  onMount,
  onCleanup,
  useContext
} from "solid-js";
import { createStore, produce, reconcile } from "solid-js/store";
import {
  findLeafInTree,
  collectLeaves,
  findBranchById,
  findSiblingLeaves,
  removeLeaf,
  replaceLeaf,
  syncCountersFromTree
} from "./panes-utils.js";

export interface Tab {
  id: string;
  label: string;
}

export type SplitDirection = "horizontal" | "vertical";

export interface LeafNode {
  type: "leaf";
  id: string;
  tabs: Tab[];
  activeTabId: string;
}

export interface BranchNode {
  type: "branch";
  id: string;
  direction: SplitDirection;
  children: [SplitNode, SplitNode];
  ratio: number;
}

export type SplitNode = LeafNode | BranchNode;

interface PanesState {
  root: SplitNode;
  activePaneId: string;
}

interface PanesContextValue {
  state: PanesState;
  activePaneId: () => string;
  activeTabId: () => string;
  setActivePane: (paneId: string) => void;
  addTab: (paneId: string) => string;
  closeTab: (paneId: string, tabId: string) => void;
  setActiveTab: (paneId: string, tabId: string) => void;
  renameTab: (paneId: string, tabId: string, label: string) => void;
  reorderTab: (paneId: string, tabId: string, targetIndex: number) => void;
  mergePane: (paneId: string) => { targetLabel: string } | null;
  confirmMerge: (paneId: string) => void;
  splitPane: (paneId: string, direction: SplitDirection) => string;
  splitPaneWithTab: (
    targetPaneId: string,
    direction: SplitDirection,
    insertBefore: boolean,
    fromPaneId: string,
    tabId: string
  ) => void;
  moveTab: (
    fromPaneId: string,
    tabId: string,
    toPaneId: string,
    index?: number
  ) => void;
  setRatio: (branchId: string, ratio: number) => void;
  findLeaf: (paneId: string) => LeafNode | undefined;
  getAllLeaves: () => LeafNode[];
  getAllTabIds: () => string[];
  getRoot: () => SplitNode;
  isNarrow: () => boolean;
}

const PanesContext = createContext<PanesContextValue>();

const STORAGE_KEY = "tether-panes";

const counters = { pane: 1, tab: 1 };

function genPaneId(): string {
  return `pane-${counters.pane++}`;
}

function genTabId(): string {
  return `tab-${counters.tab++}`;
}

function loadPersistedState(): PanesState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as PanesState;
    if (!parsed.root || !parsed.activePaneId) return null;
    syncCountersFromTree(parsed.root, counters);
    return parsed;
  } catch {
    return null;
  }
}

function persistState(state: PanesState): void {
  const data = JSON.parse(JSON.stringify(state));
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ root: data.root, activePaneId: data.activePaneId })
    );
  } catch {
    // localStorage may be unavailable
  }
}

export const PanesProvider: Component<{ children: JSX.Element }> = (props) => {
  const persisted = loadPersistedState();

  const initialPaneId = persisted ? persisted.activePaneId : genPaneId();
  const initialTabId = persisted ? "" : genTabId();

  const initialState: PanesState = persisted ?? {
    root: {
      type: "leaf",
      id: initialPaneId,
      tabs: [{ id: initialTabId, label: "Terminal 1" }],
      activeTabId: initialTabId
    },
    activePaneId: initialPaneId
  };

  const [state, setState] = createStore<PanesState>(initialState);

  createEffect(() => { persistState(state); });

  const [isNarrow, setIsNarrow] = createSignal(false);

  onMount(() => {
    const query = window.matchMedia("(max-width: 640px)");
    setIsNarrow(query.matches);
    const handler = (e: MediaQueryListEvent): void => {
      setIsNarrow(e.matches);
    };
    query.addEventListener("change", handler);
    onCleanup(() => query.removeEventListener("change", handler));
  });

  const activePaneId = (): string => state.activePaneId;

  const activeTabId = (): string => {
    const leaf = findLeafInTree(state.root, state.activePaneId);
    return leaf?.activeTabId ?? "";
  };

  const setActivePane = (paneId: string): void => {
    setState("activePaneId", paneId);
  };

  const findLeaf = (paneId: string): LeafNode | undefined =>
    findLeafInTree(state.root, paneId);

  const getAllLeaves = (): LeafNode[] => {
    const result: LeafNode[] = [];
    collectLeaves(state.root, result);
    return result;
  };

  const getAllTabIds = (): string[] =>
    getAllLeaves().flatMap(leaf => leaf.tabs.map(t => t.id));

  const getRoot = (): SplitNode => state.root;

  const addTab = (paneId: string): string => {
    const id = genTabId();
    const leaves = getAllLeaves();
    const totalTabs = leaves.reduce((n, l) => n + l.tabs.length, 0);
    const label = `Terminal ${totalTabs + 1}`;

    setState(
      "root",
      produce((root) => {
        const leaf = findLeafInTree(root, paneId);
        if (!leaf) return;
        leaf.tabs.push({ id, label });
        leaf.activeTabId = id;
      })
    );
    return id;
  };

  const closePane = (paneId: string): void => {
    if (state.root.type === "leaf") return;
    const newRoot = removeLeaf(state.root, paneId);
    if (!newRoot) return;
    const leaves: LeafNode[] = [];
    collectLeaves(newRoot, leaves);
    const firstLeaf = leaves[0];
    const newActive = firstLeaf ? firstLeaf.id : state.activePaneId;
    setState(reconcile({ root: newRoot, activePaneId: newActive }));
  };

  const closeTab = (paneId: string, tabId: string): void => {
    const leaf = findLeafInTree(state.root, paneId);
    if (!leaf) return;
    if (leaf.tabs.length <= 1) {
      closePane(paneId);
      return;
    }
    setState(
      "root",
      produce((root) => {
        const l = findLeafInTree(root, paneId);
        if (!l) return;
        const idx = l.tabs.findIndex((t) => t.id === tabId);
        if (idx === -1) return;
        l.tabs.splice(idx, 1);
        if (l.activeTabId === tabId) {
          const newIdx = Math.min(idx, l.tabs.length - 1);
          const nextTab = l.tabs[newIdx];
          if (nextTab) l.activeTabId = nextTab.id;
        }
      })
    );
  };

  const setActiveTab = (paneId: string, tabId: string): void => {
    setState(
      "root",
      produce((root) => {
        const leaf = findLeafInTree(root, paneId);
        if (leaf) leaf.activeTabId = tabId;
      })
    );
  };

  const renameTab = (paneId: string, tabId: string, label: string): void => {
    setState(
      "root",
      produce((root) => {
        const leaf = findLeafInTree(root, paneId);
        if (!leaf) return;
        const tab = leaf.tabs.find((t) => t.id === tabId);
        if (tab) tab.label = label;
      })
    );
  };

  const spliceAndInsert = (tabs: Tab[], currentIdx: number, targetIndex: number): void => {
    const removed = tabs.splice(currentIdx, 1);
    const tab = removed[0];
    if (!tab) return;
    const insertAt = targetIndex > currentIdx ? targetIndex - 1 : targetIndex;
    tabs.splice(insertAt, 0, tab);
  };

  const reorderTab = (paneId: string, tabId: string, targetIndex: number): void => {
    setState(
      "root",
      produce((root) => {
        const leaf = findLeafInTree(root, paneId);
        if (!leaf) return;
        const currentIdx = leaf.tabs.findIndex(t => t.id === tabId);
        const isNoop = currentIdx === -1 || currentIdx === targetIndex || currentIdx + 1 === targetIndex;
        if (isNoop) return;
        spliceAndInsert(leaf.tabs, currentIdx, targetIndex);
      })
    );
  };

  const getFirstSibling = (paneId: string): LeafNode | null => {
    if (state.root.type === "leaf") return null;
    const siblings = findSiblingLeaves(state.root, paneId);
    return siblings?.[0] ?? null;
  };

  const mergePane = (paneId: string): { targetLabel: string } | null => {
    const sibling = getFirstSibling(paneId);
    if (!sibling) return null;
    return { targetLabel: sibling.tabs[0]?.label ?? sibling.id };
  };

  const confirmMerge = (paneId: string): void => {
    const sibling = getFirstSibling(paneId);
    if (!sibling) return;
    const sourceLeaf = findLeafInTree(state.root, paneId);
    if (!sourceLeaf) return;
    for (const tab of sourceLeaf.tabs) {
      moveTab(paneId, tab.id, sibling.id);
    }
  };

  const splitPane = (paneId: string, direction: SplitDirection): string => {
    const newPaneId = genPaneId();
    const newTabId = genTabId();
    const leaves = getAllLeaves();
    const totalTabs = leaves.reduce((n, l) => n + l.tabs.length, 0);

    const newLeaf: LeafNode = {
      type: "leaf",
      id: newPaneId,
      tabs: [{ id: newTabId, label: `Terminal ${totalTabs + 1}` }],
      activeTabId: newTabId
    };

    const sourceLeaf = findLeafInTree(state.root, paneId);
    if (!sourceLeaf) return newTabId;

    const branchId = `branch-${Date.now()}`;
    const newRoot = replaceLeaf(state.root, paneId, {
      type: "branch",
      id: branchId,
      direction,
      children: [sourceLeaf, newLeaf],
      ratio: 0.5
    });

    setState(reconcile({ root: newRoot, activePaneId: newPaneId }));
    return newTabId;
  };

  const removeTabFromLeaf = (leaf: LeafNode, tabId: string): void => {
    const idx = leaf.tabs.findIndex(t => t.id === tabId);
    if (idx === -1) return;
    leaf.tabs.splice(idx, 1);
    if (leaf.activeTabId !== tabId) return;
    const next = leaf.tabs[Math.min(idx, leaf.tabs.length - 1)];
    if (next) leaf.activeTabId = next.id;
  };

  const removeTabFromClone = (root: SplitNode, fromPaneId: string, tabId: string): SplitNode => {
    const clonedSource = findLeafInTree(root, fromPaneId);
    if (!clonedSource) return root;
    removeTabFromLeaf(clonedSource, tabId);
    if (clonedSource.tabs.length === 0) return removeLeaf(root, fromPaneId) ?? root;
    return root;
  };

  const splitPaneWithTab = (
    targetPaneId: string,
    direction: SplitDirection,
    insertBefore: boolean,
    fromPaneId: string,
    tabId: string
  ): void => {
    const sourceLeaf = findLeafInTree(state.root, fromPaneId);
    if (!sourceLeaf) return;
    const tab = sourceLeaf.tabs.find(t => t.id === tabId);
    if (!tab) return;

    const newPaneId = genPaneId();
    const newLeaf: LeafNode = {
      type: "leaf",
      id: newPaneId,
      tabs: [{ ...tab }],
      activeTabId: tab.id
    };

    let newRoot: SplitNode = JSON.parse(JSON.stringify(state.root));
    newRoot = removeTabFromClone(newRoot, fromPaneId, tabId);

    const targetLeaf = findLeafInTree(newRoot, targetPaneId);
    if (!targetLeaf) return;

    const branchId = `branch-${Date.now()}`;
    const children: [SplitNode, SplitNode] = insertBefore
      ? [newLeaf, { ...targetLeaf }]
      : [{ ...targetLeaf }, newLeaf];

    newRoot = replaceLeaf(newRoot, targetPaneId, {
      type: "branch",
      id: branchId,
      direction,
      children,
      ratio: 0.5
    });

    setState(reconcile({ root: newRoot, activePaneId: newPaneId }));
  };

  const transferTab = (
    source: LeafNode,
    target: LeafNode,
    tabId: string,
    insertIdx: number
  ): void => {
    const tabIdx = source.tabs.findIndex((t) => t.id === tabId);
    if (tabIdx === -1) return;
    const removed = source.tabs.splice(tabIdx, 1);
    const tab = removed[0];
    if (!tab) return;
    target.tabs.splice(insertIdx, 0, tab);
    target.activeTabId = tab.id;
    if (source.activeTabId === tabId) {
      const nextTab = source.tabs[Math.min(tabIdx, source.tabs.length - 1)];
      if (nextTab) source.activeTabId = nextTab.id;
    }
  };

  const moveTab = (
    fromPaneId: string,
    tabId: string,
    toPaneId: string,
    index?: number
  ): void => {
    if (fromPaneId === toPaneId) return;

    setState(
      "root",
      produce((root) => {
        const source = findLeafInTree(root, fromPaneId);
        const target = findLeafInTree(root, toPaneId);
        if (!source || !target) return;
        transferTab(source, target, tabId, index ?? target.tabs.length);
      })
    );

    const sourceLeaf = findLeafInTree(state.root, fromPaneId);
    if (sourceLeaf && sourceLeaf.tabs.length === 0) {
      closePane(fromPaneId);
    }

    setState("activePaneId", toPaneId);
  };

  const setRatio = (branchId: string, ratio: number): void => {
    const clamped = Math.max(0.15, Math.min(0.85, ratio));
    setState(
      "root",
      produce((root) => {
        const branch = findBranchById(root, branchId);
        if (branch) branch.ratio = clamped;
      })
    );
  };

  return (
    <PanesContext.Provider
      value={{
        state,
        activePaneId,
        activeTabId,
        setActivePane,
        addTab,
        closeTab,
        setActiveTab,
        renameTab,
        reorderTab,
        mergePane,
        confirmMerge,
        splitPane,
        splitPaneWithTab,
        moveTab,
        setRatio,
        findLeaf,
        getAllLeaves,
        getAllTabIds,
        getRoot,
        isNarrow
      }}
    >
      {props.children}
    </PanesContext.Provider>
  );
};

export function usePanes(): PanesContextValue {
  const ctx = useContext(PanesContext);
  if (!ctx) {
    throw new Error("usePanes must be used within PanesProvider");
  }
  return ctx;
}
