"use strict";

const assert = require("node:assert/strict");
const crypto = require("node:crypto");
const {EventEmitter} = require("node:events");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const hot = require("./hot-deploy-official-counter.js");
const patch = require("./patch-official-textile-count.js");
const drain = require("./backlog-drain.js");

function auditFlows() {
  const filename = process.env.NODERED_AUDIT_PATH || path.resolve(
    __dirname,
    "../../../nodered-flows.audit.json"
  );
  return JSON.parse(fs.readFileSync(filename, "utf8"));
}

function disabledCandidate() {
  return hot.prepareCandidate(auditFlows(), patch, 1).flows;
}

function sampleRow(overrides = {}) {
  const time = overrides.time || "1785467100000";
  const device =
    overrides.device || "ART_ATLAS_MAQUINA_1_PRODUCCION";
  const content = overrides.content || JSON.stringify({
    code: device,
    time,
    head: ["L1_CONTEO_1"],
    data: [4],
  });
  return drain.validateRow({
    id: 10,
    device,
    time,
    content,
    alarm: 0,
    mentor_id: 478,
    status: 0,
    restful: 0,
    ...overrides,
  }, Number(time) + 1_000);
}

class MemoryEvents {
  constructor() {
    this.events = [];
  }

  append(event) {
    this.events.push(event);
  }
}

test("descifra el formato oficial AES-256-CTR de Node-RED", () => {
  const secret = "test-secret";
  const credentials = {
    "7ba0277c.2fa1b8": {user: "mentor", password: "private"},
  };
  const iv = Buffer.from("00112233445566778899aabbccddeeff", "hex");
  const key = crypto.createHash("sha256").update(secret).digest();
  const cipher = crypto.createCipheriv("aes-256-ctr", key, iv);
  const body =
    cipher.update(JSON.stringify(credentials), "utf8", "base64") +
    cipher.final("base64");
  const encrypted = `${iv.toString("hex")}${body}`;

  assert.deepEqual(
    drain.decryptCredentialBlob(secret, encrypted),
    credentials
  );
  assert.throws(
    () => drain.decryptCredentialBlob("", encrypted),
    /material cifrado/
  );
});

test("solo acepta el baseline exacto con ambos Senders deshabilitados", () => {
  const flows = disabledCandidate();
  const topology = drain.validateAuditedTopology(flows);
  assert.equal(topology.hash, drain.EXPECTED_FLOW_HASH);
  assert.equal(topology.sender.d, true);

  const enabled = structuredClone(flows);
  delete enabled.find((node) => node.id === drain.IDS.sender).d;
  assert.throws(
    () => drain.validateAuditedTopology(enabled),
    /baseline auditado|Sender principal/
  );

  const changedBroker = structuredClone(flows);
  changedBroker.find((node) => node.id === drain.IDS.broker).broker =
    "broker.invalid";
  assert.throws(
    () => drain.validateAuditedTopology(changedBroker),
    /baseline auditado|broker externo/
  );
});

test("CLI exige confirmacion literal antes de permitir drain", () => {
  const args = drain.parseArgs([
    "--action", "drain",
    "--output-dir", "/out/run-1",
    "--confirm-live", drain.LIVE_CONFIRMATION,
    "--concurrency", "2",
  ]);
  assert.equal(args.concurrency, 2);
  assert.equal(args.outputDir, "/out/run-1");
  assert.throws(
    () => drain.parseArgs([
      "--action", "drain",
      "--output-dir", "/out/run-1",
    ]),
    /exige --confirm-live/
  );
  assert.throws(
    () => drain.parseArgs([
      "--action", "drain",
      "--output-dir", "relative",
      "--confirm-live", drain.LIVE_CONFIRMATION,
    ]),
    /debe ser absoluto/
  );
  assert.throws(
    () => drain.parseArgs([
      "--action", "drain",
      "--output-dir", "/",
      "--confirm-live", drain.LIVE_CONFIRMATION,
    ]),
    /contenido dentro de \/out/
  );
  assert.throws(
    () => drain.parseArgs([
      "--action", "drain",
      "--output-dir", "/data",
      "--confirm-live", drain.LIVE_CONFIRMATION,
    ]),
    /contenido dentro de \/out/
  );
});

