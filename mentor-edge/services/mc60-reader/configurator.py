"""
MC60 meter configurator — CLI tool for reading and writing device parameters.

All configuration writes use the MC60 "instruction register" protocol:
- Write FC 0x10 to address 300: [instruction_code, param1, param2, ...]
- Read result from register 424 (echoed instruction) and 425 (0=OK, 80-83=error)

Usage inside the container:
  python configurator.py show      --slave-id 1
  python configurator.py set-ct    --slave-id 1 --primary 200 --secondary 5
  python configurator.py set-sys   --slave-id 1 --wiring 1 --freq 50 --nominal 220 --vt 1.0 --ct 40.0
  python configurator.py set-dir   --slave-id 1 --l1 pos --l2 pos --l3 rev
  python configurator.py set-addr  --slave-id 1 --new-addr 3 --baud 9600 --parity N
  python configurator.py set-time  --slave-id 1
  python configurator.py reset     --slave-id 1 --type energy
"""
import argparse
import os
import struct
import sys
import time
from datetime import datetime, timezone

from pymodbus.client import ModbusSerialClient

# Serial port settings (same as main.py)
PORT     = os.environ.get("SERIAL_PORT", "/dev/ttyUSB0")
BAUDRATE = int(os.environ.get("SERIAL_BAUDRATE", "9600"))
PARITY   = os.environ.get("SERIAL_PARITY", "N")

# Instruction result codes (register 425)
_RESULT_CODES = {
    0:  "OK",
    80: "Invalid instruction code",
    81: "Invalid parameter value",
    82: "Invalid parameter count",
    83: "Operation not performed",
}

# Baud rate index map for instruction 1210
_BAUD_INDEX = {2400: 0, 4800: 1, 9600: 2, 19200: 3, 38400: 4, 57600: 5, 115200: 6}
_PARITY_INDEX = {"N": 0, "O": 1, "E": 2}

# Reset type codes for instruction 1301
_RESET_TYPES = {
    "maxmin":  1,
    "demand":  2,
    "tariff":  3,
    "energy":  4,
    "all":     5,
}


def _connect() -> ModbusSerialClient:
    client = ModbusSerialClient(
        port=PORT,
        baudrate=BAUDRATE,
        parity=PARITY,
        stopbits=1,
        bytesize=8,
        timeout=2,
    )
    if not client.connect():
        print(f"ERROR: Cannot connect to {PORT}", file=sys.stderr)
        sys.exit(1)
    return client


def _read_u16(client: ModbusSerialClient, address: int, slave: int) -> int:
    r = client.read_holding_registers(address=address, count=1, slave=slave)
    if r.isError():
        raise IOError(f"Read error at address {address}: {r}")
    return r.registers[0]


def _read_u32(client: ModbusSerialClient, address: int, slave: int) -> int:
    r = client.read_holding_registers(address=address, count=2, slave=slave)
    if r.isError():
        raise IOError(f"Read error at address {address}: {r}")
    return (r.registers[0] << 16) | r.registers[1]


def _read_f32(client: ModbusSerialClient, address: int, slave: int) -> float:
    r = client.read_holding_registers(address=address, count=2, slave=slave)
    if r.isError():
        raise IOError(f"Read error at address {address}: {r}")
    raw = struct.pack(">HH", r.registers[0], r.registers[1])
    return struct.unpack(">f", raw)[0]


def _u32_to_regs(value: int) -> list:
    return [(value >> 16) & 0xFFFF, value & 0xFFFF]


