package com.mentor.mobile.gstreamer

import android.content.Context
import android.media.MediaCodec
import android.view.SurfaceView
import com.mentor.mobile.camera.CameraManager
import com.mentor.mobile.camera.VideoEncoder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import timber.log.Timber
import java.nio.ByteBuffer

/**
 * Resoluciones de cámara soportadas.
 */
enum class CameraResolution(val width: Int, val height: Int) {
    VGA(640, 480),
    HD_720P(1280, 720),
    HD_1080P(1920, 1080),
    UHD_4K(3840, 2160)
}

/**
 * Estadísticas del pipeline de GStreamer.
 */
data class PipelineStats(
    val fps: Double,
    val bitrate: Long,
    val droppedFrames: Long,
    val latency: Long // en ms
)

/**
 * Gestor centralizado de GStreamer para captura y transmisión de video.
 * Integra Camera2 API, MediaCodec y GStreamer para streaming en tiempo real.
 */
class GStreamerManager(
    private val context: Context,
    private val surfaceView: SurfaceView
) {

    private var gstreamerPipeline: GStreamerPipeline? = null
    private var cameraManager: CameraManager? = null
    private var videoEncoder: VideoEncoder? = null
    
    private var isInitialized = false
    private var isStreaming = false

    companion object {
        private const val TAG = "GStreamerManager"
        
        // Cargar biblioteca nativa de GStreamer
        init {
            try {
                System.loadLibrary("gstreamer_pipeline")
                Timber.d("$TAG: Biblioteca nativa cargada")
            } catch (e: UnsatisfiedLinkError) {
                Timber.e(e, "$TAG: Error cargando biblioteca nativa")
            }
        }
    }

    /**
     * Inicializa GStreamer, cámara y codificador.
     */
    suspend fun initialize() = withContext(Dispatchers.Default) {
        try {
            // Inicializar GStreamer primero
            if (!GStreamerPipeline.initialize()) {
                throw RuntimeException("No se pudo inicializar GStreamer")
            }
            Timber.d("$TAG: GStreamer inicializado")

            // Crear pipeline de GStreamer
            gstreamerPipeline = GStreamerPipeline(
                surfaceView = surfaceView,
                cameraResolution = CameraResolution.HD_720P,
                bitrate = 2500, // kbps
                framerate = 30
            )

            // Crear gestor de cámara
            cameraManager = CameraManager(
                context = context,
                width = 1280,
                height = 720,
                fps = 30
            )
            
            // Crear codificador de video
            videoEncoder = VideoEncoder(
                width = 1280,
                height = 720,
                bitrate = 2500000, // 2.5 Mbps
                fps = 30
            )

            isInitialized = true
            Timber.d("$TAG: Sistema inicializado")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error inicializando")
            throw e
        }
    }

    /**
     * Inicia solo el preview de cámara sin transmisión.
     * Permite al operario encuadrar el pallet antes de iniciar la transmisión.
     */
    suspend fun startCameraPreview() = withContext(Dispatchers.Default) {
        if (!isInitialized) {
            throw IllegalStateException("Sistema no está inicializado")
        }

        try {
            // Obtener el Surface del SurfaceView
            val surface = surfaceView.holder.surface
            
            // Iniciar cámara en modo preview (solo muestra en SurfaceView)
            cameraManager?.openCameraForPreview(surface)
            Timber.d("$TAG: Cámara en modo preview iniciada")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error iniciando preview de cámara")
            throw e
        }
    }

    /**
     * Inicia la transmisión de video hacia el servidor Edge.
     */
    suspend fun startStreaming(
        serverIp: String = "192.168.15.13",
        serverPort: Int = 5000
    ) = withContext(Dispatchers.Default) {
        if (!isInitialized) {
            throw IllegalStateException("Sistema no está inicializado")
        }
        
        if (isStreaming) {
            Timber.w("$TAG: Ya está transmitiendo")
            return@withContext
        }

        try {
            // 1. Iniciar pipeline de GStreamer para transmisión
            gstreamerPipeline?.startStreaming(serverIp, serverPort)
            Timber.d("$TAG: Pipeline GStreamer iniciado")
            
            // 2. Iniciar codificador de video
            videoEncoder?.start { encodedData, bufferInfo ->
                // Callback cuando hay datos H.264 disponibles
                try {
                    // Extraer datos del buffer
                    val data = ByteArray(bufferInfo.size)
                    encodedData.position(bufferInfo.offset)
                    encodedData.get(data, 0, bufferInfo.size)
                    
                    // Enviar al pipeline de GStreamer
                    gstreamerPipeline?.pushH264Data(data, bufferInfo.presentationTimeUs)
                    
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error enviando datos a GStreamer")
                }
            }
            Timber.d("$TAG: Codificador iniciado")
            
            // 3. Cerrar preview y reabrir cámara con codificación + preview
            cameraManager?.closeCamera()
            delay(100) // Pequeña pausa para asegurar que se cerró
            
            // Obtener el Surface del SurfaceView para mantener el preview
            val surface = surfaceView.holder.surface
            
            cameraManager?.openCamera({ frameData, timestamp ->
                // Callback cuando hay un frame disponible
                try {
                    // Codificar para transmisión
                    videoEncoder?.encodeFrame(frameData, timestamp)
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error procesando frame")
                }
            }, surface) // Pasar el surface para mantener el preview
            
            Timber.d("$TAG: Cámara reiniciada con codificación y preview")
            
            isStreaming = true
            Timber.i("$TAG: Transmisión iniciada hacia $serverIp:$serverPort")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error iniciando transmisión")
            stopStreaming()
            throw e
        }
    }

    /**
     * Detiene la transmisión de video pero mantiene el preview de cámara.
     */
    suspend fun stopStreaming() = withContext(Dispatchers.Default) {
        if (!isStreaming) {
            Timber.w("$TAG: No está transmitiendo")
            return@withContext
        }
        
        try {
            // Detener codificador y pipeline
            videoEncoder?.stop()
            Timber.d("$TAG: Codificador detenido")
            
            gstreamerPipeline?.stopStreaming()
            Timber.d("$TAG: Pipeline detenido")
            
            // Reiniciar cámara en modo preview (sin codificación)
            cameraManager?.closeCamera()
            delay(100)
            
            val surface = surfaceView.holder.surface
            cameraManager?.openCameraForPreview(surface)
            Timber.d("$TAG: Cámara en modo preview")
            
            isStreaming = false
            Timber.i("$TAG: Transmisión detenida, preview continúa")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error deteniendo transmisión")
            throw e
        }
    }

    /**
     * Obtiene el estado actual de transmisión.
     */
    fun isStreaming(): Boolean = isStreaming

    /**
     * Obtiene estadísticas del pipeline.
     */
    fun getStats(): PipelineStats? {
        return gstreamerPipeline?.getStats()
    }

    /**
     * Maneja el ciclo de vida onResume.
     */
    fun onResume() {
        gstreamerPipeline?.onResume()
    }

    /**
     * Maneja el ciclo de vida onPause.
     */
    fun onPause() {
        gstreamerPipeline?.onPause()
    }

    /**
     * Limpia recursos.
     */
    fun cleanup() {
        try {
            if (isStreaming) {
                // Usar runBlocking porque cleanup puede ser llamado desde onDestroy
                kotlinx.coroutines.runBlocking {
                    stopStreaming()
                }
            }
            
            cameraManager = null
            videoEncoder = null
            gstreamerPipeline?.cleanup()
            gstreamerPipeline = null
            
            isInitialized = false
            Timber.d("$TAG: Recursos liberados")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error durante cleanup")
        }
    }
}
