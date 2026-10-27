function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

/**
 * Snapshot-based undo/redo history.
 *
 * The initial state is the first entry. Callers push a new snapshot after every
 * committed mutation; undo/redo return clones so callers cannot mutate the
 * stored entries.
 */
export function createHistory(initial, limit = 100) {
  let entries = [clone(initial)];
  let index = 0;

  return {
    push(next) {
      entries = entries.slice(0, index + 1);
      entries.push(clone(next));
      while (entries.length > limit) {
        entries.shift();
      }
      index = entries.length - 1;
    },
    undo() {
      if (index > 0) index--;
      return clone(entries[index]);
    },
    redo() {
      if (index < entries.length - 1) index++;
      return clone(entries[index]);
    },
    current() {
      return clone(entries[index]);
    },
    canUndo() {
      return index > 0;
    },
    canRedo() {
      return index < entries.length - 1;
    },
    size() {
      return entries.length;
    },
    reset(next) {
      entries = [clone(next)];
      index = 0;
    },
  };
}
