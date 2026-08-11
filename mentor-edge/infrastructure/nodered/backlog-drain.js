"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const TOOL_VERSION = "mentor-backlog-drain:v1";
const LIVE_CONFIRMATION = "ACK_STATUS_2_VALIDATED";
const EXPECTED_FLOW_HASH =
  "f65c87507232c7bdd4cc4aca7a0beec607b54cd60e704d65ff634569ff5e9e88";
const DATABASE_LOCK_NAME = "mentor_backlog_drain_478_v1";

const IDS = Object.freeze({
  sender: "b94f3dfa216658f3",
  secondarySender: "48f1dea3eb373643",
  interval: "90544e7288574184",
  broker: "85b6c2e7.5ca2f",
  mysql: "7ba0277c.2fa1b8",
  mentor: "41b1600ff22164b4",
});

const EXPECTED = Object.freeze({
  brokerHost: "52.11.253.25",
  brokerPort: "1883",
  databaseHost: "host.docker.internal",
  databasePort: "3306",
  databaseName: "mentor",
  mentorId: 478,
  publishTopic: "mentor_data",
});

const ALLOWED_DEVICES = new Set([
  "ART_ATLAS_LINEA_1_ALARMAS",
  "ART_ATLAS_LINEA_1_PRODUCCION",
  "ART_ATLAS_MAQUINA_1_PRODUCCION",
]);

const DEFAULTS = Object.freeze({
  concurrency: 4,
  ratePerSecond: 4,
  batchSize: 100,
  ackTimeoutMs: 20_000,
  maxAttempts: 5,
  retryBaseMs: 1_000,
  maxRuntimeSeconds: 7_200,
  maxSnapshotRows: 100_000,
});

const SELECT_PENDING_SQL = `
SELECT id, device, CAST(time AS CHAR) AS time, content,
       COALESCE(alarm, 0) AS alarm, mentor_id, status, restful
  FROM mqtt_lecturas
 WHERE mentor_id = ?
   AND status = 0
 ORDER BY CAST(time AS UNSIGNED), BINARY device, id
 LIMIT ?`;

const UPDATE_DELIVERED_SQL = `
UPDATE mqtt_lecturas
   SET status = 2,
       restful = 1
 WHERE id = ?
   AND mentor_id = ?
   AND BINARY device = BINARY ?
   AND time = ?
   AND status = 0
   AND SHA2(content, 256) = ?`;

function fail(message) {
  throw new Error(message);
}

function sha256(value) {
  return crypto.createHash("sha256").update(value).digest("hex");
}

function stableValue(value) {
  if (Array.isArray(value)) return value.map(stableValue);
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.keys(value)
        .sort()
        .map((key) => [key, stableValue(value[key])])
    );
  }
  return value;
}

function stableJson(value) {
  return JSON.stringify(stableValue(value));
}

function flowHash(flows) {
  if (!Array.isArray(flows)) fail("flows.json no contiene un arreglo");
  const ordered = [...flows].sort((left, right) =>
    String(left && left.id).localeCompare(String(right && right.id))
  );
  return sha256(stableJson(ordered));
}

function fileSha256(filename) {
  return sha256(fs.readFileSync(filename));
}

function readJson(filename) {
  try {
    return JSON.parse(fs.readFileSync(filename, "utf8"));
  } catch (error) {
    fail(`JSON invalido en ${filename}: ${error.message}`);
  }
}

function nodeMap(flows) {
  const result = new Map();
  for (const node of flows) {
    if (!node || typeof node.id !== "string" || node.id === "") {
      fail("flows.json contiene un nodo sin ID");
    }
    if (result.has(node.id)) fail(`ID de nodo duplicado: ${node.id}`);
    result.set(node.id, node);
  }
  return result;
}

function validateAuditedTopology(flows) {
  const hash = flowHash(flows);
  if (hash !== EXPECTED_FLOW_HASH) {
    fail(`flows.json no es el baseline auditado (${hash})`);
  }

  const byId = nodeMap(flows);
  const sender = byId.get(IDS.sender);
  const secondary = byId.get(IDS.secondarySender);
  const interval = byId.get(IDS.interval);
  const broker = byId.get(IDS.broker);
  const database = byId.get(IDS.mysql);
  const mentor = byId.get(IDS.mentor);

  if (
    !sender ||
    sender.type !== "Sender" ||
    sender.name !== "ART_ATLAS_SENDER_GENERAL" ||
    sender.d !== true ||
    sender.mentor !== IDS.mentor ||
    sender.mysql !== IDS.mysql ||
    sender.mqtt !== IDS.broker ||
    String(sender.priority) !== "0"
  ) {
    fail("el Sender principal no permanece deshabilitado y auditado");
  }
  if (
    !secondary ||
    secondary.type !== "Sender" ||
    secondary.d !== true ||
    secondary.mqtt !== IDS.broker
  ) {
    fail("el Sender secundario no permanece deshabilitado");
  }
  if (
    !interval ||
    interval.type !== "Interval" ||
    String(interval.interval) !== "300" ||
    Number(interval.initialdefer) !== 0 ||
    stableJson(interval.wires) !== stableJson([[IDS.sender]])
  ) {
    fail("el Interval del Sender difiere de la auditoria");
  }
  if (
    !broker ||
    broker.type !== "mqtt-broker" ||
    String(broker.broker) !== EXPECTED.brokerHost ||
    String(broker.port) !== EXPECTED.brokerPort ||
    broker.usetls !== false ||
    String(broker.protocolVersion) !== "4"
  ) {
    fail("el broker externo difiere de la auditoria");
  }
  if (
    !database ||
    database.type !== "MySQLdb" ||
    database.host !== EXPECTED.databaseHost ||
    String(database.port) !== EXPECTED.databasePort ||
    database.db !== EXPECTED.databaseName
  ) {
    fail("MariaDB del Sender difiere de la auditoria");
  }
  if (
    !mentor ||
    mentor.type !== "mentor-config" ||
    Number(mentor.mentor_id) !== EXPECTED.mentorId
  ) {
    fail("mentor_id difiere de la auditoria");
  }

  const brokerUsers = flows
    .filter((node) => node && node.mqtt === IDS.broker)
    .map((node) => node.id)
    .sort();
  const expectedUsers = [IDS.sender, IDS.secondarySender].sort();
  if (stableJson(brokerUsers) !== stableJson(expectedUsers)) {
    fail("el broker auditado tiene consumidores inesperados");
  }

  return {hash, sender, secondary, interval, broker, database, mentor};
}

