"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const root = __dirname;
const shell = fs.readFileSync(
    path.join(root, "deploy-runtime-hardening.sh"),
    "utf8"
);
const helper = fs.readFileSync(
    path.join(root, "runtime-context.js"),
    "utf8"
);
const settings = fs.readFileSync(
    path.join(root, "settings-persistent-context.js"),
    "utf8"
);
const bundledRelease = path.join(
    root,
    "release",
    "sender-release-canary-v2.js"
);
const sourceRelease = path.join(root, "..", "sender-release-canary.js");
const release = fs.readFileSync(
    fs.existsSync(bundledRelease) ? bundledRelease : sourceRelease,
    "utf8"
);

test("no hereda epoch ni conteo fijo del piloto v4", () => {
    for (const source of [shell, helper]) {
        assert.doesNotMatch(source, /1785457500/);
        assert.doesNotMatch(source, /COUNTER_PREFLIGHT/);
        assert.doesNotMatch(source, /official_counter_state/);
    }
});

test("no escribe ni despliega flows", () => {
    assert.doesNotMatch(shell, /POST\s+\/flows/i);
    assert.doesNotMatch(shell, /docker\s+compose\s+.*\bup\b/);
    assert.doesNotMatch(shell, /flows\.json\.\$stamp\.tmp/);
    assert.match(shell, /require_flow_hash/);
    assert.match(shell, /flow-hash/);
    assert.doesNotMatch(shell, /sha_file_in_container \/data\/flows\.json/);
});

test("mantiene aislamiento como precondición de cada reinicio", () => {
    const restartCount = (shell.match(/docker restart -t 30/g) || []).length;
    assert.equal(restartCount, 3);
    assert.match(shell, /run_isolation_audit/);
    assert.match(shell, /require_firewall/);
    assert.match(helper, /Sender principal NO está desactivado/);
    assert.match(helper, /existe al menos un Sender habilitado/);
});

test("runner de pruebas es compatible con Node 20 del Jetson", () => {
    assert.match(shell, /\bnode --test\s+\\/);
    assert.doesNotMatch(shell, /--test-isolation/);
});

test("contextStorage usa localfilesystem en un base dedicado", () => {
    assert.match(settings, /module:\s*"localfilesystem"/);
    assert.match(settings, /dir:\s*"\/data"/);
    assert.match(settings, /base:\s*"mentor-context"/);
    assert.match(settings, /flushInterval:\s*5/);
});

test("rollback posterior conserva settings persistentes", () => {
    const rollbackStart = shell.indexOf("rollback_sender_action()");
    assert.ok(rollbackStart > 0);
    const rollbackBody = shell.slice(rollbackStart);
    assert.doesNotMatch(rollbackBody, /restore_backup_file settings\.js/);
    assert.match(rollbackBody, /context\.rollback\.before\.json/);
    assert.match(rollbackBody, /context\.rollback\.validate\.log/);
});

test("release posterior exige perfil hardened JS y HTML", () => {
    assert.match(release, /mentor-sender-release-canary:v2/);
    assert.match(
        release,
        /f64477cd45698afff49e2209fa8cbc051aeb752a22680107a42fdafc6deb771a/
    );
    assert.match(
        release,
        /5dddf9c7d0e39c9e78968653497bdb0c9d1dc285297cb40dcfc009d06aebd178/
    );
    assert.match(release, /senderProfile:\s*"legacy"/);
    assert.match(release, /--sender-profile debe ser legacy o hardened/);
    assert.match(release, /state\.senderProfile !== args\.senderProfile/);
});
