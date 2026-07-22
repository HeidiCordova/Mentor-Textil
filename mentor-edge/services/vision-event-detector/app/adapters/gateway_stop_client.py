"""HTTP client for creating / closing stops via the edge-gateway REST API."""

import json
import logging
import urllib.request
from datetime import datetime
from typing import Optional

logger = logging.getLogger('detector.stop_client')


class GatewayStopClient:
    """Thin HTTP adapter — talks to edge-gateway ``/edge/stops``."""

    def __init__(self, gateway_url: str, device_id: str, timeout: int = 3) -> None:
        self._base = (gateway_url or '').rstrip('/')
        self._device_id = device_id
        self._timeout = timeout

    @property
    def enabled(self) -> bool:
        return bool(self._base)

    def create_stop(
        self,
        stop_type: str,
        started_at: datetime,
        ended_at: Optional[datetime] = None,
    ) -> Optional[str]:
        """POST /edge/stops → returns stop_id or None on failure."""
        if not self._base:
            return None
        body: dict = {
            'device_id': self._device_id,
            'stop_type': stop_type,
            'started_at': started_at.isoformat(),
            'source': 'detector',
        }
        if ended_at is not None:
            body['ended_at'] = ended_at.isoformat()
        try:
            data = json.dumps(body).encode()
            req = urllib.request.Request(
                f'{self._base}/edge/stops',
                data=data,
                headers={'Content-Type': 'application/json'},
                method='POST',
            )
            with urllib.request.urlopen(req, timeout=self._timeout) as resp:
                result = json.loads(resp.read().decode())
                stop_id = result.get('stop_id', '')
                logger.info('Created stop %s (%s)', stop_id, stop_type)
                return stop_id or None
        except Exception as exc:
            logger.warning('Failed to create stop: %s', exc)
            return None

    def close_stop(self, stop_id: str) -> bool:
        """POST /edge/stops/{stop_id}/close → returns True on success."""
        if not self._base or not stop_id:
            return False
        try:
            req = urllib.request.Request(
                f'{self._base}/edge/stops/{stop_id}/close',
                data=b'{}',
                headers={'Content-Type': 'application/json'},
                method='POST',
            )
            with urllib.request.urlopen(req, timeout=self._timeout) as resp:
                logger.info('Closed stop %s', stop_id)
                return resp.status in (200, 201)
        except Exception as exc:
            logger.warning('Failed to close stop %s: %s', stop_id, exc)
            return False

    def find_open_detector_stop(self) -> Optional[str]:
        """GET /edge/stops?source=detector&open=true&limit=1 → stop_id or None."""
        if not self._base:
            return None
        try:
            url = f'{self._base}/edge/stops?open=true&limit=1'
            with urllib.request.urlopen(url, timeout=self._timeout) as resp:
                stops = json.loads(resp.read().decode())
            for s in stops:
                if s.get('source') == 'detector':
                    return s.get('stop_id')
            return None
        except Exception as exc:
            logger.warning('Failed to query open detector stops: %s', exc)
            return None