function decryptCredentialBlob(secret, encrypted) {
  if (
    typeof secret !== "string" ||
    secret.length === 0 ||
    typeof encrypted !== "string" ||
    encrypted.length <= 32 ||
    !/^[0-9a-fA-F]{32}/.test(encrypted)
  ) {
    fail("material cifrado de Node-RED invalido");
  }
  try {
    const iv = Buffer.from(encrypted.slice(0, 32), "hex");
    const key = crypto.createHash("sha256").update(secret).digest();
    const decipher = crypto.createDecipheriv("aes-256-ctr", key, iv);
    const plaintext =
      decipher.update(encrypted.slice(32), "base64", "utf8") +
      decipher.final("utf8");
    return JSON.parse(plaintext);
  } catch (error) {
    fail(`no se pudieron descifrar credenciales Node-RED: ${error.message}`);
  }
}

function loadRuntime(dataDir) {
  const flowsFile = path.join(dataDir, "flows.json");
  const runtimeFile = path.join(dataDir, ".config.runtime.json");
  const credentialsFile = path.join(dataDir, "flows_cred.json");
  const flows = readJson(flowsFile);
  const topology = validateAuditedTopology(flows);
  const runtime = readJson(runtimeFile);
  const encrypted = readJson(credentialsFile);
  const credentials = decryptCredentialBlob(
    runtime._credentialSecret,
    encrypted.$
  );
  const databaseCredentials = credentials[IDS.mysql] || {};
  const brokerCredentials = credentials[IDS.broker] || {};

  if (
    typeof databaseCredentials.user !== "string" ||
    databaseCredentials.user === "" ||
    typeof databaseCredentials.password !== "string"
  ) {
    fail("credenciales MariaDB ausentes");
  }
  for (const name of ["user", "password"]) {
    if (
      Object.hasOwn(brokerCredentials, name) &&
      typeof brokerCredentials[name] !== "string"
    ) {
      fail("credenciales MQTT invalidas");
    }
  }

  return {
    ...topology,
    databaseCredentials,
    brokerCredentials,
    credentialBlobSha256: sha256(encrypted.$),
  };
}

function loadModule(candidates) {
  const errors = [];
  for (const candidate of candidates) {
    try {
      return require(candidate);
    } catch (error) {
      errors.push(`${candidate}:${error.code || error.message}`);
    }
  }
  fail(`modulo no disponible (${errors.join("; ")})`);
}

function query(connection, sql, params, timeoutMs = 30_000) {
  return new Promise((resolve, reject) => {
    connection.query({sql, timeout: timeoutMs}, params, (error, rows) => {
      if (error) reject(error);
      else resolve(rows);
    });
  });
}

function connectDatabase(mysql, options) {
  const connection = mysql.createConnection(options);
  return new Promise((resolve, reject) => {
    connection.connect((error) => {
      if (error) reject(error);
      else resolve(connection);
    });
  });
}

function closeDatabase(connection) {
  if (!connection) return Promise.resolve();
  return new Promise((resolve) => {
    let finished = false;
    const finish = () => {
      if (finished) return;
      finished = true;
      clearTimeout(timer);
      resolve();
    };
    const timer = setTimeout(() => {
      if (typeof connection.destroy === "function") connection.destroy();
      finish();
    }, 5_000);
    connection.end(finish);
  });
}

function sleep(milliseconds, signal) {
  if (signal && signal.aborted) {
    return Promise.reject(new Error("interrumpido por senal"));
  }
  return new Promise((resolve, reject) => {
    let timer = null;
    const cleanup = () => {
      if (signal) signal.removeEventListener("abort", onAbort);
    };
    const onElapsed = () => {
      cleanup();
      resolve();
    };
    const onAbort = () => {
      clearTimeout(timer);
      cleanup();
      reject(new Error("interrumpido por senal"));
    };
    timer = setTimeout(onElapsed, milliseconds);
    if (signal) signal.addEventListener("abort", onAbort, {once: true});
  });
}

function boundedInteger(raw, name, minimum, maximum) {
  if (!/^\d+$/.test(String(raw))) {
    fail(`${name} debe ser un entero`);
  }
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < minimum || value > maximum) {
    fail(`${name} debe estar entre ${minimum} y ${maximum}`);
  }
  return value;
}

