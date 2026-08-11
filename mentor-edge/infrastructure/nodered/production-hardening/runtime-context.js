"use strict";

const fs = require("node:fs");
const crypto = require("node:crypto");
const http = require("node:http");
const https = require("node:https");
const path = require("node:path");

const EXPECTED = Object.freeze({
    flowCount: 137,
    broker: Object.freeze({
        id: "85b6c2e7.5ca2f",
        host: "52.11.253.25",
        port: "1883"
    }),
    principalSender: Object.freeze({
        id: "b94f3dfa216658f3",
        flowId: "f8d4b04ebc5bac09",
        mentor: "41b1600ff22164b4",
        mysql: "7ba0277c.2fa1b8"
    }),
    secondarySender: Object.freeze({
        id: "48f1dea3eb373643",
        flowId: "f8d4b04ebc5bac09",
        mysql: "7ba0277c.2fa1b8"
    }),
    interval: Object.freeze({
        id: "90544e7288574184",
        seconds: 300
    }),
    productionNode: Object.freeze({
        id: "70d7cc1c095e63b5",
        flowId: "6b3340fb77a8dfe2"
    })
});

const REQUIRED_GLOBAL_COUNTERS = Object.freeze([
    "L1_t_disponible",
    "L1_t_microparada",
    "L1_t_parada_no_asignada",
    "L1_conteo_1"
]);

const REQUIRED_PRODUCTION_CONTEXT = Object.freeze([
    "prod_ultimo_tiempo_ms",
    "prod_tiempo_idle_s",
    "prod_pna_activa",
    "prod_estado_anterior"
]);

const LEGACY_GROUP_CONTEXT = Object.freeze({
    flowId: "f8d4b04ebc5bac09",
    mentorId: 478,
    devices: Object.freeze({
        ART_ATLAS_LINEA_1_ALARMAS: Object.freeze({
            fields: Object.freeze([
                "L1_ALARMA_1",
                "L1_ALARMA_2",
                "L1_ALARMA_3"
            ]),
            maxValue: 0xffff
        }),
        ART_ATLAS_MAQUINA_1_PRODUCCION: Object.freeze({
            fields: Object.freeze([
                "L1_T_DISPONIBLE",
                "L1_T_MICROPARADA",
                "L1_T_PARADA_NO_ASIGNADA",
                "L1_CONTEO_1"
            ]),
            maxValue: 0xffffffff
        })
    })
});

function fail(message) {
    throw new Error(message);
}

function assert(condition, message) {
    if (!condition) {
        fail(message);
    }
}

function plainObject(value) {
    return value !== null && typeof value === "object" && !Array.isArray(value);
}

function stableValue(value) {
    if (Array.isArray(value)) {
        return value.map(stableValue);
    }
    if (value && typeof value === "object") {
        return Object.fromEntries(
            Object.keys(value)
                .sort()
                .map((key) => [key, stableValue(value[key])])
        );
    }
    return value;
}

function canonicalFlows(flows) {
    assert(Array.isArray(flows), "flows debe ser un arreglo");
    return JSON.stringify(
        [...flows]
            .sort((left, right) => (
                String(left && left.id).localeCompare(
                    String(right && right.id)
                )
            ))
            .map(stableValue)
    );
}

function flowHash(flows) {
    return crypto
        .createHash("sha256")
        .update(canonicalFlows(flows))
        .digest("hex");
}

function parseArguments(argv) {
    const result = { action: argv[0] || "" };
    for (let index = 1; index < argv.length; index += 1) {
        const name = argv[index];
        if (!name.startsWith("--")) {
            fail(`argumento inesperado: ${name}`);
        }
        if (name === "--replace") {
            result.replace = true;
            continue;
        }
        const value = argv[index + 1];
        if (value === undefined || value.startsWith("--")) {
            fail(`falta valor para ${name}`);
        }
        result[name.slice(2)] = value;
        index += 1;
    }
    return result;
}

