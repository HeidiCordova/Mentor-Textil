"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const runtime = require("./runtime-context.js");

function baseFlows() {
    const expected = runtime.EXPECTED;
    const filler = [];
    for (let index = 0; index < expected.flowCount - 7; index += 1) {
        filler.push({
            id: `filler-${index}`,
            type: "debug",
            z: "flow-filler"
        });
    }
    return [
        {
            id: expected.broker.id,
            type: "mqtt-broker",
            broker: expected.broker.host,
            port: expected.broker.port,
            usetls: false
        },
        {
            id: expected.principalSender.id,
            type: "Sender",
            z: expected.principalSender.flowId,
            d: true,
            mentor: expected.principalSender.mentor,
            mysql: expected.principalSender.mysql,
            mqtt: expected.broker.id,
            priority: "0"
        },
        {
            id: expected.secondarySender.id,
            type: "Sender",
            z: expected.secondarySender.flowId,
            d: true,
            mentor: "",
            mysql: expected.secondarySender.mysql,
            mqtt: expected.broker.id,
            priority: "0"
        },
        {
            id: expected.interval.id,
            type: "Interval",
            z: expected.principalSender.flowId,
            interval: "300",
            initialdefer: 0,
            wires: [[expected.principalSender.id]]
        },
        {
            id: expected.productionNode.id,
            type: "function",
            z: expected.productionNode.flowId
        },
        { id: expected.principalSender.flowId, type: "tab" },
        { id: expected.productionNode.flowId, type: "tab" },
        ...filler
    ];
}

function requiredContext(values = {}) {
    return {
        global: Object.assign({
            L1_t_disponible: 100,
            L1_t_microparada: 20,
            L1_t_parada_no_asignada: 30,
            L1_conteo_1: 1
        }, values.global),
        flows: values.flows || {
            "flow-filler": { gpio4_estado: true }
        },
        nodes: values.nodes || {
            [runtime.EXPECTED.productionNode.id]: {
                flowId: runtime.EXPECTED.productionNode.flowId,
                values: {
                    prod_ultimo_tiempo_ms: 1_785_000_000_000,
                    prod_tiempo_idle_s: 12,
                    prod_pna_activa: false,
                    prod_estado_anterior: "beige_in"
                }
            }
        }
    };
}

function legacyGroupValue(device) {
    const time = 1_785_457_500_000;
    const samples = {
        ART_ATLAS_LINEA_1_ALARMAS: [1, 0, 1],
        ART_ATLAS_MAQUINA_1_PRODUCCION: [100, 20, 30, 1]
    };
    const schema = runtime.LEGACY_GROUP_CONTEXT.devices[device];
    return {
        n: 0,
        time,
        data: [{
            time,
            code: device,
            name: { __enc__: true, type: "undefined" },
            value: schema.fields.map((field, index) => ({
                [field]: samples[device][index]
            })),
            mentor_id: runtime.LEGACY_GROUP_CONTEXT.mentorId
        }],
        timeout: 0
    };
}

function encodedLegacyGroup(device, value = legacyGroupValue(device)) {
    return {
        format: "Object",
        msg: JSON.stringify(value)
    };
}

function legacyGroupLabel(device) {
    return `flow.${runtime.LEGACY_GROUP_CONTEXT.flowId}.${device}`;
}

test("audita exactamente los dos Sender desactivados y su Interval", () => {
    const result = runtime.auditSenderIsolation({
        rev: "rev-1",
        flows: baseFlows()
    });
    assert.equal(result.flowCount, runtime.EXPECTED.flowCount);
    assert.equal(result.flowHash, runtime.flowHash(baseFlows()));
    assert.deepEqual(result.senderIds, [
        runtime.EXPECTED.secondarySender.id,
        runtime.EXPECTED.principalSender.id
    ].sort());
});

test("hash canónico no depende del orden de nodos ni propiedades", () => {
    const flows = baseFlows();
    const reordered = [...flows]
        .reverse()
        .map((node) => Object.fromEntries(Object.entries(node).reverse()));
    assert.equal(runtime.flowHash(flows), runtime.flowHash(reordered));
    reordered[0].x = 123;
    assert.notEqual(runtime.flowHash(flows), runtime.flowHash(reordered));
});

test("aborta si el Sender principal queda habilitado", () => {
    const flows = baseFlows();
    flows.find((node) => node.id === runtime.EXPECTED.principalSender.id).d = false;
    assert.throws(
        () => runtime.auditSenderIsolation({ rev: "rev", flows }),
        /NO está desactivado/
    );
});

test("aborta si aparece otro consumidor del broker externo", () => {
    const flows = baseFlows();
    flows[flows.length - 1].mqtt = runtime.EXPECTED.broker.id;
    assert.throws(
        () => runtime.auditSenderIsolation({ rev: "rev", flows }),
        /consumidores no auditados/
    );
});