function parseArgs(argv) {
  const result = {
    action: "",
    dataDir: "/data",
    outputDir: "",
    confirmation: "",
    resume: false,
    ...DEFAULTS,
  };
  const valueOptions = new Map([
    ["--action", "action"],
    ["--data-dir", "dataDir"],
    ["--output-dir", "outputDir"],
    ["--confirm-live", "confirmation"],
    ["--concurrency", "concurrency"],
    ["--rate-per-second", "ratePerSecond"],
    ["--batch-size", "batchSize"],
    ["--ack-timeout-ms", "ackTimeoutMs"],
    ["--max-attempts", "maxAttempts"],
    ["--retry-base-ms", "retryBaseMs"],
    ["--max-runtime-seconds", "maxRuntimeSeconds"],
    ["--max-snapshot-rows", "maxSnapshotRows"],
  ]);

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token === "--resume") {
      result.resume = true;
      continue;
    }
    if (!valueOptions.has(token)) fail(`opcion desconocida: ${token}`);
    if (index + 1 >= argv.length) fail(`falta valor para ${token}`);
    result[valueOptions.get(token)] = argv[index + 1];
    index += 1;
  }

  if (!["preflight", "drain"].includes(result.action)) {
    fail("--action debe ser preflight o drain");
  }
  if (!path.isAbsolute(result.dataDir)) {
    fail("--data-dir debe ser absoluto");
  }
  if (result.action === "drain") {
    if (!path.isAbsolute(result.outputDir)) {
      fail("--output-dir debe ser absoluto para drain");
    }
    const portableOutput = path.posix.resolve(
      result.outputDir.replaceAll("\\", "/")
    );
    if (
      portableOutput !== "/out" &&
      !portableOutput.startsWith("/out/")
    ) {
      fail("--output-dir debe estar contenido dentro de /out");
    }
    if (result.confirmation !== LIVE_CONFIRMATION) {
      fail(`drain exige --confirm-live ${LIVE_CONFIRMATION}`);
    }
  } else if (result.resume) {
    fail("--resume solo aplica a drain");
  }

  result.concurrency = boundedInteger(
    result.concurrency, "--concurrency", 1, 16
  );
  result.ratePerSecond = boundedInteger(
    result.ratePerSecond, "--rate-per-second", 1, 50
  );
  result.batchSize = boundedInteger(
    result.batchSize, "--batch-size", 1, 1_000
  );
  result.ackTimeoutMs = boundedInteger(
    result.ackTimeoutMs, "--ack-timeout-ms", 1_000, 120_000
  );
  result.maxAttempts = boundedInteger(
    result.maxAttempts, "--max-attempts", 1, 20
  );
  result.retryBaseMs = boundedInteger(
    result.retryBaseMs, "--retry-base-ms", 100, 30_000
  );
  result.maxRuntimeSeconds = boundedInteger(
    result.maxRuntimeSeconds, "--max-runtime-seconds", 60, 86_400
  );
  result.maxSnapshotRows = boundedInteger(
    result.maxSnapshotRows, "--max-snapshot-rows", 1, 1_000_000
  );
  return result;
}

function normalizeRow(source) {
  const content = Buffer.isBuffer(source.content)
    ? source.content.toString("utf8")
    : String(source.content);
  return {
    id: Number(source.id),
    device: String(source.device),
    time: String(source.time),
    content,
    alarm: Number(source.alarm),
    mentor_id: Number(source.mentor_id),
    status: Number(source.status),
    restful: Number(source.restful),
  };
}