function requestJson(baseUrl, pathname, options = {}) {
    const target = new URL(pathname, baseUrl);
    const client = target.protocol === "https:" ? https : http;
    return new Promise((resolve, reject) => {
        const request = client.request(target, {
            method: "GET",
            headers: options.headers || {},
            timeout: options.timeoutMs || 5_000
        }, (response) => {
            const chunks = [];
            response.on("data", (chunk) => chunks.push(chunk));
            response.on("end", () => {
                const body = Buffer.concat(chunks).toString("utf8");
                if (response.statusCode < 200 || response.statusCode >= 300) {
                    reject(new Error(
                        `GET ${target.pathname} devolvió HTTP ${response.statusCode}: ` +
                        body.slice(0, 200)
                    ));
                    return;
                }
                try {
                    resolve(JSON.parse(body));
                } catch (error) {
                    reject(new Error(
                        `GET ${target.pathname} no devolvió JSON válido: ${error.message}`
                    ));
                }
            });
        });
        request.on("timeout", () => request.destroy(new Error(
            `timeout consultando ${target.pathname}`
        )));
        request.on("error", reject);
        request.end();
    });
}

async function readFlows(baseUrl, requester = requestJson) {
    const response = await requester(baseUrl, "/flows", {
        headers: { "Node-RED-API-Version": "v2" }
    });
    if (Array.isArray(response)) {
        return { rev: null, flows: response };
    }
    assert(
        plainObject(response) && Array.isArray(response.flows),
        "respuesta /flows v2 inválida"
    );
    return {
        rev: typeof response.rev === "string" ? response.rev : null,
        flows: response.flows
    };
}

function sameWire(wires, targetId) {
    return Array.isArray(wires) &&
        wires.length === 1 &&
        Array.isArray(wires[0]) &&
        wires[0].length === 1 &&
        wires[0][0] === targetId;
}

function auditSenderIsolation(flowEnvelope) {
    const flows = flowEnvelope.flows;
    assert(
        flows.length === EXPECTED.flowCount,
        `cantidad de nodos inesperada: ${flows.length}; se esperaban ${EXPECTED.flowCount}`
    );
    assert(
        new Set(flows.map((node) => node.id)).size === flows.length,
        "flows contiene ids duplicados"
    );
    const byId = Object.fromEntries(flows.map((node) => [node.id, node]));

    const broker = byId[EXPECTED.broker.id];
    assert(broker && broker.type === "mqtt-broker", "broker externo ausente");
    assert(
        broker.broker === EXPECTED.broker.host &&
        String(broker.port) === EXPECTED.broker.port &&
        broker.usetls === false,
        "configuración del broker externo cambió"
    );

    const principal = byId[EXPECTED.principalSender.id];
    assert(principal && principal.type === "Sender", "Sender principal ausente");
    assert(principal.z === EXPECTED.principalSender.flowId, "flow del Sender principal cambió");
    assert(principal.d === true, "Sender principal NO está desactivado");
    assert(principal.mqtt === EXPECTED.broker.id, "broker del Sender principal cambió");
    assert(principal.mentor === EXPECTED.principalSender.mentor, "Mentor del Sender principal cambió");
    assert(principal.mysql === EXPECTED.principalSender.mysql, "MySQL del Sender principal cambió");
    assert(String(principal.priority) === "0", "prioridad del Sender principal cambió");

    const secondary = byId[EXPECTED.secondarySender.id];
    assert(secondary && secondary.type === "Sender", "Sender secundario ausente");
    assert(secondary.z === EXPECTED.secondarySender.flowId, "flow del Sender secundario cambió");
    assert(secondary.d === true, "Sender secundario NO está desactivado");
    assert(secondary.mqtt === EXPECTED.broker.id, "broker del Sender secundario cambió");
    assert(secondary.mysql === EXPECTED.secondarySender.mysql, "MySQL del Sender secundario cambió");
    assert(String(secondary.priority) === "0", "prioridad del Sender secundario cambió");

    const senderNodes = flows.filter((node) => node.type === "Sender");
    assert(senderNodes.length === 2, `cantidad de Sender inesperada: ${senderNodes.length}`);
    assert(
        senderNodes.every((node) => node.d === true),
        "existe al menos un Sender habilitado"
    );

    const brokerUsers = flows.filter((node) => node.mqtt === EXPECTED.broker.id);
    assert(
        brokerUsers.length === 2 &&
        brokerUsers.every((node) => node.type === "Sender"),
        "existen consumidores no auditados del broker externo"
    );

    const interval = byId[EXPECTED.interval.id];
    assert(interval && interval.type === "Interval", "Interval del Sender ausente");
    assert(
        Number(interval.interval) === EXPECTED.interval.seconds &&
        Number(interval.initialdefer) === 0 &&
        sameWire(interval.wires, EXPECTED.principalSender.id),
        "Interval del Sender cambió"
    );

    const production = byId[EXPECTED.productionNode.id];
    assert(
        production &&
        production.type === "function" &&
        production.z === EXPECTED.productionNode.flowId,
        "función de producción auditada ausente"
    );

    return {
        flowCount: flows.length,
        flowHash: flowHash(flows),
        revision: flowEnvelope.rev,
        senderIds: senderNodes.map((node) => node.id).sort(),
        broker: `${EXPECTED.broker.host}:${EXPECTED.broker.port}`,
        productionNodeId: production.id
    };
}

