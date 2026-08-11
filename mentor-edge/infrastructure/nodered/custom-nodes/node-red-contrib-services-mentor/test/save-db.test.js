"use strict";

const assert = require("node:assert/strict");
const { EventEmitter } = require("node:events");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const registerSaveDb = require("../services/save-db.js");
const {
  SQL,
  acquireDatabaseConnection,
  parseBoolean,
  stableStringify
} = registerSaveDb._test;

function databaseError(code, message = code) {
  const result = new Error(message);
  result.code = code;
  return result;
}

function createConnection(querySteps = [], options = {}) {
  const events = [];
  const calls = [];
  const remaining = [...querySteps];

  function finish(callback, failure, value) {
    queueMicrotask(() => callback(failure || null, value));
  }

  const connection = {
    events,
    calls,
    remaining,
    beginTransaction(callback) {
      events.push("begin");
      finish(callback, options.beginError);
    },
    commit(callback) {
      events.push("commit");
      finish(callback, options.commitError);
    },
    rollback(callback) {
      events.push("rollback");
      finish(callback, options.rollbackError);
    },
    release() {
      events.push("release");
      if (options.releaseError) {
        throw options.releaseError;
      }
    },
    query(sql, params, callback) {
      const step = remaining.shift();
      calls.push({ sql, params });
      events.push(`query:${calls.length}`);

      assert.ok(step, `Unexpected query: ${sql}`);
      if (step.match) {
        assert.match(sql, step.match);
      }
      if (step.inspect) {
        step.inspect({ sql, params });
      }
      finish(callback, step.error, step.rows);
    }
  };

  return connection;
}

function createPool(connection, options = {}) {
  return {
    calls: 0,
    getConnection(callback) {
      this.calls += 1;
      connection.events.push("pool.getConnection");
      queueMicrotask(() => callback(options.acquireError || null, connection));
    }
  };
}

function createHarness({
  strictDelta = true,
  interval = 5,
  connection = createConnection(),
  databaseConnection = connection,
  pool,
  mentorId = 7
} = {}) {
  let NodeConstructor;
  const statuses = [];
  const nodeErrors = [];
  const runtimeEvents = connection.events;

  const databaseConfig = new EventEmitter();
  databaseConfig.connected = true;
  databaseConfig.connection = databaseConnection;
  if (pool) {
    databaseConfig.pool = pool;
  }
  databaseConfig.connect = () => runtimeEvents.push("connect");

  const RED = {
    nodes: {
      createNode(node) {
        const emitter = new EventEmitter();
        node.on = emitter.on.bind(emitter);
        node.emit = emitter.emit.bind(emitter);
        node.removeListener = emitter.removeListener.bind(emitter);
        node.status = (status) => statuses.push(status);
        node.error = (failure, msg) => nodeErrors.push({ failure, msg });
        node.send = (msg) => {
          runtimeEvents.push("node.send");
          node.sent.push(msg);
        };
        node.sent = [];
      },
      getNode(id) {
        if (id === "database") {
          return databaseConfig;
        }
        if (id === "mentor") {
          return { mentor_id: mentorId };
        }
        return null;
      },
      registerType(name, constructor) {
        assert.equal(name, "SaveDB");
        NodeConstructor = constructor;
      }
    }
  };

  registerSaveDb(RED);
  const node = new NodeConstructor({
    mysql: "database",
    mentor: "mentor",
    interval,
    strictDelta
  });

  async function input(msg) {
    const sent = [];
    return new Promise((resolve) => {
      node.emit(
        "input",
        msg,
        (outgoing) => {
          runtimeEvents.push("send");
          sent.push(outgoing);
        },
        (failure) => {
          runtimeEvents.push(failure ? "done:error" : "done:ok");
          resolve({ failure, sent });
        }
      );
    });
  }

  return {
    connection,
    databaseConfig,
    input,
    node,
    nodeErrors,
    statuses
  };
}

function validPayload(overrides = {}) {
  return {
    code: "linea_1",
    time: 1_800_000,
    head: ["L1_CONTEO_1"],
    data: [3],
    ...overrides
  };
}

