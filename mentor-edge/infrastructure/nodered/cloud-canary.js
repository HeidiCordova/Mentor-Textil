"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");

const EXPECTED_DEVICE = "ART_ATLAS_MAQUINA_1_PRODUCCION";
const MAX_PAST_AGE_SECONDS = 20 * 60;
const MAX_FUTURE_AGE_SECONDS = 5 * 60;
const CANARY_TIMEOUT_MS = 30_000;

function loadModule(candidates) {
    const errors = [];
    for (const candidate of candidates) {
        try {
            return require(candidate);
        } catch (error) {
            errors.push(`${candidate}:${error.code || error.message}`);
        }
    }
    throw new Error(`module unavailable (${errors.join("; ")})`);
}

function decryptCredentials() {
    const runtimeConfig = JSON.parse(
        fs.readFileSync("/data/.config.runtime.json", "utf8"),
    );
    const encryptedFile = JSON.parse(
        fs.readFileSync("/data/flows_cred.json", "utf8"),
    );
    const secret = runtimeConfig._credentialSecret;
    const encrypted = encryptedFile.$;

    if (typeof secret !== "string" || typeof encrypted !== "string") {
        throw new Error("Node-RED credential material is unavailable");
    }

    const iv = Buffer.from(encrypted.slice(0, 32), "hex");
    const key = crypto.createHash("sha256").update(secret).digest();
    const decipher = crypto.createDecipheriv("aes-256-ctr", key, iv);
    const plaintext =
        decipher.update(encrypted.slice(32), "base64", "utf8") +
        decipher.final("utf8");
    return JSON.parse(plaintext);
}

function loadRuntimeConfiguration() {
    const flows = JSON.parse(fs.readFileSync("/data/flows.json", "utf8"));
    const byId = Object.fromEntries(flows.map((node) => [node.id, node]));
    const sender = flows.find(
        (node) =>
            node.type === "Sender" &&
            node.name === "ART_ATLAS_SENDER_GENERAL",
    );

    if (!sender || sender.d !== true) {
        throw new Error("official Sender is missing or is not safely disabled");
    }

    const database = byId[sender.mysql];
    const broker = byId[sender.mqtt];
    const mentor = byId[sender.mentor];
    if (!database || !broker || !mentor) {
        throw new Error("Sender configuration nodes are incomplete");
    }

    return { database, broker, mentor };
}

