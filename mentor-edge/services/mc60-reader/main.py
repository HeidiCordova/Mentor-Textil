import json
import os
import re
import struct
import logging
import threading
import time
from datetime import datetime, timezone, timedelta
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import serial
import psycopg2
import psycopg2.extras
from pymodbus.client import ModbusSerialClient

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger(__name__)

# Register map: (column_alias, start_address, register_count, encoding)
# Source: MC60 Modbus Register List (manual MC60-EN.pdf)
# Encodings: float32be = IEEE 754 big-endian float; int64be = signed Int64 big-endian
REGISTER_MAP = [
    # Currents (A)
    ("Ia",     1000, 2, "float32be"),
    ("Ib",     1002, 2, "float32be"),
    ("Ic",     1004, 2, "float32be"),
    ("Iavg",   1006, 2, "float32be"),
    ("In",     1008, 2, "float32be"),
    # Phase voltages (V)
    ("Va",     1010, 2, "float32be"),
    ("Vb",     1012, 2, "float32be"),
    ("Vc",     1014, 2, "float32be"),
    ("Vavg",   1016, 2, "float32be"),
    ("V0",     1018, 2, "float32be"),  # zero sequence voltage
    # Line voltages (V)
    ("Vab",    1020, 2, "float32be"),
    ("Vbc",    1022, 2, "float32be"),
    ("Vca",    1024, 2, "float32be"),
    ("VLavg",  1026, 2, "float32be"),
    # Active power per phase + total (kW)
    ("Pa",     1028, 2, "float32be"),
    ("Pb",     1030, 2, "float32be"),
    ("Pc",     1032, 2, "float32be"),
    ("P",      1034, 2, "float32be"),
    # Reactive power per phase + total (kVAR)
    ("Qa",     1036, 2, "float32be"),
    ("Qb",     1038, 2, "float32be"),
    ("Qc",     1040, 2, "float32be"),
    ("Q",      1042, 2, "float32be"),
    # Apparent power per phase + total (kVA)
    ("Sa",     1044, 2, "float32be"),
    ("Sb",     1046, 2, "float32be"),
    ("Sc",     1048, 2, "float32be"),
    ("S",      1050, 2, "float32be"),
    # Power factor per phase + total
    ("PFa",    1052, 2, "float32be"),
    ("PFb",    1054, 2, "float32be"),
    ("PFc",    1056, 2, "float32be"),
    ("PF",     1058, 2, "float32be"),
    # Fundamental power factor per phase + total (DPF)
    ("DPFa",   1060, 2, "float32be"),
    ("DPFb",   1062, 2, "float32be"),
    ("DPFc",   1064, 2, "float32be"),
    ("DPF",    1066, 2, "float32be"),
    # Frequency per phase + total (Hz)
    ("FreqA",  1068, 2, "float32be"),
    ("FreqB",  1070, 2, "float32be"),
    ("FreqC",  1072, 2, "float32be"),
    ("Freq",   1074, 2, "float32be"),
    # Electric energy — Int64 (signed, Wh / VARh / VAh)
    # Active energy — positive (import)
    ("EPAImp",  2500, 4, "int64be"),
    ("EPBImp",  2504, 4, "int64be"),
    ("EPCImp",  2508, 4, "int64be"),
    ("EPImp",   2512, 4, "int64be"),
    # Active energy — negative (export)
    ("EPAExp",  2516, 4, "int64be"),
    ("EPBExp",  2520, 4, "int64be"),
    ("EPCExp",  2524, 4, "int64be"),
    ("EPExp",   2528, 4, "int64be"),
    # Reactive energy — positive (import)
    ("EQAImp",  2532, 4, "int64be"),
    ("EQBImp",  2536, 4, "int64be"),
    ("EQCImp",  2540, 4, "int64be"),
    ("EQImp",   2544, 4, "int64be"),
    # Reactive energy — negative (export)
    ("EQAExp",  2548, 4, "int64be"),
    ("EQBExp",  2552, 4, "int64be"),
    ("EQCExp",  2556, 4, "int64be"),
    ("EQExp",   2560, 4, "int64be"),
    # Apparent energy per phase + total
    ("ESA",     2564, 4, "int64be"),
    ("ESB",     2568, 4, "int64be"),
    ("ESC",     2572, 4, "int64be"),
    ("ES",      2576, 4, "int64be"),
    # Current THD %
    ("THDia",  4000, 2, "float32be"),
    ("THDib",  4002, 2, "float32be"),
    ("THDic",  4004, 2, "float32be"),
    # Voltage THD %
    ("THDua",  5000, 2, "float32be"),
    ("THDub",  5002, 2, "float32be"),
    ("THDuc",  5004, 2, "float32be"),
]

HEAD = [r[0] for r in REGISTER_MAP]

# Indices into HEAD/data for registers that must be present on any live meter.
# If both are None the meter returned no valid RS-485 response at all.
_IDX_VA    = HEAD.index("Va")     # phase voltage — block 1000-1019
_IDX_EPIMP = HEAD.index("EPImp")  # total active energy — block 2500-2519

# How many extra attempts to make when a block read fails before giving up.
_BLOCK_RETRIES = 2