function exactKeys(value, expected) {
    if (!plainObject(value)) {
        return false;
    }
    const actual = Object.keys(value).sort();
    const wanted = [...expected].sort();
    return actual.length === wanted.length &&
        actual.every((key, index) => key === wanted[index]);
}

function validEpochMilliseconds(value) {
    return Number.isSafeInteger(value) &&
        /^\d{13}$/.test(String(value)) &&
        value >= 946_684_800_000 &&
        value <= 4_102_444_800_000;
}

function legacyGroupDeviceFromLabel(label) {
    const prefix = `flow.${LEGACY_GROUP_CONTEXT.flowId}.`;
    if (!label.startsWith(prefix)) {
        return null;
    }
    const device = label.slice(prefix.length);
    return Object.prototype.hasOwnProperty.call(
        LEGACY_GROUP_CONTEXT.devices,
        device
    ) ? device : null;
}

function decodeLegacyGroupContext(encoded, label, device) {
    assert(
        typeof encoded.msg === "string" &&
        Buffer.byteLength(encoded.msg, "utf8") <= 4_096,
        `${label}: contexto agrupado inválido o truncado`
    );
    let value;
    try {
        value = JSON.parse(encoded.msg);
    } catch (error) {
        fail(`${label}: contexto agrupado no es JSON válido`);
    }

    assert(
        exactKeys(value, ["n", "time", "data", "timeout"]),
        `${label}: estructura agrupada inesperada`
    );
    assert(
        value.n === 0 && value.timeout === 0,
        `${label}: lectura Modbus no está en estado estable`
    );
    assert(
        validEpochMilliseconds(value.time),
        `${label}: timestamp agrupado inválido`
    );
    assert(
        Array.isArray(value.data) && value.data.length === 1,
        `${label}: lote agrupado inválido`
    );

    const record = value.data[0];
    const keysWithName = ["time", "code", "name", "value", "mentor_id"];
    const keysWithoutName = ["time", "code", "value", "mentor_id"];
    assert(
        exactKeys(record, keysWithName) || exactKeys(record, keysWithoutName),
        `${label}: registro agrupado inesperado`
    );
    assert(
        record.time === value.time &&
        record.code === device &&
        record.mentor_id === LEGACY_GROUP_CONTEXT.mentorId,
        `${label}: identidad del registro agrupado cambió`
    );
    if (Object.prototype.hasOwnProperty.call(record, "name")) {
        assert(
            exactKeys(record.name, ["__enc__", "type"]) &&
            record.name.__enc__ === true &&
            record.name.type === "undefined",
            `${label}: marcador undefined inválido`
        );
    }

    const schema = LEGACY_GROUP_CONTEXT.devices[device];
    assert(
        Array.isArray(record.value) &&
        record.value.length === schema.fields.length,
        `${label}: cantidad de variables agrupadas cambió`
    );
    const normalizedValues = record.value.map((entry, index) => {
        const field = schema.fields[index];
        assert(
            exactKeys(entry, [field]) &&
            Number.isSafeInteger(entry[field]) &&
            entry[field] >= 0 &&
            entry[field] <= schema.maxValue,
            `${label}: variable ${field} inválida`
        );
        return { [field]: entry[field] };
    });

    // The context API represents nested undefined values with a display-only
    // sentinel. localfilesystem persists via JSON.stringify, where the
    // original undefined property is omitted. Never persist that sentinel as
    // a truthy application object.
    return {
        present: true,
        value: {
            n: 0,
            time: value.time,
            data: [{
                time: record.time,
                code: record.code,
                value: normalizedValues,
                mentor_id: record.mentor_id
            }],
            timeout: 0
        }
    };
}

