from abc import ABC, abstractmethod
from typing import Dict, Any


class ConfigPort(ABC):

    @abstractmethod
    def get_config(self) -> Dict[str, Any]:
        pass

    @abstractmethod
    def get_config_version(self) -> int:
        pass
