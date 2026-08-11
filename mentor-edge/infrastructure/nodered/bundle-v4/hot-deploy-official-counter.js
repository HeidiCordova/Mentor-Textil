#!/usr/bin/env node
"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const TOOL_VERSION = "mentor-textile-nodered-hot:v1";
const DEFAULT_API_URL = "http://127.0.0.1:1880";
const DEFAULT_CREDENTIALS = "/data/flows_cred.json";
const DEFAULT_DISK_FLOWS = "/data/flows.json";
const DEFAULT_SETTINGS = "/data/settings.js";
const EXTERNAL_BROKER_CONFIG_ID = "85b6c2e7.5ca2f";
const EXTERNAL_BROKER_HOST = "52.11.253.25";
const EXTERNAL_BROKER_PORT = "1883";
const EXTERNAL_SENDER_ID = "b94f3dfa216658f3";
const DISABLED_EXTERNAL_SENDER_ID = "48f1dea3eb373643";
const API_HEADERS = Object.freeze({
  "Node-RED-API-Version": "v2",
});
const DEPLOY_HEADERS = Object.freeze({
  ...API_HEADERS,
  "Content-Type": "application/json",
  "Node-RED-Deployment-Type": "nodes",
});

function fail(message) {
  throw new Error(message);
}

function parsePositiveInteger(raw, option, maximum = Number.MAX_SAFE_INTEGER) {
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value <= 0 || value > maximum) {
    fail(`${option} requiere un entero entre 1 y ${maximum}`);
  }
  return value;
}

function parseArgs(argv) {
  const result = {
    action: "",
    apiUrl: DEFAULT_API_URL,
    credentials: DEFAULT_CREDENTIALS,
    diskFlows: DEFAULT_DISK_FLOWS,
    settings: DEFAULT_SETTINGS,
    lineaId: 1,
    patch: "",
    stateDir: "",
    help: false,
  };
  const valueOptions = new Map([
    ["--action", "action"],
    ["--api-url", "apiUrl"],
    ["--credentials", "credentials"],
    ["--disk-flows", "diskFlows"],
    ["--settings", "settings"],
    ["--linea-id", "lineaId"],
    ["--patch", "patch"],
    ["--state-dir", "stateDir"],
  ]);

  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--help" || argument === "-h") {
      result.help = true;
      continue;
    }
    const equals = argument.indexOf("=");
    const option = equals === -1 ? argument : argument.slice(0, equals);
    if (!valueOptions.has(option)) {
      fail(`argumento desconocido: ${argument}`);
    }
    let value;
    if (equals !== -1) {
      value = argument.slice(equals + 1);
    } else {
      value = argv[index + 1];
      if (!value) fail(`${option} requiere un valor`);
      index += 1;
    }
    if (!value) fail(`${option} requiere un valor`);
    result[valueOptions.get(option)] = value;
  }

  if (result.help) return result;
  if (!["prepare", "deploy", "status", "rollback"].includes(result.action)) {
    fail("--action debe ser prepare, deploy, status o rollback");
  }
  if (!result.stateDir) fail("--state-dir es obligatorio");
  if (result.action === "prepare" || result.action === "deploy") {
    if (!result.patch) fail("--patch es obligatorio");
  }
  result.lineaId = parsePositiveInteger(
    result.lineaId,
    "--linea-id",
    10000
  );
  result.apiUrl = result.apiUrl.replace(/\/+$/, "");
  result.patch = result.patch ? path.resolve(result.patch) : "";
  result.stateDir = path.resolve(result.stateDir);
  result.credentials = path.resolve(result.credentials);
  result.diskFlows = path.resolve(result.diskFlows);
  result.settings = path.resolve(result.settings);
  return result;
}