def _write_instruction(client: ModbusSerialClient, slave: int, instr_code: int, params: list) -> bool:
    """
    Write instruction to register 300 and verify result via registers 424/425.
    params: list of uint16 values (uint32 fields must be pre-split into [hi, lo]).
    """
    values = [instr_code] + params
    r = client.write_registers(address=300, values=values, slave=slave)
    if r.isError():
        print(f"ERROR: Write failed for instruction {instr_code}: {r}", file=sys.stderr)
        return False

    time.sleep(0.15)

    try:
        echo   = _read_u16(client, 424, slave)
        status = _read_u16(client, 425, slave)
    except IOError as e:
        print(f"WARNING: Could not read result registers: {e}", file=sys.stderr)
        return True

    msg = _RESULT_CODES.get(status, f"Unknown ({status})")
    if status == 0:
        print(f"  Instruction {instr_code}: {msg}")
        return True
    else:
        print(f"  ERROR Instruction {instr_code} (echo={echo}): {msg}", file=sys.stderr)
        return False


def cmd_show(client: ModbusSerialClient, slave: int):
    """Read and display current configuration from the meter."""
    print(f"\n--- MC60 slave {slave} current configuration ---")

    wiring_labels = {0: "3P4W_4CT", 1: "3P4W_3CT", 2: "3P3W_3CT", 3: "3P3W_2CT", 4: "1P3W", 5: "1P2W"}
    baud_labels   = {0: "2400", 1: "4800", 2: "9600", 3: "19200", 4: "38400", 5: "57600", 6: "115200"}
    parity_labels = {0: "None", 1: "Odd", 2: "Even"}
    dir_labels    = {0: "Positive", 1: "Reverse"}

    wiring  = _read_u16(client, 500, slave)
    freq    = _read_u16(client, 501, slave)
    nom_v   = _read_u16(client, 502, slave)
    vt_raw  = _read_u32(client, 503, slave)
    ct_raw  = _read_u32(client, 505, slave)

    print(f"  Wiring mode:       {wiring} = {wiring_labels.get(wiring, '?')}")
    print(f"  Grid frequency:    {freq} Hz")
    print(f"  Nominal voltage:   {nom_v} V")
    print(f"  VT ratio:          {vt_raw / 10000:.4f} ({vt_raw}/10000)")
    print(f"  CT ratio:          {ct_raw / 10000:.4f} ({ct_raw}/10000)")

    ct_mode = _read_u16(client, 510, slave)
    print(f"\n  Phase L1L2L3 CT mode: {'VCT' if ct_mode == 1 else 'Rogowski'}")
    if ct_mode == 1:
        vct_pri = _read_u32(client, 517, slave)
        vct_sec = _read_u32(client, 519, slave)
        vct_nom = _read_u32(client, 521, slave)
        print(f"    VCT primary:   {vct_pri} A")
        print(f"    VCT secondary: {vct_sec / 100:.2f} mV")
        print(f"    VCT nominal:   {vct_nom} A")

    d1 = _read_u16(client, 550, slave)
    d2 = _read_u16(client, 551, slave)
    d3 = _read_u16(client, 552, slave)
    print(f"\n  Current direction: L1={dir_labels.get(d1,'?')}  L2={dir_labels.get(d2,'?')}  L3={dir_labels.get(d3,'?')}")

    ch1 = _read_u16(client, 553, slave)
    ch2 = _read_u16(client, 554, slave)
    ch3 = _read_u16(client, 555, slave)
    print(f"  Current channel:   L1=Ch{ch1+1}  L2=Ch{ch2+1}  L3=Ch{ch3+1}")

    addr    = _read_u16(client, 80, slave)
    baud    = _read_u16(client, 81, slave)
    parity  = _read_u16(client, 82, slave)
    stop    = _read_u16(client, 83, slave)
    print(f"\n  Slave address:  {addr}")
    print(f"  Baud rate:      {baud_labels.get(baud, '?')}")
    print(f"  Parity:         {parity_labels.get(parity, '?')}")
    print(f"  Stop bits:      {stop}")

    # Live measurements
    print(f"\n  -- Live readings --")
    for label, addr in [("Va", 1010), ("Vb", 1012), ("Vc", 1014), ("Ia", 1000), ("Ib", 1002), ("Ic", 1004)]:
        try:
            val = _read_f32(client, addr, slave)
            print(f"  {label:4s}: {val:.2f}")
        except IOError:
            print(f"  {label:4s}: error")
    print()


