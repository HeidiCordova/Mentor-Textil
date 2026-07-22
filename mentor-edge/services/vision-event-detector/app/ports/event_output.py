from abc import ABC, abstractmethod
from typing import Dict, Any

class EventOutput(ABC):
    @abstractmethod
    def send_event(self, event: Dict[str, Any]) -> bool:
        pass
