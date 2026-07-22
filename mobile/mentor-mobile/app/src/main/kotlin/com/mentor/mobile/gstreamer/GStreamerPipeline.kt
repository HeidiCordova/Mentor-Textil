package com.mentor.mobile.gstreamer

import android.view.SurfaceView
import timber.log.Timber

/**
 * Implementación del pipeline de GStreamer para captura y transmisión.
 * Pipeline: camerabin -> h264parse -> rtph264pay -> udpsink
 */
class GStreamerPipeline(
    private val surfaceView: SurfaceView,
    private val cameraResolution: CameraResolution,
    private val bitrate: Int, // kbps
    private val framerate: Int
) {

    private var nativeHandle: Long = 0
    private var isRunning = false

    companion object {
        private const val TAG = "GStreamerPipeline"
        
        private var gstreamerInitialized = false

        // Cargar librería nativa de GStreamer
        init {
            try {
                System.loadLibrary("gstreamer_pipeline")
                Timber.d("$TAG: Librería nativa cargada")
            } catch (e: UnsatisfiedLinkError) {
                Timber.e(e, "$TAG: Error cargando librerías nativas")
            }
        }
        
        /**
         * Inicializa GStreamer. Debe llamarse una vez antes de usar cualquier pipeline.
         */
        @JvmStatic
        fun initialize(): Boolean {
            if (gstreamerInitialized) {
                Timber.d("$TAG: GStreamer ya está inicializado")
                return true
            }
            
            try {
                val result = nativeInit()
                gstreamerInitialized = result
                if (result) {
                    Timber.i("$TAG: GStreamer inicializado correctamente")
                } else {
                    Timber.e("$TAG: Error inicializando GStreamer")
                }
                return result
            } catch (e: Exception) {
                Timber.e(e, "$TAG: Excepción inicializando GStreamer")
                return false
            }
        }
        
        // Método nativo para inicializar GStreamer
        @JvmStatic
        private external fun nativeInit(): Boolean
    }

    /**
     * Inicia la transmisión de video.
     */
    fun startStreaming(serverIp: String, serverPort: Int) {
        if (isRunning) {
            Timber.w("$TAG: Pipeline ya está en ejecución")
            return
        }
        
        // Asegurar que GStreamer esté inicializado
        if (!gstreamerInitialized) {
            Timber.w("$TAG: GStreamer no inicializado, inicializando ahora...")
            if (!initialize()) {
                throw RuntimeException("No se pudo inicializar GStreamer")
            }
        }

        try {
            val pipelineString = buildPipeline(serverIp, serverPort)
            Timber.d("$TAG: Pipeline: $pipelineString")

            // Crear y ejecutar pipeline nativo
            // Para pipelines de prueba sin preview, pasamos null como surfaceView
            nativeHandle = nativeCreatePipeline(pipelineString, null)
            
            if (nativeHandle == 0L) {
                throw RuntimeException("Error creando pipeline nativo")
            }

            nativeStartPipeline(nativeHandle)
            isRunning = true
            Timber.d("$TAG: Pipeline iniciado")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error iniciando pipeline")
            throw e
        }
    }

    /**
     * Detiene la transmisión de video.
     */
    fun stopStreaming() {
        if (!isRunning) {
            Timber.w("$TAG: Pipeline no está en ejecución")
            return
        }

        try {
            nativeStopPipeline(nativeHandle)
            nativeDestroyPipeline(nativeHandle)
            isRunning = false
            nativeHandle = 0L
            Timber.d("$TAG: Pipeline detenido")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error deteniendo pipeline")
        }
    }

    /**
     * Construye el string del pipeline de GStreamer.
     */
    private fun buildPipeline(serverIp: String, serverPort: Int): String {
        // Pipeline simplificado usando appsrc para recibir datos H.264 del encoder
        // appsrc -> h264parse -> rtph264pay -> udpsink
        return "appsrc name=videosrc ! " +
                "video/x-h264,stream-format=byte-stream ! " +
                "h264parse ! " +
                "rtph264pay config-interval=1 pt=96 ! " +
                "udpsink host=$serverIp port=$serverPort"
    }

    /**
     * Obtiene estadísticas del pipeline.
     */
    fun getStats(): PipelineStats? {
        return if (isRunning && nativeHandle != 0L) {
            val fps = nativeGetFps(nativeHandle)
            val bitrate = nativeGetBitrate(nativeHandle)
            val droppedFrames = nativeGetDroppedFrames(nativeHandle)
            val latency = nativeGetLatency(nativeHandle)

            PipelineStats(
                fps = fps,
                bitrate = bitrate,
                droppedFrames = droppedFrames,
                latency = latency
            )
        } else {
            null
        }
    }

    /**
     * Maneja onResume del ciclo de vida.
     */
    fun onResume() {
        if (isRunning && nativeHandle != 0L) {
            nativeResumePipeline(nativeHandle)
        }
    }

    /**
     * Maneja onPause del ciclo de vida.
     */
    fun onPause() {
        if (isRunning && nativeHandle != 0L) {
            nativePausePipeline(nativeHandle)
        }
    }

    /**
     * Limpia recursos del pipeline.
     */
    fun cleanup() {
        if (isRunning) {
            stopStreaming()
        }
    }
    
    /**
     * Envía datos H.264 codificados al pipeline.
     * Llamado desde VideoEncoder cuando hay datos disponibles.
     */
    fun pushH264Data(data: ByteArray, timestamp: Long) {
        if (isRunning && nativeHandle != 0L) {
            nativePushH264Data(nativeHandle, data, data.size, timestamp)
        }
    }

    // ============ Métodos nativos JNI ============

    private external fun nativeCreatePipeline(
        pipelineString: String,
        surfaceView: SurfaceView?  // Ahora puede ser null
    ): Long

    private external fun nativeStartPipeline(handle: Long)

    private external fun nativeStopPipeline(handle: Long)

    private external fun nativeDestroyPipeline(handle: Long)

    private external fun nativePausePipeline(handle: Long)

    private external fun nativeResumePipeline(handle: Long)

    private external fun nativeGetFps(handle: Long): Double

    private external fun nativeGetBitrate(handle: Long): Long

    private external fun nativeGetDroppedFrames(handle: Long): Long

    private external fun nativeGetLatency(handle: Long): Long
    
    // Método para enviar datos H.264 al appsrc
    private external fun nativePushH264Data(handle: Long, data: ByteArray, size: Int, timestamp: Long)
}
