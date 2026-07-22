import logging
from typing import Dict, Tuple, Set

logger = logging.getLogger('yolo.counter')


class LineCounter:
    """Counts objects crossing a line (horizontal or vertical) using supervision ByteTrack."""

    def __init__(self, line_y_ratio: float = 0.5, direction: str = 'top_to_bottom', line_x_ratio: float = 0.5):
        self._line_y_ratio = line_y_ratio
        self._line_x_ratio = line_x_ratio
        self._direction = direction
        self._zone = None
        self._tracker = None
        self._frame_shape: Tuple[int, int] = (0, 0)
        self._in_count = 0
        self._out_count = 0
        # Mantener registro de IDs que ya cruzaron para evitar conteo doble durante transiciones
        self._crossed_ids: Set[int] = set()
        # Límite de IDs históricos para evitar crecimiento infinito de memoria
        self._max_crossed_ids = 1000

    @property
    def is_horizontal_dir(self) -> bool:
        return self._direction in ('left_to_right', 'right_to_left')

    def _ensure_zone(self, h: int, w: int):
        if self._frame_shape == (h, w) and self._zone is not None:
            return
        import supervision as sv
        self._frame_shape = (h, w)
        if self.is_horizontal_dir:
            x = int(w * self._line_x_ratio)
            start = sv.Point(x, 0)
            end = sv.Point(x, h)
        else:
            y = int(h * self._line_y_ratio)
            start = sv.Point(0, y)
            end = sv.Point(w, y)
        self._zone = sv.LineZone(start=start, end=end)
        
        # Solo crear un nuevo tracker si no existe
        # Esto preserva los IDs de tracking durante cambios de configuración
        if self._tracker is None:
            self._tracker = sv.ByteTrack(
                track_activation_threshold=0.25,
                lost_track_buffer=30,  # Aumentado de 15 a 30 para mejor persistencia con frame_skip
                minimum_matching_threshold=0.8,
                frame_rate=15,
            )
        
        self._in_count = 0
        self._out_count = 0

    def update(self, detections, frame_h: int, frame_w: int) -> int:
        self._ensure_zone(frame_h, frame_w)
        tracked = self._tracker.update_with_detections(detections)
        prev_in = self._zone.in_count
        prev_out = self._zone.out_count
        
        # Obtener los IDs que están actualmente siendo trackeados
        current_tracked_ids = set(tracked.tracker_id) if hasattr(tracked, 'tracker_id') and len(tracked) > 0 else set()
        
        self._zone.trigger(tracked)

        # Determinar qué contador usar según la dirección
        if self._direction in ('top_to_bottom', 'left_to_right'):
            raw_delta = self._zone.in_count - prev_in
        else:
            raw_delta = self._zone.out_count - prev_out
        
        # Si no hubo cruces, retornar 0
        if raw_delta <= 0:
            return 0
        
        # Estrategia mejorada: solo contar IDs que están activos Y no han sido contados antes
        # Esto previene conteo doble durante transiciones de panel
        actual_delta = 0
        
        if len(current_tracked_ids) > 0:
            # Identificar qué IDs probablemente cruzaron
            # Asumimos que los objetos detectados cerca de la línea son los que cruzaron
            for track_id in current_tracked_ids:
                if track_id not in self._crossed_ids:
                    self._crossed_ids.add(track_id)
                    actual_delta += 1
                    logger.debug('object %d crossed line (total crossed: %d)', track_id, len(self._crossed_ids))
            
            # Limitar el tamaño del set para evitar crecimiento infinito
            if len(self._crossed_ids) > self._max_crossed_ids:
                # Remover los IDs más antiguos (aproximadamente la mitad)
                ids_to_remove = list(self._crossed_ids)[:self._max_crossed_ids // 2]
                self._crossed_ids.difference_update(ids_to_remove)
                logger.debug('cleaned old crossed IDs, remaining: %d', len(self._crossed_ids))
        else:
            # Fallback: si no hay IDs disponibles, usar el delta raw
            # Esto puede pasar si el tracker falla o en frames sin detecciones
            actual_delta = raw_delta
            logger.warning('no tracker IDs available, using raw delta: %d', raw_delta)
        
        # Asegurar que no contamos más de lo que la zona reportó
        actual_delta = min(actual_delta, raw_delta)
        
        return actual_delta

    @property
    def total_count(self) -> int:
        if self._zone is None:
            return 0
        if self._direction in ('top_to_bottom', 'left_to_right'):
            return self._zone.in_count
        return self._zone.out_count

    @property
    def line_y_ratio(self) -> float:
        return self._line_y_ratio

    @property
    def line_x_ratio(self) -> float:
        return self._line_x_ratio

    def update_line(self, y_ratio: float = None, x_ratio: float = None, direction: str = None):
        changed = False
        if direction is not None and direction != self._direction:
            self._direction = direction
            changed = True
        if y_ratio is not None and y_ratio != self._line_y_ratio:
            self._line_y_ratio = y_ratio
            changed = True
        if x_ratio is not None and x_ratio != self._line_x_ratio:
            self._line_x_ratio = x_ratio
            changed = True
        if changed:
            # Solo resetear la zona, NO el tracker
            # Esto preserva los IDs de tracking durante cambios de configuración
            self._frame_shape = (0, 0)
            self._zone = None
            # NO resetear self._tracker ni self._crossed_ids
            logger.info('line updated: y=%.2f, x=%.2f, dir=%s (tracker preserved)', 
                       self._line_y_ratio, self._line_x_ratio, self._direction)

    def reset(self):
        self._frame_shape = (0, 0)
        self._zone = None
        self._tracker = None
        self._in_count = 0
        self._out_count = 0
        self._crossed_ids.clear()
