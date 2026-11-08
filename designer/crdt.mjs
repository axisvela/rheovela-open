// Dependency-free, op-based CRDT for the Community Process Designer.
//
// The document is a set of last-writer-wins registers keyed by path:
//   - workflow_id / title / source / any other top-level field
//   - stage:<id>            one stage object
//   - transition:<from>-><to>
//   - compensation:<key>
//   - stages.order / transitions.order / compensations.order
//
// Edits stay in the design draft. Publishing a definition version and migrating
// running instances remain explicit core operations.

const ORDER_KEYS = new Set(["stages.order", "transitions.order", "compensations.order"]);

function clone(value) {
  return value === undefined ? undefined : JSON.parse(JSON.stringify(value));
}

function opWins(candidate, current) {
  if (!current) return true;
  if (candidate.counter !== current.counter) return candidate.counter > current.counter;
  return candidate.actor > current.actor;
}

function stageKey(id) {
  return `stage:${id}`;
}

function transitionKey(tr) {
  return `transition:${tr.from}->${tr.to}`;
}

function compensationKey(c) {
  return `compensation:${c.for_activity || c.id || JSON.stringify(c)}`;
}

export function createCrdt(actor, initial = {}) {
  if (typeof actor !== "string" || !actor.trim()) {
    throw new Error("crdt actor is required");
  }
  const registers = new Map();
  const pending = new Map();
  let clock = 0;

  function seed(key, value) {
    registers.set(key, {
      id: `seed:${key}`,
      key,
      value: clone(value),
      deleted: false,
      counter: 0,
      actor: "seed",
    });
  }

  function setRaw(key, value, deleted) {
    clock += 1;
    const op = {
      id: `${actor}:${clock}`,
      key,
      value: clone(value),
      deleted,
      counter: clock,
      actor,
    };
    apply(op);
    pending.set(op.id, op);
    return op;
  }

  function apply(op) {
    if (!op || typeof op.key !== "string" || typeof op.counter !== "number") {
      return false;
    }
    const current = registers.get(op.key);
    if (!opWins(op, current)) return false;
    registers.set(op.key, {
      id: op.id,
      key: op.key,
      value: clone(op.value),
      deleted: Boolean(op.deleted),
      counter: op.counter,
      actor: op.actor,
    });
    return true;
  }

  function orderFor(key) {
    const reg = registers.get(key);
    return reg && !reg.deleted && Array.isArray(reg.value) ? reg.value.slice() : [];
  }

  function addToOrder(orderKey, id) {
    const order = orderFor(orderKey);
    if (!order.includes(id)) {
      order.push(id);
      setRaw(orderKey, order, false);
    }
  }

  function removeFromOrder(orderKey, id) {
    const order = orderFor(orderKey).filter((value) => value !== id);
    setRaw(orderKey, order, false);
  }

  function materialize() {
    const wf = {};
    for (const [key, reg] of registers) {
      if (reg.deleted || ORDER_KEYS.has(key)) continue;
      if (key.startsWith("stage:") || key.startsWith("transition:") || key.startsWith("compensation:")) {
        continue;
      }
      wf[key] = clone(reg.value);
    }

    const stageOrder = orderFor("stages.order");
    const stageIDs = stageOrder.length ? stageOrder : [...registers.keys()]
      .filter((key) => key.startsWith("stage:"))
      .map((key) => key.slice("stage:".length));
    wf.stages = stageIDs
      .map((id) => registers.get(stageKey(id)))
      .filter((reg) => reg && !reg.deleted)
      .map((reg) => clone(reg.value));

    const transitionOrder = orderFor("transitions.order");
    const transitionKeys = transitionOrder.length ? transitionOrder : [...registers.keys()]
      .filter((key) => key.startsWith("transition:"))
      .map((key) => key.slice("transition:".length));
    wf.transitions = transitionKeys
      .map((key) => registers.get(`transition:${key}`))
      .filter((reg) => reg && !reg.deleted)
      .map((reg) => clone(reg.value));

    const compensationOrder = orderFor("compensations.order");
    const compensationKeys = compensationOrder.length ? compensationOrder : [...registers.keys()]
      .filter((key) => key.startsWith("compensation:"))
      .map((key) => key.slice("compensation:".length));
    const compensations = compensationKeys
      .map((key) => registers.get(`compensation:${key}`))
      .filter((reg) => reg && !reg.deleted)
      .map((reg) => clone(reg.value));
    if (compensations.length > 0 || Array.isArray(wf.compensations)) {
      wf.compensations = compensations;
    }
    return wf;
  }

  // Seed from an existing draft. Seed registers use counter 0, so any local or
  // remote edit wins over the initial value.
  if (initial && typeof initial === "object") {
    for (const [key, value] of Object.entries(initial)) {
      if (key === "stages" && Array.isArray(value)) {
        value.forEach((stage) => {
          if (!stage || !stage.id) return;
          seed(stageKey(stage.id), stage);
          addToOrder("stages.order", stage.id);
        });
        continue;
      }
      if (key === "transitions" && Array.isArray(value)) {
        value.forEach((tr) => {
          if (!tr || !tr.from || !tr.to) return;
          seed(`transition:${tr.from}->${tr.to}`, tr);
          addToOrder("transitions.order", `${tr.from}->${tr.to}`);
        });
        continue;
      }
      if (key === "compensations" && Array.isArray(value)) {
        value.forEach((c) => {
          if (!c) return;
          const id = c.for_activity || c.id || JSON.stringify(c);
          seed(`compensation:${id}`, c);
          addToOrder("compensations.order", id);
        });
        continue;
      }
      seed(key, value);
    }
  }

  return {
    actor,
    set(key, value) {
      return setRaw(key, value, false);
    },
    remove(key) {
      return setRaw(key, undefined, true);
    },
    setWorkflowField(name, value) {
      return setRaw(name, value, false);
    },
    setStage(stage) {
      if (!stage || !stage.id) throw new Error("stage id is required");
      setRaw(stageKey(stage.id), stage, false);
      addToOrder("stages.order", stage.id);
    },
    removeStage(id) {
      setRaw(stageKey(id), undefined, true);
      removeFromOrder("stages.order", id);
    },
    setTransition(tr) {
      if (!tr || !tr.from || !tr.to) throw new Error("transition from/to are required");
      const key = `${tr.from}->${tr.to}`;
      setRaw(`transition:${key}`, tr, false);
      addToOrder("transitions.order", key);
    },
    removeTransition(from, to) {
      const key = `${from}->${to}`;
      setRaw(`transition:${key}`, undefined, true);
      removeFromOrder("transitions.order", key);
    },
    setCompensation(compensation) {
      const id = compensation.for_activity || compensation.id || JSON.stringify(compensation);
      setRaw(`compensation:${id}`, compensation, false);
      addToOrder("compensations.order", id);
    },
    removeCompensation(id) {
      setRaw(`compensation:${id}`, undefined, true);
      removeFromOrder("compensations.order", id);
    },
    apply,
    merge(ops) {
      let applied = 0;
      for (const op of ops || []) {
        if (apply(op)) applied += 1;
      }
      return applied;
    },
    receive(ops) {
      return this.merge(ops);
    },
    snapshot() {
      return [...registers.values()].map((reg) => ({ ...reg }));
    },
    pending() {
      return [...pending.values()].map((op) => ({ ...op }));
    },
    ack(ops) {
      for (const op of ops || []) {
        pending.delete(op.id);
      }
    },
    materialize,
    state(key) {
      const reg = registers.get(key);
      return reg && !reg.deleted ? clone(reg.value) : undefined;
    },
  };
}

// createSync wires the CRDT to a transport. flush() sends unacknowledged local
// ops; the caller acks them when the peer confirms. Reconnect/offline replay is
// simply calling flush() again: pending ops stay queued until ack.
export function createSync(crdt, { send }) {
  if (typeof send !== "function") {
    throw new Error("sync send function is required");
  }
  return {
    flush() {
      const ops = crdt.pending();
      if (ops.length === 0) return 0;
      send(ops.map((op) => ({ ...op })));
      return ops.length;
    },
    receive(ops) {
      return crdt.receive(ops);
    },
    pendingCount() {
      return crdt.pending().length;
    },
    ack(ops) {
      crdt.ack(ops);
    },
    snapshot() {
      return crdt.snapshot();
    },
    materialize() {
      return crdt.materialize();
    },
  };
}
