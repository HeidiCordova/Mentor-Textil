import logging
import requests
from typing import Dict, Any
from ..ports.config_port import ConfigPort

logger = logging.getLogger('detector.config')


class ConfigClient(ConfigPort):

    def __init__(self, config_service_url: str, linea_id: str):
        self.config_service_url = config_service_url
        self.linea_id = linea_id
        self._cached_config: Dict[str, Any] = {}
        self._cached_version = 0

    def get_config(self) -> Dict[str, Any]:
        try:
            response = requests.get(
                f"{self.config_service_url}/config",
                params={"linea_id": self.linea_id},
                timeout=5,
            )
            if response.status_code == 200:
                self._cached_config = response.json()
                self._cached_version = self._cached_config.get('config_version', 0)
                return self._cached_config
        except Exception:
            logger.debug('Config fetch failed, using cached')
        return self._cached_config

    def get_config_version(self) -> int:
        try:
            response = requests.get(
                f"{self.config_service_url}/config/version",
                params={"linea_id": self.linea_id},
                timeout=2,
            )
            if response.status_code == 200:
                data = response.json()
                return data.get('version', self._cached_version)
        except Exception:
            logger.debug('Config version fetch failed, using cached')
        return self._cached_version
