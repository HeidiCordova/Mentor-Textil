#!/usr/bin/env python3
"""
Verificador Simple de Transmisión
Solo recibe y muestra estadísticas del video UDP
Sin procesamiento de IA - Solo para verificar conectividad
"""

import socket
import time
import threading
from datetime import datetime

class VerificadorTransmision:
    def __init__(self, host='0.0.0.0', puerto=5000):
        self.host = host
        self.puerto = puerto
        self.corriendo = False
        self.cliente_ip = None
        
        # Estadísticas
        self.paquetes_recibidos = 0
        self.bytes_recibidos = 0
        self.inicio = time.time()
        self.ultimo_paquete = 0
        self.fps_contador = 0
        self.ultimo_fps_tiempo = time.time()
        self.fps_actual = 0
        
    def iniciar(self):
        """Inicia el receptor UDP"""
        self.corriendo = True
        self.inicio = time.time()
        
        print(f"""
╔══════════════════════════════════════════════════════════════╗
║              VERIFICADOR DE TRANSMISIÓN UDP                  ║
╠══════════════════════════════════════════════════════════════╣
║ 📡 Escuchando en: {self.host}:{self.puerto}                           ║
║ 📱 Configura tu app con esta IP y puerto                    ║
║ ▶️  Presiona 'Iniciar Transmisión' en la app                ║
╚══════════════════════════════════════════════════════════════╝
        """)
        
        # Thread para recibir datos
        receptor_thread = threading.Thread(target=self.recibir_datos, daemon=True)
        receptor_thread.start()
        
        # Thread para mostrar estadísticas
        stats_thread = threading.Thread(target=self.mostrar_estadisticas, daemon=True)
        stats_thread.start()
        
    def recibir_datos(self):
        """Recibe paquetes UDP y cuenta estadísticas"""
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        
        try:
            sock.bind((self.host, self.puerto))
            print(f"✅ Socket UDP creado correctamente en puerto {self.puerto}")
            print("⏳ Esperando datos de la app móvil...")
            
            while self.corriendo:
                try:
                    data, addr = sock.recvfrom(65535)  # Buffer 64KB
                    
                    # Primera conexión
                    if not self.cliente_ip:
                        self.cliente_ip = addr[0]
                        print(f"\n🎉 ¡CONEXIÓN ESTABLECIDA!")
                        print(f"📱 Cliente: {addr[0]}:{addr[1]}")
                        print(f"📦 Primer paquete: {len(data)} bytes")
                        print("🚀 Transmisión iniciada correctamente\n")
                    
                    # Actualizar estadísticas
                    self.paquetes_recibidos += 1
                    self.bytes_recibidos += len(data)
                    self.ultimo_paquete = time.time()
                    self.fps_contador += 1
                    
                    # Calcular FPS cada segundo
                    tiempo_actual = time.time()
                    if tiempo_actual - self.ultimo_fps_tiempo >= 1.0:
                        self.fps_actual = self.fps_contador
                        self.fps_contador = 0
                        self.ultimo_fps_tiempo = tiempo_actual
                    
                    # Log cada 100 paquetes para no saturar
                    if self.paquetes_recibidos % 100 == 0:
                        print(f"📊 Paquete #{self.paquetes_recibidos}: {len(data)} bytes desde {addr[0]}")
                        
                except socket.timeout:
                    continue
                except Exception as e:
                    if self.corriendo:
                        print(f"❌ Error recibiendo datos: {e}")
                        
        except Exception as e:
            print(f"❌ Error en socket: {e}")
        finally:
            sock.close()
            
    def mostrar_estadisticas(self):
        """Muestra estadísticas cada 3 segundos"""
        while self.corriendo:
            time.sleep(3)
            
            if self.paquetes_recibidos > 0:
                tiempo_transcurrido = time.time() - self.inicio
                tiempo_sin_datos = time.time() - self.ultimo_paquete if self.ultimo_paquete > 0 else 0
                
                # Estado de conexión
                if tiempo_sin_datos < 2:
                    estado = "🟢 RECIBIENDO"
                elif tiempo_sin_datos < 5:
                    estado = "🟡 PAUSADO"
                else:
                    estado = "🔴 SIN DATOS"
                
                print(f"""
┌─────────────────────────────────────────────────────────────┐
│                    ESTADÍSTICAS DE RECEPCIÓN                │
├─────────────────────────────────────────────────────────────┤
│ Estado: {estado}                                    │
│ 📱 Cliente: {self.cliente_ip or 'Ninguno'}                                  │
│ ⏱️  Tiempo Activo: {tiempo_transcurrido:.0f}s                              │
│ 📦 Paquetes: {self.paquetes_recibidos}                                     │
│ 📊 Datos: {self.bytes_recibidos/1024/1024:.1f} MB                                │
│ 🎯 FPS: {self.fps_actual}                                           │
│ 📈 Promedio: {self.paquetes_recibidos/tiempo_transcurrido if tiempo_transcurrido > 0 else 0:.1f} paq/s                     │
│ ⏰ Último paquete: hace {tiempo_sin_datos:.1f}s                      │
└─────────────────────────────────────────────────────────────┘
                """)
            else:
                print("⏳ Esperando conexión de la app móvil...")
                
    def detener(self):
        """Detiene el receptor"""
        self.corriendo = False
        tiempo_total = time.time() - self.inicio
        
        print(f"""
╔══════════════════════════════════════════════════════════════╗
║                      RESUMEN FINAL                           ║
╠══════════════════════════════════════════════════════════════╣
║ ⏱️  Tiempo Total: {tiempo_total:.0f}s                                ║
║ 📦 Paquetes Recibidos: {self.paquetes_recibidos}                           ║
║ 📊 MB Totales: {self.bytes_recibidos/1024/1024:.1f}                                ║
║ 📈 Promedio FPS: {self.paquetes_recibidos/tiempo_total if tiempo_total > 0 else 0:.1f}                              ║
║ 📱 Cliente: {self.cliente_ip or 'Ninguno'}                                  ║
╚══════════════════════════════════════════════════════════════╝
        """)
        
        if self.paquetes_recibidos > 0:
            print("✅ VERIFICACIÓN EXITOSA: La app SÍ está transmitiendo")
        else:
            print("❌ NO se recibieron datos. Verificar:")
            print("   - IP correcta en la app")
            print("   - Puerto 5000")
            print("   - Firewall/antivirus")
            print("   - Misma red WiFi")

def main():
    print("🔍 Verificador de Transmisión UDP")
    print("📱 Configura tu app móvil con:")
    
    # Obtener IP local
    try:
        import socket
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip_local = s.getsockname()[0]
        s.close()
        print(f"   IP: {ip_local}")
    except:
        print("   IP: [IP de esta computadora]")
    
    print("   Puerto: 5000")
    print()
    
    verificador = VerificadorTransmision(host='0.0.0.0', puerto=5000)
    
    try:
        verificador.iniciar()
        
        print("⏹️  Presiona Ctrl+C para detener")
        while True:
            time.sleep(1)
            
    except KeyboardInterrupt:
        print("\n🛑 Deteniendo verificador...")
        verificador.detener()
        print("👋 Verificación completada")

if __name__ == "__main__":
    main()