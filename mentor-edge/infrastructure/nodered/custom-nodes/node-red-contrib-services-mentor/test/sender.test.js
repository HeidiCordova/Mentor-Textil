"use strict";

const assert = require("node:assert/strict");
const { EventEmitter } = require("node:events");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const senderModule = require("../services/sender.js");

function createHarness(options = {}) {
    const registeredTypes = {};
    const queryCalls = [];
    const published = [];
    const sent = [];
    const warnings = [];
    const errors = [];
    const statuses = [];
    let subscription = null;
    let nextNodeId = 1;

    const database = new EventEmitter();
    database.connected = options.databaseConnected !== false;
    database.connect = function connect() {};
    database.connection = {
        query(sql, params, callback) {
            const call = { sql, params: Array.from(params) };
            queryCalls.push(call);
            if (typeof options.onQuery === "function") {
                options.onQuery(call, callback, queryCalls.length);
                return;
            }
            callback(null, []);
        }
    };

    const broker = {
        connected: options.brokerConnected !== false,
        register() {},
        subscribe(topic, qos, callback, id) {
            subscription = { topic, qos, callback, id };
        },
        publish(message) {
            published.push(Object.assign({}, message));
            if (typeof options.onPublish === "function") {
                options.onPublish(message, published.length);
            }
        },
        deregister(node, done) {
            done();
        }
    };

    const mentor = options.mentorPresent === false
        ? null
        : { mentor_id: options.mentorId === undefined ? 42 : options.mentorId };
    const nodesById = { db: database, broker };
    if (mentor) {
        nodesById.mentor = mentor;
    }
    const RED = {
        log: {
            error(message) {
                errors.push(String(message));
            }
        },
        nodes: {
            createNode(node) {
                const emitter = new EventEmitter();
                node.id = `sender-${nextNodeId++}`;
                node.on = emitter.on.bind(emitter);
                node.once = emitter.once.bind(emitter);
                node.emit = emitter.emit.bind(emitter);
                node.removeListener = emitter.removeListener.bind(emitter);
                node.status = (status) => statuses.push(status);
                node.send = (message) => sent.push(message);
                node.warn = (message) => warnings.push(String(message));
                node.error = (message) => errors.push(String(message));
            },
            getNode(id) {
                return nodesById[id] || null;
            },
            registerType(name, constructor) {
                registeredTypes[name] = constructor;
            }
        }
    };

    const register = senderModule.createSenderModule({
        request: options.request,
        utils: options.utils || { encode: (value) => `encoded:${value}` },
        retryDelayMs: options.retryDelayMs,
        limits: options.limits
    });
    register(RED);

    const config = Object.assign({
        mysql: "db",
        mqtt: "broker",
        mentor: "mentor",
        priority: "0",
        url: "https://rest.invalid/data",
        url_alarm: "https://rest.invalid/alarm"
    }, options.config);
    const node = new registeredTypes.Sender(config);

    return {
        broker,
        database,
        errors,
        node,
        published,
        queryCalls,
        sent,
        statuses,
        warnings,
        deliverAck(value) {
            assert.ok(subscription, "Sender must subscribe to its ACK channel");
            const payload = Buffer.isBuffer(value)
                ? value
                : Buffer.from(typeof value === "string" ? value : JSON.stringify(value));
            subscription.callback(subscription.topic, payload, {});
        },
        get subscription() {
            return subscription;
        },
        async close() {
            await new Promise((resolve) => node.emit("close", resolve));
        }
    };
}

function isDevicesQuery(sql) {
    return sql.includes("SELECT DISTINCT device");
}

function isRowsQuery(sql) {
    return sql.includes("SELECT device, time, content, alarm");
}

function isUpdate(sql) {
    return sql.startsWith("UPDATE mqtt_lecturas");
}

async function waitFor(predicate, timeoutMs = 500) {
    const deadline = Date.now() + timeoutMs;
    while (!predicate()) {
        if (Date.now() >= deadline) {
            assert.fail("Timed out waiting for condition");
        }
        await new Promise((resolve) => setTimeout(resolve, 5));
    }
}

