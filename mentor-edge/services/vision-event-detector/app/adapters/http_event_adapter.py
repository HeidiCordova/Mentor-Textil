import logging
import requests
from typing import Dict, Any
from ..ports.event_output import EventOutput

logger = logging.getLogger('detector.event')


class HTTPEventAdapter(EventOutput):

    def __init__(self, resiliencia_url: str, linea_id: str = '', timeout: int = 5):
        self.resiliencia_url = resiliencia_url
        self.linea_id = linea_id
        self.timeout = timeout

    def send_event(self, event: Dict[str, Any]) -> bool:
        try:
            url = f"{self.resiliencia_url}/events"
            if self.linea_id:
                url += f"?linea_id={self.linea_id}"
            response = requests.post(
                url,
                json=event,
                timeout=self.timeout,
            )
            return response.status_code in (200, 201)
        except Exception:
            logger.warning('Failed to send event to resiliencia', exc_info=True)
            return False
