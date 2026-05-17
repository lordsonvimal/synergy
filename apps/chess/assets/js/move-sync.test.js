import { describe, it, expect } from "vitest";
import { shouldApplyIncomingSeq, predictNextSeq, createPostMoveQueue } from "./move-sync.js";

describe("shouldApplyIncomingSeq", () => {
  it("applies the first observation regardless of incoming seq", () => {
    expect(shouldApplyIncomingSeq(-1, 0)).toBe(true);
    expect(shouldApplyIncomingSeq(-1, 17)).toBe(true);
  });

  it("applies forward and same-seq patches", () => {
    expect(shouldApplyIncomingSeq(5, 5)).toBe(true);
    expect(shouldApplyIncomingSeq(5, 6)).toBe(true);
  });

  it("drops strictly older patches", () => {
    expect(shouldApplyIncomingSeq(5, 4)).toBe(false);
    expect(shouldApplyIncomingSeq(10, 0)).toBe(false);
  });

  it("applies when incomingSeq is missing/non-finite (cannot reason without it)", () => {
    expect(shouldApplyIncomingSeq(5, NaN)).toBe(true);
    expect(shouldApplyIncomingSeq(5, undefined)).toBe(true);
  });
});

describe("predictNextSeq", () => {
  it("bumps a real seq by one", () => {
    expect(predictNextSeq(0)).toBe(1);
    expect(predictNextSeq(42)).toBe(43);
  });

  it("does not invent a seq when none has been observed", () => {
    expect(predictNextSeq(-1)).toBe(-1);
  });
});

describe("createPostMoveQueue", () => {
  it("serializes async tasks in submission order", async () => {
    const enqueue = createPostMoveQueue();
    const order = [];
    const wait = (ms, label) => new Promise((r) => setTimeout(() => { order.push(label); r(label); }, ms));

    const a = enqueue(() => wait(30, "a"));
    const b = enqueue(() => wait(10, "b"));
    const c = enqueue(() => wait(5, "c"));

    await Promise.all([a, b, c]);
    expect(order).toEqual(["a", "b", "c"]);
  });

  it("subsequent tasks still run after a prior task rejects", async () => {
    const enqueue = createPostMoveQueue();
    const order = [];

    const a = enqueue(() => Promise.reject(new Error("net fail")).catch(() => { order.push("a-failed"); throw new Error("net fail"); }));
    const b = enqueue(() => Promise.resolve().then(() => order.push("b-ran")));

    await Promise.all([a, b]);
    expect(order).toEqual(["a-failed", "b-ran"]);
  });

  it("returns the per-task promise so callers can await individual completion", async () => {
    const enqueue = createPostMoveQueue();
    const p = enqueue(() => Promise.resolve("done"));
    await expect(p).resolves.toBe("done");
  });
});