function validateRow(source, now = Date.now()) {
  const row = normalizeRow(source);
  if (!Number.isSafeInteger(row.id) || row.id <= 0) {
    fail("fila pendiente con id invalido");
  }
  if (!ALLOWED_DEVICES.has(row.device)) {
    fail(`device no auditado en id=${row.id}: ${row.device}`);
  }
  if (!/^\d{13}$/.test(row.time)) {
    fail(`time invalido en id=${row.id}`);
  }
  if (Number(row.time) > now + 5 * 60_000) {
    fail(`time futuro en id=${row.id}`);
  }
  if (row.mentor_id !== EXPECTED.mentorId) {
    fail(`mentor_id inesperado en id=${row.id}`);
  }
  if (row.status !== 0 || ![0, 1].includes(row.restful)) {
    fail(`estado local inesperado en id=${row.id}`);
  }
  if (![0, 1].includes(row.alarm)) {
    fail(`alarm invalido en id=${row.id}`);
  }

  let content;
  try {
    content = JSON.parse(row.content);
  } catch {
    fail(`content JSON invalido en id=${row.id}`);
  }
  if (
    !content ||
    typeof content !== "object" ||
    Array.isArray(content) ||
    String(content.code) !== row.device ||
    String(content.time) !== row.time ||
    !Array.isArray(content.head) ||
    !Array.isArray(content.data) ||
    content.head.length !== content.data.length
  ) {
    fail(`content no cumple el contrato en id=${row.id}`);
  }
  if (/['\\\u0000\n\r\u001a]/.test(row.content)) {
    fail(
      `content inseguro para el Receiver legacy no parametrizado en id=${row.id}`
    );
  }
  return row;
}

function rowKey(row) {
  return `${row.device}\u0000${row.time}`;
}

function rowRecord(row) {
  return {
    id: row.id,
    device: row.device,
    time: row.time,
    content: row.content,
    alarm: row.alarm,
    mentor_id: row.mentor_id,
    status: row.status,
    restful: row.restful,
    content_sha256: sha256(row.content),
  };
}

function rowRecordHash(row) {
  return sha256(stableJson(rowRecord(row)));
}

function writeAll(fd, text) {
  const buffer = Buffer.from(text, "utf8");
  let offset = 0;
  while (offset < buffer.length) {
    offset += fs.writeSync(fd, buffer, offset, buffer.length - offset);
  }
}

function loadBackup(filename) {
  const byId = new Map();
  const byKey = new Map();
  if (!fs.existsSync(filename)) return {byId, byKey};
  const raw = fs.readFileSync(filename, "utf8");
  if (raw !== "" && !raw.endsWith("\n")) {
    fail("backup NDJSON termina en una linea incompleta");
  }
  for (const [index, line] of raw.split("\n").entries()) {
    if (line === "") continue;
    let entry;
    try {
      entry = JSON.parse(line);
    } catch {
      fail(`backup NDJSON invalido en linea ${index + 1}`);
    }
    if (
      entry.type !== "mqtt_lectura_backup_v1" ||
      typeof entry.backed_up_at !== "string" ||
      !entry.row
    ) {
      fail(`registro de backup invalido en linea ${index + 1}`);
    }
    const row = normalizeRow(entry.row);
    const hash = rowRecordHash(row);
    if (entry.row.content_sha256 !== sha256(row.content)) {
      fail(`hash de content invalido en backup id=${row.id}`);
    }
    if (byId.has(row.id) || byKey.has(rowKey(row))) {
      fail(`fila duplicada en backup id=${row.id}`);
    }
    byId.set(row.id, hash);
    byKey.set(rowKey(row), row.id);
  }
  return {byId, byKey};
}

function recoverTrailingPartial(filename, label) {
  if (!fs.existsSync(filename)) return;
  const raw = fs.readFileSync(filename);
  if (raw.length === 0 || raw[raw.length - 1] === 0x0a) return;
  const lastNewline = raw.lastIndexOf(0x0a);
  const completeLength = lastNewline === -1 ? 0 : lastNewline + 1;
  const partial = raw.subarray(completeLength);
  const evidence = `${filename}.partial-${Date.now()}`;
  fs.writeFileSync(evidence, partial, {mode: 0o600});
  fs.chmodSync(evidence, 0o600);
  fs.truncateSync(filename, completeLength);
  const fd = fs.openSync(filename, "r+");
  fs.fsyncSync(fd);
  fs.closeSync(fd);
  if (label === "events" && completeLength > 0) {
    for (const [index, line] of fs
      .readFileSync(filename, "utf8")
      .split("\n")
      .entries()) {
      if (line === "") continue;
      try {
        JSON.parse(line);
      } catch {
        fail(`journal de eventos corrupto en linea ${index + 1}`);
      }
    }
  }
}

class BackupJournal {
  constructor(outputDir, resume = false) {
    this.filename = path.join(outputDir, "backlog-before.ndjson");
    if (!resume && fs.existsSync(this.filename)) {
      fail(`backup ya existe: ${this.filename}`);
    }
    if (resume) recoverTrailingPartial(this.filename, "backup");
    const loaded = loadBackup(this.filename);
    this.byId = loaded.byId;
    this.byKey = loaded.byKey;
    this.fd = fs.openSync(this.filename, "a", 0o600);
    fs.chmodSync(this.filename, 0o600);
  }

  appendRows(rows) {
    const additions = [];
    for (const row of rows) {
      const hash = rowRecordHash(row);
      if (this.byId.has(row.id)) {
        if (this.byId.get(row.id) !== hash) {
          fail(`fila id=${row.id} cambio desde su backup`);
        }
        continue;
      }
      if (this.byKey.has(rowKey(row))) {
        fail(`clave device+time duplicada antes del backup: ${rowKey(row)}`);
      }
      additions.push({
        type: "mqtt_lectura_backup_v1",
        backed_up_at: new Date().toISOString(),
        row: rowRecord(row),
      });
    }
    if (additions.length > 0) {
      writeAll(
        this.fd,
        additions.map((entry) => `${JSON.stringify(entry)}\n`).join("")
      );
      fs.fsyncSync(this.fd);

      const verified = loadBackup(this.filename);
      for (const row of rows) {
        if (verified.byId.get(row.id) !== rowRecordHash(row)) {
          fail(`backup durable no contiene id=${row.id}`);
        }
      }
      this.byId = verified.byId;
      this.byKey = verified.byKey;
    }
    return additions.length;
  }

  assertExact(row) {
    if (this.byId.get(row.id) !== rowRecordHash(row)) {
      fail(`se intento publicar id=${row.id} sin backup durable exacto`);
    }
  }

  get count() {
    return this.byId.size;
  }

  close() {
    if (this.fd !== null) {
      fs.closeSync(this.fd);
      this.fd = null;
    }
  }
}

class EventJournal {
  constructor(outputDir, resume = false) {
    this.filename = path.join(outputDir, "drain-events.ndjson");
    if (!resume && fs.existsSync(this.filename)) {
      fail(`journal ya existe: ${this.filename}`);
    }
    if (resume) recoverTrailingPartial(this.filename, "events");
    this.fd = fs.openSync(this.filename, "a", 0o600);
    fs.chmodSync(this.filename, 0o600);
  }

  append(event) {
    writeAll(
      this.fd,
      `${JSON.stringify({
        at: new Date().toISOString(),
        ...event,
      })}\n`
    );
    fs.fsyncSync(this.fd);
  }

  close() {
    if (this.fd !== null) {
      fs.closeSync(this.fd);
      this.fd = null;
    }
  }
}

function writePrivateJson(filename, value) {
  const temporary = `${filename}.tmp-${process.pid}`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, {
    encoding: "utf8",
    mode: 0o600,
  });
  const fd = fs.openSync(temporary, "r+");
  fs.fsyncSync(fd);
  fs.closeSync(fd);
  fs.renameSync(temporary, filename);
  fs.chmodSync(filename, 0o600);
}

function prepareRunDirectory(outputDir, resume) {
  const portableOutput = path.posix.resolve(outputDir.replaceAll("\\", "/"));
  if (
    portableOutput !== "/out" &&
    !portableOutput.startsWith("/out/")
  ) {
    fail("output-dir fuera de /out");
  }
  const portableParts = portableOutput.split("/").filter(Boolean);
  let portableCursor = "";
  for (const part of portableParts) {
    portableCursor += `/${part}`;
    if (
      fs.existsSync(portableCursor) &&
      fs.lstatSync(portableCursor).isSymbolicLink()
    ) {
      fail(`output-dir contiene symlink: ${portableCursor}`);
    }
  }

  const exists = fs.existsSync(outputDir);
  if (!exists && resume) fail(`no existe run para reanudar: ${outputDir}`);
  if (!exists) fs.mkdirSync(outputDir, {recursive: true, mode: 0o700});
  if (!exists) fs.chmodSync(outputDir, 0o700);

  const manifestFile = path.join(outputDir, "manifest.json");
  const lockFile = path.join(outputDir, "run.lock");
  if (fs.existsSync(lockFile)) {
    if (!resume) {
      fail(`run bloqueado; comprueba que no siga activo: ${lockFile}`);
    }
    fs.renameSync(
      lockFile,
      `${lockFile}.stale-${Date.now()}`
    );
  }
  if (!resume && fs.existsSync(manifestFile)) {
    fail(`output-dir ya contiene un run: ${outputDir}`);
  }
  if (resume && !fs.existsSync(manifestFile)) {
    fail(`manifest ausente para reanudar: ${manifestFile}`);
  }
  const lockFd = fs.openSync(lockFile, "wx", 0o600);
  writeAll(
    lockFd,
    `${JSON.stringify({
      toolVersion: TOOL_VERSION,
      pid: process.pid,
      startedAt: new Date().toISOString(),
    })}\n`
  );
  fs.fsyncSync(lockFd);
  fs.closeSync(lockFd);
  return {manifestFile, lockFile};
}