function usage(io = console) {
  io.log([
    "Uso:",
    "  node hot-deploy-official-counter.js \\",
    "    --action prepare|deploy|status|rollback \\",
    "    --state-dir /data/mentor-hot-deploy/ESTADO \\",
    "    --patch /tmp/patch-official-textile-count.js",
    "",
    "prepare crea respaldo y candidato, pero no despliega.",
    "deploy usa rev optimista y Node-RED-Deployment-Type=nodes.",
    "rollback solo revierte si el runtime aun coincide con el candidato.",
    "El modo de parche siempre preserva SaveDB legacy byte por byte.",
    "El candidato deshabilita el Sender externo; rollback restaura su estado.",
  ].join("\n"));
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function canonicalize(value) {
  if (Array.isArray(value)) {
    return value.map(canonicalize);
  }
  if (value && typeof value === "object") {
    const result = {};
    for (const key of Object.keys(value).sort()) {
      result[key] = canonicalize(value[key]);
    }
    return result;
  }
  return value;
}

function stableJson(value) {
  return JSON.stringify(canonicalize(value));
}

function nodeMap(flows) {
  if (!Array.isArray(flows)) fail("flows no es un arreglo");
  const result = new Map();
  for (const node of flows) {
    if (!node || typeof node !== "object" || typeof node.id !== "string") {
      fail("todos los nodos deben tener id string");
    }
    if (result.has(node.id)) fail(`id duplicado: ${node.id}`);
    result.set(node.id, node);
  }
  return result;
}

function canonicalFlows(flows) {
  const byId = nodeMap(flows);
  const ordered = [...byId.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([, node]) => canonicalize(node));
  return JSON.stringify(ordered);
}

function sha256(value) {
  return crypto.createHash("sha256").update(value).digest("hex");
}

function flowHash(flows) {
  return sha256(canonicalFlows(flows));
}

function fileHash(filename) {
  return sha256(fs.readFileSync(filename));
}

function diffFlows(before, after) {
  const left = nodeMap(before);
  const right = nodeMap(after);
  const added = [];
  const removed = [];
  const modified = [];

  for (const [id, node] of right) {
    if (!left.has(id)) {
      added.push(id);
    } else if (stableJson(left.get(id)) !== stableJson(node)) {
      modified.push(id);
    }
  }
  for (const id of left.keys()) {
    if (!right.has(id)) removed.push(id);
  }
  return {
    added: added.sort(),
    removed: removed.sort(),
    modified: modified.sort(),
  };
}

function assertSameList(actual, expected, label) {
  const left = [...actual].sort();
  const right = [...expected].sort();
  if (JSON.stringify(left) !== JSON.stringify(right)) {
    fail(
      `${label} inesperados: actual=${JSON.stringify(left)} ` +
      `esperado=${JSON.stringify(right)}`
    );
  }
}

function isolateExternalSender(flows) {
  const byId = nodeMap(flows);
  const broker = byId.get(EXTERNAL_BROKER_CONFIG_ID);
  const sender = byId.get(EXTERNAL_SENDER_ID);
  const alreadyDisabled = byId.get(DISABLED_EXTERNAL_SENDER_ID);
  if (
    !broker ||
    broker.type !== "mqtt-broker" ||
    String(broker.broker) !== EXTERNAL_BROKER_HOST ||
    String(broker.port) !== EXTERNAL_BROKER_PORT
  ) {
    fail("configuracion del broker externo no coincide con el audit");
  }
  if (
    !sender ||
    sender.type !== "Sender" ||
    sender.mqtt !== EXTERNAL_BROKER_CONFIG_ID
  ) {
    fail("Sender externo activo no coincide con el audit");
  }
  if (
    !alreadyDisabled ||
    alreadyDisabled.type !== "Sender" ||
    alreadyDisabled.mqtt !== EXTERNAL_BROKER_CONFIG_ID ||
    alreadyDisabled.d !== true
  ) {
    fail("Sender externo secundario no esta deshabilitado como en el audit");
  }
  const users = flows
    .filter((node) =>
      node &&
      (node.mqtt === EXTERNAL_BROKER_CONFIG_ID ||
       node.broker === EXTERNAL_BROKER_CONFIG_ID)
    )
    .map((node) => node.id)
    .sort();
  assertSameList(
    users,
    [EXTERNAL_SENDER_ID, DISABLED_EXTERNAL_SENDER_ID],
    "usuarios del broker externo"
  );
  const changed = sender.d !== true;
  sender.d = true;
  for (const id of users) {
    if (byId.get(id).d !== true) {
      fail(`usuario externo no quedo deshabilitado: ${id}`);
    }
  }
  return changed;
}

function prepareCandidate(beforeFlows, patchModule, lineaId = 1) {
  if (!patchModule || typeof patchModule.patchFlows !== "function") {
    fail("modulo de parche invalido");
  }
  const inputSnapshot = stableJson(beforeFlows);
  const patched = patchModule.patchFlows(beforeFlows, {
    lineaId,
    preserveLegacySaveDb: true,
  });
  if (stableJson(beforeFlows) !== inputSnapshot) {
    fail("el parche muto el arreglo de entrada");
  }
  if (patched.saveDbMode !== patchModule.SAVE_DB_MODE_PRESERVE) {
    fail(`modo SaveDB inesperado: ${String(patched.saveDbMode)}`);
  }
  const senderIsolated = isolateExternalSender(patched.flows);
  if (!senderIsolated) {
    fail("Sender externo ya estaba deshabilitado; audit de entrada cambio");
  }

  const idempotent = patchModule.patchFlows(patched.flows, {
    lineaId,
    preserveLegacySaveDb: true,
  });
  if (idempotent.changes.length !== 0) {
    fail(
      `el candidato no es idempotente: ${idempotent.changes.join("; ")}`
    );
  }
  if (flowHash(idempotent.flows) !== flowHash(patched.flows)) {
    fail("la segunda pasada cambio el candidato");
  }
  if (isolateExternalSender(idempotent.flows)) {
    fail("aislamiento del Sender no es idempotente");
  }
  if (flowHash(idempotent.flows) !== flowHash(patched.flows)) {
    fail("la segunda pasada de aislamiento cambio el candidato");
  }

  const diff = diffFlows(beforeFlows, patched.flows);
  assertSameList(
    diff.added,
    Object.values(patchModule.IDS),
    "nodos agregados"
  );
  assertSameList(
    diff.modified,
    [
      patched.productionId,
      patchModule.AUDIT.productionInterval,
      patchModule.AUDIT.manualInject,
      patchModule.AUDIT.globalPublisher,
      EXTERNAL_SENDER_ID,
    ],
    "nodos modificados"
  );
  assertSameList(diff.removed, [], "nodos eliminados");

  const beforeById = nodeMap(beforeFlows);
  const afterById = nodeMap(patched.flows);
  const saveDbId = patchModule.AUDIT.saveDb;
  if (
    stableJson(beforeById.get(saveDbId)) !==
    stableJson(afterById.get(saveDbId))
  ) {
    fail("SaveDB no fue preservado exactamente");
  }
  if (patched.flows.length !== beforeFlows.length + diff.added.length) {
    fail("cantidad de nodos inesperada en candidato");
  }

  return {
    flows: patched.flows,
    changes: [
      ...patched.changes,
      "Sender externo deshabilitado durante el piloto",
    ],
    diff,
    productionId: patched.productionId,
    saveDbMode: patched.saveDbMode,
  };
}

function readJson(filename) {
  let parsed;
  try {
    parsed = JSON.parse(fs.readFileSync(filename, "utf8"));
  } catch (error) {
    fail(`${filename} no es JSON valido: ${error.message}`);
  }
  return parsed;
}

function writePrivate(filename, value) {
  const serialized = `${JSON.stringify(value, null, 2)}\n`;
  fs.writeFileSync(filename, serialized, {
    encoding: "utf8",
    flag: "wx",
    mode: 0o600,
  });
}

function replacePrivate(filename, value) {
  const temporary = `${filename}.tmp.${process.pid}`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, {
    encoding: "utf8",
    flag: "wx",
    mode: 0o600,
  });
  fs.renameSync(temporary, filename);
  fs.chmodSync(filename, 0o600);
}

