// Pure helpers for client-side move synchronization. Kept dependency-free so
// they can be unit-tested without a DOM, datastar, or chessops.

// Decide whether an incoming server signal (carrying `incomingSeq`) should be
// applied given the highest seq the client has already applied. Returns true
// for the first-ever observation (currentSeq < 0) or any forward/same-seq
// patch; false only for strictly older patches. A non-finite incomingSeq
// (e.g. missing from the payload) is always applied — we can't reason about
// ordering without it.
export function shouldApplyIncomingSeq(currentSeq, incomingSeq) {
  if (!Number.isFinite(incomingSeq)) return true;
  if (currentSeq < 0) return true;
  return incomingSeq >= currentSeq;
}

// Bump the locally-applied seq by one to reflect an optimistic move. If the
// client has never observed a real seq yet (currentSeq < 0), do not invent
// one — the next server signal will seed it authoritatively.
export function predictNextSeq(currentSeq) {
  return currentSeq >= 0 ? currentSeq + 1 : currentSeq;
}

// Returns an enqueue function that serializes async work: each `enqueue(fn)`
// resolves only after the prior queued fn has settled. Errors are swallowed
// so one rejected POST doesn't poison the chain.
export function createPostMoveQueue() {
  let chain = Promise.resolve();
  return function enqueue(fn) {
    const next = chain.then(() => fn()).catch(() => {});
    chain = next;
    return next;
  };
}