function removeLock(lockFile) {
  if (lockFile && fs.existsSync(lockFile)) fs.unlinkSync(lockFile);
}

function envelopeForRow(row) {
  return {
    device: row.device,
    time: row.time,
    content: row.content,
    alarm: row.alarm,
    mentor_id: row.mentor_id,
  };
}

function ackKey(device, time) {
  return `${String(device)}\u0000${String(time)}`;
}

function parseExactAck(topic, payload, mentorId, expectedKeys) {
  if (topic !== `mentor_answer_${mentorId}`) return null;
  let ack;
  try {
    ack = JSON.parse(Buffer.isBuffer(payload)
      ? payload.toString("utf8")
      : String(payload));
  } catch {
    return null;
  }
  if (
    !ack ||
    ack.status !== 2 ||
    typeof ack.device !== "string" ||
    !/^\d{13}$/.test(String(ack.time))
  ) {
    return null;
  }
  const key = ackKey(ack.device, ack.time);
  if (expectedKeys && !expectedKeys.has(key)) return null;
  return {key, device: ack.device, time: String(ack.time), status: 2};
}

class MqttAckTransport {
  constructor(mqtt, broker, credentials, mentorId, io = console) {
    this.mqtt = mqtt;
    this.broker = broker;
    this.credentials = credentials;
    this.mentorId = mentorId;
    this.io = io;
    this.client = null;
    this.connected = false;
    this.closed = false;
    this.inflight = new Map();
  }

  connect() {
    if (this.client) fail("MQTT transport ya inicializado");
    return new Promise((resolve, reject) => {
      const options = {
        clientId: `mentor_backlog_${process.pid}_${Date.now()
          .toString()
          .slice(-8)}`,
        protocolVersion: 4,
        clean: true,
        reconnectPeriod: 0,
        connectTimeout: 8_000,
        keepalive: 30,
        resubscribe: false,
      };
      if (this.credentials.user) options.username = this.credentials.user;
      if (this.credentials.password) {
        options.password = this.credentials.password;
      }
      const url = `mqtt://${this.broker.broker}:${this.broker.port}`;
      const client = this.mqtt.connect(url, options);
      this.client = client;
      let initial = true;
      const initialTimeout = setTimeout(() => {
        failInitial(new Error("timeout esperando CONNACK/SUBACK MQTT"));
        client.end(true);
      }, 15_000);

      const failInitial = (error) => {
        if (!initial) return;
        initial = false;
        clearTimeout(initialTimeout);
        reject(error);
      };
      client.once("connect", () => {
        if (!initial) return;
        const topic = `mentor_answer_${this.mentorId}`;
        client.subscribe(topic, {qos: 2}, (error, granted) => {
          if (!initial) return;
          if (error) return failInitial(error);
          const grant = Array.isArray(granted)
            ? granted.find((item) => item.topic === topic)
            : null;
          if (!grant || grant.qos === 128) {
            return failInitial(
              new Error(`suscripcion ACK rechazada: ${JSON.stringify(granted)}`)
            );
          }
          this.connected = true;
          initial = false;
          clearTimeout(initialTimeout);
          this.io.log(`MQTT_ACK_SUBSCRIBED topic=${topic} qos=${grant.qos}`);
          resolve();
        });
      });
      client.on("message", (topic, payload) => {
        const ack = parseExactAck(
          topic,
          payload,
          this.mentorId,
          new Set(this.inflight.keys())
        );
        if (!ack) return;
        const pending = this.inflight.get(ack.key);
        if (!pending) return;
        pending.applicationAck = true;
        pending.ack = ack;
        this.maybeResolve(ack.key);
      });
      client.on("error", (error) => {
        failInitial(error);
        this.rejectAll(error);
      });
      client.on("close", () => {
        this.connected = false;
        if (initial) failInitial(new Error("conexion MQTT cerrada"));
        else if (!this.closed) {
          this.rejectAll(new Error("conexion MQTT cerrada"));
        }
      });
    });
  }

  maybeResolve(key) {
    const pending = this.inflight.get(key);
    if (!pending || !pending.brokerPuback || !pending.applicationAck) return;
    clearTimeout(pending.timer);
    this.inflight.delete(key);
    pending.resolve({
      brokerPuback: true,
      applicationAck: true,
      ack: pending.ack,
    });
  }

  rejectAll(error) {
    for (const [key, pending] of this.inflight) {
      clearTimeout(pending.timer);
      this.inflight.delete(key);
      pending.reject(error);
    }
  }

  send(envelope, timeoutMs) {
    if (!this.connected || !this.client) {
      return Promise.reject(new Error("MQTT no conectado"));
    }
    const key = ackKey(envelope.device, envelope.time);
    if (this.inflight.has(key)) {
      return Promise.reject(new Error(`envio duplicado en vuelo: ${key}`));
    }
    return new Promise((resolve, reject) => {
      const pending = {
        resolve,
        reject,
        brokerPuback: false,
        applicationAck: false,
        ack: null,
        timer: null,
      };
      pending.timer = setTimeout(() => {
        this.inflight.delete(key);
        reject(new Error("timeout esperando PUBACK y ACK exacto"));
      }, timeoutMs);
      this.inflight.set(key, pending);

      this.client.publish(
        EXPECTED.publishTopic,
        JSON.stringify(envelope),
        {qos: 1, retain: false},
        (error) => {
          if (!this.inflight.has(key)) return;
          if (error) {
            clearTimeout(pending.timer);
            this.inflight.delete(key);
            reject(error);
            return;
          }
          pending.brokerPuback = true;
          this.maybeResolve(key);
        }
      );
    });
  }