function copyPrivate(source, destination) {
  fs.copyFileSync(source, destination, fs.constants.COPYFILE_EXCL);
  fs.chmodSync(destination, 0o600);
}

function loadEncryptedCredentials(filename) {
  const credentials = readJson(filename);
  if (
    !credentials ||
    typeof credentials !== "object" ||
    Array.isArray(credentials) ||
    Object.keys(credentials).length !== 1 ||
    typeof credentials.$ !== "string" ||
    credentials.$.length === 0
  ) {
    fail("flows_cred.json no contiene un blob cifrado completo");
  }
  return credentials;
}

function validateRuntime(value) {
  if (
    !value ||
    typeof value !== "object" ||
    typeof value.rev !== "string" ||
    value.rev.length === 0 ||
    !Array.isArray(value.flows)
  ) {
    fail("respuesta v2 de /flows invalida");
  }
  nodeMap(value.flows);
  return value;
}

async function requestJson(url, options = {}, timeoutMs = 10000) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  let response;
  let text;
  try {
    response = await fetch(url, {
      ...options,
      signal: controller.signal,
    });
    text = await response.text();
  } catch (error) {
    fail(`HTTP ${url} fallo: ${error.message}`);
  } finally {
    clearTimeout(timeout);
  }
  let body = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch (_) {
      body = text;
    }
  }
  if (!response.ok) {
    const detail = typeof body === "string"
      ? body.slice(0, 500)
      : JSON.stringify(body);
    fail(`HTTP ${response.status} ${url}: ${detail}`);
  }
  return {status: response.status, body};
}

