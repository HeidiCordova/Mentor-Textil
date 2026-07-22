import hashlib
import json
from typing import Any, Mapping, Sequence
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

from .model import HISTOGRAM_ALGORITHM_PARAMETERS, HISTOGRAM_ALGORITHM_VERSION


def _safe_camera_identity(camera_url: str) -> str:
    """Return a stable camera identity without persisting RTSP credentials."""
    if not camera_url:
        return ""

    parsed = urlsplit(camera_url)
    if not parsed.scheme or not parsed.hostname:
        return camera_url.split("@")[-1]

    host = parsed.hostname.lower()
    if parsed.port:
        host = f"{host}:{parsed.port}"

    # Keep stream selectors such as channel/subtype, but remove credentials
    # and rotating signatures that should not invalidate a visual reference.
    sensitive_parts = ("token", "password", "passwd", "secret", "auth", "signature")
    safe_query = [
        (key, value)
        for key, value in parse_qsl(parsed.query, keep_blank_values=True)
        if not any(part in key.lower() for part in sensitive_parts)
    ]
    safe_query.sort()
    return urlunsplit((
        parsed.scheme.lower(),
        host,
        parsed.path,
        urlencode(safe_query),
        "",
    ))


def build_calibration_fingerprint(
    *,
    roi: Sequence[int],
    camera_url: str,
    camera_config: Mapping[str, Any] | None = None,
    signal_scale: float = 1.0,
    algorithm_version: str = HISTOGRAM_ALGORITHM_VERSION,
) -> str:
    """Build a deterministic SHA-256 compatibility key for a calibration."""
    camera = dict(camera_config or {})
    visual_keys = {
        "width", "height", "resolution", "exposure", "auto_exposure",
        "gain", "brightness", "contrast", "saturation", "hue",
        "white_balance", "auto_white_balance", "frame_backend",
        "signal_scale",
    }
    safe_camera = {key: value for key, value in camera.items() if key in visual_keys}
    safe_camera["source"] = _safe_camera_identity(camera_url)

    payload = {
        "algorithm_version": algorithm_version,
        "histogram": HISTOGRAM_ALGORITHM_PARAMETERS,
        "roi": [int(value) for value in roi],
        "signal_scale": round(float(signal_scale), 6),
        "camera": safe_camera,
    }
    canonical = json.dumps(
        payload,
        sort_keys=True,
        separators=(",", ":"),
        ensure_ascii=True,
    )
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()