test("safe default sends one pending record per device with QoS 1 and waits for ACK", async () => {
    const rows = {
        "LINE'1": [
            { device: "LINE'1", time: 1000, content: "first", alarm: 0 },
            { device: "LINE'1", time: 1001, content: "second", alarm: 0 }
        ],
        "LINE-2": [
            { device: "LINE-2", time: "2000", content: "other", alarm: 1 },
            { device: "LINE-2", time: "2001", content: "other-2", alarm: 1 }
        ]
    };
    const updates = [];

    const harness = createHarness({
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                assert.deepEqual(call.params, [42]);
                callback(null, Object.keys(rows).map((device) => ({ device })));
                return;
            }
            if (isRowsQuery(call.sql)) {
                const [mentorId, device, batchSize] = call.params;
                assert.equal(mentorId, 42);
                assert.equal(batchSize, 1);
                callback(null, rows[device]);
                return;
            }
            if (isUpdate(call.sql)) {
                updates.push(call);
                callback(null, { affectedRows: 1 });
                return;
            }
            callback(new Error("unexpected query"));
        }
    });

    harness.node.emit("input", { payload: "tick" });

    assert.equal(harness.published.length, 2);
    for (const publication of harness.published) {
        assert.equal(publication.topic, "mentor_data");
        assert.equal(publication.qos, 1);
        assert.equal(publication.retain, false);
    }
    assert.equal(JSON.parse(harness.published[0].payload).mentor_id, 42);
    assert.equal(updates.length, 0, "Publishing must not mark the row as sent");
    assert.equal(harness.node.inflight.size, 2);

    harness.deliverAck({ device: "LINE'1", time: 1000, status: 2 });
    assert.equal(updates.length, 1);
    assert.match(updates[0].sql, /mentor_id = \?/);
    assert.match(updates[0].sql, /device = \?/);
    assert.match(updates[0].sql, /time = \?/);
    assert.doesNotMatch(updates[0].sql, /CAST\(time AS UNSIGNED\).*=/);
    assert.doesNotMatch(updates[0].sql, /LINE'1/);
    assert.deepEqual(updates[0].params, [2, 42, "LINE'1", "1000"]);

    harness.deliverAck({ device: "LINE-2", time: "2000", status: 2 });
    assert.equal(updates.length, 2);
    assert.equal(
        harness.queryCalls.filter((call) => isDevicesQuery(call.sql)).length,
        1,
        "Default batch must not continuously drain after ACK"
    );

    await harness.close();
});

test("explicit batchSize is applied independently to every device", async () => {
    const rows = {
        A: [
            { device: "A", time: 1, content: "A1" },
            { device: "A", time: 2, content: "A2" },
            { device: "A", time: 3, content: "A3" }
        ],
        B: [
            { device: "B", time: 1, content: "B1" },
            { device: "B", time: 2, content: "B2" },
            { device: "B", time: 3, content: "B3" }
        ]
    };

    const harness = createHarness({
        config: { batchSize: 2 },
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                callback(null, [{ device: "A" }, { device: "B" }]);
            } else if (isRowsQuery(call.sql)) {
                assert.equal(call.params[2], 2);
                assert.match(call.sql, /ORDER BY CAST\(time AS UNSIGNED\) ASC/);
                callback(null, rows[call.params[1]]);
            } else {
                callback(null, { affectedRows: 1 });
            }
        }
    });

    harness.node.emit("input", {});
    assert.equal(harness.published.length, 4);
    assert.deepEqual(
        harness.published.map((entry) => {
            const payload = JSON.parse(entry.payload);
            return `${payload.device}:${payload.time}`;
        }),
        ["A:1", "A:2", "B:1", "B:2"]
    );
    await harness.close();
});

test("malformed, wrong-status and non-inflight MQTT ACKs cannot update the database", async () => {
    let updates = 0;
    const harness = createHarness({
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                callback(null, [{ device: "D1" }]);
            } else if (isRowsQuery(call.sql)) {
                callback(null, [{ device: "D1", time: 10, content: "payload" }]);
            } else if (isUpdate(call.sql)) {
                updates += 1;
                callback(null, { affectedRows: 1 });
            }
        }
    });

    harness.node.emit("input", {});
    harness.deliverAck("{broken");
    harness.deliverAck({ device: "D1", time: 10, status: 1 });
    harness.deliverAck({ device: "UNKNOWN", time: 10, status: 2 });
    harness.deliverAck({ device: "D1", time: "not-a-time", status: 2 });

    assert.equal(updates, 0);
    assert.equal(harness.node.inflight.size, 1);
    assert.ok(harness.warnings.length >= 4);

    harness.deliverAck({ device: "D1", time: "10", status: "2" });
    assert.equal(updates, 1);
    assert.equal(harness.node.inflight.size, 0);
    await harness.close();
});

test("busy is a real query lock and an overlapping trigger is coalesced", async () => {
    let devicesCallback;
    let deviceQueries = 0;
    let updates = 0;

    const harness = createHarness({
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                if (!devicesCallback) {
                    devicesCallback = callback;
                } else {
                    callback(null, []);
                }
            } else if (isRowsQuery(call.sql)) {
                deviceQueries += 1;
                callback(null, [{ device: "D1", time: 50, content: "payload" }]);
            } else if (isUpdate(call.sql)) {
                updates += 1;
                callback(null, { affectedRows: 1 });
            }
        }
    });

    harness.node.emit("input", { payload: 1 });
    harness.node.emit("input", { payload: 2 });
    assert.equal(
        harness.queryCalls.filter((call) => isDevicesQuery(call.sql)).length,
        1,
        "Second input must not start a concurrent SELECT"
    );
    assert.equal(harness.node.busy, true);

    devicesCallback(null, [{ device: "D1" }]);
    assert.equal(deviceQueries, 1);
    assert.equal(harness.published.length, 1);

    harness.deliverAck({ device: "D1", time: 50, status: 2 });
    assert.equal(updates, 1);
    await waitFor(
        () => harness.queryCalls.filter((call) => isDevicesQuery(call.sql)).length === 2
    );

    await harness.close();
});

