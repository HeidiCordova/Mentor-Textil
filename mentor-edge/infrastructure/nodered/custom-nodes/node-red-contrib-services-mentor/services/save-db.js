'use strict';

// mentor-textile-window:v3 — hardened, immutable five-minute delta persistence.

const SQL = Object.freeze({
  insertReading:
    'INSERT INTO mqtt_lecturas (mentor_id, time, device, content, result) VALUES (?, ?, ?, ?, ?)',
  selectReading:
    'SELECT mentor_id, content, result FROM mqtt_lecturas WHERE device = ? AND time = ? LIMIT 1',
  selectSnapshot:
    'SELECT content FROM mqtt_snapshot WHERE device = ? LIMIT 1',
  upsertSnapshot: [
    'INSERT INTO mqtt_snapshot (mentor_id, time, device, content, result)',
    'VALUES (?, ?, ?, ?, ?)',
    'ON DUPLICATE KEY UPDATE',
    'mentor_id = VALUES(mentor_id),',
    'time = VALUES(time),',
    'content = VALUES(content),',
    'result = VALUES(result)'
  ].join(' ')
});

class SaveDbError extends Error {
  constructor(code, message, cause) {
    super(message);
    this.name = 'SaveDbError';
    this.code = code;
    if (cause) {
      Object.defineProperty(this, 'cause', {
        configurable: true,
        enumerable: false,
        value: cause
      });
    }
  }
}

const sharedConnectionLocks = new WeakMap();

function error(code, message, cause) {
  return new SaveDbError(code, message, cause);
}

function isPlainObject(value) {
  if (value === null || typeof value !== 'object') {
    return false;
  }
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function normalizeJsonValue(value, seen) {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') {
    return value;
  }

  if (typeof value === 'number') {
    if (!Number.isFinite(value)) {
      throw error('SAVE_DB_INVALID_JSON_VALUE', 'Payload contains a non-finite number');
    }
    return value;
  }

  if (typeof value !== 'object') {
    throw error('SAVE_DB_INVALID_JSON_VALUE', 'Payload contains a value that JSON cannot represent');
  }

  if (seen.has(value)) {
    throw error('SAVE_DB_INVALID_JSON_VALUE', 'Payload contains a circular reference');
  }

  seen.add(value);
  let normalized;

  if (Array.isArray(value)) {
    normalized = value.map((item) => normalizeJsonValue(item, seen));
  } else {
    if (!isPlainObject(value)) {
      seen.delete(value);
      throw error('SAVE_DB_INVALID_JSON_VALUE', 'Payload must contain only plain JSON objects');
    }

    normalized = {};
    for (const key of Object.keys(value).sort()) {
      normalized[key] = normalizeJsonValue(value[key], seen);
    }
  }

  seen.delete(value);
  return normalized;
}

function stableStringify(value) {
  return JSON.stringify(normalizeJsonValue(value, new Set()));
}

function parsePayload(payload) {
  let parsed;

  if (typeof payload === 'string') {
    try {
      parsed = JSON.parse(payload);
    } catch (cause) {
      throw error('SAVE_DB_INVALID_JSON', 'msg.payload is not valid JSON', cause);
    }
  } else if (isPlainObject(payload)) {
    parsed = payload;
  } else {
    throw error('SAVE_DB_INVALID_JSON', 'msg.payload must be a JSON object or its string representation');
  }

  if (!isPlainObject(parsed)) {
    throw error('SAVE_DB_INVALID_PAYLOAD', 'msg.payload must decode to a JSON object');
  }

  return normalizeJsonValue(parsed, new Set());
}

function normalizeTime(value) {
  if (typeof value === 'string' && !/^\d+$/.test(value)) {
    throw error('SAVE_DB_INVALID_TIME', 'Payload field "time" must be an epoch-millisecond integer');
  }

  const normalized = Number(value);
  if (!Number.isSafeInteger(normalized) || normalized < 0) {
    throw error('SAVE_DB_INVALID_TIME', 'Payload field "time" must be a safe epoch-millisecond integer');
  }
  return normalized;
}