function decodeContextValue(encoded, label) {
    assert(plainObject(encoded), `${label}: valor codificado inválido`);
    const format = encoded.format;
    if (format === "number") {
        const value = Number(encoded.msg);
        assert(Number.isFinite(value), `${label}: número no finito`);
        return { present: true, value };
    }
    if (format === "boolean") {
        assert(
            encoded.msg === "true" || encoded.msg === "false",
            `${label}: booleano inválido`
        );
        return { present: true, value: encoded.msg === "true" };
    }
    if (typeof format === "string" && /^string\[\d+\]$/.test(format)) {
        assert(typeof encoded.msg === "string", `${label}: string inválido`);
        const expectedLength = Number(format.slice(7, -1));
        assert(
            encoded.msg.length === expectedLength,
            `${label}: string truncado por la API de contexto`
        );
        return { present: true, value: encoded.msg };
    }
    if (format === "null") {
        return { present: true, value: null };
    }
    if (format === "undefined") {
        return { present: false };
    }
    if (
        format === "Object" &&
        /\.mentor_textile_count_v4_samples$/.test(label)
    ) {
        assert(
            typeof encoded.msg === "string" && encoded.msg.length <= 1_000,
            `${label}: lock de muestras inválido o truncado`
        );
        let value;
        try {
            value = JSON.parse(encoded.msg);
        } catch (error) {
            fail(`${label}: lock de muestras no es JSON válido`);
        }
        assert(plainObject(value), `${label}: lock de muestras no es objeto`);
        assert(
            Object.keys(value).length <= 4,
            `${label}: demasiados locks de muestras`
        );
        for (const [sampleId, entry] of Object.entries(value)) {
            assert(
                /^\d+:\d{13}$/.test(sampleId) &&
                plainObject(entry) &&
                Object.keys(entry).length === 1 &&
                Number.isSafeInteger(entry.expiresAt) &&
                entry.expiresAt > 0,
                `${label}: entrada de lock inválida`
            );
        }
        return { present: true, value };
    }
    if (format === "Object") {
        const device = legacyGroupDeviceFromLabel(label);
        if (device) {
            return decodeLegacyGroupContext(encoded, label, device);
        }
    }
    fail(
        `${label}: formato ${JSON.stringify(format)} no se migra automáticamente; ` +
        "se aborta para evitar truncar/corromper contexto complejo"
    );
}

function decodeContextResponse(response, label) {
    assert(plainObject(response), `${label}: respuesta de contexto inválida`);
    const stores = Object.entries(response).filter(([, value]) => (
        plainObject(value) && Object.keys(value).length > 0
    ));
    if (stores.length === 0) {
        return {};
    }
    assert(stores.length === 1, `${label}: más de un almacén de contexto activo`);
    const [storeName, entries] = stores[0];
    assert(
        storeName === "memory" || storeName === "default",
        `${label}: almacén inesperado ${storeName}`
    );

    const decoded = {};
    for (const [key, encoded] of Object.entries(entries)) {
        const result = decodeContextValue(encoded, `${label}.${key}`);
        if (result.present) {
            decoded[key] = result.value;
        }
    }
    return decoded;
}

