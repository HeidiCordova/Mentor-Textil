"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");

const TOOL_VERSION = "mentor-sender-release-canary:v2";
const API_HEADERS = Object.freeze({
  Accept: "application/json",
  "Node-RED-API-Version": "v2",
});
const DEPLOY_HEADERS = Object.freeze({
  ...API_HEADERS,
  "Content-Type": "application/json",
  "Node-RED-Deployment-Type": "nodes",
});

const IDS = Object.freeze({
  sender: "b94f3dfa216658f3",
  secondarySender: "48f1dea3eb373643",
  interval: "90544e7288574184",
  broker: "85b6c2e7.5ca2f",
  mysql: "7ba0277c.2fa1b8",
  saveDbMysql: "dfee56952fb9dd3b",
  mentor: "41b1600ff22164b4",
});

const EXPECTED = Object.freeze({
  disabledFlowHash:
    "f65c87507232c7bdd4cc4aca7a0beec607b54cd60e704d65ff634569ff5e9e88",
  legacySenderSha256:
    "3d9b929beaf2f3b195e008442938e7f05b76a78bb5893e1d13a91cd8fcb58a7b",
  legacySenderHtmlSha256:
    "d17cbaf2f926d3ba897827236298aad8b6076e6378575cfba2ba92da8e4c0e01",
  hardenedSenderSha256:
    "f64477cd45698afff49e2209fa8cbc051aeb752a22680107a42fdafc6deb771a",
  hardenedSenderHtmlSha256:
    "5dddf9c7d0e39c9e78968653497bdb0c9d1dc285297cb40dcfc009d06aebd178",
  legacyIntervalSha256:
    "b4ded442cc92d9fd47aa34d87a9c5589a3d7219a38ad128ec78036fa77677683",
  brokerHost: "52.11.253.25",
  brokerPort: "1883",
  mentorId: "478",
});

const SENDER_PROFILES = Object.freeze({
  legacy: Object.freeze({
    jsSha256: EXPECTED.legacySenderSha256,
    htmlSha256: EXPECTED.legacySenderHtmlSha256,
  }),
  hardened: Object.freeze({
    jsSha256: EXPECTED.hardenedSenderSha256,
    htmlSha256: EXPECTED.hardenedSenderHtmlSha256,
  }),
});

const MODES = Object.freeze({
  disabled: Object.freeze({senderDisabled: true, interval: "300"}),
  canary: Object.freeze({senderDisabled: false, interval: "30"}),
  drain: Object.freeze({senderDisabled: false, interval: "10"}),
  steady: Object.freeze({senderDisabled: false, interval: "60"}),
});

