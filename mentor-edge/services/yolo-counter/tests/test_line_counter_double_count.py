"""
Tests para verificar que las soluciones de conteo doble funcionan correctamente.

Estos tests verifican:
1. Que el tracker se preserve durante cambios de configuración
2. Que no se cuenten objetos dos veces con el mismo ID
3. Que el sistema maneje correctamente detecciones vacías (frame_skip)
"""

import pytest
import numpy as np
from unittest.mock import Mock, MagicMock
import sys
import os

# Agregar el path del módulo
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'app'))

from domain.line_counter import LineCounter


class MockDetections:
    """Mock de supervision.Detections para testing"""
    def __init__(self, tracker_ids=None, xyxy=None, confidence=None):
        self.tracker_id = np.array(tracker_ids if tracker_ids else [])
        self.xyxy = np.array(xyxy if xyxy else [])
        self.confidence = np.array(confidence if confidence else [])
    
    def __len__(self):
        return len(self.tracker_id)


class TestLineCounterDoubleCount:
    """Tests para prevención de conteo doble"""
    
    def test_tracker_preserved_on_line_update(self):
        """Verificar que el tracker se preserve al actualizar la línea"""
        counter = LineCounter(line_y_ratio=0.5, direction='top_to_bottom')
        
        # Simular primer frame para inicializar el tracker
        detections = MockDetections(tracker_ids=[1], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
        counter.update(detections, 480, 640)
        
        # Guardar referencia al tracker original
        original_tracker = counter._tracker
        
        # Actualizar la línea
        counter.update_line(y_ratio=0.6)
        
        # Verificar que el tracker es el mismo objeto
        assert counter._tracker is original_tracker, "El tracker debe preservarse durante update_line()"
        assert counter._tracker is not None, "El tracker no debe ser None"
    
    def test_no_double_count_same_id(self):
        """Verificar que un objeto con el mismo ID no se cuente dos veces"""
        counter = LineCounter(line_y_ratio=0.5, direction='top_to_bottom')
        
        # Mock de supervision para simular cruce de línea
        import supervision as sv
        
        # Primer frame: objeto con ID 1 cruza la línea
        detections1 = MockDetections(tracker_ids=[1], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
        
        # Mockear el tracker y la zona
        counter._tracker = Mock()
        counter._zone = Mock()
        counter._zone.in_count = 0
        counter._zone.out_count = 0
        
        tracked1 = MockDetections(tracker_ids=[1], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
        counter._tracker.update_with_detections.return_value = tracked1
        
        # Simular que la zona detectó un cruce
        counter._zone.trigger = Mock(side_effect=lambda x: setattr(counter._zone, 'in_count', 1))
        
        delta1 = counter.update(detections1, 480, 640)
        assert delta1 == 1, "Primer cruce debe contar 1"
        
        # Segundo frame: mismo objeto (ID 1) todavía visible
        # No debe contarse de nuevo
        counter._zone.in_count = 1  # Sin cambio
        tracked2 = MockDetections(tracker_ids=[1], xyxy=[[100, 200, 150, 250]], confidence=[0.9])
        counter._tracker.update_with_detections.return_value = tracked2
        counter._zone.trigger = Mock()  # Sin nuevo cruce
        
        delta2 = counter.update(detections1, 480, 640)
        assert delta2 == 0, "Mismo ID no debe contarse dos veces"
    
    def test_empty_detections_no_crash(self):
        """Verificar que detecciones vacías (frame_skip) no causen errores"""
        counter = LineCounter(line_y_ratio=0.5, direction='top_to_bottom')
        
        # Primer frame con detección
        detections1 = MockDetections(tracker_ids=[1], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
        counter.update(detections1, 480, 640)
        
        # Frame vacío (simulando frame_skip)
        empty_detections = MockDetections(tracker_ids=[], xyxy=[], confidence=[])
        
        # No debe crashear
        try:
            delta = counter.update(empty_detections, 480, 640)
            assert delta == 0, "Detecciones vacías deben retornar delta 0"
        except Exception as e:
            pytest.fail(f"Detecciones vacías causaron error: {e}")
    
    def test_crossed_ids_cleanup(self):
        """Verificar que el set de IDs cruzados se limpie cuando alcanza el límite"""
        counter = LineCounter(line_y_ratio=0.5, direction='top_to_bottom')
        counter._max_crossed_ids = 10  # Límite bajo para testing
        
        # Mockear el tracker y la zona
        counter._tracker = Mock()
        counter._zone = Mock()
        counter._zone.in_count = 0
        counter._zone.out_count = 0
        
        # Simular muchos objetos cruzando
        for i in range(15):
            detections = MockDetections(tracker_ids=[i], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
            tracked = MockDetections(tracker_ids=[i], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
            counter._tracker.update_with_detections.return_value = tracked
            
            # Simular cruce
            prev_count = counter._zone.in_count
            counter._zone.in_count = prev_count + 1
            counter._zone.trigger = Mock()
            
            counter.update(detections, 480, 640)
        
        # Verificar que el set no creció indefinidamente
        assert len(counter._crossed_ids) <= counter._max_crossed_ids, \
            f"Set de IDs debe limitarse a {counter._max_crossed_ids}, pero tiene {len(counter._crossed_ids)}"
    
    def test_reset_clears_crossed_ids(self):
        """Verificar que reset() limpia el set de IDs cruzados"""
        counter = LineCounter(line_y_ratio=0.5, direction='top_to_bottom')
        
        # Agregar algunos IDs
        counter._crossed_ids.add(1)
        counter._crossed_ids.add(2)
        counter._crossed_ids.add(3)
        
        assert len(counter._crossed_ids) == 3, "Debe haber 3 IDs antes del reset"
        
        # Reset
        counter.reset()
        
        assert len(counter._crossed_ids) == 0, "Reset debe limpiar todos los IDs cruzados"
        assert counter._tracker is None, "Reset debe limpiar el tracker"
        assert counter._zone is None, "Reset debe limpiar la zona"


class TestLineCounterConfiguration:
    """Tests para cambios de configuración"""
    
    def test_direction_change_preserves_tracker(self):
        """Verificar que cambiar dirección preserve el tracker"""
        counter = LineCounter(line_y_ratio=0.5, direction='top_to_bottom')
        
        # Inicializar
        detections = MockDetections(tracker_ids=[1], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
        counter.update(detections, 480, 640)
        original_tracker = counter._tracker
        
        # Cambiar dirección
        counter.update_line(direction='bottom_to_top')
        
        assert counter._tracker is original_tracker, "Cambiar dirección debe preservar el tracker"
        assert counter._direction == 'bottom_to_top', "Dirección debe actualizarse"
    
    def test_line_position_change_preserves_tracker(self):
        """Verificar que cambiar posición de línea preserve el tracker"""
        counter = LineCounter(line_y_ratio=0.5, direction='top_to_bottom')
        
        # Inicializar
        detections = MockDetections(tracker_ids=[1], xyxy=[[100, 100, 150, 150]], confidence=[0.9])
        counter.update(detections, 480, 640)
        original_tracker = counter._tracker
        
        # Cambiar posición
        counter.update_line(y_ratio=0.7)
        
        assert counter._tracker is original_tracker, "Cambiar posición debe preservar el tracker"
        assert counter._line_y_ratio == 0.7, "Posición debe actualizarse"


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
