#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");

const PATCH_VERSION = "mentor-textile-count:v4-cumulative-pilot";
const DEFAULT_INPUT = "/data/flows.json";
const DEFAULT_LINE_ID = 1;
const SAMPLE_INTERVAL_SECONDS = 300;
const SAMPLE_INTERVAL_MS = SAMPLE_INTERVAL_SECONDS * 1000;
const SETTLE_DELAY_SECONDS = 10;
const MIN_AS_OF_LAG_MS = SETTLE_DELAY_SECONDS * 1000;
const SAMPLE_LOCK_TTL_MS = 240000;
const TOO_EARLY_RETRY_SECONDS = 2;
const MAX_TOO_EARLY_RETRIES = 5;
const UINT32_MAX = 0xFFFFFFFF;
const VISION_COUNTER_URL =
  "http://host.docker.internal:8002/vision/counter";
const LOCK_STORE_KEY = "mentor_textile_count_v4_samples";
const SAVE_DB_MODE_STRICT = "strict-delta";
const SAVE_DB_MODE_PRESERVE = "preserve-legacy";

// Nodos existentes verificados en nodered-flows.audit.json.
const AUDIT = Object.freeze({
  productionInterval: "570d133c1babc579",
  manualInject: "e776b3254b93967f",
  genericIn: "fa2fab697e9c3b6c",
  flexGetter: "6548475195d0bdcb",
  genericOut: "d99995b9cb8c231a",
  sanitizeProduction: "f9d49532f4f6753e",
  saveDb: "b0a365a2d443baa2",
  senderInterval: "90544e7288574184",
  globalPublisher: "9d1c621a18c65202",
  legacyCountConverter: "4201b6b3d72b77c1",
});

// IDs administrados por v4. Se reutilizan IDs compatibles de v3 para migrar
// sin duplicar nodos.
const IDS = Object.freeze({
  http: "c07e000000000002",
  validate: "c07e000000000003",
  catch: "c07e000000000004",
  delay: "c07e000000000006",
  prepare: "c07e000000000007",
  guard: "c07e000000000008",
  modbusWrite: "c07e00000000000a",
  restore: "c07e00000000000b",
  fail: "c07e00000000000d",
  retryDelay: "c07e00000000000e",
});

// Nodos v2/v3 que v4 elimina: ya no hay inject de arranque, recuperación de
// ventana, rate-limit de catch-up ni marca de ventana completada.
const OBSOLETE_IDS = Object.freeze({
  startup: "c07e000000000001",
  recover: "c07e000000000005",
  rateLimit: "c07e000000000009",
  complete: "c07e00000000000c",
});

const COUNT_OWNER_MARKER =
  `// ${PATCH_VERSION} - CONTEO_1 es el contador CORTE acumulativo del backend.`;
const READ_ONLY_MARKER =
  `// ${PATCH_VERSION} - L1_conteo_1 es de solo lectura en este nodo.`;
const MOTION_STOP_VERSION = "mentor-motion-stops:v1";
const MOTION_STOP_MARKER =
  `// ${MOTION_STOP_VERSION} - las paradas usan presence_motion, no fsm_state.`;
const MOTION_STOP_CONTEXT_KEY = "prod_stop_logic_version";
const MOTION_READY_CONTEXT_KEY = "prod_motion_observation_ready";
const PROGRESS_DEBUG_VERSION = "mentor-estimated-progress:v1";
const PROGRESS_DEBUG_MARKER =
  `// ${PROGRESS_DEBUG_VERSION} - Python calcula; Node-RED solo refleja.`;
const COUNT_OWNER_MARKER_RE =
  /\/\/ mentor-textile-count:v[\w.-]+ - CONTEO_1 [^\r\n]+/;
const READ_ONLY_MARKER_RE =
  /\/\/ mentor-textile-count:v[\w.-]+ - L1_conteo_1 es de solo lectura en este nodo\./;