async function getRuntime(apiUrl) {
  const response = await requestJson(`${apiUrl}/flows`, {
    headers: API_HEADERS,
  });
  return validateRuntime(response.body);
}

async function postRuntime(apiUrl, payload) {
  const response = await requestJson(`${apiUrl}/flows`, {
    method: "POST",
    headers: DEPLOY_HEADERS,
    body: JSON.stringify(payload),
  }, 30000);
  if (
    !response.body ||
    typeof response.body.rev !== "string" ||
    response.body.rev.length === 0
  ) {
    fail("POST /flows no devolvio un rev v2");
  }
  return response.body.rev;
}

async function waitForFlowHash(apiUrl, expectedHash) {
  let last = null;
  for (let attempt = 1; attempt <= 20; attempt += 1) {
    last = await getRuntime(apiUrl);
    if (flowHash(last.flows) === expectedHash) return last;
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  fail(
    `runtime no alcanzo hash esperado; actual=${flowHash(last.flows)}`
  );
}

async function observeTransition(
  apiUrl,
  sourceHash,
  targetHash,
  attempts = 30
) {
  let last = null;
  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    last = await getRuntime(apiUrl);
    const hash = flowHash(last.flows);
    if (hash === targetHash) return {match: "target", runtime: last};
    if (hash !== sourceHash) return {match: "other", runtime: last};
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  return {match: "source", runtime: last};
}

function loadState(stateDir) {
  const filename = path.join(stateDir, "state.json");
  if (!fs.existsSync(filename)) fail(`no existe ${filename}`);
  const state = readJson(filename);
  if (
    state.toolVersion !== TOOL_VERSION ||
    typeof state.toolSha256 !== "string" ||
    typeof state.beforeFlowHash !== "string" ||
    typeof state.rollbackFlowHash !== "string" ||
    typeof state.candidateFlowHash !== "string"
  ) {
    fail("state.json invalido o incompatible");
  }
  if (fileHash(__filename) !== state.toolSha256) {
    fail("hot-deploy.before.js fue alterado");
  }
  return state;
}

function updateState(stateDir, state) {
  replacePrivate(path.join(stateDir, "state.json"), state);
}

function loadPatch(filename) {
  if (!fs.existsSync(filename)) fail(`no existe parche: ${filename}`);
  delete require.cache[require.resolve(filename)];
  return require(filename);
}

async function prepare(args) {
  if (fs.existsSync(args.stateDir)) {
    fail(`state-dir ya existe: ${args.stateDir}`);
  }
  fs.mkdirSync(args.stateDir, {recursive: true, mode: 0o700});
  fs.chmodSync(args.stateDir, 0o700);

  for (const filename of [
    args.patch,
    args.credentials,
    args.diskFlows,
    args.settings,
  ]) {
    if (!fs.existsSync(filename)) fail(`archivo requerido ausente: ${filename}`);
  }

  const runtime = await getRuntime(args.apiUrl);
  const diskFlows = readJson(args.diskFlows);
  if (flowHash(diskFlows) !== flowHash(runtime.flows)) {
    fail("flows.json en disco no coincide con el runtime API");
  }
  const credentials = loadEncryptedCredentials(args.credentials);
  const patchModule = loadPatch(args.patch);
  const candidate = prepareCandidate(
    runtime.flows,
    patchModule,
    args.lineaId
  );
  const patchSha256 = fileHash(args.patch);
  const toolSha256 = fileHash(__filename);
  const beforeFlowHash = flowHash(runtime.flows);
  const candidateFlowHash = flowHash(candidate.flows);
  const rollbackFlows = clone(runtime.flows);
  if (!isolateExternalSender(rollbackFlows)) {
    fail("no se pudo construir rollback con Sender aislado");
  }
  const rollbackFlowHash = flowHash(rollbackFlows);
  const credentialBlobHash = sha256(credentials.$);

  copyPrivate(
    args.diskFlows,
    path.join(args.stateDir, "flows.disk.before.json")
  );
  copyPrivate(
    args.credentials,
    path.join(args.stateDir, "flows_cred.before.json")
  );
  copyPrivate(
    args.settings,
    path.join(args.stateDir, "settings.before.js")
  );
  copyPrivate(
    args.patch,
    path.join(args.stateDir, "patch.before.js")
  );
  copyPrivate(
    __filename,
    path.join(args.stateDir, "hot-deploy.before.js")
  );
  writePrivate(
    path.join(args.stateDir, "runtime.before.v2.json"),
    runtime
  );
  writePrivate(
    path.join(args.stateDir, "candidate.flows.json"),
    candidate.flows
  );
  writePrivate(
    path.join(args.stateDir, "rollback.safe.flows.json"),
    rollbackFlows
  );
  writePrivate(
    path.join(args.stateDir, "candidate.v2.json"),
    {
      rev: runtime.rev,
      flows: candidate.flows,
      credentials,
    }
  );
  const state = {
    toolVersion: TOOL_VERSION,
    toolSha256,
    status: "prepared",
    preparedAt: new Date().toISOString(),
    apiUrl: args.apiUrl,
    lineaId: args.lineaId,
    beforeRev: runtime.rev,
    beforeFlowHash,
    diskFlowHash: flowHash(diskFlows),
    rollbackFlowHash,
    candidateFlowHash,
    credentialBlobHash,
    patchSha256,
    saveDbMode: candidate.saveDbMode,
    changes: candidate.changes,
    diff: candidate.diff,
  };
  writePrivate(path.join(args.stateDir, "state.json"), state);
  return state;
}

function validatePreparedFiles(args, state) {
  if (fileHash(args.patch) !== state.patchSha256) {
    fail("hash del parche no coincide con el estado preparado");
  }
  const before = validateRuntime(
    readJson(path.join(args.stateDir, "runtime.before.v2.json"))
  );
  const candidateFlows = readJson(
    path.join(args.stateDir, "candidate.flows.json")
  );
  const rollbackFlows = readJson(
    path.join(args.stateDir, "rollback.safe.flows.json")
  );
  if (flowHash(before.flows) !== state.beforeFlowHash) {
    fail("respaldo runtime.before fue alterado");
  }
  if (flowHash(candidateFlows) !== state.candidateFlowHash) {
    fail("candidate.flows fue alterado");
  }
  if (flowHash(rollbackFlows) !== state.rollbackFlowHash) {
    fail("rollback.safe.flows fue alterado");
  }
  const patchModule = loadPatch(args.patch);
  const rebuilt = prepareCandidate(
    before.flows,
    patchModule,
    state.lineaId
  );
  if (flowHash(rebuilt.flows) !== state.candidateFlowHash) {
    fail("el parche actual no reproduce el candidato");
  }
  const credentials = loadEncryptedCredentials(
    path.join(args.stateDir, "flows_cred.before.json")
  );
  if (sha256(credentials.$) !== state.credentialBlobHash) {
    fail("blob cifrado de respaldo fue alterado");
  }
  return {before, candidateFlows, rollbackFlows, credentials};
}

async function deploy(args) {
  let state = loadState(args.stateDir);
  if (!["prepared", "deploying"].includes(state.status)) {
    fail(`estado no desplegable: ${state.status}`);
  }
  const prepared = validatePreparedFiles(args, state);
  const currentCredentials = loadEncryptedCredentials(args.credentials);
  if (sha256(currentCredentials.$) !== state.credentialBlobHash) {
    fail("credenciales activas cambiaron desde prepare");
  }
  const current = await getRuntime(args.apiUrl);
  const currentHash = flowHash(current.flows);
  if (currentHash === state.candidateFlowHash) {
    if (flowHash(readJson(args.diskFlows)) !== state.candidateFlowHash) {
      fail("runtime candidato pero flows.json en disco no coincide");
    }
    const updated = {
      ...state,
      status: "deployed",
      deployedAt: new Date().toISOString(),
      deployedRev: current.rev,
      reconciled: true,
    };
    replacePrivate(
      path.join(args.stateDir, "runtime.after.v2.json"),
      current
    );
    updateState(args.stateDir, updated);
    return updated;
  }
  if (currentHash !== state.beforeFlowHash) {
    fail("runtime cambio desde prepare; no se fuerza el despliegue");
  }
  if (current.rev !== state.beforeRev) {
    fail("rev cambio aunque los flows parezcan anteriores; se aborta");
  }

  state = {
    ...state,
    status: "deploying",
    deployStartedAt: state.deployStartedAt || new Date().toISOString(),
  };
  updateState(args.stateDir, state);

  let returnedRev = null;
  let postError = null;
  try {
    returnedRev = await postRuntime(args.apiUrl, {
      rev: current.rev,
      flows: prepared.candidateFlows,
      credentials: prepared.credentials,
    });
  } catch (error) {
    postError = error;
  }
  const observed = returnedRev
    ? {
        match: "target",
        runtime: await waitForFlowHash(
          args.apiUrl,
          state.candidateFlowHash
        ),
      }
    : await observeTransition(
        args.apiUrl,
        state.beforeFlowHash,
        state.candidateFlowHash
      );
  if (observed.match === "other") {
    fail("runtime quedo en estado ajeno tras intento de deploy");
  }
  if (observed.match === "source") {
    const reset = {
      ...state,
      status: "prepared",
      lastDeployError: postError ? postError.message : "sin transicion",
    };
    updateState(args.stateDir, reset);
    throw postError || new Error("deploy no produjo transicion");
  }
  const after = observed.runtime;
  if (flowHash(readJson(args.diskFlows)) !== state.candidateFlowHash) {
    fail("flows.json en disco no coincide con candidato desplegado");
  }
  if (returnedRev && after.rev !== returnedRev) {
    fail(
      `rev post-deploy inesperado: POST=${returnedRev} GET=${after.rev}`
    );
  }
  const afterCredentials = loadEncryptedCredentials(args.credentials);
  if (sha256(afterCredentials.$) !== state.credentialBlobHash) {
    fail("blob cifrado cambio durante el despliegue");
  }
  replacePrivate(
    path.join(args.stateDir, "runtime.after.v2.json"),
    after
  );
  const updated = {
    ...state,
    status: "deployed",
    deployedAt: new Date().toISOString(),
    deployedRev: after.rev,
    reconciled: Boolean(postError),
  };
  updateState(args.stateDir, updated);
  return updated;
}

async function status(args) {
  const state = loadState(args.stateDir);
  const current = await getRuntime(args.apiUrl);
  const currentHash = flowHash(current.flows);
  let matches = "other";
  if (currentHash === state.candidateFlowHash) matches = "candidate";
  if (currentHash === state.rollbackFlowHash) matches = "safe_rollback";
  if (currentHash === state.beforeFlowHash) matches = "before";
  return {
    stateStatus: state.status,
    currentRev: current.rev,
    currentFlowHash: currentHash,
    matches,
    beforeFlowHash: state.beforeFlowHash,
    rollbackFlowHash: state.rollbackFlowHash,
    candidateFlowHash: state.candidateFlowHash,
  };
}

async function rollback(args) {
  let state = loadState(args.stateDir);
  const rollbackFlows = readJson(
    path.join(args.stateDir, "rollback.safe.flows.json")
  );
  if (flowHash(rollbackFlows) !== state.rollbackFlowHash) {
    fail("rollback.safe.flows fue alterado");
  }
  const credentials = loadEncryptedCredentials(
    path.join(args.stateDir, "flows_cred.before.json")
  );
  if (sha256(credentials.$) !== state.credentialBlobHash) {
    fail("blob cifrado de rollback fue alterado");
  }
  const activeCredentials = loadEncryptedCredentials(args.credentials);
  if (sha256(activeCredentials.$) !== state.credentialBlobHash) {
    fail("credenciales activas cambiaron; rollback automatico abortado");
  }
  const current = await getRuntime(args.apiUrl);
  const currentHash = flowHash(current.flows);
  if (currentHash === state.rollbackFlowHash) {
    if (flowHash(readJson(args.diskFlows)) !== state.rollbackFlowHash) {
      fail("runtime rollback pero flows.json en disco no coincide");
    }
    const updated = {
      ...state,
      status: "rolled_back",
      rolledBackAt: state.rolledBackAt || new Date().toISOString(),
      rolledBackRev: current.rev,
    };
    updateState(args.stateDir, updated);
    return {...updated, alreadySafe: true};
  }
  if (
    currentHash !== state.candidateFlowHash &&
    currentHash !== state.beforeFlowHash
  ) {
    fail("runtime no admite rollback automatico seguro");
  }

  state = {
    ...state,
    status: "rolling_back",
    rollbackStartedAt: state.rollbackStartedAt || new Date().toISOString(),
  };
  updateState(args.stateDir, state);
  let returnedRev = null;
  let postError = null;
  try {
    returnedRev = await postRuntime(args.apiUrl, {
      rev: current.rev,
      flows: rollbackFlows,
      credentials,
    });
  } catch (error) {
    postError = error;
  }
  const observed = returnedRev
    ? {
        match: "target",
        runtime: await waitForFlowHash(args.apiUrl, state.rollbackFlowHash),
      }
    : await observeTransition(
        args.apiUrl,
        currentHash,
        state.rollbackFlowHash
      );
  if (observed.match === "other") {
    fail("runtime quedo en estado ajeno tras intento de rollback");
  }
  if (observed.match === "source") {
    const reset = {
      ...state,
      status: currentHash === state.candidateFlowHash
        ? "deployed"
        : "prepared",
      lastRollbackError: postError ? postError.message : "sin transicion",
    };
    updateState(args.stateDir, reset);
    throw postError || new Error("rollback no produjo transicion");
  }
  const after = observed.runtime;
  if (flowHash(readJson(args.diskFlows)) !== state.rollbackFlowHash) {
    fail("flows.json en disco no coincide con rollback");
  }
  if (returnedRev && after.rev !== returnedRev) {
    fail(
      `rev post-rollback inesperado: POST=${returnedRev} GET=${after.rev}`
    );
  }
  const afterCredentials = loadEncryptedCredentials(args.credentials);
  if (sha256(afterCredentials.$) !== state.credentialBlobHash) {
    fail("blob cifrado cambio durante rollback");
  }
  replacePrivate(
    path.join(args.stateDir, "runtime.rollback.v2.json"),
    after
  );
  const updated = {
    ...state,
    status: "rolled_back",
    rolledBackAt: new Date().toISOString(),
    rolledBackRev: after.rev,
    reconciledRollback: Boolean(postError),
  };
  updateState(args.stateDir, updated);
  return updated;
}

async function main(argv = process.argv.slice(2), io = console) {
  const args = parseArgs(argv);
  if (args.help) {
    usage(io);
    return {help: true};
  }
  let result;
  if (args.action === "prepare") result = await prepare(args);
  if (args.action === "deploy") result = await deploy(args);
  if (args.action === "status") result = await status(args);
  if (args.action === "rollback") result = await rollback(args);
  io.log(JSON.stringify(result, null, 2));
  return result;
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`ERROR: ${error.message}`);
    process.exitCode = 1;
  });
}

module.exports = {
  API_HEADERS,
  DEPLOY_HEADERS,
  DISABLED_EXTERNAL_SENDER_ID,
  EXTERNAL_BROKER_CONFIG_ID,
  EXTERNAL_BROKER_HOST,
  EXTERNAL_BROKER_PORT,
  EXTERNAL_SENDER_ID,
  TOOL_VERSION,
  canonicalFlows,
  diffFlows,
  flowHash,
  main,
  parseArgs,
  prepareCandidate,
};