# Block read ranges: (start_address, register_count)
# Groups contiguous addresses into a single FC03 request.
# addr 1000-1075 → all measurements (currents, voltages, powers, PF, freq)
# addr 2500-2579 → all energy Int64 blocks (per-phase + totals, active/reactive/apparent)
# addr 4000-4005 → current THD (3 × Float32)
# addr 5000-5005 → voltage THD (3 × Float32)
_READ_BLOCKS = [
    (1000, 20),   # 1000-1019: Ia Ib Ic Iavg In Va Vb Vc Vavg V0
    (1020, 20),   # 1020-1039: Vab Vbc Vca VLavg Pa Pb Pc P Qa Qb
    (1040, 20),   # 1040-1059: Qc Q Sa Sb Sc S PFa PFb PFc PF
    (1060, 16),   # 1060-1075: DPFa DPFb DPFc DPF FreqA FreqB FreqC Freq
    (2500, 20),   # 2500-2519: EPAImp EPBImp EPCImp EPImp EPAExp
    (2520, 20),   # 2520-2539: EPBExp EPCExp EPExp EQAImp EQBImp
    (2540, 20),   # 2540-2559: EQCImp EQImp EQAExp EQBExp EQCExp
    (2560, 20),   # 2560-2579: EQExp ESA ESB ESC ES
    (4000,  6),   # THD current
    (5000,  6),   # THD voltage
]


def _parse_registers(registers: list, encoding: str) -> float:
    if encoding == "float32be":
        raw = struct.pack(">HH", registers[0], registers[1])
        return round(struct.unpack(">f", raw)[0], 2)
    if encoding == "int64be":
        raw = struct.pack(">HHHH", registers[0], registers[1], registers[2], registers[3])
        return float(struct.unpack(">q", raw)[0])  # signed Int64 per manual
    raise ValueError(f"Unknown encoding: {encoding}")


def _snapshot_timestamp() -> str:
    now = datetime.now(timezone.utc)
    rounded_min = round(now.minute / 5) * 5
    ts = now.replace(second=0, microsecond=0, minute=0) + timedelta(minutes=rounded_min)
    return ts.isoformat()


def _crc16(data: bytes) -> int:
    crc = 0xFFFF
    for b in data:
        crc ^= b
        for _ in range(8):
            crc = (crc >> 1) ^ 0xA001 if crc & 1 else crc >> 1
    return crc


def _parse_fc03(resp: bytes, slave: int, count: int):
    """Validate and decode a Modbus FC03 response frame."""
    resp_len = 5 + count * 2
    if len(resp) < resp_len:
        return None
    if resp[0] != slave or resp[1] != 0x03 or resp[2] != count * 2:
        return None
    crc_calc = _crc16(resp[:resp_len - 2])
    crc_got  = struct.unpack("<H", resp[resp_len - 2:resp_len])[0]
    if crc_calc != crc_got:
        return None
    n = resp[2]
    return list(struct.unpack(f">{n // 2}H", resp[3:3 + n]))


def _raw_read_regs(s: serial.Serial, slave: int, addr: int, count: int):
    """Read holding registers via raw serial, handling both echo and no-echo cases.

    The ch341 USB-RS485 adapter may or may not echo transmitted bytes on RX
    (half-duplex without hardware direction control).  Reading req_len+resp_len
    bytes in one shot and searching for the valid frame is robust to both modes.
    """
    req      = struct.pack(">BBHH", slave, 0x03, addr, count)
    req      += struct.pack("<H", _crc16(req))
    req_len  = len(req)       # always 8
    resp_len = 5 + count * 2

    s.reset_input_buffer()
    s.write(req)

    # Read up to echo + response in one shot.
    # At 9600 baud the full echo+response arrives within ~100ms.
    # The ch341 USB adapter may raise SerialException when readiness is signalled
    # but data hasn't landed in the kernel buffer yet; one retry after a brief
    # pause is sufficient for the USB pipe to settle.
    try:
        buf = s.read(req_len + resp_len)
    except serial.SerialException:
        time.sleep(0.1)
        try:
            buf = s.read(req_len + resp_len)
        except serial.SerialException:
            return None

    # Case 1: echo present — response starts after req_len bytes
    if len(buf) >= req_len + resp_len:
        result = _parse_fc03(buf[req_len:req_len + resp_len], slave, count)
        if result is not None:
            return result

    # Case 2: no echo — response starts at byte 0
    if len(buf) >= resp_len:
        return _parse_fc03(buf[:resp_len], slave, count)

    return None


def read_meter(s: serial.Serial, unit_id: int) -> list:
    blocks: dict = {}
    for start, count in _READ_BLOCKS:
        regs = None
        for attempt in range(_BLOCK_RETRIES + 1):
            try:
                regs = _raw_read_regs(s, unit_id, start, count)
            except serial.SerialException as exc:
                log.warning("Serial exception unit_id=%d start=%d attempt=%d: %s",
                            unit_id, start, attempt + 1, exc)
                regs = None
            if regs is not None:
                break
            if attempt < _BLOCK_RETRIES:
                time.sleep(0.15)
        if regs is None:
            log.warning("Block read error unit_id=%d start=%d count=%d (all %d attempts failed)",
                        unit_id, start, count, _BLOCK_RETRIES + 1)
        blocks[start] = regs
        time.sleep(0.05)  # RS-485 inter-request settling

    data = []
    for name, address, count, encoding in REGISTER_MAP:
        val = None
        for block_start, regs in blocks.items():
            if regs is None:
                continue
            offset = address - block_start
            if 0 <= offset and offset + count <= len(regs):
                try:
                    val = _parse_registers(regs[offset : offset + count], encoding)
                except Exception as exc:
                    log.warning("Parse error name=%s addr=%d: %s", name, address, exc)
                break
        if val is None:
            log.warning("Missing value unit_id=%d name=%s addr=%d", unit_id, name, address)
        data.append(val)
    return data


