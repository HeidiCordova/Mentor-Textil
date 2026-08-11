"use strict";

const DEFAULT_BATCH_SIZE = 1;
const MAX_BATCH_SIZE = 100;
const DEFAULT_INFLIGHT_TIMEOUT_MS = 60_000;
const DEFAULT_RETRY_DELAY_MS = 5_000;
const DEFAULT_REQUEST_TIMEOUT_MS = 120_000;
const MQTT_DATA_TOPIC = "mentor_data";

const SQL = Object.freeze({
    selectMqttDevices: [
        "SELECT DISTINCT device",
        "FROM mqtt_lecturas",
        "WHERE mentor_id = ? AND status = 0",
        "ORDER BY device ASC"
    ].join(" "),
    selectMqttForDevice: [
        "SELECT device, time, content, alarm",
        "FROM mqtt_lecturas",
        "WHERE mentor_id = ? AND status = 0 AND device = ?",
        "ORDER BY CAST(time AS UNSIGNED) ASC",
        "LIMIT ?"
    ].join(" "),
    selectRestDevices: [
        "SELECT DISTINCT device",
        "FROM mqtt_lecturas",
        "WHERE mentor_id = ? AND restful = 0",
        "ORDER BY device ASC"
    ].join(" "),
    selectRestForDevice: [
        "SELECT device, time, content, alarm",
        "FROM mqtt_lecturas",
        "WHERE mentor_id = ? AND restful = 0 AND device = ?",
        "ORDER BY CAST(time AS UNSIGNED) ASC",
        "LIMIT ?"
    ].join(" "),
    acknowledgeMqtt: [
        "UPDATE mqtt_lecturas",
        "SET restful = 1, status = ?",
        "WHERE mentor_id = ? AND device = ?",
        "AND time = ?",
        "AND status = 0"
    ].join(" "),
    acknowledgeRest: [
        "UPDATE mqtt_lecturas",
        "SET status = 1, restful = ?",
        "WHERE mentor_id = ? AND device = ?",
        "AND time = ?",
        "AND restful = 0"
    ].join(" ")
});