  close() {
    if (!this.client) return Promise.resolve();
    this.closed = true;
    this.rejectAll(new Error("MQTT transport cerrado"));
    return new Promise((resolve) => {
      this.client.end(true, {}, resolve);
    });
  }
}

class RateLimiter {
  constructor(ratePerSecond, now = () => Date.now(), wait = sleep) {
    this.intervalMs = 1_000 / ratePerSecond;
    this.nextAt = 0;
    this.now = now;
    this.wait = wait;
  }

  async acquire(signal) {
    const current = this.now();
    const scheduled = Math.max(current, this.nextAt);
    this.nextAt = scheduled + this.intervalMs;
    const delay = scheduled - current;
    if (delay > 0) await this.wait(delay, signal);
  }
}

async function readPendingRows(connection, maximum, now = Date.now()) {
  const rows = await query(
    connection,
    SELECT_PENDING_SQL,
    [EXPECTED.mentorId, maximum + 1]
  );
  if (rows.length > maximum) {
    fail(
      `cola supera --max-snapshot-rows (${maximum}); no se envio ninguna fila`
    );
  }
  const normalized = rows.map((row) => validateRow(row, now));
  const ids = new Set();
  const keys = new Set();
  for (const row of normalized) {
    if (ids.has(row.id) || keys.has(rowKey(row))) {
      fail(`consulta devolvio fila duplicada id=${row.id}`);
    }
    ids.add(row.id);
    keys.add(rowKey(row));
  }
  return normalized;
}

async function pendingSummary(connection) {
  return query(
    connection,
    `SELECT device, COUNT(*) AS pending,
            MIN(CAST(time AS UNSIGNED)) AS oldest_time,
            MAX(CAST(time AS UNSIGNED)) AS newest_time
       FROM mqtt_lecturas
      WHERE mentor_id = ?
        AND status = 0
      GROUP BY device
      ORDER BY BINARY device`,
    [EXPECTED.mentorId]
  );
}

async function deliveredState(connection, row) {
  const rows = await query(
    connection,
    `SELECT status, restful, SHA2(content, 256) AS content_sha256
       FROM mqtt_lecturas
      WHERE id = ?
        AND mentor_id = ?
        AND BINARY device = BINARY ?
        AND time = ?`,
    [row.id, row.mentor_id, row.device, row.time]
  );
  if (rows.length !== 1) return null;
  return {
    status: Number(rows[0].status),
    restful: Number(rows[0].restful),
    contentSha256: String(rows[0].content_sha256),
  };
}

async function acquireDatabaseLock(connection) {
  const rows = await query(
    connection,
    "SELECT GET_LOCK(?, 0) AS acquired",
    [DATABASE_LOCK_NAME]
  );
  if (rows.length !== 1 || Number(rows[0].acquired) !== 1) {
    fail("otro drenador conserva el lock global de MariaDB");
  }
}

async function releaseDatabaseLock(connection) {
  try {
    await query(
      connection,
      "SELECT RELEASE_LOCK(?) AS released",
      [DATABASE_LOCK_NAME]
    );
  } catch {
    // The server releases advisory locks when the connection closes.
  }
}

async function assertCurrentPendingRow(connection, expected, now = Date.now()) {
  const rows = await query(
    connection,
    `SELECT id, device, CAST(time AS CHAR) AS time, content,
            COALESCE(alarm, 0) AS alarm, mentor_id, status, restful
       FROM mqtt_lecturas
      WHERE id = ?
        AND mentor_id = ?
        AND BINARY device = BINARY ?
        AND time = ?`,
    [expected.id, expected.mentor_id, expected.device, expected.time]
  );
  if (rows.length !== 1) {
    fail(`fila desaparecio antes de publish id=${expected.id}`);
  }
  const current = validateRow(rows[0], now);
  if (rowRecordHash(current) !== rowRecordHash(expected)) {
    fail(`fila cambio despues del backup id=${expected.id}`);
  }
  return current;
}

async function assertLiveRuntimeDisabled(
  apiUrl = "http://127.0.0.1:1880"
) {
  const response = await fetch(`${apiUrl}/flows`, {
    method: "GET",
    headers: {
      Accept: "application/json",
      "Node-RED-API-Version": "v2",
    },
    signal: AbortSignal.timeout(10_000),
  });
  if (!response.ok) {
    fail(`Node-RED /flows devolvio HTTP ${response.status}`);
  }
  const body = await response.json();
  const flows = Array.isArray(body) ? body : body.flows;
  validateAuditedTopology(flows);
}

async function markDelivered(
  connection,
  row,
  attempts = 3,
  wait = sleep,
  signal
) {
  const contentHash = sha256(row.content);
  let lastError = null;
  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    try {
      const result = await query(
        connection,
        UPDATE_DELIVERED_SQL,
        [row.id, row.mentor_id, row.device, row.time, contentHash]
      );
      if (Number(result.affectedRows) === 1) {
        return {affectedRows: 1, reconciled: false};
      }
      const state = await deliveredState(connection, row);
      if (
        state &&
        state.status === 2 &&
        state.restful === 1 &&
        state.contentSha256 === contentHash
      ) {
        return {affectedRows: 0, reconciled: true};
      }
      fail(`UPDATE protegido no afecto id=${row.id}`);
    } catch (error) {
      lastError = error;
      if (attempt < attempts) {
        await wait(Math.min(5_000, 500 * 2 ** (attempt - 1)), signal);
      }
    }
  }
  throw lastError;
}