test("missing application ACK releases inflight and retries later", async () => {
    const harness = createHarness({
        config: { inflightTimeoutSeconds: 0.01 },
        retryDelayMs: 5,
        limits: {
            minInflightTimeoutMs: 5,
            maxInflightTimeoutMs: 1_000
        },
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                callback(null, [{ device: "D1" }]);
            } else if (isRowsQuery(call.sql)) {
                callback(null, [{ device: "D1", time: 70, content: "retry-me" }]);
            } else {
                callback(null, { affectedRows: 1 });
            }
        }
    });

    harness.node.emit("input", {});
    assert.equal(harness.published.length, 1);
    await waitFor(() => harness.published.length >= 2, 300);
    assert.ok(
        harness.warnings.some((message) => message.includes("ACK timeout")),
        "Timeout must be observable"
    );

    await harness.close();
});

test("REST applies a real timeout and handles network errors without marking success", async () => {
    let configuredTimeout = null;
    let updates = 0;
    const request = {
        post() {
            const chain = {
                set() {
                    return chain;
                },
                timeout(value) {
                    configuredTimeout = value;
                    return chain;
                },
                send() {
                    return chain;
                },
                end(callback) {
                    const error = new Error("request timed out");
                    error.code = "ETIMEDOUT";
                    callback(error);
                }
            };
            return chain;
        }
    };

    const harness = createHarness({
        request,
        config: { priority: "1", requestTimeoutSeconds: 2 },
        retryDelayMs: 100,
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                callback(null, [{ device: "REST-1" }]);
            } else if (isRowsQuery(call.sql)) {
                callback(null, [{ device: "REST-1", time: 90, content: "content", alarm: 0 }]);
            } else if (isUpdate(call.sql)) {
                updates += 1;
                callback(null, { affectedRows: 1 });
            }
        }
    });

    harness.node.emit("input", {});
    assert.deepEqual(configuredTimeout, { response: 2000, deadline: 3000 });
    assert.equal(updates, 0);
    assert.ok(harness.errors.some((message) => message.includes("ETIMEDOUT")));
    assert.equal(harness.node.inflight.size, 0);
    await harness.close();
});

test("validated REST response persists a parameterized mentor-scoped ACK", async () => {
    const updates = [];
    const request = {
        post() {
            const chain = {
                set() {
                    return chain;
                },
                timeout() {
                    return chain;
                },
                send() {
                    return chain;
                },
                end(callback) {
                    callback(null, {
                        statusCode: 200,
                        headers: {},
                        body: { code: "REST'2", time: 91, status: 1 }
                    });
                }
            };
            return chain;
        }
    };

    const harness = createHarness({
        request,
        config: { priority: "1" },
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                callback(null, [{ device: "REST'2" }]);
            } else if (isRowsQuery(call.sql)) {
                callback(null, [{ device: "REST'2", time: 91, content: "content", alarm: 0 }]);
            } else if (isUpdate(call.sql)) {
                updates.push(call);
                callback(null, { affectedRows: 1 });
            }
        }
    });

    harness.node.emit("input", {});
    assert.equal(updates.length, 1);
    assert.doesNotMatch(updates[0].sql, /REST'2/);
    assert.match(updates[0].sql, /mentor_id = \?/);
    assert.deepEqual(updates[0].params, [1, 42, "REST'2", "91"]);
    await harness.close();
});

test("editor definition persists safe batch and timeout properties", () => {
    const htmlPath = path.join(__dirname, "..", "services", "sender.html");
    const html = fs.readFileSync(htmlPath, "utf8");
    assert.match(html, /batchSize:\s*\{\s*value:\s*1/);
    assert.match(html, /inflightTimeoutSeconds:\s*\{\s*value:\s*60/);
    assert.match(html, /requestTimeoutSeconds:\s*\{\s*value:\s*120/);
});

test("missing Mentor config preserves the legacy mentor_id=0 scope", async () => {
    let observedMentorId = null;
    const harness = createHarness({
        mentorPresent: false,
        onQuery(call, callback) {
            if (isDevicesQuery(call.sql)) {
                observedMentorId = call.params[0];
                callback(null, []);
            } else {
                callback(null, []);
            }
        }
    });

    harness.node.emit("input", {});
    assert.equal(observedMentorId, 0);
    assert.equal(harness.subscription.topic, "mentor_answer_0");
    await harness.close();
});

test("device identity follows the VARCHAR(50) storage limit", () => {
    const { normalizeIdentity } = senderModule._internals;
    assert.deepEqual(normalizeIdentity({ device: "D".repeat(50), time: "001" }), {
        device: "D".repeat(50),
        time: "001"
    });
    assert.equal(normalizeIdentity({ device: "D".repeat(51), time: "001" }), null);
});