test("valida contrato, mentor, estado, fecha y devices auditados", () => {
  assert.equal(sampleRow().mentor_id, 478);
  assert.throws(
    () => sampleRow({mentor_id: 0}),
    /mentor_id inesperado/
  );
  assert.throws(
    () => sampleRow({status: 2}),
    /estado local inesperado/
  );
  assert.throws(
    () => sampleRow({
      device: "OTRO",
      content: JSON.stringify({
        code: "OTRO",
        time: "1785467100000",
        head: [],
        data: [],
      }),
    }),
    /device no auditado/
  );
  assert.throws(
    () => sampleRow({
      content: JSON.stringify({
        code: "ART_ATLAS_MAQUINA_1_PRODUCCION",
        time: "1785467100999",
        head: [],
        data: [],
      }),
    }),
    /no cumple el contrato/
  );
  assert.throws(
    () => sampleRow({
      content: JSON.stringify({
        code: "ART_ATLAS_MAQUINA_1_PRODUCCION",
        time: "1785467100000",
        head: ["descripcion"],
        data: ["operario's"],
      }),
    }),
    /Receiver legacy no parametrizado/
  );
});

test("backup NDJSON exacto y durable precede al primer publish", async () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "mentor-drain-"));
  const backup = new drain.BackupJournal(directory);
  const events = new MemoryEvents();
  const row = sampleRow();
  const order = [];
  try {
    assert.equal(backup.appendRows([row]), 1);
    const hashBefore = drain.rowRecordHash(row);
    const persisted = drain.loadBackup(backup.filename);
    assert.equal(persisted.byId.get(row.id), hashBefore);

    const transport = {
      async send(envelope) {
        backup.assertExact(row);
        order.push(`publish:${envelope.device}:${envelope.time}`);
        return {brokerPuback: true, applicationAck: true};
      },
    };
    let updated = 0;
    const delivered = await drain.processBatch([row], {
      backupJournal: backup,
      eventJournal: events,
      connection: {},
      transport,
      rateLimiter: {async acquire() {}},
      options: {
        concurrency: 1,
        maxAttempts: 1,
        retryBaseMs: 1,
        ackTimeoutMs: 100,
      },
      markDeliveredFn: async () => {
        updated += 1;
        order.push("update");
        return {affectedRows: 1, reconciled: false};
      },
      assertCurrentRowFn: async () => {
        order.push("revalidate");
        return row;
      },
      wait: async () => {},
    });
    assert.equal(delivered, 1);
    assert.equal(updated, 1);
    assert.deepEqual(order, [
      "revalidate",
      `publish:${row.device}:${row.time}`,
      "update",
    ]);
    assert.deepEqual(
      events.events.map((event) => event.type),
      ["publish_attempt", "exact_ack", "local_delivered"]
    );
  } finally {
    backup.close();
    fs.rmSync(directory, {recursive: true, force: true});
  }
});

test("sin ACK nunca ejecuta UPDATE y agota reintentos", async () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "mentor-drain-"));
  const backup = new drain.BackupJournal(directory);
  const events = new MemoryEvents();
  const row = sampleRow();
  backup.appendRows([row]);
  let sends = 0;
  let updates = 0;
  try {
    await assert.rejects(
      () => drain.processBatch([row], {
        backupJournal: backup,
        eventJournal: events,
        connection: {},
        transport: {
          async send() {
            sends += 1;
            throw new Error("timeout");
          },
        },
        rateLimiter: {async acquire() {}},
        options: {
          concurrency: 1,
          maxAttempts: 3,
          retryBaseMs: 1,
          ackTimeoutMs: 10,
        },
        markDeliveredFn: async () => {
          updates += 1;
          return {affectedRows: 1, reconciled: false};
        },
        assertCurrentRowFn: async () => row,
        wait: async () => {},
      }),
      /sin ACK exacto tras 3 intentos/
    );
    assert.equal(sends, 3);
    assert.equal(updates, 0);
  } finally {
    backup.close();
    fs.rmSync(directory, {recursive: true, force: true});
  }
});