function safeId(value, label) {
    assert(
        typeof value === "string" && /^[A-Za-z0-9_.-]+$/.test(value),
        `${label} contiene un id inseguro`
    );
    return value;
}

async function captureContext(flowEnvelope, baseUrl, requester = requestJson) {
    const globalResponse = await requester(baseUrl, "/context/global");
    const snapshot = {
        global: decodeContextResponse(globalResponse, "global"),
        flows: {},
        nodes: {}
    };

    const flowNodes = flowEnvelope.flows.filter((node) => (
        node.type === "tab" || node.type === "subflow"
    ));
    for (const flow of flowNodes) {
        const id = safeId(flow.id, "flow");
        const response = await requester(baseUrl, `/context/flow/${encodeURIComponent(id)}`);
        const values = decodeContextResponse(response, `flow.${id}`);
        if (Object.keys(values).length > 0) {
            snapshot.flows[id] = values;
        }
    }

    for (const node of flowEnvelope.flows) {
        if (!node.id || node.type === "tab" || node.type === "subflow") {
            continue;
        }
        const id = safeId(node.id, "node");
        const response = await requester(baseUrl, `/context/node/${encodeURIComponent(id)}`);
        const values = decodeContextResponse(response, `node.${id}`);
        if (Object.keys(values).length === 0) {
            continue;
        }
        assert(
            typeof node.z === "string" && node.z.length > 0,
            `node.${id}: tiene contexto pero no pertenece a un flow`
        );
        snapshot.nodes[id] = {
            flowId: safeId(node.z, `node.${id}.flowId`),
            values
        };
    }
    validateRequiredContext(snapshot);
    return snapshot;
}

function nonNegativeSafeInteger(value) {
    return Number.isSafeInteger(value) && value >= 0;
}

function validateRequiredContext(snapshot) {
    assert(plainObject(snapshot.global), "contexto global ausente");
    for (const key of REQUIRED_GLOBAL_COUNTERS) {
        assert(
            nonNegativeSafeInteger(snapshot.global[key]),
            `contexto global ${key} debe ser entero no negativo`
        );
    }

    const production = snapshot.nodes[EXPECTED.productionNode.id];
    assert(production, "contexto local de Producción Art Atlas ausente");
    assert(
        production.flowId === EXPECTED.productionNode.flowId,
        "scope del contexto de producción cambió"
    );
    for (const key of REQUIRED_PRODUCTION_CONTEXT) {
        assert(
            Object.prototype.hasOwnProperty.call(production.values, key),
            `contexto de producción ${key} ausente`
        );
    }
    assert(
        nonNegativeSafeInteger(production.values.prod_ultimo_tiempo_ms),
        "prod_ultimo_tiempo_ms inválido"
    );
    assert(
        Number.isFinite(production.values.prod_tiempo_idle_s) &&
        production.values.prod_tiempo_idle_s >= 0,
        "prod_tiempo_idle_s inválido"
    );
    assert(
        typeof production.values.prod_pna_activa === "boolean",
        "prod_pna_activa inválido"
    );
    assert(
        typeof production.values.prod_estado_anterior === "string" &&
        production.values.prod_estado_anterior.length > 0,
        "prod_estado_anterior inválido"
    );
}

function atomicWriteJson(target, value) {
    fs.mkdirSync(path.dirname(target), { recursive: true, mode: 0o700 });
    const temporary = `${target}.${process.pid}.${Date.now()}.tmp`;
    fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, {
        encoding: "utf8",
        mode: 0o600,
        flag: "wx"
    });
    fs.renameSync(temporary, target);
}

