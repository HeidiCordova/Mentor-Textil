# Archivos Importantes - Mentor Mobile

## 📁 Estructura Final Organizada

### 📖 Documentación Principal
- `README.md` - **Descripción completa del proyecto**
- `GUIA_COMPLETA.md` - **Manual de uso detallado**
- `HISTORIAL_DESARROLLO.md` - **Cronología y decisiones técnicas**
- `ARCHIVOS_IMPORTANTES.md` - **Este archivo (índice)**

### 🔧 Scripts de Desarrollo
- `COMPILAR_CON_ANDROID_STUDIO.md` - Guía de compilación
- `VERIFICAR_RAPIDO.bat` - Verificar transmisión UDP
- `INSTALAR_SERVIDOR.bat` - Instalar servidor Edge con IA

### 🐍 Servidores Python
- `verificar_transmision.py` - **Receptor simple para pruebas**
- `servidor_edge_completo.py` - **Servidor completo con IA**
- `RECIBIR_VIDEO_COMPLETO.py` - Receptor que guarda video

### 📱 Código Android (Principales)
```
app/src/main/
├── kotlin/com/mentor/mobile/
│   ├── ui/MainActivity.kt              ⭐ Actividad principal
│   ├── ui/CameraOverlayView.kt         ⭐ Overlay de guía
│   ├── gstreamer/GStreamerManager.kt   ⭐ Gestor de video
│   ├── camera/CameraManager.kt         ⭐ Gestor de cámara
│   └── network/EdgeConnectionManager.kt ⭐ Conexión Edge
├── cpp/gstreamer_pipeline.cpp          ⭐ Pipeline nativo
└── res/layout/activity_main.xml        ⭐ Layout principal
```

### 🗂️ Archivos de Configuración
- `build.gradle.kts` - Configuración del proyecto
- `app/build.gradle.kts` - Configuración de la app
- `app/src/main/cpp/CMakeLists.txt` - Configuración nativa
- `gradle.properties` - Propiedades de Gradle
- `local.properties` - Configuración local (SDK paths)

### 📋 Archivos de Soporte
- `gradlew.bat` / `gradlew` - Gradle wrapper
- `.gitignore` - Archivos ignorados por Git
- `settings.gradle.kts` - Configuración de módulos

---

## 🎯 Archivos por Tarea

### Para Compilar la App
1. `README.md` - Instrucciones básicas
2. `COMPILAR_CON_ANDROID_STUDIO.md` - Guía detallada
3. Android Studio + carpeta `mentor-mobile/`

### Para Verificar Transmisión
1. `VERIFICAR_RAPIDO.bat` - Script automático
2. `verificar_transmision.py` - Receptor Python

### Para Servidor Edge Completo
1. `INSTALAR_SERVIDOR.bat` - Instalación automática
2. `servidor_edge_completo.py` - Servidor con IA

### Para Entender el Código
1. `GUIA_COMPLETA.md` - Explicación técnica
2. `HISTORIAL_DESARROLLO.md` - Decisiones y problemas
3. Código fuente en `app/src/main/`

### Para la Tesis
1. `README.md` - Descripción del sistema
2. `HISTORIAL_DESARROLLO.md` - Proceso de desarrollo
3. `GUIA_COMPLETA.md` - Métricas y resultados
4. Logs de la app (latencia, FPS, etc.)

---

## 🚮 Archivos Eliminados (Limpieza)

### Documentación Obsoleta (50+ archivos)
- Múltiples README duplicados
- Checklists obsoletos
- Guías de compilación antiguas
- Archivos de progreso temporales
- Soluciones a problemas ya resueltos

### Scripts Obsoletos (20+ archivos)
- Scripts PowerShell antiguos
- Múltiples versiones de compilación
- Configuraciones de GStreamer obsoletas
- Scripts de verificación duplicados

### Archivos de Debug Temporales
- Logs de compilación antiguos
- Archivos de diagnóstico
- Configuraciones de prueba

---

## ✅ Estado Final

### Archivos Totales: ~25 (antes: ~95)
- **Reducción**: 70+ archivos eliminados
- **Organización**: 4 categorías claras
- **Documentación**: 4 archivos principales
- **Scripts**: 3 archivos esenciales
- **Código**: Estructura limpia

### Beneficios de la Limpieza
- ✅ **Navegación más fácil**
- ✅ **Información consolidada**
- ✅ **Sin duplicados**
- ✅ **Archivos actualizados**
- ✅ **Estructura clara**

---

## 🎯 Próximos Pasos

### Inmediatos
1. **Compilar app**: Usar `README.md`
2. **Verificar transmisión**: Usar `VERIFICAR_RAPIDO.bat`
3. **Servidor Edge**: Usar `INSTALAR_SERVIDOR.bat`

###
1. **Documentar métricas**: Usar datos de `GUIA_COMPLETA.md`
2. **Escribir capítulos**: Usar `HISTORIAL_DESARROLLO.md`
3. **Crear presentación**: Usar `README.md`

---

**Limpieza completada**: 2026-04-19  
**Archivos organizados**: ✅  
**Documentación consolidada**: ✅  
**Listo para uso**: ✅