function validateEnvelope(payload, strictDelta) {
  if (typeof payload.code !== 'string' || payload.code.trim() === '') {
    throw error('SAVE_DB_INVALID_CODE', 'Payload field "code" must be a non-empty string');
  }

  const code = payload.code.trim();
  if (code.length > 50) {
    throw error('SAVE_DB_INVALID_CODE', 'Payload field "code" exceeds mqtt_lecturas.device VARCHAR(50)');
  }

  payload.code = code;
  payload.time = normalizeTime(payload.time);

  if (!strictDelta) {
    return payload;
  }

  if (!Object.prototype.hasOwnProperty.call(payload, 'data')) {
    throw error('SAVE_DB_INVALID_DATA', 'strictDelta payload requires a "data" field');
  }

  const dataIsArray = Array.isArray(payload.data);
  const dataIsObject = isPlainObject(payload.data);
  if (!dataIsArray && !dataIsObject) {
    throw error('SAVE_DB_INVALID_DATA', 'strictDelta field "data" must be an array or object');
  }

  if (
    (dataIsArray && payload.data.length === 0) ||
    (dataIsObject && Object.keys(payload.data).length === 0)
  ) {
    throw error('SAVE_DB_INVALID_DATA', 'strictDelta field "data" must not be empty');
  }

  if (dataIsArray) {
    if (!Array.isArray(payload.head) || payload.head.length !== payload.data.length) {
      throw error(
        'SAVE_DB_INVALID_HEAD',
        'strictDelta array payload requires a same-length "head" array'
      );
    }
  }

  if (
    Object.prototype.hasOwnProperty.call(payload, 'head') &&
    (!Array.isArray(payload.head) ||
      payload.head.some((item) => typeof item !== 'string' || item.trim() === ''))
  ) {
    throw error('SAVE_DB_INVALID_HEAD', 'Payload field "head" must contain non-empty strings');
  }

  return payload;
}

function containsInvalidReading(value) {
  if (typeof value === 'number') {
    return !Number.isFinite(value);
  }
  if (typeof value === 'string') {
    return /timeout/i.test(value) || /(^|[^a-z])nan([^a-z]|$)/i.test(value);
  }
  if (Array.isArray(value)) {
    return value.some(containsInvalidReading);
  }
  if (isPlainObject(value)) {
    return Object.values(value).some(containsInvalidReading);
  }
  return false;
}

function normalizeMentorId(mentor) {
  const raw = mentor && mentor.mentor_id !== undefined ? mentor.mentor_id : 0;
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < 0) {
    throw error('SAVE_DB_INVALID_MENTOR_ID', 'Configured mentor_id must be a non-negative integer');
  }
  return value;
}

function parseBoolean(value) {
  return value === true || value === 1 || value === '1' || value === 'true';
}

function parseIntervalMs(value) {
  const minutes = Number(value);
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return 0;
  }
  return minutes * 60 * 1000;
}

function query(connection, sql, params) {
  return new Promise((resolve, reject) => {
    connection.query(sql, params, (queryError, rows) => {
      if (queryError) {
        reject(queryError);
        return;
      }
      resolve(rows);
    });
  });
}

function connectionCall(connection, method) {
  return new Promise((resolve, reject) => {
    if (!connection || typeof connection[method] !== 'function') {
      reject(error('SAVE_DB_TRANSACTION_UNSUPPORTED', `Database connection lacks ${method}()`));
      return;
    }

    connection[method]((connectionError) => {
      if (connectionError) {
        reject(connectionError);
        return;
      }
      resolve();
    });
  });
}

