import { SplitNode, LeafNode, BranchNode } from "./panes.js";

export function findLeafInTree(
  node: SplitNode,
  paneId: string
): LeafNode | undefined {
  if (node.type === "leaf") {
    return node.id === paneId ? node : undefined;
  }
  return (
    findLeafInTree(node.children[0], paneId) ??
    findLeafInTree(node.children[1], paneId)
  );
}

export function collectLeaves(
  node: SplitNode,
  result: LeafNode[]
): void {
  if (node.type === "leaf") {
    result.push(node);
  } else {
    collectLeaves(node.children[0], result);
    collectLeaves(node.children[1], result);
  }
}

export function findBranchById(
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

function collectSiblingLeaves(
  node: SplitNode,
  paneId: string,
  other: SplitNode
): LeafNode[] | null {
  if (node.type !== "leaf" || node.id !== paneId) return null;
  const result: LeafNode[] = [];
  collectLeaves(other, result);
  return result;
}

export function findSiblingLeaves(
  node: SplitNode,
  paneId: string
): LeafNode[] | null {
  if (node.type === "leaf") return null;
  const [left, right] = node.children;
  const fromLeft = collectSiblingLeaves(left, paneId, right);
  if (fromLeft) return fromLeft;
  const fromRight = collectSiblingLeaves(right, paneId, left);
  if (fromRight) return fromRight;
  return (
    findSiblingLeaves(left, paneId) ??
    findSiblingLeaves(right, paneId)
  );
}

function removeDirectChild(
  node: BranchNode,
  paneId: string
): SplitNode | null {
  if (node.children[0].type === "leaf" && node.children[0].id === paneId) {
    return node.children[1];
  }
  if (node.children[1].type === "leaf" && node.children[1].id === paneId) {
    return node.children[0];
  }
  return null;
}

export function removeLeaf(
  node: SplitNode,
  paneId: string
): SplitNode | null {
  if (node.type === "leaf") return null;

  const direct = removeDirectChild(node, paneId);
  if (direct) return direct;

  const left = removeLeaf(node.children[0], paneId);
  if (left) return { ...node, children: [left, node.children[1]] };

  const right = removeLeaf(node.children[1], paneId);
  if (right) return { ...node, children: [node.children[0], right] };

  return null;
}

export function replaceLeaf(
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

function updateCounter(
  id: string,
  prefix: string,
  current: number
): number {
  const num = parseInt(id.replace(prefix, ""), 10);
  if (!isNaN(num) && num >= current) return num + 1;
  return current;
}

function syncLeafCounters(
  node: LeafNode,
  counters: { pane: number; tab: number }
): void {
  counters.pane = updateCounter(node.id, "pane-", counters.pane);
  for (const tab of node.tabs) {
    counters.tab = updateCounter(tab.id, "tab-", counters.tab);
  }
}

export function syncCountersFromTree(
  node: SplitNode,
  counters: { pane: number; tab: number }
): void {
  if (node.type === "leaf") {
    syncLeafCounters(node, counters);
  } else {
    syncCountersFromTree(node.children[0], counters);
    syncCountersFromTree(node.children[1], counters);
  }
}