const LEGACY_TRANSITION_RE =
  /if\s*\(\s*estadoAnterior\s*===\s*["']en_prenda["']\s*&&\s*estadoActual\s*===\s*["']idle["']\s*\)\s*\{\s*conteo\s*\+=\s*1\s*;\s*\}/m;
const LEGACY_COUNT_COMMENTS_RE =
  /[ \t]*\/\/ Se considera un corte terminado[^\r\n]*\r?\n[ \t]*\/\/ en_prenda[^\r\n]*\r?\n/m;
const LEGACY_GLOBAL_WRITE_RE =
  /global\.set\s*\(\s*["']L1_conteo_1["']\s*,\s*Math\.floor\s*\(\s*conteo\s*\)\s*\)\s*;/m;

function validateVisionCounterResponse(message, expected) {
  if (!message || typeof message !== "object") {
    return {ok: false, error: "mensaje HTTP ausente"};
  }
  if (message.error) {
    return {ok: false, error: "error de transporte HTTP"};
  }
  if (message.statusCode !== 200) {
    return {
      ok: false,
      error: `HTTP ${String(message.statusCode ?? "sin estado")}`,
    };
  }
  if (!expected ||
      !Number.isInteger(expected.lineaId) ||
      expected.lineaId <= 0 ||
      !Number.isSafeInteger(expected.sampleTime) ||
      expected.sampleTime < 300000 ||
      expected.sampleTime % 300000 !== 0) {
    return {ok: false, error: "metadatos de muestra invalidos"};
  }

  let data = message.payload;
  if (typeof data === "string") {
    try {
      data = JSON.parse(data);
    } catch (_) {
      return {ok: false, error: "respuesta JSON invalida"};
    }
  }
  if (!data || typeof data !== "object" || Array.isArray(data)) {
    return {ok: false, error: "respuesta vacia o invalida"};
  }
  if (!Number.isInteger(data.linea_id) ||
      data.linea_id !== expected.lineaId) {
    return {ok: false, error: "linea_id no coincide"};
  }
  if (data.event_type !== "CORTE") {
    return {ok: false, error: "event_type debe ser CORTE"};
  }

  const until = typeof data.until === "string"
    ? Date.parse(data.until)
    : NaN;
  const asOf = typeof data.as_of === "string"
    ? Date.parse(data.as_of)
    : NaN;
  const counterEpoch = typeof data.counter_epoch === "string"
    ? Date.parse(data.counter_epoch)
    : NaN;
  const stateUpdatedAt = typeof data.state_updated_at === "string"
    ? Date.parse(data.state_updated_at)
    : NaN;
  if (!Number.isFinite(until) || until !== expected.sampleTime) {
    return {ok: false, error: "until no coincide con B"};
  }
  if (!Number.isFinite(asOf)) {
    return {ok: false, error: "as_of ausente o invalido"};
  }
  if (asOf < expected.sampleTime + 10000) {
    return {ok: false, error: "as_of anterior al cierre seguro"};
  }
  if (!Number.isFinite(counterEpoch) ||
      counterEpoch < 0 ||
      counterEpoch > expected.sampleTime) {
    return {ok: false, error: "counter_epoch invalido"};
  }
  if (!Number.isFinite(stateUpdatedAt) ||
      stateUpdatedAt < counterEpoch ||
      stateUpdatedAt > asOf) {
    return {ok: false, error: "state_updated_at invalido"};
  }
  if (!Number.isSafeInteger(data.count) ||
      data.count < 0 ||
      data.count > 0xFFFFFFFF) {
    return {ok: false, error: "count fuera de UINT32"};
  }

  return {
    ok: true,
    count: data.count,
    counterEpoch,
    counterEpochISO: new Date(counterEpoch).toISOString(),
    stateUpdatedAt,
    data,
  };
}

function buildPrepareFunction(lineaId) {
  return [
    `// ${PATCH_VERSION}`,
    `const SAMPLE_MS = ${SAMPLE_INTERVAL_MS};`,
    `const LINEA_ID = ${lineaId};`,
    `const BASE_URL = ${JSON.stringify(VISION_COUNTER_URL)};`,
    "const sampleTime = Number(msg.payload);",
    "",
    "if (!Number.isSafeInteger(sampleTime) ||",
    "    sampleTime < SAMPLE_MS ||",
    "    sampleTime % SAMPLE_MS !== 0) {",
    "    node.warn(\"CONTADOR v4: timestamp B invalido\");",
    "    node.status({fill:\"red\", shape:\"ring\", text:\"B invalido\"});",
    "    return null;",
    "}",
    "",
    "const until = new Date(sampleTime).toISOString();",
    "msg._counterLineId = LINEA_ID;",
    "msg._counterSampleTime = sampleTime;",
    "msg._counterSampleId = `${LINEA_ID}:${sampleTime}`;",
    "msg._counterExpectedUntil = until;",
    "msg.method = \"GET\";",
    "msg.url = `${BASE_URL}?linea_id=${LINEA_ID}` +",
    "    `&until=${encodeURIComponent(until)}`;",
    "return msg;",
  ].join("\n");
}

function buildGuardFunction() {
  return [
    `// ${PATCH_VERSION}`,
    `const STORE_KEY = ${JSON.stringify(LOCK_STORE_KEY)};`,
    `const TTL_MS = ${SAMPLE_LOCK_TTL_MS};`,
    "const now = Date.now();",
    "const sampleId = msg._counterSampleId;",
    "let samples = flow.get(STORE_KEY);",
    "",
    "if (!samples || typeof samples !== \"object\" || Array.isArray(samples)) {",
    "    samples = {};",
    "}",
    "for (const [id, value] of Object.entries(samples)) {",
    "    if (!value || !Number.isFinite(value.expiresAt) || value.expiresAt <= now) {",
    "        delete samples[id];",
    "    }",
    "}",
    "if (typeof sampleId !== \"string\" || sampleId === \"\") {",
    "    node.warn(\"CONTADOR v4: sample_id ausente\");",
    "    flow.set(STORE_KEY, samples);",
    "    return null;",
    "}",
    "if (samples[sampleId]) {",
    "    node.status({fill:\"yellow\", shape:\"ring\", text:\"B duplicado\"});",
    "    flow.set(STORE_KEY, samples);",
    "    return null;",
    "}",
    "",
    "samples[sampleId] = {expiresAt: now + TTL_MS};",
    "flow.set(STORE_KEY, samples);",
    "global.set(\"L1_conteo_1_counter_valid\", false);",
    "node.status({fill:\"blue\", shape:\"dot\", text:\"consulta acumulativa\"});",
    "return msg;",
  ].join("\n");
}

function buildValidateFunction() {
  return [
    `// ${PATCH_VERSION}`,
    validateVisionCounterResponse.toString(),
    "",
    `const MAX_425_RETRIES = ${MAX_TOO_EARLY_RETRIES};`,
    `const COUNTER_URL = ${JSON.stringify(VISION_COUNTER_URL)};`,
    "function reject(reason) {",
    "    msg._counterFailureReason = reason;",
    "    global.set(\"L1_conteo_1_counter_valid\", false);",
    "    node.warn(`CONTADOR v4 bloqueado: ${reason}`);",
    "    node.status({fill:\"red\", shape:\"ring\", text:\"contador rechazado\"});",
    "    return [null, msg, null];",
    "}",
    "",
    "if (msg.statusCode === 425) {",
    "    const sampleTime = Number(msg._counterSampleTime);",
    "    const lineaId = Number(msg._counterLineId);",
    "    const retry = Number(msg._counterRetry425) || 0;",
    "    if (!Number.isSafeInteger(sampleTime) ||",
    "        sampleTime % 300000 !== 0 ||",
    "        !Number.isInteger(lineaId) || lineaId <= 0) {",
    "        return reject(\"HTTP 425 sin B valido\");",
    "    }",
    "    if (retry >= MAX_425_RETRIES) {",
    "        return reject(\"HTTP 425 excedio reintentos\");",
    "    }",
    "    msg._counterRetry425 = retry + 1;",
    "    const until = new Date(sampleTime).toISOString();",
    "    msg.payload = sampleTime;",
    "    msg._counterExpectedUntil = until;",
    "    msg.url = `${COUNTER_URL}?linea_id=${lineaId}` +",
    "        `&until=${encodeURIComponent(until)}`;",
    "    delete msg.statusCode;",
    "    delete msg.error;",
    "    delete msg.headers;",
    "    delete msg.responseUrl;",
    "    node.status({fill:\"yellow\", shape:\"ring\", text:\"425: reintentar mismo B\"});",
    "    return [null, null, msg];",
    "}",
    "",
    "const expected = {",
    "    lineaId: msg._counterLineId,",
    "    sampleTime: msg._counterSampleTime",
    "};",
    "const validation = validateVisionCounterResponse(msg, expected);",
    "if (!validation.ok) return reject(validation.error);",
    "",
    "const previousEpochRaw = global.get(\"L1_conteo_1_counter_epoch\");",
    "const previousEpoch = typeof previousEpochRaw === \"string\"",
    "    ? Date.parse(previousEpochRaw)",
    "    : NaN;",
    "const previousCount = Number(global.get(\"L1_conteo_1\"));",
    "const sameEpoch = Number.isFinite(previousEpoch) &&",
    "    previousEpoch === validation.counterEpoch;",
    "if (sameEpoch &&",
    "    Number.isSafeInteger(previousCount) &&",
    "    previousCount >= 0 &&",
    "    validation.count < previousCount) {",
    "    return reject(",
    "        `retroceso ${validation.count} < ${previousCount} en el mismo epoch`",
    "    );",
    "}",
    "",
    "const count = validation.count;",
    "const highWord = Math.floor(count / 65536);",
    "const lowWord = count % 65536;",
    "// El backend es dueño del epoch y del acumulado; Node-RED no suma deltas.",
    "global.set(\"L1_conteo_1\", count);",
    "global.set(\"L1_conteo_1_counter_epoch\", validation.counterEpochISO);",
    "global.set(\"L1_conteo_1_counter_valid\", true);",
    "global.set(\"L1_conteo_1_sample_time\", msg._counterSampleTime);",
    "msg._counterCount = count;",
    "msg._counterEpoch = validation.counterEpochISO;",
    "msg.payload = [highWord, lowWord];",
    "node.status({fill:\"green\", shape:\"dot\", text:`TOTAL ${count}`});",
    "return [msg, null, null];",
  ].join("\n");
}

function buildRestoreFunction() {
  return [
    `// ${PATCH_VERSION}`,
    `const SAMPLE_MS = ${SAMPLE_INTERVAL_MS};`,
    "const sampleTime = Number(msg._counterSampleTime);",
    "if (!Number.isSafeInteger(sampleTime) || sampleTime % SAMPLE_MS !== 0) {",
    "    msg._counterFailureReason = \"Modbus no preservo B\";",
    "    global.set(\"L1_conteo_1_counter_valid\", false);",
    "    node.warn(\"CONTADOR v4: ACK Modbus sin B\");",
    "    return [null, msg];",
    "}",
    "",
    "// Generic In conserva code/head/data y publica la muestra con time=B.",
    "msg.payload = sampleTime;",
    "delete msg.statusCode;",
    "delete msg.responseUrl;",
    "delete msg.url;",
    "delete msg.headers;",
    "node.status({fill:\"green\", shape:\"dot\", text:\"Modbus confirmado\"});",
    "return [msg, null];",
  ].join("\n");
}

function buildFailureFunction() {
  return [
    `// ${PATCH_VERSION}`,
    "global.set(\"L1_conteo_1_counter_valid\", false);",
    "const reason = String(msg._counterFailureReason ||",
    "    (msg.error && msg.error.message) || \"flujo incompleto\").slice(0, 80);",
    "node.warn(`CONTADOR v4 sin muestra: ${reason}`);",
    "node.status({fill:\"red\", shape:\"ring\", text:\"muestra bloqueada\"});",
    "return null;",
  ].join("\n");
}

function usage(io = console) {
  io.log([
    "Uso:",
    "  node patch-official-textile-count.js [opciones]",
    "",
    "Opciones:",
    "  --input RUTA     flows.json de entrada",
    "  --apply          crea backup y reemplaza atomico",
    "  --linea-id N     linea de vision (default: 1)",
    "  --preserve-legacy-save-db",
    "                   no modifica SaveDB mientras se endurece el runtime",
    "  --help           muestra esta ayuda",
    "",
    `Entrada predeterminada: ${DEFAULT_INPUT}`,
    "Sin --apply solo valida y reporta; no escribe el archivo.",
    "v4 consulta GET /vision/counter?linea_id=N&until=B.",
    "counter_epoch y count son persistidos y calculados por el backend.",
    "Node-RED no fija un epoch, no suma deltas y no recupera ventanas al arrancar.",
    "El Interval del Sender se conserva siempre sin cambios.",
  ].join("\n"));
}

function parsePositiveInteger(raw, option, maximum = Number.MAX_SAFE_INTEGER) {
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value <= 0 || value > maximum) {
    throw new Error(`${option} requiere un entero entre 1 y ${maximum}`);
  }
  return value;
}

function parseArgs(argv) {
  let input = DEFAULT_INPUT;
  let apply = false;
  let help = false;
  let lineaId = DEFAULT_LINE_ID;
  let preserveLegacySaveDb = false;

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--apply") {
      apply = true;
    } else if (arg === "--preserve-legacy-save-db") {
      preserveLegacySaveDb = true;
    } else if (arg === "--help" || arg === "-h") {
      help = true;
    } else if (arg === "--input" || arg === "--linea-id") {
      if (!argv[i + 1]) throw new Error(`${arg} requiere un valor`);
      const value = argv[i + 1];
      i += 1;
      if (arg === "--input") input = value;
      if (arg === "--linea-id") {
        lineaId = parsePositiveInteger(value, "--linea-id", 10000);
      }
    } else if (arg.startsWith("--input=")) {
      input = arg.slice("--input=".length);
    } else if (arg.startsWith("--linea-id=")) {
      lineaId = parsePositiveInteger(
        arg.slice("--linea-id=".length),
        "--linea-id",
        10000
      );
    } else {
      throw new Error(`argumento desconocido: ${arg}`);
    }
  }
  return {
    input: path.resolve(input),
    apply,
    help,
    lineaId,
    preserveLegacySaveDb,
  };
}

function normalizeOptions(options = {}) {
  if (
    options.preserveLegacySaveDb !== undefined &&
    typeof options.preserveLegacySaveDb !== "boolean"
  ) {
    throw new Error("preserveLegacySaveDb requiere un booleano");
  }
  return {
    lineaId: options.lineaId === undefined
      ? DEFAULT_LINE_ID
      : parsePositiveInteger(options.lineaId, "lineaId", 10000),
    preserveLegacySaveDb: options.preserveLegacySaveDb === true,
  };
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function getRequiredNode(flows, id, type) {
  const node = flows.find((candidate) => candidate && candidate.id === id);
  if (!node) throw new Error(`falta nodo auditado ${id}`);
  if (node.type !== type) {
    throw new Error(
      `tipo inesperado para ${id}: ${String(node.type)}; esperado ${type}`
    );
  }
  return node;
}

function findProductionNode(flows) {
  const candidates = flows.filter((node) =>
    node &&
    node.type === "function" &&
    typeof node.func === "string" &&
    node.func.includes("prod_estado_anterior") &&
    node.func.includes("L1_conteo_1") &&
    node.func.includes("L1_t_disponible")
  );
  if (candidates.length !== 1) {
    throw new Error(
      `se esperaba un nodo Produccion Art Atlas; encontrados: ${candidates.length}`
    );
  }
  return candidates[0];
}

function removeLegacyCountLogic(node, changes) {
  let source = node.func;
  if (LEGACY_TRANSITION_RE.test(source)) {
    source = source.replace(LEGACY_COUNT_COMMENTS_RE, "");
    source = source.replace(
      LEGACY_TRANSITION_RE,
      `${COUNT_OWNER_MARKER}\n    // Los tiempos continúan; este nodo no cuenta prendas.`
    );
    changes.push("eliminado incremento legado en_prenda -> idle");
  } else if (COUNT_OWNER_MARKER_RE.test(source)) {
    const migrated = source.replace(COUNT_OWNER_MARKER_RE, COUNT_OWNER_MARKER);
    if (migrated !== source) {
      source = migrated;
      changes.push("actualizado marcador de CONTEO_1 acumulativo");
    }
  } else {
    throw new Error(
      "no se encontro el incremento legado ni un marcador compatible"
    );
  }

  if (LEGACY_GLOBAL_WRITE_RE.test(source)) {
    source = source.replace(LEGACY_GLOBAL_WRITE_RE, READ_ONLY_MARKER);
    changes.push("marcado L1_conteo_1 como solo lectura en Produccion");
  } else if (READ_ONLY_MARKER_RE.test(source)) {
    const migrated = source.replace(READ_ONLY_MARKER_RE, READ_ONLY_MARKER);
    if (migrated !== source) {
      source = migrated;
      changes.push("actualizado marcador read-only de L1_conteo_1");
    }
  } else {
    throw new Error(
      "no se encontro la escritura global legada ni un marcador compatible"
    );
  }
  node.func = source;
}

function migrateMotionStopLogic(node, changes) {
  let source = node.func;
  if (source.includes(MOTION_STOP_MARKER)) {
    if (/estadoActual\s*===\s*["']idle["']/.test(source)) {
      throw new Error(
        "Produccion tiene marcador de movimiento pero conserva parada por idle"
      );
    }
    return;
  }

  const legacyValidation = [
    "// Validar respuesta del detector",
    "if (!datos || typeof datos.fsm_state !== \"string\") {",
    "    node.warn(\"Producción: mensaje sin fsm_state\");",
    "    return null;",
    "}",
    "",
    "const estadoActual = datos.fsm_state;",
    "const ahora = Date.now();",
  ].join("\n");
  const motionValidation = [
    "// Validar el contrato completo de /status. Un dato incompleto no debe",
    "// convertirse silenciosamente en una parada.",
    "if (!datos ||",
    "    typeof datos.fsm_state !== \"string\" ||",
    "    typeof datos.presence_motion !== \"boolean\" ||",
    "    typeof datos.motion_ready !== \"boolean\" ||",
    "    typeof datos.motion_fresh !== \"boolean\" ||",
    "    !Number.isFinite(Number(datos.micro_stop_max_s)) ||",
    "    Number(datos.micro_stop_max_s) <= 3) {",
    "    node.warn(\"Producción: contrato /status de movimiento incompleto\");",
    "    return null;",
    "}",
    "",
    "const ahora = Date.now();",
    "",
    "// Durante calentamiento, caída de cámara o muestra inválida no se",
    "// clasifica tiempo: false todavía no representa una parada observable.",
    "if (!datos.motion_ready || !datos.motion_fresh) {",
    `    context.set(${JSON.stringify(MOTION_READY_CONTEXT_KEY)}, false);`,
    "    context.set(\"prod_ultimo_tiempo_ms\", ahora);",
    "    return null;",
    "}",
    "",
    `const primeraMuestraLista = context.get(${JSON.stringify(MOTION_READY_CONTEXT_KEY)}) !== true;`,
    `context.set(${JSON.stringify(MOTION_READY_CONTEXT_KEY)}, true);`,
    "",
    "const estadoActual = datos.fsm_state;",
    "const hayMovimiento = datos.presence_motion;",
    "const limiteMicroparadaS = Number(datos.micro_stop_max_s);",
  ].join("\n");
  if (!source.includes(legacyValidation)) {
    throw new Error(
      "no se encontró la validación legacy de /status en Produccion"
    );
  }
  source = source.replace(legacyValidation, motionValidation);

  const elapsedGuard = [
    "    if (segundos < 0 || segundos > 15) {",
    "        segundos = 0;",
    "    }",
  ].join("\n");
  const readyElapsedGuard = [
    "    if (primeraMuestraLista || segundos < 0 || segundos > 15) {",
    "        segundos = 0;",
    "    }",
  ].join("\n");
  if (!source.includes(elapsedGuard)) {
    throw new Error("no se encontró la protección de intervalo en Produccion");
  }
  source = source.replace(elapsedGuard, readyElapsedGuard);

  const sectionStart = source.indexOf(
    "// 3. MICROPARADA Y PARADA NO ASIGNADA"
  );
  const sectionEndMarker = [
    "// -----------------------------------------------------",
    "// Guardar variables globales para Modbus",
  ].join("\n");
  const sectionEnd = source.indexOf(sectionEndMarker, sectionStart);
  if (sectionStart < 0 || sectionEnd < 0) {
    throw new Error("no se encontró la sección legacy de paradas");
  }

  const motionSection = [
    "// 3. MICROPARADA Y PARADA NO ASIGNADA",
    "// -----------------------------------------------------",
    "",
    MOTION_STOP_MARKER,
    `const VERSION_LOGICA_PARADAS = ${JSON.stringify(MOTION_STOP_VERSION)};`,
    "",
    "// Un hot-deploy conserva el contexto del nodo. Se descarta solamente el",
    "// candidato interno creado por la lógica legacy; nunca los acumulados T_*.",
    `if (context.get(${JSON.stringify(MOTION_STOP_CONTEXT_KEY)}) !== VERSION_LOGICA_PARADAS) {`,
    "    tiempoIdle = 0;",
    "    pnaActiva = false;",
    `    context.set(${JSON.stringify(MOTION_STOP_CONTEXT_KEY)}, VERSION_LOGICA_PARADAS);`,
    "}",
    "",
    "// fsm_state describe la etapa textil (idle, beige_in, en_prenda,",
    "// cooldown). No se usa para decidir si la máquina está parada.",
    "// presence_motion es la señal estable de movimiento del detector.",
    "if (!hayMovimiento) {",
    "",
    "    tiempoIdle += segundos;",
    "",
    "    // Al superar el límite, todo el periodo se reclasifica como PNA.",
    "    if (!pnaActiva && tiempoIdle >= limiteMicroparadaS) {",
    "        tiempoParadaNoAsignada += Math.floor(tiempoIdle);",
    "        pnaActiva = true;",
    "    }",
    "    else if (pnaActiva) {",
    "        tiempoParadaNoAsignada += segundos;",
    "    }",
    "",
    "}",
    "else {",
    "",
    "    // Cuando vuelve el movimiento, una pausa breve se confirma de forma",
    "    // retroactiva como microparada. Menos de 3 s se descarta como ruido.",
    "    if (",
    "        !pnaActiva &&",
    "        tiempoIdle >= 3 &&",
    "        tiempoIdle < limiteMicroparadaS",
    "    ) {",
    "        tiempoMicroparada += Math.floor(tiempoIdle);",
    "    }",
    "",
    "    tiempoIdle = 0;",
    "    pnaActiva = false;",
    "}",
    "",
  ].join("\n");
  source =
    source.slice(0, sectionStart) + motionSection + source.slice(sectionEnd);

  const debugStart = [
    "    estado_actual: estadoActual,",
    "    estado_anterior: estadoAnterior || \"inicio\",",
    "    intervalo_segundos: segundos,",
  ].join("\n");
  const motionDebug = [
    "    estado_actual: estadoActual,",
    "    estado_anterior: estadoAnterior || \"inicio\",",
    "    presencia_movimiento: hayMovimiento,",
    "    puntaje_movimiento: Number(datos.motion_score) || 0,",
    "    estado_gestor_parada: datos.stop_tracker_state || \"sin_dato\",",
    "    movimiento_listo: datos.motion_ready,",
    "    movimiento_fresco: datos.motion_fresh,",
    "    limite_microparada_segundos: limiteMicroparadaS,",
    "    intervalo_segundos: segundos,",
  ].join("\n");
  if (!source.includes(debugStart)) {
    throw new Error("no se encontró el payload debug de Produccion");
  }
  source = source.replace(debugStart, motionDebug);

  node.func = source;
  changes.push(
    "paradas de Produccion migradas de fsm_state idle a presence_motion"
  );
}

function migrateEstimatedProgressDebug(node, changes) {
  let source = node.func;
  if (source.includes(PROGRESS_DEBUG_MARKER)) return;

  const anchor =
    '    estado_gestor_parada: datos.stop_tracker_state || "sin_dato",';
  if (!source.includes(anchor)) {
    throw new Error(
      "no se encontrÃ³ el payload debug de movimiento para reflejar avance"
    );
  }

  const progressDebug = [
    anchor,
    `    ${PROGRESS_DEBUG_MARKER}`,
    "    avance_estimado_pct: (",
    "        datos.progress_valid === true &&",
    "        typeof datos.progress_estimated_pct === \"number\" &&",
    "        Number.isFinite(datos.progress_estimated_pct) &&",
    "        datos.progress_estimated_pct >= 0 &&",
    "        datos.progress_estimated_pct <= 100",
    "    ) ? datos.progress_estimated_pct : null,",
    "    avance_estimado_valido: (",
    "        datos.progress_valid === true &&",
    "        typeof datos.progress_estimated_pct === \"number\" &&",
    "        Number.isFinite(datos.progress_estimated_pct) &&",
    "        datos.progress_estimated_pct >= 0 &&",
    "        datos.progress_estimated_pct <= 100",
    "    ),",
    "    estado_avance: typeof datos.progress_state === \"string\"",
    "        ? datos.progress_state : \"unavailable\",",
    "    avance_observable: datos.progress_observable === true,",
    "    segundos_activos_prenda: Number.isFinite(Number(datos.active_cycle_s))",
    "        ? Number(datos.active_cycle_s) : null,",
    "    tiempo_ideal_prenda_segundos: datos.ideal_cycle_s !== null &&",
    "        datos.ideal_cycle_s !== undefined &&",
    "        Number.isFinite(Number(datos.ideal_cycle_s))",
    "        ? Number(datos.ideal_cycle_s) : null,",
    "    producto_avance_id: datos.progress_product_id ?? null,",
    "    corrida_avance_id: datos.progress_run_id || null,",
    "    motivo_avance: datos.progress_reason || null,",
    "    ultimo_corte_avance_id: datos.progress_last_completion_event_id || null,",
    "    contexto_ultimo_corte_valido: datos.progress_last_completion_context_valid === true",
    "        ? true : (datos.progress_last_completion_context_valid === false ? false : null),",
  ].join("\n");

  node.func = source.replace(anchor, progressDebug);
  changes.push("avance estimado de Python reflejado solo en debug de Produccion");
}

function ensureOutput(node, outputIndex) {
  if (!Array.isArray(node.wires)) node.wires = [];
  while (node.wires.length <= outputIndex) node.wires.push([]);
  if (!Array.isArray(node.wires[outputIndex])) node.wires[outputIndex] = [];
  return node.wires[outputIndex];
}

function rewireProductionInterval(node, changes) {
  if (!Array.isArray(node.wires) || !Array.isArray(node.wires[0])) {
    throw new Error("Interval de produccion no tiene salida auditada");
  }
  const output = node.wires[0];
  const isLegacy = output.length === 1 && output[0] === AUDIT.genericIn;
  const isManaged = output.length === 1 && output[0] === IDS.delay;
  if (!isLegacy && !isManaged) {
    throw new Error("Interval de produccion contiene wires inesperados");
  }
  if (isLegacy) {
    node.wires[0] = [IDS.delay];
    changes.push("Interval 300 s redirigido al contador acumulativo v4");
  }
}

function disableManualInject(node, changes) {
  if (node.payloadType !== "date" ||
      String(node.repeat || "") !== "" ||
      node.once !== false) {
    throw new Error(
      "inject manual auditado debe ser date, sin repeat y once=false"
    );
  }
  if (!Array.isArray(node.wires) || !Array.isArray(node.wires[0])) {
    throw new Error("inject manual no tiene la salida auditada");
  }
  const output = node.wires[0];
  const allowed =
    output.length === 0 ||
    (output.length === 1 && output[0] === AUDIT.genericIn) ||
    (output.length === 1 && output[0] === OBSOLETE_IDS.recover);
  if (!allowed) {
    throw new Error("inject manual contiene wires inesperados");
  }
  if (output.length > 0) {
    node.wires[0] = [];
    changes.push("inject manual deshabilitado para evitar filas duplicadas");
  }
}

function disconnectLegacyCountWriter(node, changes) {
  if (!Array.isArray(node.wires) || !Array.isArray(node.wires[3])) {
    throw new Error("publisher global no tiene la salida 4 auditada");
  }
  const output = node.wires[3];
  if (output.length > 0 &&
      (output.length !== 1 || output[0] !== AUDIT.legacyCountConverter)) {
    throw new Error(
      "salida 4 del publisher global contiene destinos no auditados"
    );
  }
  if (output.length > 0) {
    node.wires[3] = [];
    changes.push("desconectada solo la salida 4 del publisher global");
  }
}

function removeManagedWire(node, outputIndex, target, changes, description) {
  if (!Array.isArray(node.wires) ||
      !Array.isArray(node.wires[outputIndex])) {
    return;
  }
  const before = node.wires[outputIndex];
  const after = before.filter((candidate) => candidate !== target);
  if (after.length !== before.length) {
    node.wires[outputIndex] = after;
    changes.push(description);
  }
}

function isObsoleteOwned(node, id) {
  if (!node || typeof node.name !== "string") return false;
  if (node.name.startsWith("[PILOTO v3]")) return true;
  if (id === OBSOLETE_IDS.startup) {
    return (
      node.name.includes("conteo CORTE oficial") ||
      node.name.includes("conteo de corrida activa") ||
      node.name.includes("Sincronizar conteo")
    );
  }
  return false;
}

function removeObsoleteNodes(flows, changes) {
  for (const id of Object.values(OBSOLETE_IDS)) {
    const index = flows.findIndex((node) => node && node.id === id);
    if (index === -1) continue;
    const current = flows[index];
    if (!isObsoleteOwned(current, id)) {
      throw new Error(`colision con ID obsoleto reservado ${id}`);
    }
    flows.splice(index, 1);
    changes.push(`eliminado nodo de recuperacion v3 ${id}`);
  }
}

function desiredNodes(tabId, lineaId, modbusServer, unitId) {
  const prefix = "[PILOTO v4 ACUMULATIVO]";
  return [
    {
      id: IDS.delay,
      type: "delay",
      z: tabId,
      name: `${prefix} esperar 10 s tras B`,
      pauseType: "delay",
      timeout: String(SETTLE_DELAY_SECONDS),
      timeoutUnits: "seconds",
      rate: "1",
      nbRateUnits: "1",
      rateUnits: "second",
      randomFirst: "1",
      randomLast: "5",
      randomUnits: "seconds",
      drop: false,
      allowrate: false,
      outputs: 1,
      x: 380,
      y: 600,
      wires: [[IDS.prepare]],
    },
    {
      id: IDS.prepare,
      type: "function",
      z: tabId,
      name: `${prefix} fijar B y GET /vision/counter`,
      func: buildPrepareFunction(lineaId),
      outputs: 1,
      timeout: 0,
      noerr: 0,
      initialize: "",
      finalize: "",
      libs: [],
      x: 650,
      y: 600,
      wires: [[IDS.guard]],
    },
    {
      id: IDS.guard,
      type: "function",
      z: tabId,
      name: `${prefix} guard por B`,
      func: buildGuardFunction(),
      outputs: 1,
      timeout: 0,
      noerr: 0,
      initialize: "",
      finalize: "",
      libs: [],
      x: 900,
      y: 600,
      wires: [[IDS.http]],
    },
    {
      id: IDS.http,
      type: "http request",
      z: tabId,
      name: `${prefix} GET contador CORTE`,
      method: "GET",
      ret: "obj",
      paytoqs: "ignore",
      url: "",
      tls: "",
      persist: false,
      proxy: "",
      insecureHTTPParser: false,
      authType: "",
      senderr: true,
      headers: [],
      x: 1120,
      y: 600,
      wires: [[IDS.validate]],
    },
    {
      id: IDS.validate,
      type: "function",
      z: tabId,
      name: `${prefix} validar total y convertir UINT32`,
      func: buildValidateFunction(),
      outputs: 3,
      timeout: 0,
      noerr: 0,
      initialize: "",
      finalize: "",
      libs: [],
      x: 1370,
      y: 600,
      wires: [[IDS.modbusWrite], [IDS.fail], [IDS.retryDelay]],
    },
    {
      id: IDS.retryDelay,
      type: "delay",
      z: tabId,
      name: `${prefix} reintentar HTTP 425 con el mismo B`,
      pauseType: "delay",
      timeout: String(TOO_EARLY_RETRY_SECONDS),
      timeoutUnits: "seconds",
      rate: "1",
      nbRateUnits: "1",
      rateUnits: "second",
      randomFirst: "1",
      randomLast: "5",
      randomUnits: "seconds",
      drop: false,
      allowrate: false,
      outputs: 1,
      x: 1370,
      y: 660,
      wires: [[IDS.http]],
    },
    {
      id: IDS.modbusWrite,
      type: "modbus-write",
      z: tabId,
      name: `${prefix} escribir TOTAL 8:9`,
      showStatusActivities: false,
      showErrors: true,
      showWarnings: true,
      unitid: String(unitId),
      dataType: "MHoldingRegisters",
      adr: "8",
      quantity: "2",
      server: modbusServer,
      emptyMsgOnFail: false,
      keepMsgProperties: true,
      delayOnStart: false,
      startDelayTime: "",
      x: 1630,
      y: 580,
      wires: [[IDS.restore], [IDS.fail]],
    },
    {
      id: IDS.restore,
      type: "function",
      z: tabId,
      name: `${prefix} restaurar B y leer snapshot`,
      func: buildRestoreFunction(),
      outputs: 2,
      timeout: 0,
      noerr: 0,
      initialize: "",
      finalize: "",
      libs: [],
      x: 1880,
      y: 580,
      wires: [[AUDIT.genericIn], [IDS.fail]],
    },
    {
      id: IDS.fail,
      type: "function",
      z: tabId,
      name: `${prefix} bloquear muestra fallida`,
      func: buildFailureFunction(),
      outputs: 1,
      timeout: 0,
      noerr: 0,
      initialize: "",
      finalize: "",
      libs: [],
      x: 1610,
      y: 720,
      wires: [[]],
    },
    {
      id: IDS.catch,
      type: "catch",
      z: tabId,
      name: `${prefix} capturar fallo sin muestra`,
      scope: [
        IDS.prepare,
        IDS.guard,
        IDS.http,
        IDS.validate,
        IDS.retryDelay,
        IDS.modbusWrite,
        IDS.restore,
        AUDIT.genericIn,
        AUDIT.flexGetter,
        AUDIT.genericOut,
        AUDIT.sanitizeProduction,
        AUDIT.saveDb,
      ],
      uncaught: false,
      x: 1360,
      y: 720,
      wires: [[IDS.fail]],
    },
  ];
}

function isOwnedNode(current, desired) {
  if (!current || current.type !== desired.type ||
      typeof current.name !== "string") {
    return false;
  }
  if (current.name === desired.name ||
      current.name.startsWith("[PILOTO v4 ACUMULATIVO]") ||
      current.name.startsWith("[PILOTO v3]")) {
    return true;
  }
  const legacyIds = new Set([IDS.http, IDS.validate, IDS.catch]);
  return legacyIds.has(current.id) &&
    (
      current.name.includes("conteo CORTE oficial") ||
      current.name.includes("CONTEO_1 oficial") ||
      current.name.includes("conteo de corrida activa") ||
      current.name.includes("CONTEO_1 de corrida") ||
      current.name.includes("Bloquear conteo si API falla")
    );
}

function upsertOwnedNode(flows, desired, changes) {
  const index = flows.findIndex((node) => node && node.id === desired.id);
  if (index === -1) {
    flows.push(desired);
    changes.push(`anadido nodo ${desired.name}`);
    return;
  }
  const current = flows[index];
  if (!isOwnedNode(current, desired)) {
    throw new Error(`colision con ID reservado ${desired.id}`);
  }
  if (JSON.stringify(current) !== JSON.stringify(desired)) {
    flows[index] = desired;
    changes.push(`actualizado nodo ${desired.name}`);
  }
}

function assertUniqueIds(flows) {
  const seen = new Set();
  for (const node of flows) {
    if (!node || typeof node.id !== "string" || node.id === "") {
      throw new Error("todos los nodos deben tener un ID");
    }
    if (seen.has(node.id)) throw new Error(`ID duplicado: ${node.id}`);
    seen.add(node.id);
  }
}

function findIncomingNodes(flows, targetId) {
  const incoming = [];
  for (const node of flows) {
    if (!Array.isArray(node.wires)) continue;
    const targets = node.wires.flatMap((output) =>
      Array.isArray(output) ? output : []
    );
    if (targets.includes(targetId)) incoming.push(node.id);
  }
  return incoming.sort();
}

function assertValid(
  flows,
  production,
  expectedNodes,
  untouchedById,
  expectedSaveDbWires,
  expectedGenericInWires,
  senderSerialized,
  saveDbMode,
  saveDbSerialized
) {
  assertUniqueIds(flows);
  if (LEGACY_TRANSITION_RE.test(production.func) ||
      LEGACY_GLOBAL_WRITE_RE.test(production.func)) {
    throw new Error("Produccion conserva escritura/incremento legado");
  }
  if (!production.func.includes(COUNT_OWNER_MARKER) ||
      !production.func.includes(READ_ONLY_MARKER)) {
    throw new Error("faltan marcadores v4 en Produccion");
  }
  if (!production.func.includes(MOTION_STOP_MARKER) ||
      !production.func.includes("typeof datos.presence_motion !== \"boolean\"") ||
      !production.func.includes("typeof datos.motion_ready !== \"boolean\"") ||
      !production.func.includes("typeof datos.motion_fresh !== \"boolean\"") ||
      !production.func.includes("Number(datos.micro_stop_max_s)") ||
      /estadoActual\s*===\s*["']idle["']/.test(production.func)) {
    throw new Error("Produccion no usa exclusivamente movimiento para paradas");
  }
  if (!production.func.includes(PROGRESS_DEBUG_MARKER) ||
      !production.func.includes("datos.progress_estimated_pct") ||
      /global\.set\s*\(\s*["']L1_avance/i.test(production.func)) {
    throw new Error("Produccion no refleja el avance estimado de forma read-only");
  }
  for (const [id, serialized] of untouchedById) {
    const node = flows.find((candidate) => candidate.id === id);
    if (!node || JSON.stringify(node) !== serialized) {
      throw new Error(`el parche altero un nodo fuera de alcance: ${id}`);
    }
  }
  for (const desired of expectedNodes) {
    const actual = flows.find((node) => node.id === desired.id);
    if (!actual || JSON.stringify(actual) !== JSON.stringify(desired)) {
      throw new Error(`nodo administrado invalido: ${desired.id}`);
    }
  }
  for (const obsoleteId of Object.values(OBSOLETE_IDS)) {
    if (flows.some((node) => node.id === obsoleteId)) {
      throw new Error(`nodo de recuperacion v3 sigue presente: ${obsoleteId}`);
    }
  }

  const interval = getRequiredNode(
    flows,
    AUDIT.productionInterval,
    "Interval"
  );
  if (Number(interval.interval) !== SAMPLE_INTERVAL_SECONDS ||
      interval.payload !== "" ||
      Number(interval.initialdefer) !== 0 ||
      JSON.stringify(interval.wires) !== JSON.stringify([[IDS.delay]])) {
    throw new Error("Interval de produccion no quedo aislado por v4");
  }
  const manual = getRequiredNode(flows, AUDIT.manualInject, "inject");
  if (manual.payloadType !== "date" ||
      String(manual.repeat || "") !== "" ||
      manual.once !== false ||
      JSON.stringify(manual.wires) !== JSON.stringify([[]])) {
    throw new Error("inject manual no quedo deshabilitado");
  }
  const publisher = getRequiredNode(
    flows,
    AUDIT.globalPublisher,
    "function"
  );
  if (ensureOutput(publisher, 3).length !== 0) {
    throw new Error("el writer legado sigue conectado");
  }
  const saveDb = getRequiredNode(flows, AUDIT.saveDb, "SaveDB");
  if (saveDbMode === SAVE_DB_MODE_STRICT) {
    if (saveDb.strictDelta !== true ||
        JSON.stringify(saveDb.wires) !== JSON.stringify(expectedSaveDbWires)) {
      throw new Error("SaveDB no conserva wires/strictDelta esperados");
    }
  } else if (saveDbMode === SAVE_DB_MODE_PRESERVE) {
    if (JSON.stringify(saveDb) !== saveDbSerialized) {
      throw new Error("SaveDB legacy fue modificado en modo preservacion");
    }
  } else {
    throw new Error(`modo SaveDB inesperado: ${String(saveDbMode)}`);
  }
  const genericIn = getRequiredNode(flows, AUDIT.genericIn, "Generic In");
  if (JSON.stringify(genericIn.wires) !==
      JSON.stringify(expectedGenericInWires)) {
    throw new Error("Generic In no conserva sus wires no administrados");
  }
  const writer = getRequiredNode(flows, IDS.modbusWrite, "modbus-write");
  const getter = getRequiredNode(
    flows,
    AUDIT.flexGetter,
    "modbus-flex-getter"
  );
  if (writer.server !== getter.server ||
      writer.adr !== "8" ||
      writer.quantity !== "2" ||
      writer.keepMsgProperties !== true) {
    throw new Error("writer acumulativo no coincide con Modbus 8:9");
  }
  const sender = getRequiredNode(flows, AUDIT.senderInterval, "Interval");
  if (JSON.stringify(sender) !== senderSerialized) {
    throw new Error("Interval del Sender fue modificado");
  }
  const incoming = findIncomingNodes(flows, AUDIT.genericIn);
  if (JSON.stringify(incoming) !== JSON.stringify([IDS.restore])) {
    throw new Error(
      `Generic In conserva bypasses: ${incoming.join(",") || "ninguno"}`
    );
  }
}

function patchFlows(inputFlows, rawOptions = {}) {
  if (!Array.isArray(inputFlows)) {
    throw new Error("flows.json debe contener un arreglo JSON");
  }
  const options = normalizeOptions(rawOptions);
  const flows = clone(inputFlows);
  assertUniqueIds(flows);

  const production = findProductionNode(flows);
  const productionWiresBefore = JSON.stringify(production.wires || []);
  const interval = getRequiredNode(
    flows,
    AUDIT.productionInterval,
    "Interval"
  );
  const manual = getRequiredNode(flows, AUDIT.manualInject, "inject");
  const genericIn = getRequiredNode(flows, AUDIT.genericIn, "Generic In");
  const getter = getRequiredNode(
    flows,
    AUDIT.flexGetter,
    "modbus-flex-getter"
  );
  const genericOut = getRequiredNode(
    flows,
    AUDIT.genericOut,
    "Generic Out"
  );
  const sanitize = getRequiredNode(
    flows,
    AUDIT.sanitizeProduction,
    "function"
  );
  const saveDb = getRequiredNode(flows, AUDIT.saveDb, "SaveDB");
  const saveDbSerialized = JSON.stringify(saveDb);
  const sender = getRequiredNode(
    flows,
    AUDIT.senderInterval,
    "Interval"
  );
  const publisher = getRequiredNode(
    flows,
    AUDIT.globalPublisher,
    "function"
  );
  getRequiredNode(flows, AUDIT.legacyCountConverter, "function");

  const tabId = interval.z;
  for (const node of [manual, genericIn, getter, genericOut, sanitize, saveDb]) {
    if (node.z !== tabId) {
      throw new Error(`nodo ${node.id} no pertenece al tab de produccion`);
    }
  }
  if (Number(interval.interval) !== SAMPLE_INTERVAL_SECONDS ||
      interval.payload !== "" ||
      Number(interval.initialdefer) !== 0) {
    throw new Error(
      "Interval 570d133c1babc579 debe ser 300 s, payload vacio y defer 0"
    );
  }
  if (typeof getter.server !== "string" || getter.server === "") {
    throw new Error("Modbus getter no tiene servidor");
  }

  const senderSerialized = JSON.stringify(sender);
  const expectedSaveDbWires = clone(saveDb.wires || []);
  if (Array.isArray(expectedSaveDbWires[0])) {
    expectedSaveDbWires[0] = expectedSaveDbWires[0]
      .filter((target) => target !== OBSOLETE_IDS.complete);
  }
  const expectedGenericInWires = clone(genericIn.wires || []);
  if (Array.isArray(expectedGenericInWires[1])) {
    expectedGenericInWires[1] = expectedGenericInWires[1]
      .filter((target) => target !== IDS.fail);
  }

  const allManagedIds = new Set([
    ...Object.values(IDS),
    ...Object.values(OBSOLETE_IDS),
  ]);
  const modifiedIds = new Set([
    production.id,
    AUDIT.productionInterval,
    AUDIT.manualInject,
    AUDIT.genericIn,
    AUDIT.globalPublisher,
    ...allManagedIds,
  ]);
  if (!options.preserveLegacySaveDb) {
    modifiedIds.add(AUDIT.saveDb);
  }
  const untouchedById = new Map(
    inputFlows
      .filter((node) => !modifiedIds.has(node.id))
      .map((node) => [node.id, JSON.stringify(node)])
  );
  const changes = [];

  removeLegacyCountLogic(production, changes);
  migrateMotionStopLogic(production, changes);
  migrateEstimatedProgressDebug(production, changes);
  if (JSON.stringify(production.wires || []) !== productionWiresBefore) {
    throw new Error("se modificaron conexiones de Produccion");
  }
  rewireProductionInterval(interval, changes);
  disableManualInject(manual, changes);
  disconnectLegacyCountWriter(publisher, changes);
  removeManagedWire(
    genericIn,
    1,
    IDS.fail,
    changes,
    "retirado wire de timeout administrado por v3"
  );
  if (options.preserveLegacySaveDb) {
    if (
      !Array.isArray(saveDb.wires) ||
      saveDb.wires.some((output) => !Array.isArray(output))
    ) {
      throw new Error(
        "modo preserve-legacy-save-db requiere wires SaveDB validos"
      );
    }
    const hasObsoleteSaveDbWire = saveDb.wires.some((output) =>
      output.includes(OBSOLETE_IDS.complete)
    );
    if (hasObsoleteSaveDbWire) {
      throw new Error(
        "modo preserve-legacy-save-db no admite wire SaveDB de v3"
      );
    }
  } else {
    removeManagedWire(
      saveDb,
      0,
      OBSOLETE_IDS.complete,
      changes,
      "retirada marca de ventana SaveDB de v3"
    );
    if (saveDb.strictDelta !== true) {
      saveDb.strictDelta = true;
      changes.push("SaveDB configurado strictDelta=true");
    }
  }
  removeObsoleteNodes(flows, changes);

  const unitId = parsePositiveInteger(
    genericIn.unitid === undefined ? 1 : genericIn.unitid,
    "Generic In unitid",
    247
  );
  const expectedNodes = desiredNodes(
    tabId,
    options.lineaId,
    getter.server,
    unitId
  );
  for (const desired of expectedNodes) {
    upsertOwnedNode(flows, desired, changes);
  }

  assertValid(
    flows,
    production,
    expectedNodes,
    untouchedById,
    expectedSaveDbWires,
    expectedGenericInWires,
    senderSerialized,
    options.preserveLegacySaveDb
      ? SAVE_DB_MODE_PRESERVE
      : SAVE_DB_MODE_STRICT,
    saveDbSerialized
  );

  return {
    flows,
    changes,
    productionId: production.id,
    tabId,
    lineaId: options.lineaId,
    senderInterval: String(sender.interval),
    saveDbMode: options.preserveLegacySaveDb
      ? SAVE_DB_MODE_PRESERVE
      : SAVE_DB_MODE_STRICT,
  };
}

function timestamp() {
  return new Date()
    .toISOString()
    .replace(/[-:]/g, "")
    .replace(/\.\d{3}Z$/, "Z");
}

function writeAtomically(inputPath, flows) {
  const directory = path.dirname(inputPath);
  const basename = path.basename(inputPath);
  const stat = fs.statSync(inputPath);
  const backup = `${inputPath}.bak.${timestamp()}.${process.pid}`;
  const temporary = path.join(
    directory,
    `.${basename}.${PATCH_VERSION.replace(/[^a-z0-9]+/gi, "-")}.${process.pid}.tmp`
  );
  const serialized = `${JSON.stringify(flows, null, 2)}\n`;

  fs.copyFileSync(inputPath, backup, fs.constants.COPYFILE_EXCL);
  fs.chmodSync(backup, 0o600);
  let fd;
  try {
    fd = fs.openSync(temporary, "wx", stat.mode & 0o777);
    fs.writeFileSync(fd, serialized, "utf8");
    fs.fsyncSync(fd);
    fs.closeSync(fd);
    fd = undefined;
    fs.renameSync(temporary, inputPath);
    try {
      const directoryFd = fs.openSync(directory, "r");
      fs.fsyncSync(directoryFd);
      fs.closeSync(directoryFd);
    } catch (_) {
      // Algunos sistemas de archivos no admiten fsync sobre directorios.
    }
  } catch (error) {
    if (fd !== undefined) fs.closeSync(fd);
    try {
      fs.unlinkSync(temporary);
    } catch (_) {
      // El temporal puede no haberse creado.
    }
    throw error;
  }
  return backup;
}

function main(argv = process.argv.slice(2), io = console) {
  const args = parseArgs(argv);
  if (args.help) {
    usage(io);
    return {help: true};
  }
  const raw = fs.readFileSync(args.input, "utf8");
  let inputFlows;
  try {
    inputFlows = JSON.parse(raw);
  } catch (error) {
    throw new Error(`flows.json invalido: ${error.message}`);
  }
  const result = patchFlows(inputFlows, {
    lineaId: args.lineaId,
    preserveLegacySaveDb: args.preserveLegacySaveDb,
  });

  io.log(args.apply ? "MODO: APPLY" : "MODO: DRY-RUN");
  io.log(`linea_id: ${result.lineaId}`);
  io.log("Fuente: /vision/counter; counter_epoch administrado por backend.");
  io.log(`Sender interval: conservado en ${result.senderInterval} s.`);
  io.log(`SaveDB mode: ${result.saveDbMode}.`);
  if (result.changes.length === 0) {
    io.log("Cambios: ninguno; el flujo ya coincide con v4.");
  } else {
    io.log(`Cambios propuestos: ${result.changes.length}`);
    for (const change of result.changes) io.log(`- ${change}`);
  }

  let backup = null;
  if (!args.apply) {
    io.log("DRY-RUN: no se escribio flows.json.");
  } else if (result.changes.length === 0) {
    io.log("APPLY: no se escribio; no habia cambios.");
  } else {
    backup = writeAtomically(args.input, result.flows);
    io.log(`APPLY: backup creado en ${backup}`);
    io.log("APPLY: flows.json reemplazado de forma atomica.");
  }
  return {...result, backup, applied: Boolean(backup)};
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(`ERROR: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = {
  AUDIT,
  COUNT_OWNER_MARKER,
  DEFAULT_LINE_ID,
  IDS,
  LOCK_STORE_KEY,
  MIN_AS_OF_LAG_MS,
  MOTION_STOP_MARKER,
  MOTION_STOP_CONTEXT_KEY,
  MOTION_READY_CONTEXT_KEY,
  MOTION_STOP_VERSION,
  OBSOLETE_IDS,
  PATCH_VERSION,
  PROGRESS_DEBUG_MARKER,
  PROGRESS_DEBUG_VERSION,
  READ_ONLY_MARKER,
  SAMPLE_INTERVAL_MS,
  SAMPLE_INTERVAL_SECONDS,
  SAMPLE_LOCK_TTL_MS,
  SAVE_DB_MODE_PRESERVE,
  SAVE_DB_MODE_STRICT,
  SETTLE_DELAY_SECONDS,
  MAX_TOO_EARLY_RETRIES,
  TOO_EARLY_RETRY_SECONDS,
  UINT32_MAX,
  VISION_COUNTER_URL,
  buildGuardFunction,
  buildPrepareFunction,
  buildValidateFunction,
  parseArgs,
  patchFlows,
  runCLI: main,
  validateVisionCounterResponse,
};