function acquirePoolConnection(pool) {
  return new Promise((resolve, reject) => {
    try {
      pool.getConnection((connectionError, connection) => {
        if (connectionError) {
          reject(
            error(
              'SAVE_DB_CONNECTION_ACQUIRE_FAILED',
              'Could not acquire a dedicated database connection',
              connectionError
            )
          );
          return;
        }

        if (!connection || typeof connection.query !== 'function') {
          if (connection) {
            try {
              if (typeof connection.release === 'function') {
                connection.release();
              } else if (typeof pool.releaseConnection === 'function') {
                pool.releaseConnection(connection);
              }
            } catch (_) {
              // The acquisition error below is the actionable failure.
            }
          }
          reject(
            error(
              'SAVE_DB_CONNECTION_ACQUIRE_FAILED',
              'Database pool returned an invalid connection'
            )
          );
          return;
        }

        resolve(connection);
      });
    } catch (cause) {
      reject(
        error(
          'SAVE_DB_CONNECTION_ACQUIRE_FAILED',
          'Could not acquire a dedicated database connection',
          cause
        )
      );
    }
  });
}

function acquireSharedConnectionLock(databaseConfig) {
  let state = sharedConnectionLocks.get(databaseConfig);
  if (!state) {
    state = { locked: false, waiters: [] };
    sharedConnectionLocks.set(databaseConfig, state);
  }

  return new Promise((resolve) => {
    const grant = () => {
      state.locked = true;
      let released = false;

      resolve(() => {
        if (released) {
          return;
        }
        released = true;

        const next = state.waiters.shift();
        if (next) {
          next();
        } else {
          state.locked = false;
        }
      });
    };

    if (state.locked) {
      state.waiters.push(grant);
    } else {
      grant();
    }
  });
}

async function acquireDatabaseConnection(databaseConfig) {
  const pool = databaseConfig && databaseConfig.pool;

  if (pool && typeof pool.getConnection === 'function') {
    // A transaction must own its connection. Never fall back to the config
    // node's long-lived shared connection after a pool acquisition error,
    // because another Node-RED node may be using it concurrently.
    const connection = await acquirePoolConnection(pool);
    let released = false;

    return {
      connection,
      dedicated: true,
      release() {
        if (released) {
          return;
        }
        released = true;

        if (typeof connection.release === 'function') {
          connection.release();
          return;
        }
        if (typeof pool.releaseConnection === 'function') {
          pool.releaseConnection(connection);
          return;
        }

        throw error(
          'SAVE_DB_CONNECTION_RELEASE_UNSUPPORTED',
          'Dedicated database connection cannot be returned to its pool'
        );
      }
    };
  }

  // Compatibility path for tests and older MySQLdb config nodes that expose
  // only one connection. It remains safe for those installations because this
  // path is used only when no pool API exists.
  const connection = databaseConfig && databaseConfig.connection;
  if (!connection || typeof connection.query !== 'function') {
    throw error('SAVE_DB_NOT_CONNECTED', 'Database not connected');
  }

  const releaseLock = await acquireSharedConnectionLock(databaseConfig);
  return {
    connection,
    dedicated: false,
    release: releaseLock
  };
}

async function withTransaction(connection, operation) {
  await connectionCall(connection, 'beginTransaction');
  let open = true;

  try {
    const result = await operation();
    await connectionCall(connection, 'commit');
    open = false;
    return result;
  } catch (operationError) {
    if (open) {
      try {
        await connectionCall(connection, 'rollback');
      } catch (_) {
        // Keep the original failure. A retry is safe because mqtt_lecturas is immutable by key.
      }
    }
    throw operationError;
  }
}

function isDuplicateError(databaseError) {
  return Boolean(
    databaseError &&
      (databaseError.code === 'ER_DUP_ENTRY' ||
        databaseError.errno === 1062 ||
        databaseError.number === 1062)
  );
}

function contentsEqual(left, right) {
  const leftString = String(left);
  const rightString = String(right);
  if (leftString === rightString) {
    return true;
  }

  try {
    return stableStringify(JSON.parse(leftString)) === stableStringify(JSON.parse(rightString));
  } catch (_) {
    return false;
  }
}

function integerFieldEquals(actual, expected) {
  if (
    (typeof actual !== 'number' && typeof actual !== 'string') ||
    (typeof actual === 'string' && !/^\d+$/.test(actual))
  ) {
    return false;
  }

  const normalized = Number(actual);
  return Number.isSafeInteger(normalized) && normalized === expected;
}

