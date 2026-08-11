"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const patch = require("./patch-official-textile-count.js");
const hot = require("./hot-deploy-official-counter.js");
const release = require("./sender-release-canary.js");

function auditFlows() {
  const filename = process.env.NODERED_AUDIT_PATH || path.resolve(
    __dirname,
    "../../../nodered-flows.audit.json"
  );
  assert.equal(fs.existsSync(filename), true, filename);
  return JSON.parse(fs.readFileSync(filename, "utf8"));
}

function disabledCandidate() {
  return hot.prepareCandidate(auditFlows(), patch, 1).flows;
}

test("variantes exactas modifican solamente Sender e Interval", () => {
  const baseline = disabledCandidate();
  release.validateDisabledBaseline(baseline);
  assert.equal(
    release.flowHash(baseline),
    release.EXPECTED.disabledFlowHash
  );

  const expectedHashes = {
    disabled:
      "f65c87507232c7bdd4cc4aca7a0beec607b54cd60e704d65ff634569ff5e9e88",
    canary:
      "4bf469865d1b817ed21df7047ff289c12b331f44c04edf69774265a7f9484eed",
    drain:
      "8b94acfff321c4dcd4768cd5de8dcb3d9f58916e6c637109697b0f52d9b2c8e7",
    steady:
      "60e094378827ff42ce9fa29ff0e9abc7665c5048c9aca6ae9a6d8f0fb88356c1",
  };
  for (const [mode, expectedHash] of Object.entries(expectedHashes)) {
    const flows = release.buildVariant(baseline, mode);
    const byId = Object.fromEntries(flows.map((node) => [node.id, node]));
    assert.equal(release.flowHash(flows), expectedHash);
    assert.equal(
      byId[release.IDS.sender].d === true,
      release.MODES[mode].senderDisabled
    );
    assert.equal(
      byId[release.IDS.interval].interval,
      release.MODES[mode].interval
    );
    const diff = release.diffFlows(baseline, flows);
    assert.deepEqual(diff.added, []);
    assert.deepEqual(diff.removed, []);
    assert.deepEqual(
      diff.modified,
      mode === "disabled"
        ? []
        : [release.IDS.interval, release.IDS.sender].sort()
    );
  }
});

test("topologia fail-closed rechaza broker, usuarios o baseline ajenos", () => {
  const baseline = disabledCandidate();
  const changedBroker = structuredClone(baseline);
  changedBroker.find((node) => node.id === release.IDS.broker).broker =
    "broker.invalid";
  assert.throws(
    () => release.validateAuditedTopology(changedBroker),
    /broker externo/
  );

  const extraUser = structuredClone(baseline);
  extraUser.push({
    id: "unexpected",
    type: "Sender",
    mqtt: release.IDS.broker,
  });
  assert.throws(
    () => release.validateAuditedTopology(extraUser),
    /usuarios del broker/
  );

  const changedOtherNode = structuredClone(baseline);
  changedOtherNode.find((node) => node.type === "tab").label += " changed";
  assert.throws(
    () => release.validateDisabledBaseline(changedOtherNode),
    /hash baseline inesperado/
  );
});

test("CLI exige confirmacion explicita y paths absolutos", () => {
  const args = release.parseArgs([
    "--action", "apply",
    "--mode", "drain",
    "--state-dir", "/data/release",
    "--confirm-canary",
  ]);
  assert.equal(args.mode, "drain");
  assert.equal(args.confirmCanary, true);
  assert.equal(args.senderProfile, "legacy");
  const hardened = release.parseArgs([
    "--action", "prepare",
    "--state-dir", "/data/release-hardened",
    "--sender-profile", "hardened",
  ]);
  assert.equal(hardened.senderProfile, "hardened");
  assert.throws(
    () => release.parseArgs([
      "--action", "apply",
      "--mode", "fast",
      "--state-dir", "/data/release",
    ]),
    /--mode debe/
  );
  assert.throws(
    () => release.parseArgs([
      "--action", "status",
      "--state-dir", "relative",
    ]),
    /state-dir debe ser absoluto/
  );
  assert.throws(
    () => release.parseArgs([
      "--action", "prepare",
      "--state-dir", "/data/release",
      "--sender-profile", "unknown",
    ]),
    /legacy o hardened/
  );
});

test("perfil hardened exige exactamente sender.js y sender.html candidatos", () => {
  const intervalModule = path.resolve(
    __dirname,
    "../../../nodered-custom-nodes.audit/" +
    "node-red-contrib-services-mentor/services/interval-timer.js"
  );
  const senderModule = path.resolve(
    __dirname,
    "custom-nodes/node-red-contrib-services-mentor/services/sender.js"
  );
  const senderHtml = path.resolve(
    __dirname,
    "custom-nodes/node-red-contrib-services-mentor/services/sender.html"
  );
  const result = release.validateModules({
    senderProfile: "hardened",
    senderModule,
    senderHtml,
    intervalModule,
  });
  assert.equal(
    result.senderSha256,
    release.EXPECTED.hardenedSenderSha256
  );
  assert.equal(
    result.senderHtmlSha256,
    release.EXPECTED.hardenedSenderHtmlSha256
  );
});