function fail(message) {
  throw new Error(message);
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
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

function canonicalFlows(flows) {
  if (!Array.isArray(flows)) fail("flows debe ser un arreglo");
  return stableJson(
    [...flows].sort((left, right) =>
      String(left && left.id).localeCompare(String(right && right.id))
    )
  );
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

function nodeMap(flows) {
  const result = new Map();
  for (const node of flows) {
    if (!node || typeof node.id !== "string" || node.id === "") {
      fail("flow contiene nodo sin ID");
    }
    if (result.has(node.id)) fail(`ID duplicado: ${node.id}`);
    result.set(node.id, node);
  }
  return result;
}

function diffFlows(before, after) {
  const left = nodeMap(before);
  const right = nodeMap(after);
  const added = [];
  const removed = [];
  const modified = [];
  for (const [id, node] of right) {
    if (!left.has(id)) added.push(id);
    else if (stableJson(left.get(id)) !== stableJson(node)) modified.push(id);
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

function sameList(actual, expected, label) {
  const left = [...actual].sort();
  const right = [...expected].sort();
  if (JSON.stringify(left) !== JSON.stringify(right)) {
    fail(
      `${label} inesperados: actual=${JSON.stringify(left)} ` +
      `esperado=${JSON.stringify(right)}`
    );
  }
}

function validateAuditedTopology(flows) {
  const byId = nodeMap(flows);
  const sender = byId.get(IDS.sender);
  const secondary = byId.get(IDS.secondarySender);
  const interval = byId.get(IDS.interval);
  const broker = byId.get(IDS.broker);
  const mysql = byId.get(IDS.mysql);
  const saveDbMysql = byId.get(IDS.saveDbMysql);
  const mentor = byId.get(IDS.mentor);

  if (
    !sender ||
    sender.type !== "Sender" ||
    sender.mentor !== IDS.mentor ||
    sender.mysql !== IDS.mysql ||
    sender.mqtt !== IDS.broker ||
    String(sender.priority) !== "0"
  ) {
    fail("Sender principal no coincide con la auditoria");
  }
  if (
    !secondary ||
    secondary.type !== "Sender" ||
    secondary.d !== true ||
    secondary.mqtt !== IDS.broker
  ) {
    fail("Sender secundario no permanece deshabilitado");
  }
  if (
    !interval ||
    interval.type !== "Interval" ||
    Number(interval.initialdefer) !== 0 ||
    stableJson(interval.wires) !== stableJson([[IDS.sender]])
  ) {
    fail("Interval exclusivo del Sender no coincide con la auditoria");
  }
  if (
    !broker ||
    broker.type !== "mqtt-broker" ||
    String(broker.broker) !== EXPECTED.brokerHost ||
    String(broker.port) !== EXPECTED.brokerPort ||
    broker.usetls !== false
  ) {
    fail("broker externo no coincide con la auditoria");
  }
  if (
    !mysql ||
    mysql.type !== "MySQLdb" ||
    mysql.host !== "host.docker.internal" ||
    String(mysql.port) !== "3306" ||
    mysql.db !== "mentor"
  ) {
    fail("MySQL del Sender no coincide con la auditoria");
  }
  if (
    !saveDbMysql ||
    saveDbMysql.type !== "MySQLdb" ||
    saveDbMysql.id === mysql.id
  ) {
    fail("SaveDB no conserva un MySQL separado");
  }
  if (
    !mentor ||
    mentor.type !== "mentor-config" ||
    String(mentor.mentor_id) !== EXPECTED.mentorId
  ) {
    fail("mentor_id del Sender no coincide con la auditoria");
  }

  const brokerUsers = flows
    .filter((node) => node && node.mqtt === IDS.broker)
    .map((node) => node.id);
  sameList(
    brokerUsers,
    [IDS.sender, IDS.secondarySender],
    "usuarios del broker"
  );
  return {sender, interval};
}

function buildVariant(disabledFlows, mode) {
  if (!Object.hasOwn(MODES, mode)) fail(`modo invalido: ${mode}`);
  const flows = clone(disabledFlows);
  const {sender, interval} = validateAuditedTopology(flows);
  if (MODES[mode].senderDisabled) sender.d = true;
  else delete sender.d;
  interval.interval = MODES[mode].interval;
  validateAuditedTopology(flows);

  const diff = diffFlows(disabledFlows, flows);
  const expectedModified = mode === "disabled"
    ? []
    : [IDS.interval, IDS.sender];
  sameList(diff.added, [], `altas ${mode}`);
  sameList(diff.removed, [], `bajas ${mode}`);
  sameList(diff.modified, expectedModified, `cambios ${mode}`);
  return flows;
}

function validateDisabledBaseline(flows) {
  const {sender, interval} = validateAuditedTopology(flows);
  if (sender.d !== true) fail("Sender principal no esta deshabilitado");
  if (String(interval.interval) !== MODES.disabled.interval) {
    fail("Interval de baseline no es 300 s");
  }
  const hash = flowHash(flows);
  if (hash !== EXPECTED.disabledFlowHash) {
    fail(`hash baseline inesperado: ${hash}`);
  }
}

function readJson(filename) {
  try {
    return JSON.parse(fs.readFileSync(filename, "utf8"));
  } catch (error) {
    fail(`${filename} no es JSON valido: ${error.message}`);
  }
}

function writePrivate(filename, value) {
  fs.writeFileSync(filename, `${JSON.stringify(value, null, 2)}\n`, {
    encoding: "utf8",
    mode: 0o600,
    flag: "wx",
  });
  fs.chmodSync(filename, 0o600);
}

function replacePrivate(filename, value) {
  const temporary = `${filename}.tmp.${process.pid}`;
  if (fs.existsSync(temporary)) fail(`temporal ya existe: ${temporary}`);
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, {
    encoding: "utf8",
    mode: 0o600,
    flag: "wx",
  });
  fs.renameSync(temporary, filename);
  fs.chmodSync(filename, 0o600);
}

function copyPrivate(source, destination) {
  fs.copyFileSync(source, destination, fs.constants.COPYFILE_EXCL);
  fs.chmodSync(destination, 0o600);
}

function loadEncryptedCredentials(filename) {
  const value = readJson(filename);
  if (
    !value ||
    typeof value !== "object" ||
    typeof value.$ !== "string" ||
    value.$ === ""
  ) {
    fail("flows_cred no contiene blob cifrado");
  }
  return value;
}

function parseArgs(argv) {
  const result = {
    action: "",
    mode: "",
    stateDir: "",
    apiUrl: "http://127.0.0.1:1880",
    diskFlows: "/data/flows.json",
    credentials: "/data/flows_cred.json",
    senderModule:
      "/data/node_modules/node-red-contrib-services-mentor/services/sender.js",
    senderHtml:
      "/data/node_modules/node-red-contrib-services-mentor/services/sender.html",
    senderProfile: "legacy",
    intervalModule:
      "/data/node_modules/node-red-contrib-services-mentor/services/interval-timer.js",
    confirmCanary: false,
  };
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (token === "--confirm-canary") {
      result.confirmCanary = true;
      continue;
    }
    if (!token.startsWith("--")) fail(`argumento inesperado: ${token}`);
    const equals = token.indexOf("=");
    const key = equals >= 0 ? token.slice(2, equals) : token.slice(2);
    const value = equals >= 0 ? token.slice(equals + 1) : argv[++index];
    if (value === undefined || value.startsWith("--")) {
      fail(`falta valor para --${key}`);
    }
    const property = {
      action: "action",
      mode: "mode",
      "state-dir": "stateDir",
      "api-url": "apiUrl",
      "disk-flows": "diskFlows",
      credentials: "credentials",
      "sender-module": "senderModule",
      "sender-html": "senderHtml",
      "sender-profile": "senderProfile",
      "interval-module": "intervalModule",
    }[key];
    if (!property) fail(`opcion desconocida: --${key}`);
    result[property] = value;
  }
  if (!["prepare", "apply", "status"].includes(result.action)) {
    fail("--action debe ser prepare, apply o status");
  }
  if (!path.isAbsolute(result.stateDir)) fail("--state-dir debe ser absoluto");
  for (const property of [
    "diskFlows",
    "credentials",
    "senderModule",
    "senderHtml",
    "intervalModule",
  ]) {
    if (!path.isAbsolute(result[property])) {
      fail(`--${property} debe ser absoluto`);
    }
  }
  if (result.action === "apply" && !Object.hasOwn(MODES, result.mode)) {
    fail("--mode debe ser disabled, canary, drain o steady");
  }
  if (result.action !== "apply" && result.mode !== "") {
    fail("--mode solo aplica a action=apply");
  }
  if (!Object.hasOwn(SENDER_PROFILES, result.senderProfile)) {
    fail("--sender-profile debe ser legacy o hardened");
  }
  return result;
}

async function requestJson(url, options = {}) {
  const response = await fetch(url, {
    ...options,
    signal: AbortSignal.timeout(30_000),
  });
  const raw = await response.text();
  let body = {};
  if (raw !== "") {
    try {
      body = JSON.parse(raw);
    } catch {
      fail(`respuesta no JSON de ${url}`);
    }
  }
  if (!response.ok) {
    fail(`HTTP ${response.status} ${url}: ${JSON.stringify(body)}`);
  }
  return body;
}

async function getRuntime(apiUrl) {
  const body = await requestJson(`${apiUrl}/flows`, {
    method: "GET",
    headers: API_HEADERS,
  });
  if (
    !body ||
    typeof body.rev !== "string" ||
    !Array.isArray(body.flows)
  ) {
    fail("GET /flows v2 devolvio formato inesperado");
  }
  return {rev: body.rev, flows: body.flows};
}

async function postRuntime(apiUrl, body) {
  const response = await requestJson(`${apiUrl}/flows`, {
    method: "POST",
    headers: DEPLOY_HEADERS,
    body: JSON.stringify(body),
  });
  if (typeof response.rev !== "string") {
    fail("POST /flows no devolvio rev");
  }
  return response.rev;
}

async function waitForHash(apiUrl, targetHash, attempts = 40) {
  let runtime = null;
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    runtime = await getRuntime(apiUrl);
    if (flowHash(runtime.flows) === targetHash) return runtime;
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  fail(`runtime no alcanzo hash ${targetHash}`);
}

async function waitForDisk(filename, targetHash, attempts = 40) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    if (fs.existsSync(filename)) {
      const flows = readJson(filename);
      if (flowHash(flows) === targetHash) return;
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  fail(`flows.json no alcanzo hash ${targetHash}`);
}

function validateModules(args, state = null) {
  for (const filename of [
    args.senderModule,
    args.senderHtml,
    args.intervalModule,
  ]) {
    if (!fs.existsSync(filename)) fail(`modulo ausente: ${filename}`);
  }
  const profile = SENDER_PROFILES[args.senderProfile];
  if (!profile) fail(`perfil Sender inválido: ${args.senderProfile}`);
  const senderSha256 = fileHash(args.senderModule);
  const senderHtmlSha256 = fileHash(args.senderHtml);
  const intervalSha256 = fileHash(args.intervalModule);
  if (senderSha256 !== profile.jsSha256) {
    fail(
      `Sender instalado no coincide con perfil ${args.senderProfile}: ` +
      senderSha256
    );
  }
  if (senderHtmlSha256 !== profile.htmlSha256) {
    fail(
      `sender.html no coincide con perfil ${args.senderProfile}: ` +
      senderHtmlSha256
    );
  }
  if (intervalSha256 !== EXPECTED.legacyIntervalSha256) {
    fail(`Interval instalado no coincide con audit: ${intervalSha256}`);
  }
  if (
    state &&
    (
      state.senderModuleSha256 !== senderSha256 ||
      state.senderHtmlSha256 !== senderHtmlSha256 ||
      state.senderProfile !== args.senderProfile ||
      state.intervalModuleSha256 !== intervalSha256
    )
  ) {
    fail("modulos cambiaron desde prepare");
  }
  return {senderSha256, senderHtmlSha256, intervalSha256};
}

function variantFilename(stateDir, mode) {
  return path.join(stateDir, `flows.${mode}.json`);
}

function loadState(args) {
  const filename = path.join(args.stateDir, "state.json");
  if (!fs.existsSync(filename)) fail(`estado ausente: ${filename}`);
  const state = readJson(filename);
  if (
    state.toolVersion !== TOOL_VERSION ||
    typeof state.toolSha256 !== "string" ||
    !state.hashes ||
    !Object.keys(MODES).every(
      (mode) => typeof state.hashes[mode] === "string"
    )
  ) {
    fail("state.json invalido");
  }
  if (fileHash(__filename) !== state.toolSha256) {
    fail("helper difiere del preparado");
  }
  validateModules(args, state);
  for (const mode of Object.keys(MODES)) {
    const flows = readJson(variantFilename(args.stateDir, mode));
    if (flowHash(flows) !== state.hashes[mode]) {
      fail(`artefacto ${mode} fue alterado`);
    }
  }
  return state;
}

async function prepare(args) {
  if (fs.existsSync(args.stateDir)) {
    fail(`state-dir ya existe: ${args.stateDir}`);
  }
  for (const filename of [
    args.diskFlows,
    args.credentials,
    args.senderModule,
    args.senderHtml,
    args.intervalModule,
  ]) {
    if (!fs.existsSync(filename)) fail(`archivo ausente: ${filename}`);
  }
  const modules = validateModules(args);
  const runtime = await getRuntime(args.apiUrl);
  const disk = readJson(args.diskFlows);
  if (flowHash(runtime.flows) !== flowHash(disk)) {
    fail("runtime y flows.json no coinciden");
  }
  validateDisabledBaseline(runtime.flows);
  const credentials = loadEncryptedCredentials(args.credentials);
  const variants = {};
  const hashes = {};
  for (const mode of Object.keys(MODES)) {
    variants[mode] = buildVariant(runtime.flows, mode);
    hashes[mode] = flowHash(variants[mode]);
  }
  if (hashes.disabled !== EXPECTED.disabledFlowHash) {
    fail("variante disabled no preserva baseline");
  }

  fs.mkdirSync(args.stateDir, {recursive: true, mode: 0o700});
  fs.chmodSync(args.stateDir, 0o700);
  copyPrivate(__filename, path.join(args.stateDir, "helper.prepared.js"));
  copyPrivate(args.diskFlows, path.join(args.stateDir, "flows.disk.before.json"));
  copyPrivate(
    args.credentials,
    path.join(args.stateDir, "flows_cred.before.json")
  );
  for (const mode of Object.keys(MODES)) {
    writePrivate(variantFilename(args.stateDir, mode), variants[mode]);
  }
  writePrivate(path.join(args.stateDir, "runtime.before.json"), runtime);
  const state = {
    toolVersion: TOOL_VERSION,
    toolSha256: fileHash(__filename),
    status: "prepared",
    currentMode: "disabled",
    preparedAt: new Date().toISOString(),
    beforeRev: runtime.rev,
    credentialBlobSha256: sha256(credentials.$),
    senderModuleSha256: modules.senderSha256,
    senderHtmlSha256: modules.senderHtmlSha256,
    senderProfile: args.senderProfile,
    intervalModuleSha256: modules.intervalSha256,
    hashes,
  };
  writePrivate(path.join(args.stateDir, "state.json"), state);
  return state;
}

function matchingMode(state, hash) {
  const matches = Object.entries(state.hashes)
    .filter(([, expected]) => expected === hash)
    .map(([mode]) => mode);
  if (matches.length !== 1) return "other";
  return matches[0];
}

function assertTransition(state, currentMode, targetMode, confirmCanary) {
  if (targetMode === currentMode) return;
  if (targetMode === "disabled") return;
  if (currentMode === "disabled" && targetMode === "canary") return;
  if (
    currentMode === "canary" &&
    targetMode === "drain" &&
    confirmCanary
  ) {
    return;
  }
  if (currentMode === "drain" && targetMode === "steady") return;
  if (
    currentMode === "steady" &&
    targetMode === "drain" &&
    confirmCanary
  ) {
    return;
  }
  fail(
    `transicion no permitida ${currentMode}->${targetMode}; ` +
    "canary->drain requiere --confirm-canary"
  );
}

async function status(args) {
  const state = loadState(args);
  const credentials = loadEncryptedCredentials(args.credentials);
  const credentialBlobMatches =
    sha256(credentials.$) === state.credentialBlobSha256;
  const runtime = await getRuntime(args.apiUrl);
  const runtimeHash = flowHash(runtime.flows);
  const diskHash = flowHash(readJson(args.diskFlows));
  return {
    stateStatus: state.status,
    recordedMode: state.currentMode,
    runtimeMode: matchingMode(state, runtimeHash),
    currentRev: runtime.rev,
    currentFlowHash: runtimeHash,
    diskFlowHash: diskHash,
    diskMatchesRuntime: diskHash === runtimeHash,
    credentialBlobMatches,
    hashes: state.hashes,
  };
}

async function applyMode(args) {
  let state = loadState(args);
  const credentials = loadEncryptedCredentials(args.credentials);
  if (sha256(credentials.$) !== state.credentialBlobSha256) {
    fail("credenciales activas cambiaron desde prepare");
  }
  const runtime = await getRuntime(args.apiUrl);
  const currentHash = flowHash(runtime.flows);
  const currentMode = matchingMode(state, currentHash);
  if (currentMode === "other") {
    fail(`runtime ajeno al release: ${currentHash}`);
  }
  await waitForDisk(args.diskFlows, currentHash);
  assertTransition(state, currentMode, args.mode, args.confirmCanary);

  const targetFlows = readJson(variantFilename(args.stateDir, args.mode));
  const targetHash = state.hashes[args.mode];
  if (currentMode === args.mode) {
    const updated = {
      ...state,
      status: args.mode === "disabled" ? "safe" : "active",
      currentMode: args.mode,
      reconciledAt: new Date().toISOString(),
      currentRev: runtime.rev,
    };
    replacePrivate(path.join(args.stateDir, "state.json"), updated);
    return {...updated, alreadyApplied: true};
  }

  state = {
    ...state,
    status: "applying",
    currentMode,
    targetMode: args.mode,
    applyStartedAt: new Date().toISOString(),
  };
  replacePrivate(path.join(args.stateDir, "state.json"), state);

  let returnedRev = null;
  let postError = null;
  try {
    returnedRev = await postRuntime(args.apiUrl, {
      rev: runtime.rev,
      flows: targetFlows,
      credentials,
    });
  } catch (error) {
    postError = error;
  }

  let after;
  try {
    after = await waitForHash(args.apiUrl, targetHash);
  } catch (error) {
    const observed = await getRuntime(args.apiUrl);
    const observedMode = matchingMode(state, flowHash(observed.flows));
    const reset = {
      ...state,
      status: observedMode === "disabled" ? "safe" : "active",
      currentMode: observedMode,
      lastApplyError: postError
        ? postError.message
        : error.message,
    };
    replacePrivate(path.join(args.stateDir, "state.json"), reset);
    if (observedMode === "other") {
      fail("runtime quedo en estado ajeno tras POST");
    }
    throw postError || error;
  }
  await waitForDisk(args.diskFlows, targetHash);
  const afterCredentials = loadEncryptedCredentials(args.credentials);
  if (sha256(afterCredentials.$) !== state.credentialBlobSha256) {
    fail("blob cifrado cambio durante apply");
  }
  if (returnedRev && returnedRev !== after.rev) {
    fail(`rev inesperado POST=${returnedRev} GET=${after.rev}`);
  }
  const updated = {
    ...state,
    status: args.mode === "disabled" ? "safe" : "active",
    currentMode: args.mode,
    targetMode: undefined,
    appliedAt: new Date().toISOString(),
    currentRev: after.rev,
    reconciled: Boolean(postError),
    canaryConfirmedAt:
      args.confirmCanary && args.mode === "drain"
        ? new Date().toISOString()
        : state.canaryConfirmedAt,
  };
  delete updated.targetMode;
  replacePrivate(path.join(args.stateDir, "state.json"), updated);
  return updated;
}

async function main(argv = process.argv.slice(2), io = console) {
  const args = parseArgs(argv);
  let result;
  if (args.action === "prepare") result = await prepare(args);
  if (args.action === "apply") result = await applyMode(args);
  if (args.action === "status") result = await status(args);
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
  EXPECTED,
  IDS,
  MODES,
  SENDER_PROFILES,
  TOOL_VERSION,
  buildVariant,
  canonicalFlows,
  diffFlows,
  flowHash,
  main,
  parseArgs,
  validateAuditedTopology,
  validateDisabledBaseline,
  validateModules,
};