function readingValues(record) {
  // The deployed schema stores time as VARCHAR(20); bind a string so the
  // UNIQUE(device,time) lookup does not require numeric coercion.
  return [record.mentorId, String(record.time), record.code, record.content, record.result];
}

async function upsertSnapshot(connection, record) {
  await query(connection, SQL.upsertSnapshot, readingValues(record));
}

async function persistStrictDelta(connection, record) {
  return withTransaction(connection, async () => {
    let duplicate = false;

    try {
      await query(connection, SQL.insertReading, readingValues(record));
    } catch (databaseError) {
      if (!isDuplicateError(databaseError)) {
        throw databaseError;
      }

      const rows = await query(connection, SQL.selectReading, [
        record.code,
        String(record.time)
      ]);
      if (!Array.isArray(rows) || rows.length === 0) {
        throw error(
          'SAVE_DB_DUPLICATE_NOT_FOUND',
          'Database reported a duplicate window but its immutable row was not found',
          databaseError
        );
      }

      const existing = rows[0];
      if (
        !contentsEqual(existing.content, record.content) ||
        !integerFieldEquals(existing.mentor_id, record.mentorId) ||
        !integerFieldEquals(existing.result, record.result)
      ) {
        throw error(
          'SAVE_DB_IDEMPOTENCY_CONFLICT',
          'The same device/time window already exists with different immutable fields'
        );
      }
      duplicate = true;
    }

    // This never updates mqtt_lecturas, so sender ACK/retry columns remain untouched.
    await upsertSnapshot(connection, record);
    return { content: record.content, duplicate };
  });
}

async function persistLegacy(connection, record, intervalMs, invalidReading) {
  if (invalidReading) {
    return withTransaction(connection, async () => {
      const rows = await query(connection, SQL.selectSnapshot, [record.code]);
      if (!Array.isArray(rows) || rows.length === 0) {
        throw error(
          'SAVE_DB_SNAPSHOT_NOT_FOUND',
          'No previous snapshot exists for the legacy Timeout/NaN fallback'
        );
      }

      let previous;
      try {
        previous = JSON.parse(String(rows[0].content));
      } catch (cause) {
        throw error('SAVE_DB_INVALID_SNAPSHOT', 'Previous snapshot is not valid JSON', cause);
      }
      if (!isPlainObject(previous)) {
        throw error('SAVE_DB_INVALID_SNAPSHOT', 'Previous snapshot is not a JSON object');
      }

      previous.time = record.time;
      const fallbackRecord = {
        ...record,
        content: JSON.stringify(previous)
      };
      await query(connection, SQL.insertReading, readingValues(fallbackRecord));
      return { content: fallbackRecord.content, duplicate: false };
    });
  }

  return withTransaction(connection, async () => {
    if (intervalMs > 0 && record.time % intervalMs === 0) {
      await query(connection, SQL.insertReading, readingValues(record));
    }
    await upsertSnapshot(connection, record);
    return { content: record.content, duplicate: false };
  });
}

function asPersistenceError(cause) {
  if (cause instanceof SaveDbError) {
    return cause;
  }
  const databaseCode = cause && (cause.code || cause.errno || cause.number);
  const suffix = databaseCode ? ` (${String(databaseCode)})` : '';
  return error(
    'SAVE_DB_PERSISTENCE_FAILED',
    `Database persistence failed${suffix}`,
    cause
  );
}

function buildRecord(msg, mentor, strictDelta) {
  const payload = validateEnvelope(parsePayload(msg.payload), strictDelta);
  const invalidReading = containsInvalidReading(strictDelta ? payload.data : payload);

  if (strictDelta && invalidReading) {
    throw error(
      'SAVE_DB_INVALID_DELTA_READING',
      'strictDelta rejects Timeout/NaN readings instead of copying a previous snapshot'
    );
  }

  return {
    record: {
      mentorId: normalizeMentorId(mentor),
      time: payload.time,
      code: payload.code,
      content:
        strictDelta || typeof msg.payload !== 'string'
          ? stableStringify(payload)
          : msg.payload,
      result: 1
    },
    invalidReading
  };
}

