import logging
import requests
from typing import Dict, Any
from ..ports.event_output import EventOutput

logger = logging.getLogger('detector.event')


class HTTPEventAdapter(EventOutput):

    def __init__(self, resiliencia_url: str, linea_id: str = '', timeout: float = 5):
        self.resiliencia_url = resiliencia_url.rstrip('/')
        self.linea_id = linea_id
        self.timeout = float(timeout)
        self._session = requests.Session()

    def send_event(self, event: Dict[str, Any]) -> bool:
        try:
            url = f"{self.resiliencia_url}/events"
            with self._session.post(
                url,
                json=event,
                params={'linea_id': self.linea_id} if self.linea_id else None,
                headers={'Idempotency-Key': str(event.get('event_id', ''))},
                timeout=self.timeout,
            ) as response:
                if 200 <= response.status_code < 300:
                    return True
                logger.warning(
                    'Resiliencia rejected event with HTTP %s',
                    response.status_code,
                )
                return False
        except Exception:
            logger.warning('Failed to send event to resiliencia', exc_info=True)
            return False

    def close(self) -> None:
        self._session.close()
