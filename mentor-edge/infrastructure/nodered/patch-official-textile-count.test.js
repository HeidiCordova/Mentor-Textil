"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const {
  AUDIT,
  COUNT_OWNER_MARKER,
  IDS,
  LOCK_STORE_KEY,
  MIN_AS_OF_LAG_MS,
  MAX_TOO_EARLY_RETRIES,
  OBSOLETE_IDS,
  READ_ONLY_MARKER,
  SAMPLE_INTERVAL_MS,
  SAMPLE_LOCK_TTL_MS,
  SAVE_DB_MODE_PRESERVE,
  SAVE_DB_MODE_STRICT,
  SETTLE_DELAY_SECONDS,
  TOO_EARLY_RETRY_SECONDS,
  UINT32_MAX,
  VISION_COUNTER_URL,
  buildGuardFunction,
  buildPrepareFunction,
  buildValidateFunction,
  parseArgs,
  patchFlows,
  runCLI,
  validateVisionCounterResponse,
} = require("./patch-official-textile-count.js");

const B = 1710000000000;
const COUNTER_EPOCH = 1700000000000;
const TAB = "tab-production";

function contextStore(initial = {}) {
  const values = new Map(Object.entries(initial));
  return {
    values,
    api: {
      get(key) {
        return values.get(key);
      },
      set(key, value) {
        values.set(key, value);
      },
    },
  };
}

function functionRuntime(initialGlobal = {}, initialFlow = {}) {
  const globalStore = contextStore(initialGlobal);
  const flowStore = contextStore(initialFlow);
  const statuses = [];
  const warnings = [];
  const node = {
    status(value) {
      statuses.push(value);
    },
    warn(value) {
      warnings.push(String(value));
    },
    error(value) {
      warnings.push(String(value));
    },
  };
  return {
    global: globalStore.api,
    globalValues: globalStore.values,
    flow: flowStore.api,
    flowValues: flowStore.values,
    statuses,
    warnings,
    node,
  };
}

function executeFunction(source, msg, runtime) {
  const execute = new Function(
    "msg",
    "global",
    "flow",
    "context",
    "node",
    "env",
    source
  );
  return execute(
    msg,
    runtime.global,
    runtime.flow,
    runtime.flow,
    runtime.node,
    {get() { return undefined; }}
  );
}

function validPayload(overrides = {}) {
  return {
    linea_id: 1,
    count: 42,
    counter_epoch: new Date(COUNTER_EPOCH).toISOString(),
    until: new Date(B).toISOString(),
    as_of: new Date(B + MIN_AS_OF_LAG_MS).toISOString(),
    state_updated_at: new Date(B + 5000).toISOString(),
    event_type: "CORTE",
    ...overrides,
  };
}

function expectedSample() {
  return {lineaId: 1, sampleTime: B};
}

function legacyProductionNode() {
  return {
    id: "production-function",
    type: "function",
    z: "tab-detector",
    name: "Produccion Art Atlas",
    func: [
      "let conteo = leerGlobal(\"L1_conteo_1\");",
      "let tiempoDisponible = leerGlobal(\"L1_t_disponible\");",
      "const estadoAnterior = context.get(\"prod_estado_anterior\");",
      "const estadoActual = \"idle\";",
      "// Se considera un corte terminado cuando la prenda sale:",
      "// en_prenda -> idle",
      "if (",
      "    estadoAnterior === \"en_prenda\" &&",
      "    estadoActual === \"idle\"",
      ") {",
      "    conteo += 1;",
      "}",
      "global.set(",
      "    \"L1_conteo_1\",",
      "    Math.floor(conteo)",
      ");",
      "return msg;",
    ].join("\n"),
    outputs: 1,
    wires: [["production-debug"]],
  };
}

