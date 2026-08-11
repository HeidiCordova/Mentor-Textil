from abc import ABC, abstractmethod
from typing import Dict, Any


class EventOutputError(RuntimeError):
    """Raised when an event cannot be accepted without risking data loss."""


class EventOutput(ABC):
    @abstractmethod
    def send_event(self, event: Dict[str, Any]) -> bool:
        """Return True once the event has been accepted by the output.

        An output may deliver synchronously or commit to a durable local queue
        for asynchronous delivery.
        """
        raise NotImplementedError
