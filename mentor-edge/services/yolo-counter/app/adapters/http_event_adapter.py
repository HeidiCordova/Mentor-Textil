import logging
import requests
from typing import Dict, Any
from ..ports.event_output import EventOutput

logger = logging.getLogger('yolo.event')


class HTTPEventAdapter(EventOutput):

    def __init__(self, resiliencia_url: str, linea_id: str = '', timeout: int = 5):
        self._url = resiliencia_url
        self._linea_id = linea_id
        self._timeout = timeout

    def send_event(self, event: Dict[str, Any]) -> bool:
        try:
            url = f"{self._url}/events"
            if self._linea_id:
                url += f"?linea_id={self._linea_id}"
            r = requests.post(url, json=event, timeout=self._timeout)
            return r.status_code in (200, 201)
        except Exception:
            logger.warning('failed to send event to resiliencia')
            return False