function query(connection, sql, params) {
    return new Promise((resolve, reject) => {
        connection.query(sql, params, (error, rows) => {
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
    return new Promise((resolve) => connection.end(() => resolve()));
}

function validateEnvelope(envelope, mentorId) {
    if (
        envelope.device !== EXPECTED_DEVICE ||
        String(envelope.mentor_id) !== String(mentorId) ||
        !/^\d{13}$/.test(String(envelope.time)) ||
        ![0, 1].includes(Number(envelope.alarm))
    ) {
        throw new Error("selected MariaDB envelope is invalid");
    }

    const content = JSON.parse(envelope.content);
    const expectedHead = [
        "L1_T_DISPONIBLE",
        "L1_T_MICROPARADA",
        "L1_T_PARADA_NO_ASIGNADA",
        "L1_CONTEO_1",
    ];
    if (
        content.code !== envelope.device ||
        String(content.time) !== String(envelope.time) ||
        JSON.stringify(content.head) !== JSON.stringify(expectedHead) ||
        !Array.isArray(content.data) ||
        content.data.length !== 4
    ) {
        throw new Error("selected MariaDB content does not match the contract");
    }

    const ageSeconds = (Date.now() - Number(envelope.time)) / 1000;
    if (
        ageSeconds > MAX_PAST_AGE_SECONDS ||
        ageSeconds < -MAX_FUTURE_AGE_SECONDS
    ) {
        throw new Error(`selected row is not recent (age=${ageSeconds}s)`);
    }

    return content;
}

function mqttCanary(mqtt, broker, brokerCredentials, envelope, mentorId) {
    return new Promise((resolve, reject) => {
        const ackTopic = `mentor_answer_${mentorId}`;
        const clientId = `mentor_canary_${process.pid}_${Date.now()
            .toString()
            .slice(-8)}`;
        let finished = false;
        let brokerPuback = false;
        let applicationAck = false;

        const options = {
            clientId,
            protocolVersion: 4,
            clean: true,
            reconnectPeriod: 0,
            connectTimeout: 8_000,
            keepalive: 30,
            resubscribe: false,
        };
        if (brokerCredentials.user) options.username = brokerCredentials.user;
        if (brokerCredentials.password) {
            options.password = brokerCredentials.password;
        }

        const client = mqtt.connect(
            `mqtt://${broker.broker}:${broker.port}`,
            options,
        );
        const timeout = setTimeout(() => {
            finish(
                new Error(
                    `application ACK timeout (puback=${brokerPuback}, exact_ack=${applicationAck})`,
                ),
            );
        }, CANARY_TIMEOUT_MS);

        function finish(error) {
            if (finished) return;
            finished = true;
            clearTimeout(timeout);
            client.end(true, {}, () => {
                if (error) reject(error);
                else resolve({ ackTopic, brokerPuback, applicationAck });
            });
        }

        function maybeFinish() {
            if (brokerPuback && applicationAck) finish();
        }

        client.on("connect", () => {
            console.log(`MQTT_CONNECT_OK client=${clientId}`);
            client.subscribe(ackTopic, { qos: 2 }, (subscribeError, granted) => {
                if (subscribeError) return finish(subscribeError);
                const grant = Array.isArray(granted)
                    ? granted.find((item) => item.topic === ackTopic)
                    : undefined;
                if (!grant || grant.qos === 128) {
                    return finish(
                        new Error(
                            `subscription rejected: ${JSON.stringify(granted)}`,
                        ),
                    );
                }
                console.log(
                    `MQTT_SUBSCRIBE_OK topic=${ackTopic} qos=${grant.qos}`,
                );
                client.publish(
                    "mentor_data",
                    JSON.stringify(envelope),
                    { qos: 1, retain: false },
                    (publishError) => {
                        if (publishError) return finish(publishError);
                        brokerPuback = true;
                        console.log("MQTT_PUBACK_OK topic=mentor_data qos=1");
                        maybeFinish();
                    },
                );
            });
        });

        client.on("message", (topic, payload) => {
            if (topic !== ackTopic) return;
            let ack;
            try {
                ack = JSON.parse(payload.toString("utf8"));
            } catch {
                console.log("ACK_IGNORED malformed_json");
                return;
            }
            const exact =
                String(ack.device) === envelope.device &&
                String(ack.time) === String(envelope.time) &&
                Number(ack.status) === 2;
            if (!exact) {
                console.log(
                    `ACK_IGNORED ${JSON.stringify({
                        device: ack.device,
                        time: ack.time,
                        status: ack.status,
                    })}`,
                );
                return;
            }
            applicationAck = true;
            console.log("APPLICATION_ACK_EXACT_OK status=2");
            maybeFinish();
        });

        client.on("error", finish);
        client.on("close", () => {
            if (!finished) finish(new Error("MQTT connection closed"));
        });
    });
}

async function main() {
    const mysql = loadModule([
        "/data/node_modules/mysql",
        "/data/node_modules/mysql2",
        "mysql",
    ]);
    const mqtt = loadModule([
        "/data/node_modules/mqtt",
        "/usr/src/node-red/node_modules/mqtt",
        "mqtt",
    ]);
    const credentials = decryptCredentials();
    const { database, broker, mentor } = loadRuntimeConfiguration();
    const dbCredentials = credentials[database.id] || {};
    const brokerCredentials = credentials[broker.id] || {};
    const mentorId = Number(mentor.mentor_id);
    if (!Number.isInteger(mentorId) || mentorId <= 0) {
        throw new Error("invalid mentor_id");
    }

    const connection = await connectDatabase(mysql, {
        host:
            database.host === "host.docker.internal"
                ? "127.0.0.1"
                : database.host,
        port: Number(database.port || 3306),
        database: database.db,
        user: dbCredentials.user,
        password: dbCredentials.password,
        charset: database.charset || "UTF8MB4_GENERAL_CI",
        timezone: database.tz || "local",
    });

    try {
        const rows = await query(
            connection,
            `SELECT device, CAST(time AS CHAR) AS time, content,
                    COALESCE(alarm, 0) AS alarm, mentor_id
               FROM mqtt_lecturas
              WHERE BINARY device = BINARY ?
                AND mentor_id = ?
                AND status = 0
                AND restful = 0
              ORDER BY CAST(time AS UNSIGNED) DESC
              LIMIT 1`,
            [EXPECTED_DEVICE, mentorId],
        );
        if (rows.length !== 1) {
            throw new Error("no recent pending production row was found");
        }

        const envelope = {
            device: rows[0].device,
            time: String(rows[0].time),
            content: rows[0].content,
            alarm: Number(rows[0].alarm),
            mentor_id: Number(rows[0].mentor_id),
        };
        const content = validateEnvelope(envelope, mentorId);
        const contentHash = crypto
            .createHash("sha256")
            .update(envelope.content)
            .digest("hex");
        console.log(
            `CANARY_SELECTED device=${envelope.device} time=${envelope.time} count=${content.data[3]}`,
        );

        const result = await mqttCanary(
            mqtt,
            broker,
            brokerCredentials,
            envelope,
            mentorId,
        );

        const localRows = await query(
            connection,
            `SELECT COUNT(*) AS row_count, MIN(status) AS status,
                    MIN(restful) AS restful
               FROM mqtt_lecturas
              WHERE BINARY device = BINARY ?
                AND mentor_id = ?
                AND time = ?`,
            [EXPECTED_DEVICE, mentorId, envelope.time],
        );
        const local = localRows[0];
        if (
            Number(local.row_count) !== 1 ||
            Number(local.status) !== 0 ||
            Number(local.restful) !== 0
        ) {
            throw new Error(
                `local row changed unexpectedly (${local.row_count}|${local.status}|${local.restful})`,
            );
        }

        console.log(
            `CANARY_ACK_OK ${JSON.stringify({
                device: envelope.device,
                time: envelope.time,
                status: 2,
                topic: result.ackTopic,
                broker_puback: result.brokerPuback,
                content_sha256: contentHash,
            })}`,
        );
        console.log("CANARY_LOCAL_UNCHANGED=1|0|0");
    } finally {
        await closeDatabase(connection);
    }
}

main().catch((error) => {
    console.error(`CANARY_ERROR ${error.message}`);
    process.exitCode = 1;
});