test("aborta si cambia TLS o aparecen ids duplicados", () => {
    const tlsFlows = baseFlows();
    tlsFlows.find((node) => node.id === runtime.EXPECTED.broker.id).usetls = "false";
    assert.throws(
        () => runtime.auditSenderIsolation({ rev: "rev", flows: tlsFlows }),
        /broker externo cambió/
    );

    const duplicateFlows = baseFlows();
    duplicateFlows[duplicateFlows.length - 1].id = duplicateFlows[0].id;
    assert.throws(
        () => runtime.auditSenderIsolation({ rev: "rev", flows: duplicateFlows }),
        /ids duplicados/
    );
});

test("decodifica solo primitivas completas", () => {
    assert.deepEqual(
        runtime.decodeContextValue({ format: "number", msg: "42" }, "n"),
        { present: true, value: 42 }
    );
    assert.deepEqual(
        runtime.decodeContextValue({ format: "boolean", msg: "false" }, "b"),
        { present: true, value: false }
    );
    assert.deepEqual(
        runtime.decodeContextValue({ format: "string[4]", msg: "idle" }, "s"),
        { present: true, value: "idle" }
    );
    assert.throws(
        () => runtime.decodeContextValue(
            { format: "string[10]", msg: "trunc..." },
            "truncated"
        ),
        /truncado/
    );
    assert.throws(
        () => runtime.decodeContextValue(
            { format: "Object", msg: "{\"a\":1}" },
            "complex"
        ),
        /no se migra automáticamente/
    );
    assert.deepEqual(
        runtime.decodeContextValue({
            format: "Object",
            msg: "{\"1:1785457500000\":{\"expiresAt\":1785457740000}}"
        }, "flow.id.mentor_textile_count_v4_samples"),
        {
            present: true,
            value: {
                "1:1785457500000": { expiresAt: 1785457740000 }
            }
        }
    );
});

test("decodifica estrictamente los dos grupos Modbus heredados", () => {
    for (const device of Object.keys(runtime.LEGACY_GROUP_CONTEXT.devices)) {
        const result = runtime.decodeContextValue(
            encodedLegacyGroup(device),
            legacyGroupLabel(device)
        );
        assert.equal(result.present, true);
        assert.equal(result.value.n, 0);
        assert.equal(result.value.timeout, 0);
        assert.equal(result.value.data[0].code, device);
        assert.equal(
            Object.prototype.hasOwnProperty.call(result.value.data[0], "name"),
            false
        );
        assert.equal(JSON.stringify(result.value).includes("__enc__"), false);

        const afterLocalfilesystemReload = structuredClone(result.value);
        assert.deepEqual(
            runtime.decodeContextValue(
                encodedLegacyGroup(device, afterLocalfilesystemReload),
                legacyGroupLabel(device)
            ),
            result
        );
    }
});

test("rechaza objetos Modbus activos, alterados o fuera de rango", () => {
    const device = "ART_ATLAS_MAQUINA_1_PRODUCCION";
    const label = legacyGroupLabel(device);

    const active = legacyGroupValue(device);
    active.timeout = 1;
    assert.throws(
        () => runtime.decodeContextValue(encodedLegacyGroup(device, active), label),
        /no está en estado estable/
    );

    const badSentinel = legacyGroupValue(device);
    badSentinel.data[0].name = { __enc__: true, type: "Buffer" };
    assert.throws(
        () => runtime.decodeContextValue(
            encodedLegacyGroup(device, badSentinel),
            label
        ),
        /marcador undefined inválido/
    );

    const extra = legacyGroupValue(device);
    extra.data[0].unexpected = true;
    assert.throws(
        () => runtime.decodeContextValue(encodedLegacyGroup(device, extra), label),
        /registro agrupado inesperado/
    );

    const wrongField = legacyGroupValue(device);
    wrongField.data[0].value[3] = { L1_CONTEO_2: 1 };
    assert.throws(
        () => runtime.decodeContextValue(
            encodedLegacyGroup(device, wrongField),
            label
        ),
        /L1_CONTEO_1 inválida/
    );

    const outOfRange = legacyGroupValue(device);
    outOfRange.data[0].value[0].L1_T_DISPONIBLE = 0x1_0000_0000;
    assert.throws(
        () => runtime.decodeContextValue(
            encodedLegacyGroup(device, outOfRange),
            label
        ),
        /L1_T_DISPONIBLE inválida/
    );
});

test("rechaza más de un context store y stores con otro nombre", () => {
    assert.throws(
        () => runtime.decodeContextResponse({
            default: { a: { format: "number", msg: "1" } },
            file: { a: { format: "number", msg: "1" } }
        }, "scope"),
        /más de un almacén/
    );
    assert.throws(
        () => runtime.decodeContextResponse({
            memoryOnly: { a: { format: "number", msg: "1" } }
        }, "scope"),
        /almacén inesperado/
    );
    assert.deepEqual(
        runtime.decodeContextResponse({
            memory: { a: { format: "number", msg: "1" } }
        }, "scope"),
        { a: 1 }
    );
});