test("strictDelta inserts an immutable window and emits only after commit", async () => {
  const connection = createConnection([
    {
      match: /^INSERT INTO mqtt_lecturas /,
      rows: { affectedRows: 1 }
    },
    {
      match: /^INSERT INTO mqtt_snapshot /,
      rows: { affectedRows: 1 }
    }
  ]);
  const harness = createHarness({ connection });
  const msg = { payload: JSON.stringify(validPayload()) };

  const result = await harness.input(msg);

  assert.equal(result.failure, undefined);
  assert.deepEqual(result.sent, [msg]);
  assert.equal(connection.remaining.length, 0);
  assert.ok(connection.events.indexOf("commit") < connection.events.indexOf("send"));
  assert.deepEqual(connection.events.slice(-2), ["send", "done:ok"]);

  for (const call of connection.calls) {
    assert.doesNotMatch(call.sql, /linea_1|L1_CONTEO_1/);
  }
  assert.equal(connection.calls[0].sql, SQL.insertReading);
  assert.equal(connection.calls[0].params[1], "1800000");
  assert.equal(connection.calls[0].params[2], "linea_1");
  assert.equal(connection.calls[0].params[3], stableStringify(validPayload()));
  assert.match(connection.calls[1].sql, /ON DUPLICATE KEY UPDATE/);
});

test("strictDelta treats an identical UNIQUE(device,time) replay as success without updating history", async () => {
  const content = stableStringify(validPayload());
  const connection = createConnection([
    {
      match: /^INSERT INTO mqtt_lecturas /,
      error: databaseError("ER_DUP_ENTRY")
    },
    {
      match: /^SELECT mentor_id, content, result FROM mqtt_lecturas /,
      rows: [{ mentor_id: 7, content, result: 1 }]
    },
    {
      match: /^INSERT INTO mqtt_snapshot /,
      rows: { affectedRows: 2 }
    }
  ]);
  const harness = createHarness({ connection });

  const result = await harness.input({ payload: validPayload() });

  assert.equal(result.failure, undefined);
  assert.equal(result.sent.length, 1);
  assert.equal(connection.remaining.length, 0);
  assert.equal(connection.events.includes("commit"), true);
  assert.equal(connection.events.includes("rollback"), false);
  assert.deepEqual(connection.calls[1].params, ["linea_1", "1800000"]);
  assert.equal(
    connection.calls.some((call) => /UPDATE\s+mqtt_lecturas/i.test(call.sql)),
    false,
    "A replay must not overwrite content or reset sender ACK/retry columns"
  );
  assert.equal(
    harness.statuses.some((status) => status.text === "idempotent duplicate"),
    true
  );
});

test("strictDelta rejects a same-key/different-content collision without snapshot or downstream", async () => {
  const existing = stableStringify(validPayload({ data: [2] }));
  const connection = createConnection([
    {
      match: /^INSERT INTO mqtt_lecturas /,
      error: databaseError("ER_DUP_ENTRY")
    },
    {
      match: /^SELECT mentor_id, content, result FROM mqtt_lecturas /,
      rows: [{ mentor_id: 7, content: existing, result: 1 }]
    }
  ]);
  const harness = createHarness({ connection });

  const result = await harness.input({ payload: validPayload({ data: [3] }) });

  assert.equal(result.failure.code, "SAVE_DB_IDEMPOTENCY_CONFLICT");
  assert.equal(result.sent.length, 0);
  assert.equal(connection.remaining.length, 0);
  assert.equal(connection.calls.length, 2);
  assert.equal(connection.events.includes("commit"), false);
  assert.equal(connection.events.includes("rollback"), true);
});

test("strictDelta rejects replays whose immutable mentor_id or result differs", async (t) => {
  const content = stableStringify(validPayload());
  const cases = [
    {
      name: "different mentor_id",
      existing: { mentor_id: 8, content, result: 1 }
    },
    {
      name: "different result",
      existing: { mentor_id: 7, content, result: 0 }
    }
  ];

  for (const item of cases) {
    await t.test(item.name, async () => {
      const connection = createConnection([
        {
          match: /^INSERT INTO mqtt_lecturas /,
          error: databaseError("ER_DUP_ENTRY")
        },
        {
          match: /^SELECT mentor_id, content, result FROM mqtt_lecturas /,
          rows: [item.existing]
        }
      ]);
      const harness = createHarness({ connection });

      const result = await harness.input({ payload: validPayload() });

      assert.equal(result.failure.code, "SAVE_DB_IDEMPOTENCY_CONFLICT");
      assert.equal(result.sent.length, 0);
      assert.equal(connection.events.includes("rollback"), true);
      assert.equal(
        connection.calls.some((call) => /UPDATE\s+mqtt_lecturas/i.test(call.sql)),
        false,
        "A conflicting replay must never alter ACK/history fields"
      );
    });
  }
});

