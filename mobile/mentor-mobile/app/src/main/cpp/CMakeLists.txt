cmake_minimum_required(VERSION 3.22.1)
project(gstreamer_pipeline)

# Configurar GStreamer - Usar variable pasada desde build.gradle.kts
# La ruta debe configurarse en local.properties o como variable de entorno
if(NOT DEFINED GSTREAMER_ROOT)
    # Intentar obtener de variable de entorno
    if(DEFINED ENV{GSTREAMER_ROOT})
        set(GSTREAMER_ROOT "$ENV{GSTREAMER_ROOT}")
    else()
        # Ruta por defecto para desarrollo local
        set(GSTREAMER_ROOT "F:/gstreamer-android/gstreamer-1.0-android-universal-1.28.2")
    endif()
endif()

# Mapear arquitecturas de Android a directorios de GStreamer
# Android usa: arm64-v8a, armeabi-v7a, x86, x86_64
# GStreamer usa: arm64, armv7, x86, x86_64
if(ANDROID_ABI STREQUAL "arm64-v8a")
    set(GSTREAMER_ARCH "arm64")
elseif(ANDROID_ABI STREQUAL "armeabi-v7a")
    set(GSTREAMER_ARCH "armv7")
elseif(ANDROID_ABI STREQUAL "x86")
    set(GSTREAMER_ARCH "x86")
elseif(ANDROID_ABI STREQUAL "x86_64")
    set(GSTREAMER_ARCH "x86_64")
else()
    message(FATAL_ERROR "Unsupported Android ABI: ${ANDROID_ABI}")
endif()

# La estructura es: F:/si/arm64/include y F:/si/arm64/lib
set(GSTREAMER_ARCH_DIR "${GSTREAMER_ROOT}/${GSTREAMER_ARCH}")
set(GSTREAMER_INCLUDE_DIR "${GSTREAMER_ARCH_DIR}/include")
set(GSTREAMER_LIB_DIR "${GSTREAMER_ARCH_DIR}/lib")

message(STATUS "GStreamer Root: ${GSTREAMER_ROOT}")
message(STATUS "Android ABI: ${ANDROID_ABI}")
message(STATUS "GStreamer Arch: ${GSTREAMER_ARCH}")
message(STATUS "GStreamer Arch Dir: ${GSTREAMER_ARCH_DIR}")
message(STATUS "GStreamer Include: ${GSTREAMER_INCLUDE_DIR}")
message(STATUS "GStreamer Lib: ${GSTREAMER_LIB_DIR}")

# Incluir directorios
include_directories(
    ${GSTREAMER_INCLUDE_DIR}
    ${GSTREAMER_INCLUDE_DIR}/gstreamer-1.0
    ${GSTREAMER_INCLUDE_DIR}/glib-2.0
    ${GSTREAMER_LIB_DIR}/glib-2.0/include
    ${GSTREAMER_INCLUDE_DIR}/gstreamer-1.0/gst/app
)

# Crear librería compartida
add_library(gstreamer_pipeline SHARED gstreamer_pipeline.cpp)

# Configurar para usar bibliotecas estáticas de GStreamer (correcto para Android)
# Las bibliotecas .a se linkean estáticamente y se incluyen en el .so final

# Definir rutas directas a las bibliotecas estáticas
set(GSTREAMER_LIB "${GSTREAMER_LIB_DIR}/libgstreamer-1.0.a")
set(GSTAPP_LIB "${GSTREAMER_LIB_DIR}/libgstapp-1.0.a")
set(GSTVIDEO_LIB "${GSTREAMER_LIB_DIR}/libgstvideo-1.0.a")
set(GSTBASE_LIB "${GSTREAMER_LIB_DIR}/libgstbase-1.0.a")
set(GOBJECT_LIB "${GSTREAMER_LIB_DIR}/libgobject-2.0.a")
set(GLIB_LIB "${GSTREAMER_LIB_DIR}/libglib-2.0.a")
set(GMODULE_LIB "${GSTREAMER_LIB_DIR}/libgmodule-2.0.a")
set(GIO_LIB "${GSTREAMER_LIB_DIR}/libgio-2.0.a")
set(GTHREAD_LIB "${GSTREAMER_LIB_DIR}/libgthread-2.0.a")
set(INTL_LIB "${GSTREAMER_LIB_DIR}/libintl.a")
set(ICONV_LIB "${GSTREAMER_LIB_DIR}/libiconv.a")
set(FFI_LIB "${GSTREAMER_LIB_DIR}/libffi.a")
set(PCRE_LIB "${GSTREAMER_LIB_DIR}/libpcre2-8.a")
set(Z_LIB "${GSTREAMER_LIB_DIR}/libz.a")