function syntheticFlows() {
  return [
    legacyProductionNode(),
    {
      id: AUDIT.productionInterval,
      type: "Interval",
      z: TAB,
      interval: "300",
      payload: "",
      name: "",
      initialdefer: 0,
      wires: [[AUDIT.genericIn]],
    },
    {
      id: AUDIT.manualInject,
      type: "inject",
      z: TAB,
      name: "",
      props: [{p: "payload"}, {p: "topic", vt: "str"}],
      repeat: "",
      crontab: "",
      once: false,
      onceDelay: 0.1,
      topic: "",
      payload: "",
      payloadType: "date",
      wires: [[AUDIT.genericIn]],
    },
    {
      id: AUDIT.genericIn,
      type: "Generic In",
      z: TAB,
      name: "ART_ATLAS_MAQUINA_1_PRODUCCION",
      unitid: 1,
      defer: "10000",
      retries: "10",
      wires: [[AUDIT.flexGetter], []],
    },
    {
      id: AUDIT.flexGetter,
      type: "modbus-flex-getter",
      z: TAB,
      name: "ART_ATLAS_LOCAL",
      server: "modbus-production",
      keepMsgProperties: false,
      wires: [[], [AUDIT.genericOut]],
    },
    {
      id: AUDIT.genericOut,
      type: "Generic Out",
      z: TAB,
      name: "",
      wires: [[AUDIT.sanitizeProduction]],
    },
    {
      id: AUDIT.sanitizeProduction,
      type: "function",
      z: TAB,
      name: "QUITAR MENTOR_ID PRODUCCION",
      func: "return msg;",
      wires: [[AUDIT.saveDb, "debug-production"]],
    },
    {
      id: AUDIT.saveDb,
      type: "SaveDB",
      z: TAB,
      name: "",
      interval: "1",
      wires: [["save-debug"]],
    },
    {
      id: AUDIT.senderInterval,
      type: "Interval",
      z: TAB,
      name: "",
      interval: "300",
      payload: "",
      initialdefer: 0,
      wires: [["sender-general"]],
    },
    {
      id: AUDIT.globalPublisher,
      type: "function",
      z: "tab-modbus",
      name: "Datos Variables Globales",
      func: "return [msg,msg,msg,msg,msg,msg,msg,msg];",
      wires: [
        ["writer-0"],
        ["writer-1"],
        ["writer-2"],
        [AUDIT.legacyCountConverter],
        ["writer-4"],
        ["writer-5"],
        ["writer-6"],
        ["writer-7"],
      ],
    },
    {
      id: AUDIT.legacyCountConverter,
      type: "function",
      z: "tab-modbus",
      name: "Numeric to DWORD",
      func: "return msg;",
      wires: [["legacy-count-writer"]],
    },
    {
      id: "legacy-count-writer",
      type: "modbus-write",
      z: "tab-modbus",
      name: "L1_CONTEO 01:8",
      adr: "8",
      quantity: "2",
      server: "legacy-server-alias",
      keepMsgProperties: false,
      wires: [[], []],
    },
    {
      id: "sentinel",
      type: "debug",
      z: TAB,
      name: "must remain byte-for-byte",
      active: true,
      wires: [],
    },
  ];
}

test("validador acepta contador acumulativo exacto para B", () => {
  const result = validateVisionCounterResponse(
    {statusCode: 200, payload: validPayload()},
    expectedSample()
  );
  assert.equal(result.ok, true);
  assert.equal(result.count, 42);
  assert.equal(result.counterEpoch, COUNTER_EPOCH);
});