function createSenderModule(dependencies = {}) {
    const timers = dependencies.timers || {
        setTimeout: global.setTimeout,
        clearTimeout: global.clearTimeout
    };
    const limits = Object.assign({
        minInflightTimeoutMs: 1_000,
        maxInflightTimeoutMs: 30 * 60_000
    }, dependencies.limits);
    const retryDelayMs = positiveInteger(
        dependencies.retryDelayMs,
        DEFAULT_RETRY_DELAY_MS,
        1,
        60_000
    );

    let requestClient = dependencies.request || null;
    let utilsClient = dependencies.utils || null;

    function getRequestClient() {
        if (!requestClient) {
            // Loaded lazily so MQTT-only deployments do not require REST setup at startup.
            requestClient = require("superagent");
        }
        return requestClient;
    }

    function getUtilsClient() {
        if (!utilsClient) {
            // This file is intended to overlay the existing package beside utils.js.
            utilsClient = require("./utils.js");
        }
        return utilsClient;
    }

    function register(RED) {
        const log = RED.log || {};

        function SenderNode(config) {
            RED.nodes.createNode(this, config);
            const node = this;

            node.mydbConfig = RED.nodes.getNode(config.mysql);
            node.mentor = RED.nodes.getNode(config.mentor);
            node.brokerConn = RED.nodes.getNode(config.mqtt);
            node.priority = parsePriority(config.priority);
            node.mentorId = normalizeMentorId(node.mentor && node.mentor.mentor_id);
            node.batchSize = parseBatchSize(config.batchSize);
            node.inflightTimeoutMs = durationSeconds(
                config.inflightTimeoutSeconds,
                DEFAULT_INFLIGHT_TIMEOUT_MS,
                limits.minInflightTimeoutMs,
                limits.maxInflightTimeoutMs
            );
            node.requestTimeoutMs = durationSeconds(
                config.requestTimeoutSeconds,
                DEFAULT_REQUEST_TIMEOUT_MS,
                1_000,
                30 * 60_000
            );
            node.url = typeof config.url === "string" ? config.url : "";
            node.urlAlarm = typeof config.url_alarm === "string" ? config.url_alarm : "";
            node.channel = `mentor_answer_${node.mentorId}`;

            node.busy = false;
            node.closed = false;
            node.drainRequested = false;
            node.activated = false;
            node.retryTimer = null;
            node.inflight = new Map();
            node.ackUpdates = new Set();
            node.lastInputMessage = {};

            if (node.mydbConfig) {
                if (typeof node.mydbConfig.connect === "function") {
                    node.mydbConfig.connect();
                }
                node.databaseStateListener = function databaseStateListener(info) {
                    setDatabaseStatus(node, info);
                    if (info === "connected" && node.activated && node.drainRequested) {
                        scheduleDrain(node, 0);
                    }
                };
                if (typeof node.mydbConfig.on === "function") {
                    node.mydbConfig.on("state", node.databaseStateListener);
                }
            } else {
                node.error("MySQL database not configured");
            }

            if (node.brokerConn) {
                if (typeof node.brokerConn.register === "function") {
                    node.brokerConn.register(node);
                }
                if (typeof node.brokerConn.subscribe === "function") {
                    node.brokerConn.subscribe(
                        node.channel,
                        2,
                        function onBrokerMessage(topic, payload) {
                            if (topic !== node.channel) {
                                return;
                            }
                            processMqttAcknowledgement(node, payload);
                        },
                        node.id
                    );
                }
            } else if (usesMqtt(node.priority)) {
                node.error("MQTT broker not configured");
            }

            node.on("input", function onInput(msg, send, done) {
                node.activated = true;
                node.lastInputMessage = safeMessageTemplate(msg);
                if (node.busy || node.inflight.size > 0) {
                    // Coalesce an overlapping trigger and execute it only after
                    // the current application ACK set has completed.
                    node.drainRequested = true;
                } else {
                    requestDrain(node);
                }
                if (typeof done === "function") {
                    done();
                }
            });

            node.on("close", function onClose(removed, done) {
                if (typeof removed === "function") {
                    done = removed;
                }
                closeNode(node, typeof done === "function" ? done : function noop() {});
            });
        }

        function requestDrain(node) {
            if (node.closed) {
                return;
            }
            if (node.busy) {
                node.drainRequested = true;
                return;
            }
            drainPending(node);
        }

        function drainPending(node) {
            if (node.closed || node.busy) {
                return;
            }

            if (!node.mydbConfig || !node.mydbConfig.connected ||
                !node.mydbConfig.connection ||
                typeof node.mydbConfig.connection.query !== "function") {
                node.status({ fill: "red", shape: "ring", text: "database disconnected" });
                scheduleDrain(node, retryDelayMs);
                return;
            }
            if (usesMqtt(node.priority) &&
                (!node.brokerConn || node.brokerConn.connected !== true)) {
                node.status({ fill: "red", shape: "ring", text: "mqtt disconnected" });
                scheduleDrain(node, retryDelayMs);
                return;
            }

            node.busy = true;
            const devicesSql = usesMqtt(node.priority)
                ? SQL.selectMqttDevices
                : SQL.selectRestDevices;
            node.mydbConfig.connection.query(
                devicesSql,
                [node.mentorId],
                function onPendingDevices(error, rows) {
                    if (node.closed) {
                        finishDrain(node);
                        return;
                    }
                    if (error) {
                        reportError(node, "Unable to query pending devices", error);
                        finishDrain(node);
                        scheduleDrain(node, retryDelayMs);
                        return;
                    }

                    const devices = uniqueValidDevices(rows);
                    let index = 0;
                    let totalDispatched = 0;

                    function nextDevice() {
                        if (node.closed || index >= devices.length) {
                            if (totalDispatched === 0 && node.inflight.size === 0) {
                                node.status({ fill: "green", shape: "ring", text: "queue idle" });
                            }
                            finishDrain(node);
                            return;
                        }

                        const device = devices[index++];
                        const rowsSql = usesMqtt(node.priority)
                            ? SQL.selectMqttForDevice
                            : SQL.selectRestForDevice;
                        node.mydbConfig.connection.query(
                            rowsSql,
                            [node.mentorId, device, node.batchSize],
                            function onPendingRows(rowError, pendingRows) {
                                if (node.closed) {
                                    finishDrain(node);
                                    return;
                                }
                                if (rowError) {
                                    reportError(node, `Unable to query pending records for ${device}`, rowError);
                                    nextDevice();
                                    return;
                                }

                                let dispatchedForDevice = 0;
                                for (const row of Array.isArray(pendingRows) ? pendingRows : []) {
                                    if (dispatchedForDevice >= node.batchSize) {
                                        break;
                                    }
                                    const identity = normalizeIdentity(row);
                                    if (!identity || identity.device !== device) {
                                        node.warn("Skipped pending record with invalid device/time identity");
                                        continue;
                                    }
                                    const key = identityKey(identity);
                                    if (node.inflight.has(key) || node.ackUpdates.has(key)) {
                                        continue;
                                    }

                                    const sent = usesMqtt(node.priority)
                                        ? dispatchMqtt(node, row, identity)
                                        : dispatchRest(node, row, identity);
                                    if (sent) {
                                        dispatchedForDevice += 1;
                                        totalDispatched += 1;
                                    }
                                }
                                nextDevice();
                            }
                        );
                    }

                    nextDevice();
                }
            );
        }

        function finishDrain(node) {
            node.busy = false;
            if (node.drainRequested && node.inflight.size === 0) {
                node.drainRequested = false;
                scheduleDrain(node, 0);
            }
        }

        function dispatchMqtt(node, row, identity) {
            const key = identityKey(identity);
            addInflight(node, key, "mqtt", identity);

            const envelope = {
                device: identity.device,
                time: identity.time,
                content: row.content,
                alarm: normalizeAlarm(row.alarm),
                mentor_id: node.mentorId
            };
            const message = {
                topic: MQTT_DATA_TOPIC,
                qos: 1,
                retain: false,
                payload: JSON.stringify(envelope)
            };

            try {
                node.brokerConn.publish(message);
                node.send({ payload: row.content });
                node.status({
                    fill: "yellow",
                    shape: "dot",
                    text: `${node.inflight.size} awaiting ACK`
                });
                return true;
            } catch (error) {
                releaseInflight(node, key);
                reportError(node, "MQTT publish failed", error);
                scheduleDrain(node, retryDelayMs);
                return false;
            }
        }

        function dispatchRest(node, row, identity) {
            const targetUrl = normalizeAlarm(row.alarm) === 1 ? node.urlAlarm : node.url;
            if (!targetUrl) {
                node.warn("REST record remains pending: target URL is not configured");
                scheduleDrain(node, retryDelayMs);
                return false;
            }

            const key = identityKey(identity);
            addInflight(node, key, "rest", identity);

            let encodedPayload;
            try {
                encodedPayload = getUtilsClient().encode(row.content);
            } catch (error) {
                releaseInflight(node, key);
                reportError(node, "Unable to encode REST payload", error);
                scheduleDrain(node, retryDelayMs);
                return false;
            }

            const outboundMessage = Object.assign({}, node.lastInputMessage, {
                alarm: normalizeAlarm(row.alarm),
                data: row.content,
                payload: encodedPayload
            });

            let request;
            try {
                request = getRequestClient()
                    .post(targetUrl)
                    .set("Content-Type", "application/json")
                    .timeout({
                        response: node.requestTimeoutMs,
                        deadline: node.requestTimeoutMs + 1_000
                    })
                    .send(outboundMessage);
            } catch (error) {
                releaseInflight(node, key);
                reportError(node, "Unable to create REST request", error);
                scheduleDrain(node, retryDelayMs);
                return false;
            }

            request.end(function onRestResponse(error, response) {
                if (node.closed) {
                    return;
                }
                if (error) {
                    releaseInflight(node, key);
                    reportError(node, "REST request failed", error);
                    scheduleDrain(node, retryDelayMs);
                    return;
                }

                const body = parseResponseBody(response);
                const acknowledgement = validateAcknowledgement({
                    device: body && (body.device !== undefined ? body.device : body.code),
                    time: body && body.time,
                    status: body && body.status
                }, new Set([1]));

                if (!acknowledgement ||
                    identityKey(acknowledgement) !== key) {
                    releaseInflight(node, key);
                    node.warn("REST response did not contain the expected validated ACK");
                    scheduleDrain(node, retryDelayMs);
                    return;
                }

                node.send({
                    payload: body,
                    headers: response && response.headers,
                    statusCode: response && response.statusCode
                });
                persistAcknowledgement(node, acknowledgement, "rest");
            });

            node.status({
                fill: "yellow",
                shape: "dot",
                text: `${node.inflight.size} awaiting ACK`
            });
            return true;
        }

        function processMqttAcknowledgement(node, payload) {
            let parsed;
            try {
                parsed = JSON.parse(Buffer.isBuffer(payload) ? payload.toString() : String(payload));
            } catch (error) {
                node.warn("Ignored malformed MQTT ACK JSON");
                return;
            }

            const acknowledgement = validateAcknowledgement(parsed, new Set([2]));
            if (!acknowledgement) {
                node.warn("Ignored MQTT ACK with invalid device/time/status");
                return;
            }
            persistAcknowledgement(node, acknowledgement, "mqtt");
        }

        function persistAcknowledgement(node, acknowledgement, transport) {
            const key = identityKey(acknowledgement);
            const entry = node.inflight.get(key);

            // Only ACK a record this process actually published. A late ACK after
            // restart/timeout is safely ignored; the idempotent receiver can ACK
            // the retry again.
            if (!entry || entry.transport !== transport || node.ackUpdates.has(key)) {
                node.warn("Ignored ACK for a record that is not in-flight");
                return;
            }
            if (!node.mydbConfig || !node.mydbConfig.connected ||
                !node.mydbConfig.connection ||
                typeof node.mydbConfig.connection.query !== "function") {
                releaseInflight(node, key);
                node.warn("ACK could not be persisted while database is disconnected");
                scheduleDrain(node, retryDelayMs);
                return;
            }

            node.ackUpdates.add(key);
            const sql = transport === "mqtt" ? SQL.acknowledgeMqtt : SQL.acknowledgeRest;
            const params = [
                acknowledgement.status,
                node.mentorId,
                acknowledgement.device,
                acknowledgement.time
            ];

            node.mydbConfig.connection.query(sql, params, function onAckSaved(error, result) {
                node.ackUpdates.delete(key);
                if (node.closed) {
                    return;
                }
                if (error) {
                    releaseInflight(node, key);
                    reportError(node, "Unable to persist ACK", error);
                    scheduleDrain(node, retryDelayMs);
                    return;
                }

                releaseInflight(node, key);
                node.send({
                    payload: {
                        msg: `${Number(result && result.affectedRows) || 0} record(s) updated`,
                        data: acknowledgement
                    }
                });
                node.status({ fill: "green", shape: "dot", text: "ACK persisted" });
                if (node.drainRequested && node.inflight.size === 0) {
                    node.drainRequested = false;
                    scheduleDrain(node, 0);
                }
            });
        }

        function addInflight(node, key, transport, identity) {
            const timer = timers.setTimeout(function onInflightTimeout() {
                const current = node.inflight.get(key);
                if (!current || current.timer !== timer) {
                    return;
                }
                node.inflight.delete(key);
                node.warn(`ACK timeout for ${identity.device} at ${identity.time}; retry scheduled`);
                scheduleDrain(node, retryDelayMs);
            }, node.inflightTimeoutMs);
            unrefTimer(timer);
            node.inflight.set(key, {
                transport,
                identity,
                timer
            });
        }

        function releaseInflight(node, key) {
            const entry = node.inflight.get(key);
            if (!entry) {
                return;
            }
            timers.clearTimeout(entry.timer);
            node.inflight.delete(key);
        }

        function scheduleDrain(node, delayMs) {
            if (node.closed) {
                return;
            }
            if (node.retryTimer) {
                if (delayMs > 0) {
                    return;
                }
                timers.clearTimeout(node.retryTimer);
                node.retryTimer = null;
            }
            node.retryTimer = timers.setTimeout(function scheduledDrain() {
                node.retryTimer = null;
                requestDrain(node);
            }, Math.max(0, delayMs));
            unrefTimer(node.retryTimer);
        }

        function closeNode(node, done) {
            node.closed = true;
            if (node.retryTimer) {
                timers.clearTimeout(node.retryTimer);
                node.retryTimer = null;
            }
            for (const entry of node.inflight.values()) {
                timers.clearTimeout(entry.timer);
            }
            node.inflight.clear();
            node.ackUpdates.clear();

            if (node.mydbConfig && node.databaseStateListener &&
                typeof node.mydbConfig.removeListener === "function") {
                node.mydbConfig.removeListener("state", node.databaseStateListener);
            }
            node.status({});

            if (node.brokerConn && typeof node.brokerConn.deregister === "function") {
                try {
                    node.brokerConn.deregister(node, done);
                    return;
                } catch (error) {
                    reportError(node, "Unable to deregister MQTT sender", error);
                }
            }
            done();
        }

        function setDatabaseStatus(node, info) {
            if (info === "connected") {
                node.status({ fill: "green", shape: "dot", text: "connected" });
                return;
            }
            const label = info === "ECONNREFUSED"
                ? "connection refused"
                : info === "PROTOCOL_CONNECTION_LOST"
                    ? "connection lost"
                    : String(info || "disconnected");
            node.status({
                fill: label === "connecting" ? "grey" : "red",
                shape: "ring",
                text: label
            });
        }

        function reportError(node, prefix, error) {
            const detail = error && (error.code || error.message)
                ? String(error.code || error.message)
                : "unknown error";
            node.error(`${prefix}: ${detail}`);
            if (typeof log.error === "function") {
                log.error(`${prefix}: ${detail}`);
            }
        }

        RED.nodes.registerType("Sender", SenderNode);
    }

    register._internals = {
        DEFAULT_BATCH_SIZE,
        DEFAULT_INFLIGHT_TIMEOUT_MS,
        MQTT_DATA_TOPIC,
        SQL,
        durationSeconds,
        identityKey,
        normalizeIdentity,
        parseBatchSize,
        validateAcknowledgement
    };
    return register;
}