test("MQTT se suscribe antes de publicar y solo acepta ACK exacto", async () => {
  class FakeClient extends EventEmitter {
    constructor() {
      super();
      this.order = [];
    }

    subscribe(topic, options, callback) {
      this.order.push(`subscribe:${topic}:${options.qos}`);
      callback(null, [{topic, qos: options.qos}]);
    }

    publish(topic, payload, options, callback) {
      this.order.push(`publish:${topic}:${options.qos}`);
      this.lastEnvelope = JSON.parse(payload);
      setImmediate(callback);
    }

    end(force, options, callback) {
      this.order.push(`end:${force}`);
      callback();
    }
  }

  const client = new FakeClient();
  const mqtt = {
    connect() {
      setImmediate(() => client.emit("connect"));
      return client;
    },
  };
  const transport = new drain.MqttAckTransport(
    mqtt,
    {broker: "52.11.253.25", port: "1883"},
    {},
    478,
    {log() {}}
  );
  await transport.connect();
  const row = sampleRow();
  const sent = transport.send(drain.envelopeForRow(row), 1_000);

  client.emit(
    "message",
    "mentor_answer_478",
    Buffer.from(JSON.stringify({
      device: row.device,
      time: "1785467100001",
      status: 2,
    }))
  );
  let settled = false;
  sent.then(() => { settled = true; });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(settled, false);

  client.emit(
    "message",
    "mentor_answer_478",
    Buffer.from(JSON.stringify({
      device: row.device,
      time: row.time,
      status: "2",
    }))
  );
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(settled, false);

  client.emit(
    "message",
    "mentor_answer_478",
    Buffer.from(JSON.stringify({
      device: row.device,
      time: row.time,
      status: 2,
    }))
  );
  const result = await sent;
  assert.equal(result.brokerPuback, true);
  assert.equal(result.applicationAck, true);
  assert.deepEqual(client.order.slice(0, 2), [
    "subscribe:mentor_answer_478:2",
    "publish:mentor_data:1",
  ]);
  await transport.close();
});

test("UPDATE esta parametrizado y limitado a id, mentor, clave y hash", async () => {
  const row = sampleRow();
  let captured = null;
  const connection = {
    query(statement, params, callback) {
      captured = {sql: statement.sql, timeout: statement.timeout, params};
      callback(null, {affectedRows: 1});
    },
  };
  const result = await drain.markDelivered(
    connection,
    row,
    1,
    async () => {}
  );
  assert.equal(result.affectedRows, 1);
  assert.match(captured.sql, /status = 2/);
  assert.match(captured.sql, /restful = 1/);
  assert.match(captured.sql, /mentor_id = \?/);
  assert.match(captured.sql, /SHA2\(content, 256\) = \?/);
  assert.equal(captured.timeout, 30_000);
  assert.equal(captured.sql.includes(row.device), false);
  assert.deepEqual(captured.params.slice(0, 4), [
    row.id,
    row.mentor_id,
    row.device,
    row.time,
  ]);
  assert.equal(captured.params[4], crypto
    .createHash("sha256")
    .update(row.content)
    .digest("hex"));
});

test("las esperas retiradas no acumulan listeners en AbortSignal", async () => {
  const controller = new AbortController();
  const originalAdd = controller.signal.addEventListener.bind(
    controller.signal
  );
  const originalRemove = controller.signal.removeEventListener.bind(
    controller.signal
  );
  let listeners = 0;
  controller.signal.addEventListener = (...args) => {
    listeners += 1;
    return originalAdd(...args);
  };
  controller.signal.removeEventListener = (...args) => {
    listeners -= 1;
    return originalRemove(...args);
  };

  for (let index = 0; index < 25; index += 1) {
    await drain.sleep(1, controller.signal);
  }
  assert.equal(listeners, 0);
});

test("lock global MariaDB excluye otro drainer aunque use otro directorio", async () => {
  const connection = {
    query(statement, params, callback) {
      assert.match(statement.sql, /GET_LOCK/);
      assert.deepEqual(params, ["mentor_backlog_drain_478_v1"]);
      callback(null, [{acquired: 0}]);
    },
  };
  await assert.rejects(
    () => drain.acquireDatabaseLock(connection),
    /otro drenador conserva el lock global/
  );
});

test("resume conserva evidencia y trunca solo una linea NDJSON parcial", () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "mentor-resume-"));
  const filename = path.join(directory, "backlog-before.ndjson");
  const complete = `${JSON.stringify({
    type: "mqtt_lectura_backup_v1",
    backed_up_at: new Date().toISOString(),
    row: {
      ...sampleRow(),
      content_sha256: crypto
        .createHash("sha256")
        .update(sampleRow().content)
        .digest("hex"),
    },
  })}\n`;
  fs.writeFileSync(filename, `${complete}{"type":"incomplete"`, "utf8");

  drain.recoverTrailingPartial(filename, "backup");
  assert.equal(fs.readFileSync(filename, "utf8"), complete);
  assert.equal(
    fs.readdirSync(directory).some((name) => name.includes(".partial-")),
    true
  );
  const loaded = drain.loadBackup(filename);
  assert.equal(loaded.byId.size, 1);
  fs.rmSync(directory, {recursive: true, force: true});
});
