"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const patch = require("./patch-official-textile-count.js");
const {
  canonicalFlows,
  diffFlows,
  flowHash,
  main,
  parseArgs,
  prepareCandidate,
} = require("./hot-deploy-official-counter.js");

function auditFlows() {
  const auditPath = process.env.NODERED_AUDIT_PATH || path.resolve(
    __dirname,
    "../../../nodered-flows.audit.json"
  );
  assert.equal(fs.existsSync(auditPath), true, auditPath);
  return JSON.parse(fs.readFileSync(auditPath, "utf8"));
}

test("canonicalFlows ignora orden de nodos y conserva orden interno", () => {
  const first = [
    {id: "b", type: "x", wires: [["2", "1"]]},
    {id: "a", type: "x", value: {z: 2, a: 1}},
  ];
  const reordered = [
    {value: {a: 1, z: 2}, type: "x", id: "a"},
    {wires: [["2", "1"]], type: "x", id: "b"},
  ];
  const changedWire = [
    reordered[0],
    {wires: [["1", "2"]], type: "x", id: "b"},
  ];
  assert.equal(canonicalFlows(first), canonicalFlows(reordered));
  assert.equal(flowHash(first), flowHash(reordered));
  assert.notEqual(flowHash(first), flowHash(changedWire));
});

test("diffFlows informa altas, bajas y modificaciones exactas", () => {
  const before = [
    {id: "same", type: "x", value: 1},
    {id: "change", type: "x", value: 1},
    {id: "remove", type: "x"},
  ];
  const after = [
    {id: "same", type: "x", value: 1},
    {id: "change", type: "x", value: 2},
    {id: "add", type: "x"},
  ];
  assert.deepEqual(diffFlows(before, after), {
    added: ["add"],
    removed: ["remove"],
    modified: ["change"],
  });
});

test("CLI exige accion, state-dir y parche solo cuando corresponde", () => {
  const prepare = parseArgs([
    "--action=prepare",
    "--state-dir=/data/test",
    "--patch=/tmp/patch.js",
    "--linea-id=2",
  ]);
  assert.equal(prepare.action, "prepare");
  assert.equal(prepare.lineaId, 2);
  assert.ok(path.isAbsolute(prepare.stateDir));
  assert.ok(path.isAbsolute(prepare.patch));

  const status = parseArgs([
    "--action",
    "status",
    "--state-dir",
    "/data/test",
  ]);
  assert.equal(status.action, "status");
  assert.equal(status.patch, "");

  assert.throws(
    () => parseArgs(["--action=prepare", "--state-dir=/data/test"]),
    /--patch es obligatorio/
  );
  assert.throws(
    () => parseArgs(["--action=force", "--state-dir=/data/test"]),
    /prepare, deploy, status o rollback/
  );
});

test("candidato modifica cinco nodos, agrega diez y aisla el Sender", () => {
  const before = auditFlows();
  const beforeSaveDb = JSON.stringify(
    before.find((node) => node.id === patch.AUDIT.saveDb)
  );
  const candidate = prepareCandidate(before, patch, 1);

  assert.equal(candidate.saveDbMode, patch.SAVE_DB_MODE_PRESERVE);
  assert.equal(candidate.diff.added.length, 10);
  assert.deepEqual(candidate.diff.removed, []);
  assert.deepEqual(candidate.diff.modified, [
    "570d133c1babc579",
    "70d7cc1c095e63b5",
    "9d1c621a18c65202",
    "b94f3dfa216658f3",
    "e776b3254b93967f",
  ]);
  assert.equal(
    JSON.stringify(
      candidate.flows.find((node) => node.id === patch.AUDIT.saveDb)
    ),
    beforeSaveDb
  );
  assert.equal(
    candidate.flows.find(
      (node) => node.id === "b94f3dfa216658f3"
    ).d,
    true
  );
  assert.equal(
    candidate.changes.some((change) => change.includes("SaveDB")),
    false
  );
});