def _load_config(conn) -> dict:
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute("SELECT key, value FROM energy.config")
        result = {row["key"]: row["value"] for row in cur.fetchall()}
    conn.commit()
    return result


def _load_meters(conn) -> list:
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute(
            "SELECT meter_id, unit_id FROM energy.meters WHERE active = TRUE ORDER BY unit_id"
        )
        rows = cur.fetchall()

    # Create view for any meter that does not have one yet (e.g., inserted directly via DB).
    for row in rows:
        cur_views = conn.cursor()
        view = _safe_view_name(row["meter_id"])
        cur_views.execute(
            "SELECT 1 FROM information_schema.views "
            "WHERE table_schema = 'energy' AND table_name = %s",
            (view.split(".")[1],),
        )
        if cur_views.fetchone() is None:
            _ensure_meter_view(conn, row["meter_id"])
        cur_views.close()

    conn.commit()
    return rows


def _safe_view_name(meter_id: str) -> str:
    safe = re.sub(r"[^a-z0-9]+", "_", meter_id.lower()).strip("_")[:50]
    return f"energy.v_{safe}"


def _ensure_meter_view(conn, meter_id: str) -> None:
    view = _safe_view_name(meter_id)
    with conn.cursor() as cur:
        cur.execute(
            f"CREATE OR REPLACE VIEW {view} AS "
            "SELECT hora, interval_s, head, data, synced, created_at "
            "FROM energy.snapshots WHERE meter_id = %s ORDER BY hora DESC",
            (meter_id,),
        )
    conn.commit()
    log.info("View ensured: %s", view)


def _seed_meters_from_env(conn) -> None:
    rows = []
    i = 1
    while True:
        meter_id = os.environ.get(f"METER_ID_{i}", "").strip()
        unit_id_str = os.environ.get(f"METER_UNIT_ID_{i}", "").strip()
        if not meter_id:
            break
        if unit_id_str:
            rows.append((meter_id, int(unit_id_str)))
        i += 1
    if not rows:
        return
    with conn.cursor() as cur:
        for meter_id, unit_id in rows:
            cur.execute(
                "INSERT INTO energy.meters (meter_id, unit_id) VALUES (%s, %s) "
                "ON CONFLICT (meter_id) DO NOTHING",
                (meter_id, unit_id),
            )
    conn.commit()
    for meter_id, _ in rows:
        _ensure_meter_view(conn, meter_id)
    log.info("Meters seeded from environment: %s", [r[0] for r in rows])