def cmd_set_system(client: ModbusSerialClient, slave: int, args):
    """
    Instruction 1001 — system parameters.
    Reads current values first; only overrides what is explicitly provided.
    Valid wiring range for instruction 1001: 1-5 (register 500 value 0=3P4W_4CT is read-only).
    """
    wiring   = args.wiring   if args.wiring  is not None else _read_u16(client, 500, slave)
    freq     = args.freq     if args.freq    is not None else _read_u16(client, 501, slave)
    nom_v    = args.nominal  if args.nominal is not None else _read_u16(client, 502, slave)
    vt_raw   = round(args.vt * 10000) if args.vt is not None else _read_u32(client, 503, slave)
    ct_raw   = round(args.ct * 10000) if args.ct is not None else _read_u32(client, 505, slave)

    if not 1 <= wiring <= 5:
        print(f"ERROR: wiring must be 1-5 per instruction 1001 (got {wiring}). "
              "Value 0 (3P4W_4CT) is readable but not writable.", file=sys.stderr)
        return

    params = [wiring, freq, nom_v] + _u32_to_regs(vt_raw) + _u32_to_regs(ct_raw)
    print(f"  Setting system params: wiring={wiring}, freq={freq}Hz, nomV={nom_v}V, VT={vt_raw/10000:.4f}, CT={ct_raw/10000:.4f}")
    _write_instruction(client, slave, 1001, params)


def cmd_set_ct(client: ModbusSerialClient, slave: int, args):
    """
    Instruction 1001 — update CT ratio (primary/secondary).
    Only sets the CT transformation ratio in the system parameters register.
    Instruction 1002 (VCT sensor detail) must be configured separately if using
    a voltage-output CT; its secondary parameter is in mV, not amps.
    """
    primary   = args.primary
    secondary = args.secondary

    ratio   = primary / secondary
    ct_raw  = round(ratio * 10000)
    wiring  = _read_u16(client, 500, slave)
    freq    = _read_u16(client, 501, slave)
    nom_v   = _read_u16(client, 502, slave)
    vt_raw  = _read_u32(client, 503, slave)

    if not 1 <= wiring <= 5:
        print(f"WARNING: current wiring value {wiring} (3P4W_4CT) is not writable via instruction 1001. "
              "Using wiring=1 (3P4W_3CT) as fallback.", file=sys.stderr)
        wiring = 1

    print(f"  Setting CT ratio {primary}:{secondary} = {ratio:.4f} (stored as {ct_raw})")
    params_1001 = [wiring, freq, nom_v] + _u32_to_regs(vt_raw) + _u32_to_regs(ct_raw)
    _write_instruction(client, slave, 1001, params_1001)


def cmd_set_direction(client: ModbusSerialClient, slave: int, args):
    """Instruction 1010 — set current direction per phase."""
    dir_map = {"pos": 0, "positive": 0, "rev": 1, "reverse": 1}
    d1 = dir_map.get(args.l1.lower(), 0)
    d2 = dir_map.get(args.l2.lower(), 0)
    d3 = dir_map.get(args.l3.lower(), 0)
    print(f"  Setting direction: L1={'rev' if d1 else 'pos'}  L2={'rev' if d2 else 'pos'}  L3={'rev' if d3 else 'pos'}")
    _write_instruction(client, slave, 1010, [d1, d2, d3])


def cmd_set_address(client: ModbusSerialClient, slave: int, args):
    """
    Instruction 1210 — change communication parameters.
    WARNING: after this call the meter responds on the new slave address and baud rate.
    """
    baud_idx   = _BAUD_INDEX.get(args.baud, 2)       # default 9600
    parity_idx = _PARITY_INDEX.get(args.parity.upper(), 0)
    stop_bits  = args.stop if args.stop else 1

    print(f"  Changing communication: addr={args.new_addr}, baud={args.baud}, parity={args.parity}, stop={stop_bits}")
    print("  WARNING: reconnect using new parameters after this command.")
    _write_instruction(client, slave, 1210, [args.new_addr, baud_idx, parity_idx, stop_bits])