test("SaveDB prefers a dedicated pool connection and releases it before downstream", async () => {
  const dedicated = createConnection([
    {
      match: /^INSERT INTO mqtt_lecturas /,
      rows: { affectedRows: 1 }
    },
    {
      match: /^INSERT INTO mqtt_snapshot /,
      rows: { affectedRows: 1 }
    }
  ]);
  const shared = createConnection();
  const pool = createPool(dedicated);
  const harness = createHarness({
    connection: dedicated,
    databaseConnection: shared,
    pool
  });

  const result = await harness.input({ payload: validPayload() });

  assert.equal(result.failure, undefined);
  assert.equal(result.sent.length, 1);
  assert.equal(pool.calls, 1);
  assert.equal(shared.calls.length, 0);
  assert.equal(dedicated.events.filter((event) => event === "release").length, 1);
  assert.ok(dedicated.events.indexOf("commit") < dedicated.events.indexOf("release"));
  assert.ok(dedicated.events.indexOf("release") < dedicated.events.indexOf("send"));
});

test("SaveDB releases its dedicated connection after rollback", async () => {
  const dedicated = createConnection([
    {
      match: /^INSERT INTO mqtt_lecturas /,
      rows: { affectedRows: 1 }
    },
    {
      match: /^INSERT INTO mqtt_snapshot /,
      error: databaseError("ER_LOCK_WAIT_TIMEOUT")
    }
  ]);
  const pool = createPool(dedicated);
  const harness = createHarness({ connection: dedicated, pool });

  const result = await harness.input({ payload: validPayload() });

  assert.equal(result.failure.code, "SAVE_DB_PERSISTENCE_FAILED");
  assert.equal(result.sent.length, 0);
  assert.equal(dedicated.events.filter((event) => event === "release").length, 1);
  assert.ok(dedicated.events.indexOf("rollback") < dedicated.events.indexOf("release"));
});

test("SaveDB does not fall back to the shared connection when pool acquisition fails", async () => {
  const dedicated = createConnection();
  const shared = createConnection();
  const pool = createPool(dedicated, {
    acquireError: databaseError("POOL_CONNLIMIT")
  });
  const harness = createHarness({
    connection: dedicated,
    databaseConnection: shared,
    pool
  });

  const result = await harness.input({ payload: validPayload() });

  assert.equal(result.failure.code, "SAVE_DB_CONNECTION_ACQUIRE_FAILED");
  assert.equal(result.sent.length, 0);
  assert.equal(pool.calls, 1);
  assert.equal(shared.calls.length, 0);
  assert.equal(dedicated.events.includes("begin"), false);
  assert.equal(dedicated.events.includes("release"), false);
});

test("the legacy shared-connection fallback serializes transactions", async () => {
  const databaseConfig = {
    connection: createConnection()
  };
  const first = await acquireDatabaseConnection(databaseConfig);
  let secondResolved = false;
  const secondPromise = acquireDatabaseConnection(databaseConfig).then((lease) => {
    secondResolved = true;
    return lease;
  });

  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(secondResolved, false);

  first.release();
  const second = await secondPromise;
  assert.equal(secondResolved, true);

  second.release();
  second.release();
});

test("strictDelta rejects Timeout and NaN readings without querying the snapshot", async (t) => {
  for (const badValue of ["Timeout!", "NaN"]) {
    await t.test(badValue, async () => {
      const connection = createConnection();
      const harness = createHarness({ connection });

      const result = await harness.input({
        payload: validPayload({ data: [badValue] })
      });

      assert.equal(result.failure.code, "SAVE_DB_INVALID_DELTA_READING");
      assert.equal(result.sent.length, 0);
      assert.equal(connection.calls.length, 0);
      assert.equal(connection.events.includes("begin"), false);
    });
  }
});