function parsePriority(value) {
    const priority = Number.parseInt(value, 10);
    return [0, 1, 2, 3].includes(priority) ? priority : 0;
}

function usesMqtt(priority) {
    return priority === 0 || priority === 2;
}

function parseBatchSize(value) {
    // Safety default: an existing deployment that receives this overlay without
    // the new editor fields continues sending one record per trigger.
    return positiveInteger(value, DEFAULT_BATCH_SIZE, 1, MAX_BATCH_SIZE);
}

function positiveInteger(value, fallback, minimum, maximum) {
    if (value === undefined || value === null || value === "") {
        return fallback;
    }
    const parsed = Number(value);
    if (!Number.isSafeInteger(parsed) || parsed < minimum || parsed > maximum) {
        return fallback;
    }
    return parsed;
}

function durationSeconds(value, fallbackMs, minimumMs, maximumMs) {
    if (value === undefined || value === null || value === "") {
        return fallbackMs;
    }
    const milliseconds = Number(value) * 1_000;
    if (!Number.isFinite(milliseconds) ||
        milliseconds < minimumMs ||
        milliseconds > maximumMs) {
        return fallbackMs;
    }
    return Math.round(milliseconds);
}

function normalizeMentorId(value) {
    const parsed = Number(value);
    if (!Number.isSafeInteger(parsed) || parsed < 0) {
        return 0;
    }
    return parsed;
}

