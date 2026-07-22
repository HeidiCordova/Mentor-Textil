import logging
import requests
from typing import Dict, Any
from ..ports.config_port import ConfigPort

logger = logging.getLogger('yolo.config')


class ConfigClient(ConfigPort):

    def __init__(self, config_service_url: str, linea_id: str):
        self._url = config_service_url
        self._linea_id = linea_id
        self._cache: Dict[str, Any] = {}
        self._version = 0

    def get_config(self) -> Dict[str, Any]:
        try:
            r = requests.get(
                f"{self._url}/config",
                params={"linea_id": self._linea_id},
                timeout=5,
            )
            if r.status_code == 200:
                self._cache = r.json()
                self._version = self._cache.get('config_version', 0)
                return self._cache
        except Exception:
            logger.debug('config fetch failed, using cached')
        return self._cache

    def get_config_version(self) -> int:
        try:
            r = requests.get(
                f"{self._url}/config/version",
                params={"linea_id": self._linea_id},
                timeout=2,
            )
            if r.status_code == 200:
                return r.json().get('version', self._version)
        except Exception:
            pass
        return self._version