function seedContext(snapshot, seedBase, options = {}) {
    const resolved = path.resolve(seedBase);
    assert(
        resolved === "/data/mentor-context" ||
        options.allowArbitraryBase === true,
        `directorio de contexto no autorizado: ${resolved}`
    );
    if (fs.existsSync(resolved) && !options.replace) {
        fail(`${resolved} ya existe; se requiere --replace explícito`);
    }
    fs.mkdirSync(resolved, { recursive: true, mode: 0o700 });
    atomicWriteJson(path.join(resolved, "global", "global.json"), snapshot.global);

    for (const [flowId, values] of Object.entries(snapshot.flows)) {
        safeId(flowId, "flowId");
        atomicWriteJson(path.join(resolved, flowId, "flow.json"), values);
    }
    for (const [nodeId, entry] of Object.entries(snapshot.nodes)) {
        safeId(nodeId, "nodeId");
        safeId(entry.flowId, "node.flowId");
        atomicWriteJson(
            path.join(resolved, entry.flowId, `${nodeId}.json`),
            entry.values
        );
    }
}

function valueType(value) {
    if (value === null) {
        return "null";
    }
    return typeof value;
}

function validateContinuity(baseline, current) {
    validateRequiredContext(baseline.context);
    validateRequiredContext(current.context);

    const errors = [];
    for (const key of REQUIRED_GLOBAL_COUNTERS) {
        if (current.context.global[key] < baseline.context.global[key]) {
            errors.push(
                `${key} retrocedió: ${baseline.context.global[key]} -> ` +
                `${current.context.global[key]}`
            );
        }
    }

    for (const [scopeId, values] of Object.entries(baseline.context.flows)) {
        const currentValues = current.context.flows[scopeId];
        if (!currentValues) {
            errors.push(`desapareció el contexto flow.${scopeId}`);
            continue;
        }
        for (const [key, oldValue] of Object.entries(values)) {
            if (!Object.prototype.hasOwnProperty.call(currentValues, key)) {
                errors.push(`desapareció flow.${scopeId}.${key}`);
            } else if (valueType(currentValues[key]) !== valueType(oldValue)) {
                errors.push(`cambió el tipo de flow.${scopeId}.${key}`);
            }
        }
    }

    for (const [nodeId, entry] of Object.entries(baseline.context.nodes)) {
        const currentEntry = current.context.nodes[nodeId];
        if (!currentEntry) {
            errors.push(`desapareció el contexto node.${nodeId}`);
            continue;
        }
        for (const [key, oldValue] of Object.entries(entry.values)) {
            if (!Object.prototype.hasOwnProperty.call(currentEntry.values, key)) {
                errors.push(`desapareció node.${nodeId}.${key}`);
            } else if (
                valueType(currentEntry.values[key]) !== valueType(oldValue)
            ) {
                errors.push(`cambió el tipo de node.${nodeId}.${key}`);
            }
        }
    }

    const oldProduction = baseline.context.nodes[EXPECTED.productionNode.id].values;
    const newProduction = current.context.nodes[EXPECTED.productionNode.id].values;
    if (newProduction.prod_ultimo_tiempo_ms < oldProduction.prod_ultimo_tiempo_ms) {
        errors.push("prod_ultimo_tiempo_ms retrocedió");
    }

    assert(errors.length === 0, `falló continuidad:\n- ${errors.join("\n- ")}`);
    return {
        countersBefore: Object.fromEntries(
            REQUIRED_GLOBAL_COUNTERS.map((key) => [key, baseline.context.global[key]])
        ),
        countersAfter: Object.fromEntries(
            REQUIRED_GLOBAL_COUNTERS.map((key) => [key, current.context.global[key]])
        ),
        scopesBefore: (
            Object.keys(baseline.context.flows).length +
            Object.keys(baseline.context.nodes).length +
            1
        ),
        scopesAfter: (
            Object.keys(current.context.flows).length +
            Object.keys(current.context.nodes).length +
            1
        )
    };
}