test("strictDelta validates JSON and required envelope fields before touching MySQL", async (t) => {
  const cases = [
    {
      name: "invalid JSON",
      payload: '{"code":',
      expected: "SAVE_DB_INVALID_JSON"
    },
    {
      name: "missing time",
      payload: { code: "linea_1", head: ["L1_CONTEO_1"], data: [1] },
      expected: "SAVE_DB_INVALID_TIME"
    },
    {
      name: "missing data",
      payload: { code: "linea_1", time: 300_000 },
      expected: "SAVE_DB_INVALID_DATA"
    },
    {
      name: "device exceeds VARCHAR(50)",
      payload: {
        code: "x".repeat(51),
        time: 300_000,
        head: ["L1_CONTEO_1"],
        data: [1]
      },
      expected: "SAVE_DB_INVALID_CODE"
    },
    {
      name: "head/data mismatch",
      payload: {
        code: "linea_1",
        time: 300_000,
        head: ["L1_CONTEO_1", "OTHER"],
        data: [1]
      },
      expected: "SAVE_DB_INVALID_HEAD"
    }
  ];

  for (const item of cases) {
    await t.test(item.name, async () => {
      const connection = createConnection();
      const harness = createHarness({ connection });
      const result = await harness.input({ payload: item.payload });

      assert.equal(result.failure.code, item.expected);
      assert.equal(result.sent.length, 0);
      assert.equal(connection.calls.length, 0);
    });
  }
});

test("strictDelta rolls back and does not emit when snapshot persistence fails", async () => {
  const connection = createConnection([
    {
      match: /^INSERT INTO mqtt_lecturas /,
      rows: { affectedRows: 1 }
    },
    {
      match: /^INSERT INTO mqtt_snapshot /,
      error: databaseError("ER_LOCK_WAIT_TIMEOUT")
    }
  ]);
  const harness = createHarness({ connection });

  const result = await harness.input({ payload: validPayload() });

  assert.equal(result.failure.code, "SAVE_DB_PERSISTENCE_FAILED");
  assert.equal(result.sent.length, 0);
  assert.equal(connection.events.includes("rollback"), true);
  assert.equal(connection.events.includes("commit"), false);
});

test("strictDelta does not emit after an uncertain commit failure", async () => {
  const connection = createConnection(
    [
      {
        match: /^INSERT INTO mqtt_lecturas /,
        rows: { affectedRows: 1 }
      },
      {
        match: /^INSERT INTO mqtt_snapshot /,
        rows: { affectedRows: 1 }
      }
    ],
    { commitError: databaseError("PROTOCOL_CONNECTION_LOST") }
  );
  const harness = createHarness({ connection });

  const result = await harness.input({ payload: validPayload() });

  assert.equal(result.failure.code, "SAVE_DB_PERSISTENCE_FAILED");
  assert.equal(result.sent.length, 0);
  assert.equal(connection.events.includes("commit"), true);
  assert.equal(connection.events.includes("rollback"), true);
});

test("legacy mode retains snapshot fallback but waits for durable insert before emitting", async () => {
  const oldSnapshot = {
    code: "linea_1",
    time: 1_500_000,
    head: ["L1_CONTEO_1"],
    data: [4]
  };
  let insertedContent;
  const connection = createConnection([
    {
      match: /^SELECT content FROM mqtt_snapshot /,
      rows: [{ content: JSON.stringify(oldSnapshot) }]
    },
    {
      match: /^INSERT INTO mqtt_lecturas /,
      inspect({ params }) {
        insertedContent = JSON.parse(params[3]);
      },
      rows: { affectedRows: 1 }
    }
  ]);
  const harness = createHarness({ connection, strictDelta: false });

  const result = await harness.input({
    payload: JSON.stringify(validPayload({ data: ["Timeout!"] }))
  });

  assert.equal(result.failure, undefined);
  assert.equal(result.sent.length, 1);
  assert.equal(insertedContent.time, validPayload().time);
  assert.deepEqual(insertedContent.data, [4]);
  assert.ok(connection.events.indexOf("commit") < connection.events.indexOf("send"));
});

test("strictDelta can be activated by boolean-like values from flows JSON", () => {
  assert.equal(parseBoolean(true), true);
  assert.equal(parseBoolean("true"), true);
  assert.equal(parseBoolean(1), true);
  assert.equal(parseBoolean("1"), true);
  assert.equal(parseBoolean(false), false);
  assert.equal(parseBoolean("false"), false);
});

test("editor metadata preserves SaveDB compatibility and defaults strictDelta to false", () => {
  const html = fs.readFileSync(
    path.join(__dirname, "..", "services", "save-db.html"),
    "utf8"
  );

  assert.match(html, /registerType\(['"]SaveDB['"]/);
  assert.match(html, /strictDelta:\s*\{\s*value:\s*false\s*\}/);
  assert.match(html, /id="node-input-strictDelta"/);
});