def cmd_set_time(client: ModbusSerialClient, slave: int, _args):
    """Instruction 1200 — sync device clock from system UTC time."""
    now = datetime.now(timezone.utc)
    print(f"  Syncing device time to UTC: {now.strftime('%Y-%m-%d %H:%M:%S')}")
    _write_instruction(client, slave, 1200, [now.year, now.month, now.day, now.hour, now.minute, now.second])


def cmd_reset(client: ModbusSerialClient, slave: int, args):
    """Instruction 1301 — reset counters."""
    rtype = _RESET_TYPES.get(args.type.lower())
    if rtype is None:
        print(f"ERROR: Unknown reset type '{args.type}'. Valid: {list(_RESET_TYPES)}", file=sys.stderr)
        sys.exit(1)
    print(f"  Resetting: {args.type} (code {rtype})")
    _write_instruction(client, slave, 1301, [rtype])


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="MC60 meter configurator")
    p.add_argument("--slave-id", type=int, required=True, help="Modbus slave ID (1-247)")
    sub = p.add_subparsers(dest="cmd", required=True)

    sub.add_parser("show", help="Display current meter configuration and live readings")

    sp = sub.add_parser("set-sys", help="Set system parameters (instruction 1001)")
    sp.add_argument("--wiring",  type=int, choices=[1,2,3,4,5], help="1=3P4W_3CT 2=3P3W_3CT 3=3P3W_2CT 4=1P3W 5=1P2W")
    sp.add_argument("--freq",    type=int, choices=[50, 60])
    sp.add_argument("--nominal", type=int, help="Nominal phase voltage (V)")
    sp.add_argument("--vt",      type=float, help="VT ratio (e.g. 1.0 for direct)")
    sp.add_argument("--ct",      type=float, help="CT ratio primary/secondary (e.g. 40.0 for 200:5)")

    ct = sub.add_parser("set-ct", help="Configure CT ratio and VCT params (instructions 1001+1002)")
    ct.add_argument("--primary",   type=int, required=True, help="CT primary current (A), e.g. 200")
    ct.add_argument("--secondary", type=int, default=5,     help="CT secondary current (A), default 5")

    sd = sub.add_parser("set-dir", help="Set current direction per phase (instruction 1010)")
    sd.add_argument("--l1", default="pos", choices=["pos","rev"])
    sd.add_argument("--l2", default="pos", choices=["pos","rev"])
    sd.add_argument("--l3", default="pos", choices=["pos","rev"])

    sa = sub.add_parser("set-addr", help="Change communication parameters (instruction 1210)")
    sa.add_argument("--new-addr", type=int, required=True, help="New slave address (1-247)")
    sa.add_argument("--baud",     type=int, default=9600, choices=list(_BAUD_INDEX))
    sa.add_argument("--parity",   default="N", choices=["N","O","E"])
    sa.add_argument("--stop",     type=int, default=1, choices=[1, 2])

    sub.add_parser("set-time", help="Sync device clock from system UTC (instruction 1200)")

    rst = sub.add_parser("reset", help="Reset counters (instruction 1301)")
    rst.add_argument("--type", default="energy",
                     choices=list(_RESET_TYPES),
                     help="maxmin | demand | tariff | energy | all")

    return p


def main():
    parser = build_parser()
    args   = parser.parse_args()
    slave  = args.slave_id

    client = _connect()
    try:
        dispatch = {
            "show":     cmd_show,
            "set-sys":  cmd_set_system,
            "set-ct":   cmd_set_ct,
            "set-dir":  cmd_set_direction,
            "set-addr": cmd_set_address,
            "set-time": cmd_set_time,
            "reset":    cmd_reset,
        }
        dispatch[args.cmd](client, slave, args)
    finally:
        client.close()


if __name__ == "__main__":
    main()
