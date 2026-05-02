import {
  Component,
  JSX,
  createContext,
  createSignal,
  onMount,
  onCleanup,
  useContext
} from "solid-js";
import { createStore, produce, reconcile } from "solid-js/store";

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
  getRoot: () => SplitNode;
  isNarrow: () => boolean;
}

const PanesContext = createContext<PanesContextValue>();

let nextPaneId = 1;
let nextTabId = 1;

function genPaneId(): string {
  return `pane-${nextPaneId++}`;
}

function genTabId(): string {
  return `tab-${nextTabId++}`;
}

function findLeafInTree(node: SplitNode, paneId: string): LeafNode | undefined {
  if (node.type === "leaf") {
    return node.id === paneId ? node : undefined;
  }
  return (
    findLeafInTree(node.children[0], paneId) ??
    findLeafInTree(node.children[1], paneId)
  );
}

function collectLeaves(node: SplitNode, result: LeafNode[]): void {
  if (node.type === "leaf") {
    result.push(node);
  } else {
    collectLeaves(node.children[0], result);
    collectLeaves(node.children[1], result);
  }
}

function findBranchById(
  node: SplitNode,
  branchId: string
): BranchNode | undefined {
  if (node.type === "branch") {
    if (node.id === branchId) return node;
    return (
      findBranchById(node.children[0], branchId) ??
      findBranchById(node.children[1], branchId)
    );
  }
  return undefined;
}

function findSiblingLeaves(
  node: SplitNode,
  paneId: string
): LeafNode[] | null {
  if (node.type === "leaf") return null;
  const [left, right] = node.children;
  if (left.type === "leaf" && left.id === paneId) {
    const result: LeafNode[] = [];
    collectLeaves(right, result);
    return result;
  }
  if (right.type === "leaf" && right.id === paneId) {
    const result: LeafNode[] = [];
    collectLeaves(left, result);
    return result;
  }
  return (
    findSiblingLeaves(left, paneId) ?? findSiblingLeaves(right, paneId)
  );
}

function removeLeaf(
  node: SplitNode,
  paneId: string
): SplitNode | null {
  if (node.type === "leaf") return null;
  if (node.children[0].type === "leaf" && node.children[0].id === paneId) {
    return node.children[1];
  }
  if (node.children[1].type === "leaf" && node.children[1].id === paneId) {
    return node.children[0];
  }
  const left = removeLeaf(node.children[0], paneId);
  if (left) {
    return { ...node, children: [left, node.children[1]] };
  }
  const right = removeLeaf(node.children[1], paneId);
  if (right) {
    return { ...node, children: [node.children[0], right] };
  }
  return null;
}

function replaceLeaf(
  node: SplitNode,
  paneId: string,
  replacement: SplitNode
): SplitNode {
  if (node.type === "leaf") {
    return node.id === paneId ? replacement : node;
  }
  return {
    ...node,
    children: [
      replaceLeaf(node.children[0], paneId, replacement),
      replaceLeaf(node.children[1], paneId, replacement)
    ]
  };
}