for (const scenario of [
  ["error de transporte", {error: {message: "offline"}}, /transporte/],
  ["HTTP no 200", {statusCode: 503, payload: validPayload()}, /HTTP 503/],
  [
    "linea distinta",
    {statusCode: 200, payload: validPayload({linea_id: 2})},
    /linea_id/,
  ],
  [
    "evento distinto",
    {statusCode: 200, payload: validPayload({event_type: "OTRO"})},
    /event_type/,
  ],
  [
    "until distinto",
    {
      statusCode: 200,
      payload: validPayload({until: new Date(B + 1).toISOString()}),
    },
    /until/,
  ],
  [
    "as_of ausente",
    {statusCode: 200, payload: validPayload({as_of: undefined})},
    /as_of/,
  ],
  [
    "as_of antes de B+10s",
    {
      statusCode: 200,
      payload: validPayload({
        as_of: new Date(B + MIN_AS_OF_LAG_MS - 1).toISOString(),
      }),
    },
    /cierre seguro/,
  ],
  [
    "counter_epoch ausente",
    {
      statusCode: 200,
      payload: validPayload({counter_epoch: undefined}),
    },
    /counter_epoch/,
  ],
  [
    "counter_epoch posterior a B",
    {
      statusCode: 200,
      payload: validPayload({
        counter_epoch: new Date(B + 1).toISOString(),
      }),
    },
    /counter_epoch/,
  ],
  [
    "state_updated_at ausente",
    {
      statusCode: 200,
      payload: validPayload({state_updated_at: undefined}),
    },
    /state_updated_at/,
  ],
  [
    "state_updated_at anterior al epoch",
    {
      statusCode: 200,
      payload: validPayload({
        state_updated_at: new Date(COUNTER_EPOCH - 1).toISOString(),
      }),
    },
    /state_updated_at/,
  ],
  [
    "state_updated_at posterior a as_of",
    {
      statusCode: 200,
      payload: validPayload({
        state_updated_at: new Date(B + MIN_AS_OF_LAG_MS + 1).toISOString(),
      }),
    },
    /state_updated_at/,
  ],
  [
    "count negativo",
    {statusCode: 200, payload: validPayload({count: -1})},
    /UINT32/,
  ],
  [
    "count string",
    {statusCode: 200, payload: validPayload({count: "42"})},
    /UINT32/,
  ],
  [
    "count overflow",
    {statusCode: 200, payload: validPayload({count: UINT32_MAX + 1})},
    /UINT32/,
  ],
]) {
  test(`validador rechaza ${scenario[0]}`, () => {
    const result = validateVisionCounterResponse(
      scenario[1],
      expectedSample()
    );
    assert.equal(result.ok, false);
    assert.match(result.error, scenario[2]);
  });
}

test("Function escribe UINT32 y global acumulativo solo en exito", () => {
  const runtime = functionRuntime();
  const msg = {
    statusCode: 200,
    payload: validPayload({count: 0x12345678}),
    _counterLineId: 1,
    _counterSampleTime: B,
    _counterSampleId: `1:${B}`,
  };
  const [success, failure] = executeFunction(
    buildValidateFunction(),
    msg,
    runtime
  );
  assert.equal(failure, null);
  assert.deepEqual(success.payload, [0x1234, 0x5678]);
  assert.equal(runtime.globalValues.get("L1_conteo_1"), 0x12345678);
  assert.equal(
    runtime.globalValues.get("L1_conteo_1_counter_epoch"),
    new Date(COUNTER_EPOCH).toISOString()
  );
  assert.equal(
    runtime.globalValues.get("L1_conteo_1_counter_valid"),
    true
  );
});

test("HTTP 425 reintenta exactamente el mismo B y tiene limite", () => {
  const runtime = functionRuntime();
  const source = buildValidateFunction();
  const originalUrl =
    `${VISION_COUNTER_URL}?linea_id=1&until=` +
    encodeURIComponent(new Date(B).toISOString());
  const [success, failure, retry] = executeFunction(
    source,
    {
      statusCode: 425,
      payload: "too early",
      url: originalUrl,
      _counterLineId: 1,
      _counterSampleTime: B,
      _counterSampleId: `1:${B}`,
    },
    runtime
  );
  assert.equal(success, null);
  assert.equal(failure, null);
  assert.ok(retry);
  assert.equal(retry.payload, B);
  assert.equal(retry.url, originalUrl);
  assert.equal(retry._counterSampleTime, B);
  assert.equal(retry._counterRetry425, 1);

  const [, exhausted] = executeFunction(
    source,
    {
      statusCode: 425,
      payload: "too early",
      url: originalUrl,
      _counterLineId: 1,
      _counterSampleTime: B,
      _counterSampleId: `1:${B}`,
      _counterRetry425: MAX_TOO_EARLY_RETRIES,
    },
    runtime
  );
  assert.ok(exhausted);
  assert.match(exhausted._counterFailureReason, /excedio reintentos/);
});

test("mismo counter_epoch no puede retroceder", () => {
  const epochISO = new Date(COUNTER_EPOCH).toISOString();
  const runtime = functionRuntime({
    L1_conteo_1: 50,
    L1_conteo_1_counter_epoch: epochISO,
  });
  const [success, failure] = executeFunction(
    buildValidateFunction(),
    {
      statusCode: 200,
      payload: validPayload({count: 49}),
      _counterLineId: 1,
      _counterSampleTime: B,
    },
    runtime
  );
  assert.equal(success, null);
  assert.ok(failure);
  assert.equal(runtime.globalValues.get("L1_conteo_1"), 50);
  assert.equal(
    runtime.globalValues.get("L1_conteo_1_counter_valid"),
    false
  );
});

