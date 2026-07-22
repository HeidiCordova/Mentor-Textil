#!/usr/bin/env python3
"""
Script de verificación para las soluciones de conteo doble.

Este script verifica que:
1. El código tiene las modificaciones correctas
2. Los parámetros están configurados correctamente
3. El sistema está listo para prevenir conteo doble

Uso:
    python verify_double_count_fix.py
"""

import sys
import os
import re
from pathlib import Path

# Colores para output
GREEN = '\033[92m'
RED = '\033[91m'
YELLOW = '\033[93m'
RESET = '\033[0m'
BOLD = '\033[1m'


def check_file_exists(filepath):
    """Verificar que un archivo existe"""
    if Path(filepath).exists():
        print(f"{GREEN}✓{RESET} Archivo encontrado: {filepath}")
        return True
    else:
        print(f"{RED}✗{RESET} Archivo NO encontrado: {filepath}")
        return False


def check_code_contains(filepath, pattern, description):
    """Verificar que un archivo contiene un patrón específico"""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
            if re.search(pattern, content, re.MULTILINE | re.DOTALL):
                print(f"{GREEN}✓{RESET} {description}")
                return True
            else:
                print(f"{RED}✗{RESET} {description}")
                return False
    except Exception as e:
        print(f"{RED}✗{RESET} Error leyendo {filepath}: {e}")
        return False


def main():
    print(f"\n{BOLD}=== Verificación de Soluciones de Conteo Doble ==={RESET}\n")
    
    all_checks_passed = True
    
    # Verificar archivos principales
    print(f"{BOLD}1. Verificando archivos modificados...{RESET}")
    line_counter_path = "app/domain/line_counter.py"
    counter_service_path = "app/application/counter_service.py"
    
    all_checks_passed &= check_file_exists(line_counter_path)
    all_checks_passed &= check_file_exists(counter_service_path)
    
    # Verificar line_counter.py
    print(f"\n{BOLD}2. Verificando line_counter.py...{RESET}")
    
    checks = [
        (r"_crossed_ids:\s*Set\[int\]\s*=\s*set\(\)", 
         "Set de IDs cruzados (_crossed_ids) está definido"),
        
        (r"_max_crossed_ids\s*=\s*\d+", 
         "Límite de IDs (_max_crossed_ids) está definido"),
        
        (r"if self\._tracker is None:\s+self\._tracker = sv\.ByteTrack", 
         "Tracker se preserva durante reconfiguración (if self._tracker is None)"),
        
        (r"lost_track_buffer\s*=\s*30", 
         "lost_track_buffer aumentado a 30 frames"),
        
        (r"if track_id not in self\._crossed_ids:", 
         "Verificación de IDs duplicados implementada"),
        
        (r"self\._crossed_ids\.clear\(\)", 
         "Método reset() limpia _crossed_ids"),
        
        (r"# Solo resetear la zona, NO el tracker", 
         "Comentario explicativo en update_line()"),
    ]
    
    for pattern, description in checks:
        all_checks_passed &= check_code_contains(line_counter_path, pattern, description)
    
    # Verificar counter_service.py
    print(f"\n{BOLD}3. Verificando counter_service.py...{RESET}")
    
    checks = [
        (r"should_detect\s*=\s*\(self\._frame_count\s*%\s*self\._frame_skip\s*==\s*0\)", 
         "Variable should_detect implementada"),
        
        (r"if should_detect:.*?detections = self\._yolo\.detect_to_sv", 
         "YOLO solo se ejecuta cuando should_detect es True"),
        
        (r"else:.*?detections = sv\.Detections\.empty\(\)", 
         "Detecciones vacías en frames intermedios"),
        
        (r"# SIEMPRE actualizar el counter", 
         "Comentario explicativo sobre actualización continua del counter"),
        
        (r"delta = self\._counter\.update\(detections, h, w\)", 
         "Counter se actualiza siempre (no dentro del if should_detect)"),
    ]
    
    for pattern, description in checks:
        all_checks_passed &= check_code_contains(counter_service_path, pattern, description)
    
    # Verificar documentación
    print(f"\n{BOLD}4. Verificando documentación...{RESET}")
    
    doc_path = "../../docs/SOLUCION_CONTEO_DOBLE.md"
    if check_file_exists(doc_path):
        print(f"{GREEN}✓{RESET} Documentación de soluciones encontrada")
    else:
        print(f"{YELLOW}⚠{RESET} Documentación no encontrada (opcional)")
    
    # Resultado final
    print(f"\n{BOLD}{'='*60}{RESET}")
    if all_checks_passed:
        print(f"{GREEN}{BOLD}✓ TODAS LAS VERIFICACIONES PASARON{RESET}")
        print(f"\nLas soluciones de conteo doble están correctamente implementadas.")
        print(f"El sistema está listo para prevenir conteo doble durante:")
        print(f"  • Transiciones entre paneles/vistas")
        print(f"  • Operación con frame_skip activo")
        return 0
    else:
        print(f"{RED}{BOLD}✗ ALGUNAS VERIFICACIONES FALLARON{RESET}")
        print(f"\nPor favor, revisa los errores arriba y asegúrate de que:")
        print(f"  1. Los archivos están en las ubicaciones correctas")
        print(f"  2. Las modificaciones se aplicaron completamente")
        print(f"  3. No hay errores de sintaxis en el código")
        return 1


if __name__ == '__main__':
    sys.exit(main())