function normalizeIdentity(value) {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
        return null;
    }

    const device = normalizeDevice(value.device);
    if (device === null) {
        return null;
    }

    const time = normalizeUnsignedInteger(value.time);
    if (time === null) {
        return null;
    }
    return { device, time };
}

function normalizeDevice(value) {
    let device;
    if (typeof value === "string") {
        device = value;
    } else if (typeof value === "number" && Number.isFinite(value)) {
        device = String(value);
    } else {
        return null;
    }
    if (device.trim().length === 0 || device.length > 50 || device.includes("\u0000")) {
        return null;
    }
    return device;
}

function uniqueValidDevices(rows) {
    const devices = [];
    const seen = new Set();
    for (const row of Array.isArray(rows) ? rows : []) {
        const device = normalizeDevice(row && row.device);
        if (device === null || seen.has(device)) {
            continue;
        }
        seen.add(device);
        devices.push(device);
    }
    return devices;
}

function normalizeUnsignedInteger(value) {
    if (typeof value === "number") {
        if (!Number.isSafeInteger(value) || value < 0) {
            return null;
        }
        return String(value);
    }
    if (typeof value !== "string" || !/^\d{1,20}$/.test(value)) {
        return null;
    }
    try {
        const parsed = BigInt(value);
        if (parsed < 0n || parsed > 18_446_744_073_709_551_615n) {
            return null;
        }
        return value;
    } catch (error) {
        return null;
    }
}