test("un counter_epoch nuevo autoriza reinicio explicito del total", () => {
  const runtime = functionRuntime({
    L1_conteo_1: 50,
    L1_conteo_1_counter_epoch: new Date(COUNTER_EPOCH - 1000).toISOString(),
  });
  const [success, failure] = executeFunction(
    buildValidateFunction(),
    {
      statusCode: 200,
      payload: validPayload({count: 2}),
      _counterLineId: 1,
      _counterSampleTime: B,
    },
    runtime
  );
  assert.ok(success);
  assert.equal(failure, null);
  assert.equal(runtime.globalValues.get("L1_conteo_1"), 2);
});

test("repetir el mismo B conserva exactamente total, epoch y timestamp", () => {
  const runtime = functionRuntime();
  const source = buildValidateFunction();
  function request() {
    return {
      statusCode: 200,
      payload: validPayload({count: 77}),
      _counterLineId: 1,
      _counterSampleTime: B,
      _counterSampleId: `1:${B}`,
    };
  }
  const [first] = executeFunction(source, request(), runtime);
  const [second] = executeFunction(source, request(), runtime);
  assert.deepEqual(first.payload, second.payload);
  assert.equal(first._counterSampleTime, B);
  assert.equal(second._counterSampleTime, B);
  assert.equal(first._counterEpoch, second._counterEpoch);
  assert.equal(runtime.globalValues.get("L1_conteo_1"), 77);
});

test("prepare consulta /vision/counter solo con linea_id y until=B", () => {
  const runtime = functionRuntime();
  const result = executeFunction(
    buildPrepareFunction(3),
    {payload: B},
    runtime
  );
  assert.equal(result._counterLineId, 3);
  assert.equal(result._counterSampleTime, B);
  assert.equal(result._counterSampleId, `3:${B}`);
  const url = new URL(result.url);
  assert.equal(`${url.origin}${url.pathname}`, VISION_COUNTER_URL);
  assert.deepEqual(
    [...url.searchParams.keys()].sort(),
    ["linea_id", "until"]
  );
  assert.equal(url.searchParams.get("linea_id"), "3");
  assert.equal(Date.parse(url.searchParams.get("until")), B);
  assert.equal(url.searchParams.has("since"), false);
});

test("guard evita dos consultas concurrentes del mismo B y expira", () => {
  const runtime = functionRuntime();
  const prepare = buildPrepareFunction(1);
  const guard = buildGuardFunction();
  const first = executeFunction(prepare, {payload: B}, runtime);
  const duplicate = executeFunction(prepare, {payload: B}, runtime);
  assert.ok(executeFunction(guard, first, runtime));
  assert.equal(executeFunction(guard, duplicate, runtime), null);

  const samples = runtime.flowValues.get(LOCK_STORE_KEY);
  samples[`1:${B}`].expiresAt = Date.now() - 1;
  runtime.flowValues.set(LOCK_STORE_KEY, samples);
  assert.ok(
    executeFunction(
      guard,
      executeFunction(prepare, {payload: B}, runtime),
      runtime
    )
  );
  assert.ok(SAMPLE_LOCK_TTL_MS < SAMPLE_INTERVAL_MS);
});