test("captura global, flow y node desde el store memory actual", async () => {
    const flows = baseFlows();
    const expected = runtime.EXPECTED;
    const requested = [];
    const requester = async (baseUrl, pathname) => {
        assert.equal(baseUrl, "http://node-red.invalid");
        requested.push(pathname);
        if (pathname === "/context/global") {
            return {
                memory: {
                    L1_t_disponible: { format: "number", msg: "100" },
                    L1_t_microparada: { format: "number", msg: "20" },
                    L1_t_parada_no_asignada: { format: "number", msg: "30" },
                    L1_conteo_1: { format: "number", msg: "1" }
                }
            };
        }
        if (pathname === `/context/flow/${expected.productionNode.flowId}`) {
            return {
                memory: {
                    mentor_textile_count_v4_samples: {
                        format: "Object",
                        msg: "{\"1:1785457500000\":{\"expiresAt\":1785457740000}}"
                    }
                }
            };
        }
        if (pathname === `/context/node/${expected.productionNode.id}`) {
            return {
                memory: {
                    prod_ultimo_tiempo_ms: {
                        format: "number",
                        msg: "1785457500000"
                    },
                    prod_tiempo_idle_s: { format: "number", msg: "12" },
                    prod_pna_activa: { format: "boolean", msg: "false" },
                    prod_estado_anterior: {
                        format: "string[8]",
                        msg: "beige_in"
                    }
                }
            };
        }
        return {};
    };
    const snapshot = await runtime.captureContext(
        { rev: "rev", flows },
        "http://node-red.invalid",
        requester
    );
    assert.equal(snapshot.global.L1_conteo_1, 1);
    assert.equal(
        snapshot.nodes[expected.productionNode.id].values.prod_estado_anterior,
        "beige_in"
    );
    assert.ok(requested.includes("/context/global"));
    assert.ok(requested.includes(
        `/context/node/${expected.productionNode.id}`
    ));
});

test("siembra rutas compatibles con localfilesystem", () => {
    const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "mentor-context-test-"));
    const seedBase = path.join(temporary, "mentor-context");
    try {
        const context = requiredContext();
        runtime.seedContext(context, seedBase, { allowArbitraryBase: true });
        assert.deepEqual(
            JSON.parse(fs.readFileSync(
                path.join(seedBase, "global", "global.json"),
                "utf8"
            )),
            context.global
        );
        assert.deepEqual(
            JSON.parse(fs.readFileSync(
                path.join(
                    seedBase,
                    runtime.EXPECTED.productionNode.flowId,
                    `${runtime.EXPECTED.productionNode.id}.json`
                ),
                "utf8"
            )),
            context.nodes[runtime.EXPECTED.productionNode.id].values
        );
        assert.throws(
            () => runtime.seedContext(context, seedBase, {
                allowArbitraryBase: true
            }),
            /--replace/
        );
    } finally {
        fs.rmSync(temporary, { recursive: true, force: true });
    }
});

test("la semilla localfilesystem nunca persiste el sentinel undefined", () => {
    const temporary = fs.mkdtempSync(path.join(
        os.tmpdir(),
        "mentor-context-legacy-group-"
    ));
    const seedBase = path.join(temporary, "mentor-context");
    const flowId = runtime.LEGACY_GROUP_CONTEXT.flowId;
    const device = "ART_ATLAS_MAQUINA_1_PRODUCCION";
    try {
        const decoded = runtime.decodeContextValue(
            encodedLegacyGroup(device),
            legacyGroupLabel(device)
        ).value;
        const context = requiredContext({
            flows: {
                [flowId]: {
                    [device]: decoded
                }
            }
        });
        runtime.seedContext(context, seedBase, { allowArbitraryBase: true });
        const raw = fs.readFileSync(
            path.join(seedBase, flowId, "flow.json"),
            "utf8"
        );
        assert.equal(raw.includes("__enc__"), false);
        assert.equal(raw.includes("\"name\""), false);
        assert.deepEqual(JSON.parse(raw)[device], decoded);
    } finally {
        fs.rmSync(temporary, { recursive: true, force: true });
    }
});

test("continuidad permite crecimiento pero nunca retroceso", () => {
    const baseline = { context: requiredContext() };
    const currentContext = requiredContext({
        global: {
            L1_t_disponible: 105,
            L1_t_microparada: 20,
            L1_t_parada_no_asignada: 31,
            L1_conteo_1: 2
        }
    });
    currentContext.nodes[runtime.EXPECTED.productionNode.id].
        values.prod_ultimo_tiempo_ms += 5_000;
    const result = runtime.validateContinuity(
        baseline,
        { context: currentContext }
    );
    assert.equal(result.countersAfter.L1_conteo_1, 2);

    const regressed = requiredContext({
        global: { L1_conteo_1: 0 }
    });
    assert.throws(
        () => runtime.validateContinuity(
            baseline,
            { context: regressed }
        ),
        /L1_conteo_1 retrocedió/
    );
});

test("contexto obligatorio no admite valores ausentes o negativos", () => {
    const missing = requiredContext();
    delete missing.global.L1_t_disponible;
    assert.throws(
        () => runtime.validateRequiredContext(missing),
        /L1_t_disponible/
    );

    const negative = requiredContext({
        global: { L1_t_microparada: -1 }
    });
    assert.throws(
        () => runtime.validateRequiredContext(negative),
        /L1_t_microparada/
    );
});