function validateAcknowledgement(value, allowedStatuses = new Set([1, 2])) {
    const identity = normalizeIdentity(value);
    if (!identity) {
        return null;
    }
    const status = typeof value.status === "string" && /^\d+$/.test(value.status)
        ? Number(value.status)
        : value.status;
    if (!Number.isSafeInteger(status) || !allowedStatuses.has(status)) {
        return null;
    }
    return Object.assign(identity, { status });
}

function identityKey(identity) {
    return JSON.stringify([identity.device, identity.time]);
}

function normalizeAlarm(value) {
    return Number(value) === 1 ? 1 : 0;
}

function safeMessageTemplate(message) {
    if (!message || typeof message !== "object" || Array.isArray(message)) {
        return {};
    }
    return Object.assign({}, message);
}

function parseResponseBody(response) {
    if (!response) {
        return null;
    }
    if (response.body && typeof response.body === "object") {
        return response.body;
    }
    const raw = response.text !== undefined ? response.text : response.body;
    if (typeof raw !== "string") {
        return null;
    }
    try {
        return JSON.parse(raw);
    } catch (error) {
        return null;
    }
}

function unrefTimer(timer) {
    if (timer && typeof timer.unref === "function") {
        timer.unref();
    }
}

const senderModule = createSenderModule();
senderModule.createSenderModule = createSenderModule;
module.exports = senderModule;