async function buildSnapshot(baseUrl, requester = requestJson) {
    // Tras un reinicio, Node-RED reanuda el sondeo Modbus y el contexto
    // agrupado queda en n!=0 / timeout!=0 hasta que el grupo se vacía en su
    // siguiente ciclo, lo que con la línea parada puede tardar varios minutos.
    // Capturar en esa ventana haría fallar decodeLegacyGroupContext con
    // "no está en estado estable" pese a que el estado sí converge. Se reintenta
    // solo ante ese transitorio; cualquier otro error se propaga de inmediato.
    let lastError;
    for (let attempt = 0; attempt < 300; attempt += 1) {
        try {
            const flowEnvelope = await readFlows(baseUrl, requester);
            const isolation = auditSenderIsolation(flowEnvelope);
            const context = await captureContext(flowEnvelope, baseUrl, requester);
            return {
                schemaVersion: 1,
                capturedAt: new Date().toISOString(),
                isolation,
                context
            };
        } catch (error) {
            if (!/en estado estable/.test(error.message)) {
                throw error;
            }
            lastError = error;
            await new Promise((resolve) => setTimeout(resolve, 2000));
        }
    }
    throw lastError;
}

function readSnapshot(filename) {
    const value = JSON.parse(fs.readFileSync(filename, "utf8"));
    assert(value && value.schemaVersion === 1, `snapshot inválido: ${filename}`);
    return value;
}

async function runCli(argv) {
    const args = parseArguments(argv);
    const baseUrl = args.url || "http://127.0.0.1:1880";
    if (args.action === "audit") {
        const flows = await readFlows(baseUrl);
        process.stdout.write(`${JSON.stringify(auditSenderIsolation(flows), null, 2)}\n`);
        return;
    }
    if (args.action === "flow-hash") {
        const flowEnvelope = await readFlows(baseUrl);
        const runtimeHash = flowHash(flowEnvelope.flows);
        if (args.disk) {
            const diskFlows = JSON.parse(fs.readFileSync(args.disk, "utf8"));
            const diskHash = flowHash(diskFlows);
            assert(
                diskHash === runtimeHash,
                `runtime y disco difieren: runtime=${runtimeHash} disk=${diskHash}`
            );
        }
        process.stdout.write(`${runtimeHash}\n`);
        return;
    }
    if (args.action === "capture") {
        assert(args.output, "capture requiere --output");
        const snapshot = await buildSnapshot(baseUrl);
        atomicWriteJson(args.output, snapshot);
        if (args["seed-base"]) {
            seedContext(snapshot.context, args["seed-base"], {
                replace: args.replace === true
            });
        }
        process.stdout.write(`${JSON.stringify({
            capturedAt: snapshot.capturedAt,
            output: args.output,
            seedBase: args["seed-base"] || null,
            isolation: snapshot.isolation,
            counters: Object.fromEntries(
                REQUIRED_GLOBAL_COUNTERS.map((key) => [
                    key,
                    snapshot.context.global[key]
                ])
            ),
            flowContextScopes: Object.keys(snapshot.context.flows).length,
            nodeContextScopes: Object.keys(snapshot.context.nodes).length
        }, null, 2)}\n`);
        return;
    }
    if (args.action === "validate") {
        assert(args.baseline, "validate requiere --baseline");
        const baseline = readSnapshot(args.baseline);
        const current = await buildSnapshot(baseUrl);
        const continuity = validateContinuity(baseline, current);
        if (args.output) {
            atomicWriteJson(args.output, current);
        }
        process.stdout.write(`${JSON.stringify({
            validatedAt: current.capturedAt,
            isolation: current.isolation,
            continuity
        }, null, 2)}\n`);
        return;
    }
    fail("uso: runtime-context.js audit|flow-hash|capture|validate [opciones]");
}

if (require.main === module) {
    runCli(process.argv.slice(2)).catch((error) => {
        process.stderr.write(`ERROR: ${error.stack || error.message}\n`);
        process.exitCode = 1;
    });
}

module.exports = {
    EXPECTED,
    LEGACY_GROUP_CONTEXT,
    REQUIRED_GLOBAL_COUNTERS,
    REQUIRED_PRODUCTION_CONTEXT,
    auditSenderIsolation,
    canonicalFlows,
    captureContext,
    decodeContextResponse,
    decodeContextValue,
    flowHash,
    parseArguments,
    seedContext,
    validateContinuity,
    validateRequiredContext
};