test("prepare, deploy y rollback preservan rev y credenciales cifradas", async () => {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "mentor-hot-deploy-")
  );
  const stateDir = path.join(directory, "state");
  const diskFlows = path.join(directory, "flows.json");
  const credentials = path.join(directory, "flows_cred.json");
  const settings = path.join(directory, "settings.js");
  const patchPath = path.resolve(
    __dirname,
    "patch-official-textile-count.js"
  );
  const before = auditFlows();
  const encrypted = {"$": "encrypted-credential-blob"};
  fs.writeFileSync(diskFlows, JSON.stringify(before), "utf8");
  fs.writeFileSync(credentials, JSON.stringify(encrypted), "utf8");
  fs.writeFileSync(settings, "module.exports={};\n", "utf8");

  let revision = "revision-1";
  let runtimeFlows = JSON.parse(JSON.stringify(before));
  const posts = [];
  let dropNextResponse = true;
  const server = http.createServer((request, response) => {
    assert.equal(
      request.headers["node-red-api-version"],
      "v2"
    );
    if (request.method === "GET" && request.url === "/flows") {
      response.writeHead(200, {"Content-Type": "application/json"});
      response.end(JSON.stringify({rev: revision, flows: runtimeFlows}));
      return;
    }
    if (request.method === "POST" && request.url === "/flows") {
      assert.equal(
        request.headers["node-red-deployment-type"],
        "nodes"
      );
      let raw = "";
      request.setEncoding("utf8");
      request.on("data", (chunk) => { raw += chunk; });
      request.on("end", () => {
        const body = JSON.parse(raw);
        posts.push(body);
        if (body.rev !== revision) {
          response.writeHead(409, {"Content-Type": "application/json"});
          response.end(JSON.stringify({message: "version mismatch"}));
          return;
        }
        assert.deepEqual(body.credentials, encrypted);
        runtimeFlows = body.flows;
        fs.writeFileSync(diskFlows, JSON.stringify(runtimeFlows), "utf8");
        revision = `revision-${posts.length + 1}`;
        if (dropNextResponse) {
          dropNextResponse = false;
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

  await new Promise((resolve) => {
    server.listen(0, "127.0.0.1", resolve);
  });
  const address = server.address();
  const apiUrl = `http://127.0.0.1:${address.port}`;
  const io = {log() {}};
  const common = [
    "--state-dir", stateDir,
    "--api-url", apiUrl,
    "--credentials", credentials,
    "--disk-flows", diskFlows,
    "--settings", settings,
  ];

  try {
    const prepared = await main([
      "--action", "prepare",
      "--patch", patchPath,
      ...common,
    ], io);
    assert.equal(prepared.status, "prepared");
    assert.equal(posts.length, 0);
    assert.equal(
      flowHash(runtimeFlows),
      prepared.beforeFlowHash
    );

    const deployed = await main([
      "--action", "deploy",
      "--patch", patchPath,
      ...common,
    ], io);
    assert.equal(deployed.status, "deployed");
    assert.equal(deployed.reconciled, true);
    assert.equal(posts.length, 1);
    assert.equal(
      flowHash(runtimeFlows),
      deployed.candidateFlowHash
    );

    const active = await main([
      "--action", "status",
      ...common,
    ], io);
    assert.equal(active.matches, "candidate");

    fs.writeFileSync(
      credentials,
      JSON.stringify({"$": "new-encrypted-credentials"}),
      "utf8"
    );
    await assert.rejects(
      () => main([
        "--action", "rollback",
        ...common,
      ], io),
      /credenciales activas cambiaron/
    );
    assert.equal(posts.length, 1);
    fs.writeFileSync(credentials, JSON.stringify(encrypted), "utf8");

    const rolledBack = await main([
      "--action", "rollback",
      ...common,
    ], io);
    assert.equal(rolledBack.status, "rolled_back");
    assert.equal(posts.length, 2);
    assert.equal(flowHash(runtimeFlows), prepared.rollbackFlowHash);
    assert.equal(
      runtimeFlows.find(
        (node) => node.id === "b94f3dfa216658f3"
      ).d,
      true
    );
    assert.deepEqual(
      JSON.parse(fs.readFileSync(credentials, "utf8")),
      encrypted
    );
  } finally {
    await new Promise((resolve) => server.close(resolve));
    fs.rmSync(directory, {recursive: true, force: true});
  }
});