async function sendWithRetry(row, context) {
  const {
    transport,
    rateLimiter,
    eventJournal,
    options,
    signal,
    wait = sleep,
    assertCurrentRowFn = assertCurrentPendingRow,
  } = context;
  let lastError = null;
  for (let attempt = 1; attempt <= options.maxAttempts; attempt += 1) {
    if (signal && signal.aborted) fail("drenado interrumpido");
    await assertCurrentRowFn(context.connection, row);
    context.backupJournal.assertExact(row);
    await rateLimiter.acquire(signal);
    eventJournal.append({
      type: "publish_attempt",
      id: row.id,
      device: row.device,
      time: row.time,
      attempt,
    });
    try {
      const result = await transport.send(
        envelopeForRow(row),
        options.ackTimeoutMs
      );
      if (!result.brokerPuback || !result.applicationAck) {
        fail("transport devolvio exito sin ambos ACK");
      }
      eventJournal.append({
        type: "exact_ack",
        id: row.id,
        device: row.device,
        time: row.time,
        attempt,
        status: 2,
      });
      return {attempt};
    } catch (error) {
      lastError = error;
      eventJournal.append({
        type: "publish_failed",
        id: row.id,
        device: row.device,
        time: row.time,
        attempt,
        error: error.message,
      });
      if (attempt < options.maxAttempts) {
        const delay = Math.min(
          30_000,
          options.retryBaseMs * 2 ** (attempt - 1)
        );
        await wait(delay, signal);
      }
    }
  }
  throw new Error(
    `sin ACK exacto tras ${options.maxAttempts} intentos ` +
    `id=${row.id}: ${lastError && lastError.message}`
  );
}

async function processBatch(rows, context) {
  const {
    backupJournal,
    eventJournal,
    connection,
    options,
    signal,
    markDeliveredFn = markDelivered,
  } = context;
  for (const row of rows) backupJournal.assertExact(row);

  let cursor = 0;
  let firstError = null;
  let delivered = 0;
  const workers = Array.from(
    {length: Math.min(options.concurrency, rows.length)},
    async () => {
      while (!firstError) {
        if (signal && signal.aborted) {
          firstError = new Error("drenado interrumpido");
          return;
        }
        const index = cursor;
        cursor += 1;
        if (index >= rows.length) return;
        const row = rows[index];
        try {
          const ack = await sendWithRetry(row, context);
          const update = await markDeliveredFn(
            connection,
            row,
            3,
            context.wait || sleep,
            signal
          );
          eventJournal.append({
            type: "local_delivered",
            id: row.id,
            device: row.device,
            time: row.time,
            attempt: ack.attempt,
            status: 2,
            restful: 1,
            reconciled: update.reconciled,
          });
          delivered += 1;
        } catch (error) {
          firstError = error;
        }
      }
    }
  );
  await Promise.all(workers);
  if (firstError) throw firstError;
  return delivered;
}

async function runPreflight(connection, runtime, options, io = console) {
  await assertLiveRuntimeDisabled();
  const rows = await readPendingRows(
    connection,
    options.maxSnapshotRows
  );
  const summary = await pendingSummary(connection);
  const result = {
    toolVersion: TOOL_VERSION,
    flowHash: runtime.hash,
    mentorId: EXPECTED.mentorId,
    senderDisabled: true,
    pendingRows: rows.length,
    queues: summary.map((item) => ({
      device: item.device,
      pending: Number(item.pending),
      oldestTime: String(item.oldest_time),
      newestTime: String(item.newest_time),
    })),
    livePublishPerformed: false,
  };
  io.log(JSON.stringify(result, null, 2));
  return result;
}