def _save_snapshot(conn, device_id: str, meter_id: str, hora: str, interval_s: int, data: list) -> None:
    data_str = [str(v) if v is not None else "" for v in data]
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO energy.snapshots
                (device_id, meter_id, hora, interval_s, head, data, synced)
            VALUES (%s, %s, %s::timestamptz, %s, %s::text[], %s::text[], false)
            ON CONFLICT (device_id, meter_id, hora) DO NOTHING
            """,
            (device_id, meter_id, hora, interval_s, HEAD, data_str),
        )
    conn.commit()


def _connect_db(postgres_url: str):
    conn = psycopg2.connect(postgres_url)
    conn.autocommit = False
    return conn


# ─── HTTP Configuration API ──────────────────────────────────────────────────
# Background thread on port 8088.  Exposes read/write commands for the UI.
# Uses _SERIAL_LOCK to prevent simultaneous serial access with the read loop.

_SERIAL_LOCK  = threading.Lock()
_HEALTH_LOCK  = threading.Lock()
_WRITE_SERIAL = ""
_WRITE_DB_URL = ""

# Mapa meter_id -> bool actualizado en tiempo real al terminar cada poll.
# True = ultima lectura exitosa; False = fallo en registros criticos.
# Mientras no haya ocurrido ningun poll (arranque), la clave no existe.
_meter_health: dict = {}


def _health_check_loop(postgres_url: str, interval_s: int) -> None:
    """Daemon thread: lee un registro por medidor cada interval_s segundos.
    Actualiza _meter_health en tiempo real sin escribir en la base de datos."""
    conn = None
    while True:
        t_start = time.monotonic()
        try:
            if conn is None:
                conn = _connect_db(postgres_url)
            meters = _load_meters(conn)
        except Exception as exc:
            log.warning("Health-check: error cargando medidores: %s", exc)
            conn = None
            time.sleep(interval_s)
            continue

        for meter in meters:
            meter_id = meter["meter_id"]
            unit_id  = meter["unit_id"]
            alive = False
            try:
                with _SERIAL_LOCK:
                    ser = _ensure_serial()
                    regs = _raw_read_regs(ser, unit_id, 1010, 2)  # registro Va
                    if regs is None:
                        time.sleep(0.2)
                        regs = _raw_read_regs(ser, unit_id, 1010, 2)  # segundo intento
                alive = regs is not None
            except Exception as exc:
                log.debug("Health-check serial error meter=%s: %s", meter_id, exc)
            with _HEALTH_LOCK:
                prev = _meter_health.get(meter_id)
                _meter_health[meter_id] = alive
            if prev is True and not alive:
                log.warning("Health-check: meter=%s (unit_id=%d) perdio respuesta", meter_id, unit_id)
            elif prev is False and alive:
                log.info("Health-check: meter=%s (unit_id=%d) respondio de nuevo", meter_id, unit_id)

        elapsed = time.monotonic() - t_start
        time.sleep(max(0.0, interval_s - elapsed))

# Persistent serial port — opened once at startup and kept open indefinitely.
# Both the main read loop and the HTTP scan endpoint share this object.
# Keeping the port open ensures the CH340 USB adapter stays in a stable
# ("warm") state and avoids the SerialException that occurs on first read
# after a cold open.
_SER: serial.Serial | None = None


def _ensure_serial() -> serial.Serial:
    """Return the shared serial port, opening it if necessary."""
    global _SER
    if _SER is not None and _SER.is_open:
        return _SER
    s = serial.Serial(
        port=_WRITE_SERIAL, baudrate=9600, bytesize=8,
        stopbits=1, parity="N", timeout=0.8,
    )
    # Warm up: attempt reads until the CH340 adapter stops raising
    # SerialException ("device reports readiness to read but returned no data").
    # This typically settles within 2-3 attempts (~600 ms).
    for _ in range(6):
        try:
            s.reset_input_buffer()
            s.read(1)
            break
        except serial.SerialException:
            time.sleep(0.2)
    s.reset_input_buffer()
    _SER = s
    return _SER

_WIRING_LABELS  = {0: "3P4W_4CT", 1: "3P4W_3CT", 2: "3P3W_3CT", 3: "3P3W_2CT", 4: "1P3W", 5: "1P2W"}
_BAUD_LABELS    = {0: "2400", 1: "4800", 2: "9600", 3: "19200", 4: "38400", 5: "57600", 6: "115200"}
_BAUD_INDEX     = {int(v): k for k, v in _BAUD_LABELS.items()}
_PARITY_LABELS  = {0: "N", 1: "O", 2: "E"}
_PARITY_INDEX   = {v: k for k, v in _PARITY_LABELS.items()}
_RESULT_CODES   = {0: "OK", 80: "Invalid instruction code", 81: "Invalid parameter value",
                   82: "Invalid parameter count", 83: "Operation not performed"}
_RESET_TYPES    = {"maxmin": 1, "demand": 2, "tariff": 3, "energy": 4, "all": 5}


def _raw_write_regs(s: serial.Serial, slave: int, addr: int, values: list) -> bool:
    """Write multiple holding registers via raw FC16, handling CH340 echo."""
    count   = len(values)
    header  = struct.pack(">BBHHB", slave, 0x10, addr, count, count * 2)
    data    = b"".join(struct.pack(">H", v & 0xFFFF) for v in values)
    payload = header + data
    req     = payload + struct.pack("<H", _crc16(payload))
    req_len = len(req)
    resp_len = 8  # slave(1) FC(1) addr(2) count(2) CRC(2)

    s.reset_input_buffer()
    s.write(req)
    try:
        buf = s.read(req_len + resp_len)
    except serial.SerialException:
        time.sleep(0.1)
        try:
            buf = s.read(req_len + resp_len)
        except serial.SerialException:
            return False

    def _parse(frame):
        if len(frame) < resp_len:
            return False
        if frame[0] != slave or frame[1] != 0x10:
            return False
        crc_calc = _crc16(bytes(frame[:6]))
        crc_recv = struct.unpack("<H", bytes(frame[6:8]))[0]
        return crc_calc == crc_recv

    if len(buf) >= req_len + resp_len:
        if _parse(buf[req_len:req_len + resp_len]):
            return True
    if len(buf) >= resp_len:
        if _parse(buf[:resp_len]):
            return True
    return False


def _api_u32_regs(value: int) -> list:
    return [(value >> 16) & 0xFFFF, value & 0xFFFF]


def _api_write_instruction(s: serial.Serial, slave: int, code: int, params: list):
    # Send FC16 write. Some MC60 firmware versions do not echo a standard FC16
    # ACK for certain instruction codes (e.g. 1200, 1301) — they silently
    # execute and update registers 424-425. We never use the FC16 ACK as
    # validation; the status register 425 is the authoritative result.
    _raw_write_regs(s, slave, 300, [code] + params)
    time.sleep(0.15)
    for attempt in range(3):
        regs = _raw_read_regs(s, slave, 424, 2)
        if regs is not None:
            status = regs[1]  # reg 424=instruction echo, 425=status code
            return status == 0, _RESULT_CODES.get(status, f"Unknown ({status})")
        if attempt < 2:
            s.reset_input_buffer()
            time.sleep(0.1)
    return False, "Write error: device did not respond to status read"


def _api_audit_log(command: str, unit_id: int, meter_id: str, params: dict,
                   success: bool, message: str) -> None:
    try:
        conn = psycopg2.connect(_WRITE_DB_URL)
        with conn.cursor() as cur:
            cur.execute(
                "INSERT INTO energy.config_audit "
                "(command, unit_id, meter_id, params, result, message) "
                "VALUES (%s, %s, %s, %s::jsonb, %s, %s)",
                (command, unit_id, meter_id, json.dumps(params),
                 "ok" if success else "error", message),
            )
        conn.commit()
        conn.close()
    except Exception as exc:
        log.warning("Audit log failed: %s", exc)


class _ConfigAPIHandler(BaseHTTPRequestHandler):

    def log_message(self, fmt, *args):
        log.debug("cfg-api %s", fmt % args)

    def _json(self, code: int, data) -> None:
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _body(self) -> dict:
        n = int(self.headers.get("Content-Length", 0))
        return json.loads(self.rfile.read(n)) if n else {}

    def _parse_path(self):
        parts = self.path.strip("/").split("/")
        if len(parts) >= 3 and parts[0] == "write-api" and parts[1] == "meter":
            try:
                return int(parts[2]), parts[3] if len(parts) > 3 else ""
            except (ValueError, IndexError):
                pass
        return None, None

    def do_GET(self):
        if self.path == "/write-api/health":
            self._json(200, {"status": "ok"})
            return

        if self.path == "/write-api/live":
            try:
                conn = psycopg2.connect(_WRITE_DB_URL)
                with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                    cur.execute(
                        "SELECT value FROM energy.config WHERE key = 'meter_poll_interval_ms'"
                    )
                    row = cur.fetchone()
                    poll_ms = int(row["value"]) if row else 300000

                    cur.execute(
                        "SELECT meter_id, unit_id FROM energy.meters"
                        " WHERE active = TRUE ORDER BY unit_id"
                    )
                    meters = [dict(r) for r in cur.fetchall()]

                    cur.execute(
                        """
                        SELECT DISTINCT ON (s.meter_id)
                            s.meter_id,
                            s.hora,
                            s.data[array_position(s.head, 'Vavg')]  AS vavg,
                            s.data[array_position(s.head, 'Iavg')]  AS iavg,
                            s.data[array_position(s.head, 'P')]     AS p,
                            s.data[array_position(s.head, 'EPImp')] AS ep_imp,
                            s.data[array_position(s.head, 'Freq')]  AS freq,
                            s.data[array_position(s.head, 'PF')]    AS pf
                        FROM energy.snapshots s
                        ORDER BY s.meter_id, s.hora DESC
                        """
                    )
                    snaps = {r["meter_id"]: dict(r) for r in cur.fetchall()}
                conn.close()

                def _safe(v):
                    if v is None or v == "":
                        return None
                    try:
                        return round(float(v), 3)
                    except (TypeError, ValueError):
                        return None

                now = datetime.now(timezone.utc)
                result = []
                for m in meters:
                    mid  = m["meter_id"]
                    snap = snaps.get(mid)
                    if snap:
                        hora_dt = snap["hora"]
                        if hasattr(hora_dt, "tzinfo") and hora_dt.tzinfo is None:
                            hora_dt = hora_dt.replace(tzinfo=timezone.utc)
                        age_s  = int((now - hora_dt).total_seconds())
                        with _HEALTH_LOCK:
                            health = _meter_health.get(mid)
                        online = health if health is not None else age_s < (poll_ms / 1000) * 1.2
                        result.append({
                            "meter_id":  mid,
                            "unit_id":   m["unit_id"],
                            "last_hora": hora_dt.isoformat(),
                            "age_s":     age_s,
                            "online":    online,
                            "vavg":      _safe(snap["vavg"]),
                            "iavg":      _safe(snap["iavg"]),
                            "p":         _safe(snap["p"]),
                            "ep_imp":    _safe(snap["ep_imp"]),
                            "freq":      _safe(snap["freq"]),
                            "pf":        _safe(snap["pf"]),
                        })
                    else:
                        result.append({
                            "meter_id":  mid,
                            "unit_id":   m["unit_id"],
                            "last_hora": None,
                            "age_s":     None,
                            "online":    False,
                            "vavg":      None, "iavg":   None, "p":    None,
                            "ep_imp":    None, "freq":   None, "pf":   None,
                        })

                self._json(200, {
                    "poll_interval_s": poll_ms // 1000,
                    "ts":              now.isoformat(),
                    "meters":          result,
                })
            except Exception as exc:
                self._json(500, {"error": str(exc)})
            return

        if self.path.startswith("/write-api/history"):
            try:
                from urllib.parse import urlparse, parse_qs
                qs       = parse_qs(urlparse(self.path).query)
                meter_id = qs.get("meter_id", [None])[0]
                limit    = min(int(qs.get("limit", ["48"])[0]), 200)
                if not meter_id:
                    self._json(400, {"error": "meter_id required"})
                    return
                conn = psycopg2.connect(_WRITE_DB_URL)
                with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                    cur.execute(
                        """
                        SELECT s.hora,
                               s.data[array_position(s.head, 'Vavg')]   AS vavg,
                               s.data[array_position(s.head, 'VLavg')]  AS vlavg,
                               s.data[array_position(s.head, 'Iavg')]   AS iavg,
                               s.data[array_position(s.head, 'P')]      AS p,
                               s.data[array_position(s.head, 'Q')]      AS q,
                               s.data[array_position(s.head, 'S')]      AS s_va,
                               s.data[array_position(s.head, 'PF')]     AS pf,
                               s.data[array_position(s.head, 'DPF')]    AS dpf,
                               s.data[array_position(s.head, 'Freq')]   AS freq,
                               s.data[array_position(s.head, 'EPImp')]  AS ep_imp,
                               s.data[array_position(s.head, 'EPExp')]  AS ep_exp,
                               s.data[array_position(s.head, 'EQImp')]  AS eq_imp,
                               s.data[array_position(s.head, 'EQExp')]  AS eq_exp,
                               s.data[array_position(s.head, 'ES')]     AS es
                        FROM energy.snapshots s
                        WHERE s.meter_id = %s
                        ORDER BY s.hora DESC
                        LIMIT %s
                        """,
                        (meter_id, limit),
                    )
                    raw = cur.fetchall()
                conn.close()

                def _safe(v):
                    if v is None or v == "":
                        return None
                    try:
                        return round(float(v), 3)
                    except (TypeError, ValueError):
                        return None

                rows = []
                for r in reversed(raw):
                    hora_dt = r["hora"]
                    if hasattr(hora_dt, "tzinfo") and hora_dt.tzinfo is None:
                        hora_dt = hora_dt.replace(tzinfo=timezone.utc)
                    rows.append({
                        "hora":   hora_dt.isoformat(),
                        "vavg":   _safe(r["vavg"]),
                        "vlavg":  _safe(r["vlavg"]),
                        "iavg":   _safe(r["iavg"]),
                        "p":      _safe(r["p"]),
                        "q":      _safe(r["q"]),
                        "s_va":   _safe(r["s_va"]),
                        "pf":     _safe(r["pf"]),
                        "dpf":    _safe(r["dpf"]),
                        "freq":   _safe(r["freq"]),
                        "ep_imp": _safe(r["ep_imp"]),
                        "ep_exp": _safe(r["ep_exp"]),
                        "eq_imp": _safe(r["eq_imp"]),
                        "eq_exp": _safe(r["eq_exp"]),
                        "es":     _safe(r["es"]),
                    })
                self._json(200, {"meter_id": meter_id, "rows": rows})
            except Exception as exc:
                self._json(500, {"error": str(exc)})
            return

        if self.path.startswith("/write-api/audit"):
            try:
                conn = psycopg2.connect(_WRITE_DB_URL)
                with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                    cur.execute(
                        "SELECT id, ts::text, command, unit_id, meter_id, "
                        "params::text AS params, result, message "
                        "FROM energy.config_audit ORDER BY ts DESC LIMIT 100"
                    )
                    rows = [dict(r) for r in cur.fetchall()]
                conn.close()
                self._json(200, rows)
            except Exception as exc:
                self._json(500, {"error": str(exc)})
            return

        unit_id, _ = self._parse_path()
        if unit_id is None:
            self._json(404, {"error": "not found"})
            return

        def _read_config(s):
            """Read all config registers. Returns a dict or raises IOError."""
            # Inter-read delay: CH340 USB adapter needs ~20 ms between frames
            # to flush its half-duplex direction-switch pipeline.
            _IFD = 0.05

            s.reset_input_buffer()
            time.sleep(_IFD)

            b_sys = _raw_read_regs(s, unit_id, 500, 7)
            if b_sys is None:
                raise IOError(f"Read error addr=500: No response from unit {unit_id}")
            time.sleep(_IFD)

            b_ctm = _raw_read_regs(s, unit_id, 510, 1)
            if b_ctm is None:
                raise IOError(f"Read error addr=510: No response from unit {unit_id}")
            time.sleep(_IFD)

            b_dir = _raw_read_regs(s, unit_id, 550, 3)
            if b_dir is None:
                raise IOError(f"Read error addr=550: No response from unit {unit_id}")
            time.sleep(_IFD)

            b_com = _raw_read_regs(s, unit_id, 80, 3)
            if b_com is None:
                raise IOError(f"Read error addr=80: No response from unit {unit_id}")

            return {
                "wiring":      b_sys[0],
                "freq":        b_sys[1],
                "nom_v":       b_sys[2],
                "vt_raw":      (b_sys[3] << 16) | b_sys[4],
                "ct_raw":      (b_sys[5] << 16) | b_sys[6],
                "ct_mode":     b_ctm[0],
                "d1":          b_dir[0],
                "d2":          b_dir[1],
                "d3":          b_dir[2],
                "slave_addr":  b_com[0],
                "baud_idx":    b_com[1],
                "parity_i":    b_com[2],
            }

        with _SERIAL_LOCK:
            last_exc = None
            for attempt in range(3):
                try:
                    s = _ensure_serial()
                    cfg = _read_config(s)
                    self._json(200, {
                        "wiring":       cfg["wiring"],
                        "wiring_name":  _WIRING_LABELS.get(cfg["wiring"], str(cfg["wiring"])),
                        "freq":         cfg["freq"],
                        "nominal_v":    cfg["nom_v"],
                        "vt_ratio":     round(cfg["vt_raw"] / 10000, 4),
                        "ct_ratio":     round(cfg["ct_raw"] / 10000, 4),
                        "ct_mode":      "VCT" if cfg["ct_mode"] == 1 else "Rogowski",
                        "dir_l1":       "rev" if cfg["d1"] else "pos",
                        "dir_l2":       "rev" if cfg["d2"] else "pos",
                        "dir_l3":       "rev" if cfg["d3"] else "pos",
                        "slave_addr":   cfg["slave_addr"],
                        "baud":         int(_BAUD_LABELS.get(cfg["baud_idx"], "9600")),
                        "parity":       _PARITY_LABELS.get(cfg["parity_i"], "N"),
                    })
                    return
                except Exception as exc:
                    last_exc = exc
                    log.warning("Config read attempt %d failed unit_id=%d: %s",
                                attempt + 1, unit_id, exc)
                    time.sleep(0.15 * (attempt + 1))
            self._json(500, {"error": str(last_exc)})

    def do_POST(self):
        # Scan: POST /write-api/scan  body: {"start":1,"end":100}
        if self.path.rstrip("/") == "/write-api/scan":
            body  = self._body()
            start = max(1,   int(body.get("start", 1)))
            end   = min(247, int(body.get("end",  100)))
            found = []
            with _SERIAL_LOCK:
                try:
                    s = _ensure_serial()
                    s.reset_input_buffer()
                    # Pre-probe to UID 0 (Modbus reserved, never responds).
                    # Ensures the CH340 USB pipeline is fully active before the
                    # first real UID is scanned; without this, the first probe
                    # misses ~30% of the time even with a persistent port.
                    _raw_read_regs(s, 0, 1000, 1)
                    s.reset_input_buffer()
                    for uid in range(start, end + 1):
                        regs = _raw_read_regs(s, uid, 1000, 1)
                        if regs is not None:
                            found.append(uid)
                        time.sleep(0.1)
                except Exception as exc:
                    self._json(500, {"error": str(exc)})
                    return
            self._json(200, {"found": found, "scanned": end - start + 1})
            return

        unit_id, cmd = self._parse_path()
        if unit_id is None or not cmd:
            self._json(404, {"error": "not found"})
            return

        body    = self._body()
        success = False
        message = "unknown command"

        with _SERIAL_LOCK:
            try:
                s = _ensure_serial()
                s.reset_input_buffer()
                time.sleep(0.05)

                def _retry_read(addr, count):
                    """Read a register block with up to 3 attempts on failure."""
                    for _a in range(3):
                        regs = _raw_read_regs(s, unit_id, addr, count)
                        if regs is not None:
                            return regs
                        s.reset_input_buffer()
                        time.sleep(0.15)
                    raise IOError(f"Read error addr={addr}: No response from unit {unit_id}")

                # Commands without a config pre-read need a warm-up probe so the
                # CH340 USB pipeline is active before the write frame is sent.
                if cmd not in ("set-ct", "set-sys"):
                    _raw_read_regs(s, unit_id, 500, 1)
                    s.reset_input_buffer()
                    time.sleep(0.05)

                if cmd == "set-ct":
                    primary   = int(body.get("primary", 200))
                    secondary = int(body.get("secondary", 5))
                    ct_raw    = round(primary / secondary * 10000)
                    b = _retry_read(500, 5)
                    wiring = b[0]; freq = b[1]; nom_v = b[2]
                    vt_raw = (b[3] << 16) | b[4]
                    time.sleep(0.05)
                    params = [wiring, freq, nom_v] + _api_u32_regs(vt_raw) + _api_u32_regs(ct_raw)
                    success, message = _api_write_instruction(s, unit_id, 1001, params)

                elif cmd == "set-sys":
                    b = _retry_read(500, 7)
                    wiring = int(body["wiring"])  if "wiring"  in body else b[0]
                    freq   = int(body["freq"])    if "freq"    in body else b[1]
                    nom_v  = int(body["nominal"]) if "nominal" in body else b[2]
                    vt_raw = round(float(body["vt"]) * 10000) if "vt" in body else (b[3] << 16) | b[4]
                    ct_raw = round(float(body["ct"]) * 10000) if "ct" in body else (b[5] << 16) | b[6]
                    if not 1 <= wiring <= 5:
                        self._json(400, {"error": f"wiring must be 1-5 per instruction 1001 (got {wiring})"})
                        return
                    time.sleep(0.05)
                    params = [wiring, freq, nom_v] + _api_u32_regs(vt_raw) + _api_u32_regs(ct_raw)
                    success, message = _api_write_instruction(s, unit_id, 1001, params)

                elif cmd == "set-dir":
                    dir_map = {"pos": 0, "rev": 1}
                    params  = [dir_map.get(body.get("l1", "pos"), 0),
                               dir_map.get(body.get("l2", "pos"), 0),
                               dir_map.get(body.get("l3", "pos"), 0)]
                    success, message = _api_write_instruction(s, unit_id, 1010, params)

                elif cmd == "set-time":
                    TZ_LIMA = timezone(timedelta(hours=-5))  # America/Lima — UTC-5, no DST
                    now    = datetime.now(TZ_LIMA)
                    params = [now.year, now.month, now.day, now.hour, now.minute, now.second]
                    success, message = _api_write_instruction(s, unit_id, 1200, params)

                elif cmd == "reset":
                    rtype  = _RESET_TYPES.get(body.get("type", "energy"), 4)
                    success, message = _api_write_instruction(s, unit_id, 1301, [rtype])

                else:
                    self._json(400, {"error": f"Unknown command: {cmd}"})
                    return

            except Exception as exc:
                success, message = False, str(exc)

        _api_audit_log(cmd, unit_id, body.get("meter_id", ""), body, success, message)
        self._json(200 if success else 500, {"success": success, "message": message})


def _start_config_api(port: int) -> None:
    srv = ThreadingHTTPServer(("0.0.0.0", port), _ConfigAPIHandler)
    threading.Thread(target=srv.serve_forever, daemon=True, name="cfg-api").start()
    log.info("Config API started on port %d", port)


def main() -> None:
    global _WRITE_SERIAL, _WRITE_DB_URL

    postgres_url = os.environ["POSTGRES_URL"]
    device_id = os.environ.get("DEVICE_ID", "rpienergy01")
    serial_dev = os.environ.get("MODBUS_SERIAL_DEV", "/dev/ttyUSB0")
    poll_interval_s = int(os.environ.get("METER_POLL_INTERVAL_MS", "300000")) // 1000

    _WRITE_SERIAL = serial_dev
    _WRITE_DB_URL = postgres_url

    conn = _connect_db(postgres_url)
    _seed_meters_from_env(conn)

    # Alert if the system was off and there is a data gap
    try:
        with conn.cursor() as _gc:
            _gc.execute("SELECT MAX(hora) FROM energy.snapshots")
            _last_hora = _gc.fetchone()[0]
        if _last_hora is not None:
            _gap_min = (datetime.now(timezone.utc) - _last_hora).total_seconds() / 60
            if _gap_min > 15:
                log.warning(
                    "Startup gap detected: last snapshot was %.0f min ago (%s UTC) — "
                    "data was not collected during that period",
                    _gap_min, _last_hora.strftime("%Y-%m-%d %H:%M"),
                )
    except Exception as _ge:
        log.warning("Could not check startup data gap: %s", _ge)

    cfg = _load_config(conn)
    config_reload_s = int(cfg.get("config_reload_s", "60"))
    last_config_reload = time.monotonic()

    _start_config_api(int(os.environ.get("CONFIG_API_PORT", "8088")))

    health_interval_s = int(os.environ.get("HEALTH_CHECK_INTERVAL_S", "15"))
    _hc_thread = threading.Thread(
        target=_health_check_loop,
        args=(postgres_url, health_interval_s),
        daemon=True,
        name="health-check",
    )
    _hc_thread.start()
    log.info(
        "mc60-reader started device=%s serial=%s poll=%ds health_check=%ds",
        device_id, serial_dev, poll_interval_s, health_interval_s,
    )

    while True:
        loop_start = time.monotonic()

        if time.monotonic() - last_config_reload >= config_reload_s:
            try:
                cfg = _load_config(conn)
                poll_ms = int(cfg.get("meter_poll_interval_ms", str(poll_interval_s * 1000)))
                poll_interval_s = poll_ms // 1000
                config_reload_s = int(cfg.get("config_reload_s", "60"))
                last_config_reload = time.monotonic()
                log.info("Config reloaded poll=%ds", poll_interval_s)
            except Exception as exc:
                log.error("Config reload failed: %s", exc)

        try:
            meters = _load_meters(conn)
        except Exception as exc:
            log.error("Failed to load meters: %s", exc)
            try:
                conn = _connect_db(postgres_url)
                meters = _load_meters(conn)
            except Exception as exc2:
                log.error("DB reconnect failed: %s", exc2)
                time.sleep(poll_interval_s)
                continue

        if not meters:
            log.warning("No active meters in energy.meters — waiting")
            time.sleep(poll_interval_s)
            continue

        hora = _snapshot_timestamp()

        with _SERIAL_LOCK:
            try:
                ser = _ensure_serial()
                ser.reset_input_buffer()
            except serial.SerialException as exc:
                log.error("Cannot open serial %s: %s", serial_dev, exc)
                time.sleep(poll_interval_s)
                continue

            for meter in meters:
                meter_id = meter["meter_id"]
                unit_id  = meter["unit_id"]
                log.info("Reading meter=%s unit_id=%d", meter_id, unit_id)
                # Flush in-flight USB bytes before switching to a different meter
                time.sleep(0.1)
                ser.reset_input_buffer()
                data = read_meter(ser, unit_id)
                # Skip snapshot if the meter returned no valid RS-485 response at all.
                # A live meter must always provide at least Va and EPImp.
                if data[_IDX_VA] is None and data[_IDX_EPIMP] is None:
                    log.warning(
                        "Critical registers (Va, EPImp) both missing for meter=%s "
                        "— skipping snapshot to avoid polluting the timeslot",
                        meter_id,
                    )
                    with _HEALTH_LOCK:
                        _meter_health[meter_id] = False
                    continue
                try:
                    _save_snapshot(conn, device_id, meter_id, hora, poll_interval_s, data)
                    with _HEALTH_LOCK:
                        _meter_health[meter_id] = True
                    log.info("Snapshot saved meter=%s hora=%s", meter_id, hora)
                except Exception as exc:
                    log.error("Save snapshot failed meter=%s: %s", meter_id, exc)
                    try:
                        conn.rollback()
                    except Exception:
                        pass
                    try:
                        conn = _connect_db(postgres_url)
                        _save_snapshot(conn, device_id, meter_id, hora, poll_interval_s, data)
                        log.info("Snapshot saved (after DB reconnect) meter=%s hora=%s", meter_id, hora)
                    except Exception as exc2:
                        log.error("Save retry failed after reconnect meter=%s: %s", meter_id, exc2)

        elapsed = time.monotonic() - loop_start
        time.sleep(max(0.0, poll_interval_s - elapsed))


if __name__ == "__main__":
    main()
