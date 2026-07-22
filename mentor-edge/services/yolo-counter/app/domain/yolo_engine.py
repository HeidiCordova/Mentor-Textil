import logging
import os
from pathlib import Path
from typing import List, Optional, Tuple
import numpy as np

logger = logging.getLogger('yolo.engine')

MODELS_DIR = Path(os.getenv('MODELS_DIR', '/app/models'))


class YOLOEngine:
    """Wraps ultralytics YOLO model with optional TensorRT export."""

    def __init__(
        self,
        model_name: str = 'yolo26s',
        confidence: float = 0.45,
        classes: Optional[List[int]] = None,
        class_names: Optional[List[str]] = None,
        use_tensorrt: bool = True,
        imgsz: int = 640,
    ):
        self._model_name = model_name
        self._confidence = confidence
        self._classes = classes
        self._class_names = class_names
        self._use_tensorrt = use_tensorrt
        self._imgsz = imgsz
        self._model = None
        self._load()

    def _load(self):
        from ultralytics import YOLO

        MODELS_DIR.mkdir(parents=True, exist_ok=True)
        pt_path = MODELS_DIR / f"{self._model_name}.pt"
        engine_path = MODELS_DIR / f"{self._model_name}.engine"

        if self._use_tensorrt and engine_path.exists():
            logger.info('loading TensorRT engine: %s', engine_path)
            self._model = YOLO(str(engine_path))
            return

        if pt_path.exists():
            logger.info('loading model: %s', pt_path)
            self._model = YOLO(str(pt_path))
        else:
            logger.info('downloading model: %s', self._model_name)
            self._model = YOLO(f"{self._model_name}.pt")
            self._model.save(str(pt_path))

        if self._class_names and hasattr(self._model, 'set_classes'):
            self._model.set_classes(self._class_names)
            logger.info('set open-vocabulary classes: %s', self._class_names)

        if self._use_tensorrt:
            try:
                import torch
                if not torch.cuda.is_available():
                    raise RuntimeError('CUDA not available')
                logger.info('exporting to TensorRT (first run only)...')
                self._model.export(format='engine', imgsz=self._imgsz, half=True, device='0')
                if engine_path.exists():
                    self._model = YOLO(str(engine_path))
                    logger.info('TensorRT engine loaded: %s', engine_path)
            except Exception as exc:
                logger.warning('TensorRT export skipped (%s) — using PyTorch model', exc)
                self._use_tensorrt = False

    def detect(self, frame: np.ndarray) -> list:
        results = self._model(
            frame,
            conf=self._confidence,
            classes=self._classes,
            imgsz=self._imgsz,
            verbose=False,
        )
        return results

    def detect_to_sv(self, frame: np.ndarray):
        import supervision as sv
        results = self.detect(frame)
        if not results:
            return sv.Detections.empty()
        return sv.Detections.from_ultralytics(results[0])

    @property
    def model_name(self) -> str:
        return self._model_name

    def update_config(
        self,
        model_name: Optional[str] = None,
        confidence: Optional[float] = None,
        classes: Optional[List[int]] = None,
        class_names: Optional[List[str]] = None,
    ):
        reload_needed = model_name is not None and model_name != self._model_name
        if confidence is not None:
            self._confidence = confidence
        if classes is not None:
            self._classes = classes
        if class_names is not None:
            self._class_names = class_names
        if reload_needed:
            self._model_name = model_name
            self._load()