test("prepare/apply/status/rollback preservan credenciales y rev", async () => {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "mentor-sender-release-")
  );
  const stateDir = path.join(directory, "state");
  const diskFlows = path.join(directory, "flows.json");
  const credentials = path.join(directory, "flows_cred.json");
  const senderModule = path.join(directory, "sender.js");
  const senderHtml = path.join(directory, "sender.html");
  const intervalModule = path.join(directory, "interval.js");
  const baseline = disabledCandidate();
  const encrypted = {"$": "encrypted-credential-blob"};

  fs.writeFileSync(diskFlows, JSON.stringify(baseline), "utf8");
  fs.writeFileSync(credentials, JSON.stringify(encrypted), "utf8");
  fs.copyFileSync(
    path.resolve(
      __dirname,
      "../../../nodered-custom-nodes.audit/" +
      "node-red-contrib-services-mentor/services/sender.js"
    ),
    senderModule
  );
  fs.copyFileSync(
    path.resolve(
      __dirname,
      "../../../nodered-custom-nodes.audit/" +
      "node-red-contrib-services-mentor/services/sender.html"
    ),
    senderHtml
  );
  fs.copyFileSync(
    path.resolve(
      __dirname,
      "../../../nodered-custom-nodes.audit/" +
      "node-red-contrib-services-mentor/services/interval-timer.js"
    ),
    intervalModule
  );

  let revision = "rev-1";
  let runtimeFlows = structuredClone(baseline);
  let posts = 0;
  let dropFirstResponse = true;
  const server = http.createServer((request, response) => {
    assert.equal(request.headers["node-red-api-version"], "v2");
    if (request.method === "GET" && request.url === "/flows") {
      response.writeHead(200, {"Content-Type": "application/json"});
      response.end(JSON.stringify({rev: revision, flows: runtimeFlows}));
      return;
    }
    if (request.method === "POST" && request.url === "/flows") {
      assert.equal(request.headers["node-red-deployment-type"], "nodes");
      let raw = "";
      request.setEncoding("utf8");
      request.on("data", (chunk) => { raw += chunk; });
      request.on("end", () => {
        const body = JSON.parse(raw);
        assert.equal(body.rev, revision);
        assert.deepEqual(body.credentials, encrypted);
        posts += 1;
        runtimeFlows = body.flows;
        revision = `rev-${posts + 1}`;
        fs.writeFileSync(diskFlows, JSON.stringify(runtimeFlows), "utf8");
        if (dropFirstResponse) {
          dropFirstResponse = false;
          request.socket.destroy();
          return;
        }
        response.writeHead(200, {"Content-Type": "application/json"});
        response.end(JSON.stringify({rev: revision}));
      });
      return;
    }
    response.writeHead(404);
    response.end();
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const apiUrl = `http://127.0.0.1:${server.address().port}`;
  const common = [
    "--state-dir", stateDir,
    "--api-url", apiUrl,
    "--disk-flows", diskFlows,
    "--credentials", credentials,
    "--sender-module", senderModule,
    "--sender-html", senderHtml,
    "--sender-profile", "legacy",
    "--interval-module", intervalModule,
  ];
  const io = {log() {}};

  try {
    const prepared = await release.main([
      "--action", "prepare",
      ...common,
    ], io);
    assert.equal(prepared.currentMode, "disabled");
    assert.equal(posts, 0);

    const canary = await release.main([
      "--action", "apply",
      "--mode", "canary",
      ...common,
    ], io);
    assert.equal(canary.currentMode, "canary");
    assert.equal(canary.reconciled, true);
    assert.equal(posts, 1);

    await assert.rejects(
      () => release.main([
        "--action", "apply",
        "--mode", "drain",
        ...common,
      ], io),
      /requiere --confirm-canary/
    );
    assert.equal(posts, 1);

    const drain = await release.main([
      "--action", "apply",
      "--mode", "drain",
      "--confirm-canary",
      ...common,
    ], io);
    assert.equal(drain.currentMode, "drain");
    assert.equal(posts, 2);

    const steady = await release.main([
      "--action", "apply",
      "--mode", "steady",
      ...common,
    ], io);
    assert.equal(steady.currentMode, "steady");
    assert.equal(posts, 3);

    const active = await release.main([
      "--action", "status",
      ...common,
    ], io);
    assert.equal(active.runtimeMode, "steady");
    assert.equal(active.diskMatchesRuntime, true);
    assert.equal(active.credentialBlobMatches, true);

    fs.writeFileSync(
      credentials,
      JSON.stringify({"$": "changed"}),
      "utf8"
    );
    await assert.rejects(
      () => release.main([
        "--action", "apply",
        "--mode", "disabled",
        ...common,
      ], io),
      /credenciales activas cambiaron/
    );
    assert.equal(posts, 3);
    fs.writeFileSync(credentials, JSON.stringify(encrypted), "utf8");

    const safe = await release.main([
      "--action", "apply",
      "--mode", "disabled",
      ...common,
    ], io);
    assert.equal(safe.currentMode, "disabled");
    assert.equal(posts, 4);
    assert.equal(
      release.flowHash(runtimeFlows),
      release.EXPECTED.disabledFlowHash
    );

    fs.appendFileSync(senderModule, "\n// tampered\n");
    await assert.rejects(
      () => release.main([
        "--action", "status",
        ...common,
      ], io),
      /Sender instalado no coincide con perfil legacy/
    );
  } finally {
    await new Promise((resolve) => server.close(resolve));
    fs.rmSync(directory, {recursive: true, force: true});
  }
});