test("patch v4 acumulativo es idempotente y mantiene Sender/cadena", () => {
  const input = syntheticFlows();
  const senderBefore = JSON.stringify(
    input.find((node) => node.id === AUDIT.senderInterval)
  );
  const sanitizeBefore = JSON.stringify(
    input.find((node) => node.id === AUDIT.sanitizeProduction)
  );
  const converterBefore = JSON.stringify(
    input.find((node) => node.id === AUDIT.legacyCountConverter)
  );
  const legacyWriterBefore = JSON.stringify(
    input.find((node) => node.id === "legacy-count-writer")
  );
  const publisherBefore = structuredClone(
    input.find((node) => node.id === AUDIT.globalPublisher)
  );

  const first = patchFlows(input);
  const second = patchFlows(first.flows);
  assert.deepEqual(second.flows, first.flows);
  assert.deepEqual(second.changes, []);

  const production = first.flows.find(
    (node) => node.id === "production-function"
  );
  assert.ok(production.func.includes(COUNT_OWNER_MARKER));
  assert.ok(production.func.includes(READ_ONLY_MARKER));
  assert.doesNotMatch(production.func, /conteo\s*\+=\s*1/);
  assert.doesNotMatch(
    production.func,
    /global\.set\s*\(\s*["']L1_conteo_1/
  );
  assert.deepEqual(
    first.flows.find(
      (node) => node.id === AUDIT.productionInterval
    ).wires,
    [[IDS.delay]]
  );
  assert.deepEqual(
    first.flows.find((node) => node.id === AUDIT.manualInject).wires,
    [[]]
  );
  const delay = first.flows.find((node) => node.id === IDS.delay);
  assert.equal(delay.pauseType, "delay");
  assert.equal(delay.timeout, String(SETTLE_DELAY_SECONDS));
  const retryDelay = first.flows.find(
    (node) => node.id === IDS.retryDelay
  );
  assert.equal(retryDelay.timeout, String(TOO_EARLY_RETRY_SECONDS));
  assert.deepEqual(retryDelay.wires, [[IDS.http]]);

  const publisher = first.flows.find(
    (node) => node.id === AUDIT.globalPublisher
  );
  assert.deepEqual(publisher.wires[3], []);
  for (let i = 0; i < publisherBefore.wires.length; i += 1) {
    if (i !== 3) {
      assert.deepEqual(publisher.wires[i], publisherBefore.wires[i]);
    }
  }
  assert.equal(
    JSON.stringify(first.flows.find(
      (node) => node.id === AUDIT.legacyCountConverter
    )),
    converterBefore
  );
  assert.equal(
    JSON.stringify(first.flows.find(
      (node) => node.id === "legacy-count-writer"
    )),
    legacyWriterBefore
  );

  const writer = first.flows.find((node) => node.id === IDS.modbusWrite);
  assert.equal(writer.server, "modbus-production");
  assert.equal(writer.adr, "8");
  assert.equal(writer.quantity, "2");
  assert.equal(writer.keepMsgProperties, true);
  assert.deepEqual(writer.wires, [[IDS.restore], [IDS.fail]]);

  const saveDb = first.flows.find((node) => node.id === AUDIT.saveDb);
  assert.equal(saveDb.strictDelta, true);
  assert.deepEqual(saveDb.wires, [["save-debug"]]);
  assert.equal(
    JSON.stringify(first.flows.find(
      (node) => node.id === AUDIT.senderInterval
    )),
    senderBefore
  );
  assert.equal(
    JSON.stringify(first.flows.find(
      (node) => node.id === AUDIT.sanitizeProduction
    )),
    sanitizeBefore
  );
  for (const obsoleteId of Object.values(OBSOLETE_IDS)) {
    assert.equal(first.flows.some((node) => node.id === obsoleteId), false);
  }
});

test("migracion v3 elimina recuperación y sus wires administrados", () => {
  const v3 = patchFlows(syntheticFlows()).flows;
  for (const id of Object.values(IDS)) {
    const node = v3.find((candidate) => candidate.id === id);
    if (node) {
      node.name = node.name.replace(
        "[PILOTO v4 ACUMULATIVO]",
        "[PILOTO v3]"
      );
    }
  }
  const retryIndex = v3.findIndex((node) => node.id === IDS.retryDelay);
  v3.splice(retryIndex, 1);
  const manual = v3.find((node) => node.id === AUDIT.manualInject);
  manual.wires = [[OBSOLETE_IDS.recover]];
  const genericIn = v3.find((node) => node.id === AUDIT.genericIn);
  genericIn.wires[1].push(IDS.fail);
  const saveDb = v3.find((node) => node.id === AUDIT.saveDb);
  saveDb.wires[0].push(OBSOLETE_IDS.complete);
  for (const [id, suffix, type] of [
    [OBSOLETE_IDS.startup, "recuperar ultima ventana", "inject"],
    [OBSOLETE_IDS.recover, "calcular ultimo B cerrado", "function"],
    [OBSOLETE_IDS.rateLimit, "serializar ventanas", "delay"],
    [OBSOLETE_IDS.complete, "marcar ventana procesada", "function"],
  ]) {
    v3.push({
      id,
      type,
      z: TAB,
      name: `[PILOTO v3] ${suffix}`,
      wires: [[]],
    });
  }

  const migrated = patchFlows(v3);
  assert.deepEqual(
    migrated.flows.find((node) => node.id === AUDIT.manualInject).wires,
    [[]]
  );
  assert.deepEqual(
    migrated.flows.find((node) => node.id === AUDIT.genericIn).wires[1],
    []
  );
  assert.deepEqual(
    migrated.flows.find((node) => node.id === AUDIT.saveDb).wires,
    [["save-debug"]]
  );
  for (const obsoleteId of Object.values(OBSOLETE_IDS)) {
    assert.equal(migrated.flows.some((node) => node.id === obsoleteId), false);
  }
  assert.deepEqual(patchFlows(migrated.flows).changes, []);
});

test("modo conservador preserva SaveDB legacy byte por byte", () => {
  const input = syntheticFlows();
  const inputBefore = JSON.stringify(input);
  const saveDbBefore = JSON.stringify(
    input.find((node) => node.id === AUDIT.saveDb)
  );

  const first = patchFlows(input, {preserveLegacySaveDb: true});
  const saveDbAfter = JSON.stringify(
    first.flows.find((node) => node.id === AUDIT.saveDb)
  );

  assert.equal(first.saveDbMode, SAVE_DB_MODE_PRESERVE);
  assert.equal(JSON.stringify(input), inputBefore);
  assert.equal(saveDbAfter, saveDbBefore);
  assert.equal(
    first.changes.some((change) => change.includes("SaveDB")),
    false
  );

  const second = patchFlows(first.flows, {preserveLegacySaveDb: true});
  assert.deepEqual(second.changes, []);
  assert.equal(second.saveDbMode, SAVE_DB_MODE_PRESERVE);
  assert.equal(
    JSON.stringify(
      second.flows.find((node) => node.id === AUDIT.saveDb)
    ),
    saveDbBefore
  );

  const strict = patchFlows(first.flows);
  assert.equal(strict.saveDbMode, SAVE_DB_MODE_STRICT);
  assert.deepEqual(strict.changes, ["SaveDB configurado strictDelta=true"]);
  assert.equal(
    strict.flows.find((node) => node.id === AUDIT.saveDb).strictDelta,
    true
  );
});

test("modo conservador rechaza wires SaveDB obsoletos de v3", () => {
  const flows = syntheticFlows();
  flows.find((node) => node.id === AUDIT.saveDb).wires.push([
    OBSOLETE_IDS.complete,
  ]);
  const before = JSON.stringify(flows);

  assert.throws(
    () => patchFlows(flows, {preserveLegacySaveDb: true}),
    /no admite wire SaveDB de v3/
  );
  assert.equal(JSON.stringify(flows), before);
});

test("modo conservador rechaza opcion programatica no booleana", () => {
  for (const value of ["true", 1, null]) {
    assert.throws(
      () => patchFlows(syntheticFlows(), {preserveLegacySaveDb: value}),
      /requiere un booleano/
    );
  }
});

test("modo conservador rechaza estructura de wires SaveDB invalida", () => {
  for (const wires of [undefined, "invalid", [null]]) {
    const flows = syntheticFlows();
    const saveDb = flows.find((node) => node.id === AUDIT.saveDb);
    if (wires === undefined) {
      delete saveDb.wires;
    } else {
      saveDb.wires = wires;
    }
    assert.throws(
      () => patchFlows(flows, {preserveLegacySaveDb: true}),
      /requiere wires SaveDB validos/
    );
  }
});

test("patch falla cerrado ante bypasses de Interval o inject manual", () => {
  const intervalBypass = syntheticFlows();
  intervalBypass.find(
    (node) => node.id === AUDIT.productionInterval
  ).wires[0].push("destino-inesperado");
  assert.throws(
    () => patchFlows(intervalBypass),
    /Interval de produccion contiene wires inesperados/
  );

  const manualBypass = syntheticFlows();
  manualBypass.find(
    (node) => node.id === AUDIT.manualInject
  ).wires[0].push("destino-inesperado");
  assert.throws(
    () => patchFlows(manualBypass),
    /inject manual contiene wires inesperados/
  );

  const manualAutomatic = syntheticFlows();
  manualAutomatic.find(
    (node) => node.id === AUDIT.manualInject
  ).repeat = "5";
  assert.throws(
    () => patchFlows(manualAutomatic),
    /inject manual auditado debe ser date/
  );
});

test("patch no desconecta destinos inesperados del publisher global", () => {
  const flows = syntheticFlows();
  flows.find(
    (node) => node.id === AUDIT.globalPublisher
  ).wires[3].push("writer-no-auditado");
  assert.throws(
    () => patchFlows(flows),
    /destinos no auditados/
  );
});

test("patch rechaza colisiones con IDs administrados u obsoletos", () => {
  const managed = syntheticFlows();
  managed.push({
    id: IDS.delay,
    type: "debug",
    z: TAB,
    name: "unrelated",
    wires: [],
  });
  assert.throws(() => patchFlows(managed), /colision con ID reservado/);

  const obsolete = syntheticFlows();
  obsolete.push({
    id: OBSOLETE_IDS.startup,
    type: "debug",
    z: TAB,
    name: "unrelated",
    wires: [],
  });
  assert.throws(
    () => patchFlows(obsolete),
    /colision con ID obsoleto reservado/
  );
});

test("CLI solo permite linea_id; backend posee counter_epoch y Sender", () => {
  const defaults = parseArgs(["--input", "synthetic.json"]);
  assert.equal(defaults.lineaId, 1);
  assert.equal(defaults.preserveLegacySaveDb, false);
  const explicit = parseArgs([
    "--input=synthetic.json",
    "--linea-id=3",
    "--preserve-legacy-save-db",
  ]);
  assert.equal(explicit.lineaId, 3);
  assert.equal(explicit.preserveLegacySaveDb, true);
  assert.throws(
    () => parseArgs(["--counter-epoch", "1970-01-01T00:00:00Z"]),
    /argumento desconocido/
  );
  assert.throws(
    () => parseArgs(["--sender-interval", "10"]),
    /argumento desconocido/
  );
});

test("CLI dry-run usa fixture sintetico y no escribe", () => {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "mentor-count-v4-dry-")
  );
  try {
    const input = path.join(directory, "flows.synthetic.json");
    const serialized = `${JSON.stringify(syntheticFlows(), null, 2)}\n`;
    fs.writeFileSync(input, serialized, "utf8");
    const logs = [];
    const result = runCLI(
      ["--input", input],
      {log(value) { logs.push(String(value)); }}
    );
    assert.equal(result.applied, false);
    assert.equal(fs.readFileSync(input, "utf8"), serialized);
    assert.ok(logs.some((line) => line.includes("DRY-RUN")));
    assert.ok(logs.some((line) => line.includes("/vision/counter")));
    assert.ok(logs.some((line) => line.includes("conservado en 300 s")));
    assert.equal(
      fs.readdirSync(directory).some((name) => name.includes(".bak.")),
      false
    );
  } finally {
    fs.rmSync(directory, {recursive: true, force: true});
  }
});

test("CLI apply crea backup atomico y no reescribe segunda pasada", () => {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "mentor-count-v4-apply-")
  );
  try {
    const input = path.join(directory, "flows.synthetic.json");
    fs.writeFileSync(
      input,
      `${JSON.stringify(syntheticFlows(), null, 2)}\n`,
      "utf8"
    );
    const io = {log() {}};
    const first = runCLI(["--input", input, "--apply"], io);
    assert.equal(first.applied, true);
    assert.ok(first.backup);
    assert.equal(fs.existsSync(first.backup), true);
    const patched = JSON.parse(fs.readFileSync(input, "utf8"));
    assert.equal(
      patched.find((node) => node.id === AUDIT.saveDb).strictDelta,
      true
    );
    const backupsBefore = fs.readdirSync(directory)
      .filter((name) => name.includes(".bak.")).length;
    const second = runCLI(["--input", input, "--apply"], io);
    assert.equal(second.applied, false);
    assert.equal(second.backup, null);
    assert.equal(
      fs.readdirSync(directory)
        .filter((name) => name.includes(".bak.")).length,
      backupsBefore
    );
  } finally {
    fs.rmSync(directory, {recursive: true, force: true});
  }
});