export const PanesProvider: Component<{ children: JSX.Element }> = (props) => {
  const initialPaneId = genPaneId();
  const initialTabId = genTabId();

  const [state, setState] = createStore<PanesState>({
    root: {
      type: "leaf",
      id: initialPaneId,
      tabs: [{ id: initialTabId, label: "Terminal 1" }],
      activeTabId: initialTabId
    },
    activePaneId: initialPaneId
  });

  const [isNarrow, setIsNarrow] = createSignal(false);

  onMount(() => {
    const query = window.matchMedia("(max-width: 640px)");
    setIsNarrow(query.matches);
    const handler = (e: MediaQueryListEvent): void => { setIsNarrow(e.matches); };
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

  const findLeaf = (paneId: string): LeafNode | undefined => {
    return findLeafInTree(state.root, paneId);
  };

  const getAllLeaves = (): LeafNode[] => {
    const result: LeafNode[] = [];
    collectLeaves(state.root, result);
    return result;
  };

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

  const setActiveTab = (paneId: string, tabId: string): void => {
    setState(
      "root",
      produce((root) => {
        const leaf = findLeafInTree(root, paneId);
        if (leaf) leaf.activeTabId = tabId;
      })
    );
  };

  const renameTab = (
    paneId: string,
    tabId: string,
    label: string
  ): void => {
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

  const reorderTab = (
    paneId: string,
    tabId: string,
    targetIndex: number
  ): void => {
    setState(
      "root",
      produce((root) => {
        const leaf = findLeafInTree(root, paneId);
        if (!leaf) return;
        const currentIdx = leaf.tabs.findIndex(t => t.id === tabId);
        if (currentIdx === -1) return;
        if (currentIdx === targetIndex || currentIdx + 1 === targetIndex) return;
        const removed = leaf.tabs.splice(currentIdx, 1);
        const tab = removed[0];
        if (!tab) return;
        const insertAt = targetIndex > currentIdx
          ? targetIndex - 1
          : targetIndex;
        leaf.tabs.splice(insertAt, 0, tab);
      })
    );
  };

  const mergePane = (paneId: string): { targetLabel: string } | null => {
    if (state.root.type === "leaf") return null;
    const siblings = findSiblingLeaves(state.root, paneId);
    if (!siblings || siblings.length === 0) return null;
    const firstSibling = siblings[0];
    if (!firstSibling) return null;
    return { targetLabel: firstSibling.tabs[0]?.label ?? firstSibling.id };
  };

  const confirmMerge = (paneId: string): void => {
    if (state.root.type === "leaf") return;
    const siblings = findSiblingLeaves(state.root, paneId);
    if (!siblings || siblings.length === 0) return;
    const targetPaneId = siblings[0]?.id;
    if (!targetPaneId) return;

    const sourceLeaf = findLeafInTree(state.root, paneId);
    if (!sourceLeaf) return;

    for (const tab of sourceLeaf.tabs) {
      moveTab(paneId, tab.id, targetPaneId);
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

    const branchId = `branch-${Date.now()}`;
    const newRoot = replaceLeaf(state.root, paneId, {
      type: "branch",
      id: branchId,
      direction,
      children: [findLeafInTree(state.root, paneId)!, newLeaf],
      ratio: 0.5
    });

    setState(reconcile({ root: newRoot, activePaneId: newPaneId }));
    return newTabId;
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

    // Deep-clone the tree so mutations don't create aliasing bugs
    let newRoot: SplitNode = JSON.parse(JSON.stringify(state.root));

    // 1. Remove the tab from source in the cloned tree
    const clonedSource = findLeafInTree(newRoot, fromPaneId);
    if (clonedSource) {
      const idx = clonedSource.tabs.findIndex(t => t.id === tabId);
      if (idx !== -1) {
        clonedSource.tabs.splice(idx, 1);
        if (clonedSource.activeTabId === tabId) {
          const next = clonedSource.tabs[Math.min(idx, clonedSource.tabs.length - 1)];
          if (next) clonedSource.activeTabId = next.id;
        }
      }
      // If source is now empty, collapse it out of the tree
      if (clonedSource.tabs.length === 0) {
        const collapsed = removeLeaf(newRoot, fromPaneId);
        if (collapsed) newRoot = collapsed;
      }
    }

    // 2. Now split the target pane in the (already pruned) tree
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

        const tabIdx = source.tabs.findIndex((t) => t.id === tabId);
        if (tabIdx === -1) return;

        const removed = source.tabs.splice(tabIdx, 1);
        const tab = removed[0];
        if (!tab) return;

        const insertIdx = index ?? target.tabs.length;
        target.tabs.splice(insertIdx, 0, tab);
        target.activeTabId = tab.id;

        if (source.activeTabId === tabId) {
          const nextTab = source.tabs[Math.min(tabIdx, source.tabs.length - 1)];
          if (nextTab) {
            source.activeTabId = nextTab.id;
          }
        }
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