function registerSaveDbNode(RED) {
  function SaveDbNode(config) {
    RED.nodes.createNode(this, config);

    const node = this;
    node.mydb = config.mysql;
    node.mydbConfig = RED.nodes.getNode(node.mydb);
    node.mentor = RED.nodes.getNode(config.mentor);
    node.strictDelta = parseBoolean(config.strictDelta);
    node.intervalMs = parseIntervalMs(config.interval);

    if (!node.mydbConfig) {
      node.error('MySQL database not configured');
      return;
    }

    const stateHandler = (info) => {
      if (info === 'connecting') {
        node.status({ fill: 'grey', shape: 'ring', text: info });
      } else if (info === 'connected') {
        node.status({ fill: 'green', shape: 'dot', text: info });
      } else {
        const text =
          info === 'ECONNREFUSED'
            ? 'connection refused'
            : info === 'PROTOCOL_CONNECTION_LOST'
              ? 'connection lost'
              : String(info);
        node.status({ fill: 'red', shape: 'ring', text });
      }
    };

    if (typeof node.mydbConfig.on === 'function') {
      node.mydbConfig.on('state', stateHandler);
    }
    if (typeof node.mydbConfig.connect === 'function') {
      node.mydbConfig.connect();
    }

    node.on('input', async (msg, send, done) => {
      const emit = typeof send === 'function' ? send : node.send.bind(node);

      try {
        const hasPool =
          node.mydbConfig.pool &&
          typeof node.mydbConfig.pool.getConnection === 'function';
        if (
          !node.mydbConfig.connected ||
          (!hasPool && !node.mydbConfig.connection)
        ) {
          throw error('SAVE_DB_NOT_CONNECTED', 'Database not connected');
        }

        const { record, invalidReading } = buildRecord(msg, node.mentor, node.strictDelta);
        const lease = await acquireDatabaseConnection(node.mydbConfig);
        let persisted;

        try {
          persisted = node.strictDelta
            ? await persistStrictDelta(lease.connection, record)
            : await persistLegacy(
                lease.connection,
                record,
                node.intervalMs,
                invalidReading
              );
        } finally {
          try {
            lease.release();
          } catch (releaseError) {
            // COMMIT/ROLLBACK has already decided durability. Do not mask that
            // result or trigger a duplicate replay solely because release()
            // failed, but make the pool problem visible in Node-RED.
            if (typeof node.warn === 'function') {
              node.warn(asPersistenceError(releaseError));
            }
          }
        }

        msg.payload = persisted.content;
        msg.code = record.code;
        msg.time = record.time;
        msg.mentor_id = record.mentorId;
        msg.result = record.result;

        node.status({
          fill: 'green',
          shape: 'dot',
          text: persisted.duplicate ? 'idempotent duplicate' : 'persisted'
        });
        emit(msg);
        if (typeof done === 'function') {
          done();
        }
      } catch (cause) {
        const persistenceError = asPersistenceError(cause);
        node.status({ fill: 'red', shape: 'ring', text: persistenceError.code });
        if (typeof done === 'function') {
          done(persistenceError);
        } else {
          node.error(persistenceError, msg);
        }
      }
    });

    node.on('close', (removed, done) => {
      const finish = typeof removed === 'function' ? removed : done;
      if (
        typeof node.mydbConfig.removeListener === 'function'
      ) {
        node.mydbConfig.removeListener('state', stateHandler);
      }
      node.status({});
      if (typeof finish === 'function') {
        finish();
      }
    });
  }

  RED.nodes.registerType('SaveDB', SaveDbNode);
}

module.exports = registerSaveDbNode;
module.exports._test = Object.freeze({
  SQL,
  SaveDbError,
  acquireDatabaseConnection,
  buildRecord,
  contentsEqual,
  containsInvalidReading,
  integerFieldEquals,
  parseBoolean,
  persistLegacy,
  persistStrictDelta,
  stableStringify,
  validateEnvelope
});