# Bibliotecas adicionales de GStreamer necesarias para plugins
set(X264_LIB "${GSTREAMER_LIB_DIR}/libx264.a")
set(GSTPBUTILS_LIB "${GSTREAMER_LIB_DIR}/libgstpbutils-1.0.a")
set(GSTCODECPARSERS_LIB "${GSTREAMER_LIB_DIR}/libgstcodecparsers-1.0.a")
set(GSTAUDIO_LIB "${GSTREAMER_LIB_DIR}/libgstaudio-1.0.a")
set(GSTTAG_LIB "${GSTREAMER_LIB_DIR}/libgsttag-1.0.a")
set(GSTRTP_LIB "${GSTREAMER_LIB_DIR}/libgstrtp-1.0.a")
set(GSTNET_LIB "${GSTREAMER_LIB_DIR}/libgstnet-1.0.a")
set(GSTALLOCATORS_LIB "${GSTREAMER_LIB_DIR}/libgstallocators-1.0.a")
set(GSTORC_LIB "${GSTREAMER_LIB_DIR}/liborc-0.4.a")

# Plugins de GStreamer necesarios
set(GST_PLUGIN_DIR "${GSTREAMER_LIB_DIR}/gstreamer-1.0")
set(PLUGIN_COREELEMENTS "${GST_PLUGIN_DIR}/libgstcoreelements.a")
set(PLUGIN_VIDEOCONVERTSCALE "${GST_PLUGIN_DIR}/libgstvideoconvertscale.a")
set(PLUGIN_X264 "${GST_PLUGIN_DIR}/libgstx264.a")
set(PLUGIN_VIDEOPARSERSBAD "${GST_PLUGIN_DIR}/libgstvideoparsersbad.a")
set(PLUGIN_RTP "${GST_PLUGIN_DIR}/libgstrtp.a")
set(PLUGIN_UDP "${GST_PLUGIN_DIR}/libgstudp.a")
# PLUGIN_ANDROIDMEDIA comentado porque requiere libgstgl-1.0.a y libgstphotography-1.0.a
# set(PLUGIN_ANDROIDMEDIA "${GST_PLUGIN_DIR}/libgstandroidmedia.a")
set(PLUGIN_APP "${GST_PLUGIN_DIR}/libgstapp.a")
set(PLUGIN_VIDEOTESTSRC "${GST_PLUGIN_DIR}/libgstvideotestsrc.a")

# Verificar que las bibliotecas existen
message(STATUS "Verificando bibliotecas:")
message(STATUS "GSTREAMER_LIB: ${GSTREAMER_LIB}")
message(STATUS "GSTAPP_LIB: ${GSTAPP_LIB}")
message(STATUS "GSTVIDEO_LIB: ${GSTVIDEO_LIB}")
message(STATUS "GSTBASE_LIB: ${GSTBASE_LIB}")
message(STATUS "GOBJECT_LIB: ${GOBJECT_LIB}")
message(STATUS "GLIB_LIB: ${GLIB_LIB}")
message(STATUS "GMODULE_LIB: ${GMODULE_LIB}")
message(STATUS "GIO_LIB: ${GIO_LIB}")
message(STATUS "GTHREAD_LIB: ${GTHREAD_LIB}")
message(STATUS "INTL_LIB: ${INTL_LIB}")
message(STATUS "ICONV_LIB: ${ICONV_LIB}")
message(STATUS "FFI_LIB: ${FFI_LIB}")
message(STATUS "PCRE_LIB: ${PCRE_LIB}")
message(STATUS "Z_LIB: ${Z_LIB}")

# Enlazar librerías de GStreamer (estáticas) - ORDEN IMPORTANTE
target_link_libraries(gstreamer_pipeline
    # GStreamer plugins (deben ir primero)
    ${PLUGIN_COREELEMENTS}
    ${PLUGIN_VIDEOCONVERTSCALE}
    ${PLUGIN_X264}
    ${PLUGIN_VIDEOPARSERSBAD}
    ${PLUGIN_RTP}
    ${PLUGIN_UDP}
    # PLUGIN_ANDROIDMEDIA comentado (requiere OpenGL)
    ${PLUGIN_APP}
    ${PLUGIN_VIDEOTESTSRC}
    
    # Bibliotecas adicionales necesarias para plugins (ANTES de core libraries)
    ${X264_LIB}
    ${GSTPBUTILS_LIB}
    ${GSTCODECPARSERS_LIB}
    ${GSTAUDIO_LIB}
    ${GSTTAG_LIB}
    ${GSTRTP_LIB}
    ${GSTNET_LIB}
    ${GSTALLOCATORS_LIB}
    ${GSTORC_LIB}
    
    # GStreamer core libraries (orden específico)
    ${GSTREAMER_LIB}
    ${GSTAPP_LIB}
    ${GSTVIDEO_LIB}
    ${GSTBASE_LIB}
    
    # GLib ecosystem (orden específico para resolver dependencias)
    ${GOBJECT_LIB}
    ${GIO_LIB}
    ${GMODULE_LIB}
    ${GTHREAD_LIB}
    ${GLIB_LIB}
    
    # Support libraries (orden específico)
    ${INTL_LIB}
    ${ICONV_LIB}
    ${FFI_LIB}
    ${PCRE_LIB}
    ${Z_LIB}
    
    # Android system libraries
    log
    android
    atomic
    m
)