async function runDrain(
  connection,
  runtime,
  mqtt,
  options,
  signal,
  io = console
) {
  let run = null;
  let databaseLockAcquired = false;
  let backupJournal = null;
  let eventJournal = null;
  let transport = null;
  let manifest = null;
  const started = Date.now();

  try {
    await acquireDatabaseLock(connection);
    databaseLockAcquired = true;
    run = prepareRunDirectory(options.outputDir, options.resume);
    backupJournal = new BackupJournal(options.outputDir, options.resume);
    eventJournal = new EventJournal(options.outputDir, options.resume);
    if (options.resume) {
      manifest = readJson(run.manifestFile);
      if (
        manifest.toolVersion !== TOOL_VERSION ||
        manifest.toolSha256 !== fileSha256(__filename) ||
        manifest.flowHash !== runtime.hash ||
        manifest.credentialBlobSha256 !== runtime.credentialBlobSha256 ||
        manifest.mentorId !== EXPECTED.mentorId ||
        manifest.status === "complete" ||
        stableJson(manifest.settings) !== stableJson({
          concurrency: options.concurrency,
          ratePerSecond: options.ratePerSecond,
          batchSize: options.batchSize,
          ackTimeoutMs: options.ackTimeoutMs,
          maxAttempts: options.maxAttempts,
          retryBaseMs: options.retryBaseMs,
          maxRuntimeSeconds: options.maxRuntimeSeconds,
        })
      ) {
        fail("manifest no es reanudable con este runtime");
      }
      manifest = {
        ...manifest,
        status: "running",
        resumedAt: new Date().toISOString(),
      };
    } else {
      manifest = {
        toolVersion: TOOL_VERSION,
        toolSha256: fileSha256(__filename),
        status: "running",
        startedAt: new Date().toISOString(),
        flowHash: runtime.hash,
        credentialBlobSha256: runtime.credentialBlobSha256,
        mentorId: EXPECTED.mentorId,
        publishTopic: EXPECTED.publishTopic,
        ackTopic: `mentor_answer_${EXPECTED.mentorId}`,
        settings: {
          concurrency: options.concurrency,
          ratePerSecond: options.ratePerSecond,
          batchSize: options.batchSize,
          ackTimeoutMs: options.ackTimeoutMs,
          maxAttempts: options.maxAttempts,
          retryBaseMs: options.retryBaseMs,
          maxRuntimeSeconds: options.maxRuntimeSeconds,
        },
        backedUpRows: 0,
        deliveredRowsThisRun: 0,
        batches: 0,
      };
    }
    writePrivateJson(run.manifestFile, manifest);

    let snapshot = await readPendingRows(
      connection,
      options.maxSnapshotRows
    );
    const initialPending = snapshot.length;
    const added = backupJournal.appendRows(snapshot);
    manifest.initialPending = manifest.initialPending ?? initialPending;
    manifest.backedUpRows = backupJournal.count;
    manifest.backupSha256 = fileSha256(backupJournal.filename);
    manifest.lastBackupAt = new Date().toISOString();
    writePrivateJson(run.manifestFile, manifest);
    io.log(
      `BACKUP_DURABLE rows=${snapshot.length} added=${added} ` +
      `sha256=${manifest.backupSha256}`
    );

    if (snapshot.length > 0) {
      await assertLiveRuntimeDisabled();
      transport = new MqttAckTransport(
        mqtt,
        runtime.broker,
        runtime.brokerCredentials,
        EXPECTED.mentorId,
        io
      );
      await transport.connect();
    }
    const rateLimiter = new RateLimiter(options.ratePerSecond);

    while (snapshot.length > 0) {
      if (signal && signal.aborted) fail("drenado interrumpido");
      if (Date.now() - started > options.maxRuntimeSeconds * 1_000) {
        fail("se alcanzo --max-runtime-seconds");
      }

      await assertLiveRuntimeDisabled();
      const batch = snapshot.slice(0, options.batchSize);
      const delivered = await processBatch(batch, {
        backupJournal,
        eventJournal,
        connection,
        transport,
        rateLimiter,
        options,
        signal,
      });
      manifest.deliveredRowsThisRun += delivered;
      manifest.batches += 1;
      manifest.lastDeliveredAt = new Date().toISOString();
      writePrivateJson(run.manifestFile, manifest);
      io.log(
        `BATCH_OK batch=${manifest.batches} delivered=${delivered} ` +
        `run_total=${manifest.deliveredRowsThisRun}`
      );

      snapshot = await readPendingRows(
        connection,
        options.maxSnapshotRows
      );
      const newlyBackedUp = backupJournal.appendRows(snapshot);
      manifest.backedUpRows = backupJournal.count;
      manifest.backupSha256 = fileSha256(backupJournal.filename);
      manifest.lastBackupAt = new Date().toISOString();
      writePrivateJson(run.manifestFile, manifest);
      if (newlyBackedUp > 0) {
        io.log(
          `NEW_ROWS_BACKED_UP added=${newlyBackedUp} ` +
          `total=${backupJournal.count}`
        );
      }
    }

    const finalSummary = await pendingSummary(connection);
    const finalPending = finalSummary.reduce(
      (total, item) => total + Number(item.pending),
      0
    );
    if (finalPending !== 0) fail(`cola final no es cero: ${finalPending}`);

    manifest = {
      ...manifest,
      status: "complete",
      completedAt: new Date().toISOString(),
      finalPending: 0,
      backupSha256: fileSha256(backupJournal.filename),
      eventsSha256: fileSha256(eventJournal.filename),
    };
    writePrivateJson(run.manifestFile, manifest);
    io.log(
      `BACKLOG_DRAIN_OK delivered=${manifest.deliveredRowsThisRun} ` +
      `pending=0 backup=${backupJournal.filename}`
    );
    return manifest;
  } catch (error) {
    if (manifest) {
      manifest = {
        ...manifest,
        status: "failed",
        failedAt: new Date().toISOString(),
        error: error.message,
        backedUpRows: backupJournal ? backupJournal.count : 0,
      };
      try {
        writePrivateJson(run.manifestFile, manifest);
      } catch {
        // Preserve the original operational error.
      }
    }
    throw error;
  } finally {
    if (transport) await transport.close();
    if (backupJournal) backupJournal.close();
    if (eventJournal) eventJournal.close();
    if (run) removeLock(run.lockFile);
    if (databaseLockAcquired) await releaseDatabaseLock(connection);
  }
}

async function main(argv = process.argv.slice(2), io = console) {
  const options = parseArgs(argv);
  const runtime = loadRuntime(options.dataDir);
  const mysql = loadModule([
    "/data/node_modules/mysql",
    "/data/node_modules/mysql2",
    "mysql",
  ]);
  const connection = await connectDatabase(mysql, {
    host:
      runtime.database.host === "host.docker.internal"
        ? "127.0.0.1"
        : runtime.database.host,
    port: Number(runtime.database.port || 3306),
    database: runtime.database.db,
    user: runtime.databaseCredentials.user,
    password: runtime.databaseCredentials.password,
    charset: runtime.database.charset || "UTF8MB4_GENERAL_CI",
    timezone: runtime.database.tz || "local",
    connectTimeout: 10_000,
  });

  const controller = new AbortController();
  const stop = () => {
    io.error("INTERRUPCION_SOLICITADA: deteniendo nuevos envios");
    controller.abort();
  };
  process.once("SIGINT", stop);
  process.once("SIGTERM", stop);

  try {
    if (options.action === "preflight") {
      return await runPreflight(connection, runtime, options, io);
    }
    const mqtt = loadModule([
      "/data/node_modules/mqtt",
      "/usr/src/node-red/node_modules/mqtt",
      "mqtt",
    ]);
    return await runDrain(
      connection,
      runtime,
      mqtt,
      options,
      controller.signal,
      io
    );
  } finally {
    process.removeListener("SIGINT", stop);
    process.removeListener("SIGTERM", stop);
    await closeDatabase(connection);
  }
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`BACKLOG_DRAIN_ERROR ${error.message}`);
    process.exitCode = 1;
  });
}

module.exports = {
  ALLOWED_DEVICES,
  BackupJournal,
  DEFAULTS,
  EXPECTED,
  EXPECTED_FLOW_HASH,
  EventJournal,
  IDS,
  LIVE_CONFIRMATION,
  MqttAckTransport,
  RateLimiter,
  SELECT_PENDING_SQL,
  TOOL_VERSION,
  UPDATE_DELIVERED_SQL,
  ackKey,
  acquireDatabaseLock,
  assertCurrentPendingRow,
  decryptCredentialBlob,
  envelopeForRow,
  flowHash,
  loadBackup,
  markDelivered,
  normalizeRow,
  parseArgs,
  parseExactAck,
  processBatch,
  recoverTrailingPartial,
  releaseDatabaseLock,
  rowRecordHash,
  sendWithRetry,
  sleep,
  validateAuditedTopology,
  validateRow,
};
