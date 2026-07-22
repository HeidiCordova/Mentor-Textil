package com.mentor.mobile.camera

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.graphics.ImageFormat
import android.hardware.camera2.*
import android.media.ImageReader
import android.os.Handler
import android.os.HandlerThread
import android.view.Surface
import androidx.core.content.ContextCompat
import timber.log.Timber
import java.util.concurrent.Semaphore
import java.util.concurrent.TimeUnit

/**
 * Gestor de cámara usando Camera2 API.
 * Captura frames de video para ser codificados y/o mostrados en preview.
 */
class CameraManager(
    private val context: Context,
    private val width: Int = 1280,
    private val height: Int = 720,
    private val fps: Int = 30
) {
    
    private var cameraDevice: CameraDevice? = null
    private var captureSession: CameraCaptureSession? = null
    private var imageReader: ImageReader? = null
    private var previewSurface: Surface? = null
    
    private val cameraOpenCloseLock = Semaphore(1)
    private var backgroundThread: HandlerThread? = null
    private var backgroundHandler: Handler? = null
    
    private var frameCallback: ((ByteArray, Long) -> Unit)? = null
    private var previewOnly: Boolean = false
    
    companion object {
        private const val TAG = "CameraManager"
    }
    
    /**
     * Inicia el thread de background para operaciones de cámara.
     */
    private fun startBackgroundThread() {
        backgroundThread = HandlerThread("CameraBackground").also { it.start() }
        backgroundHandler = Handler(backgroundThread!!.looper)
    }
    
    /**
     * Detiene el thread de background.
     */
    private fun stopBackgroundThread() {
        backgroundThread?.quitSafely()
        try {
            backgroundThread?.join()
            backgroundThread = null
            backgroundHandler = null
        } catch (e: InterruptedException) {
            Timber.e(e, "$TAG: Error deteniendo background thread")
        }
    }
    
    /**
     * Abre la cámara para preview en un Surface (sin codificación).
     */
    fun openCameraForPreview(surface: Surface) {
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA)
            != PackageManager.PERMISSION_GRANTED) {
            Timber.e("$TAG: Permiso de cámara no otorgado")
            return
        }
        
        previewSurface = surface
        previewOnly = true
        frameCallback = null
        startBackgroundThread()
        
        val manager = context.getSystemService(Context.CAMERA_SERVICE) as android.hardware.camera2.CameraManager
        
        try {
            // Buscar cámara trasera
            val cameraId = manager.cameraIdList.firstOrNull { id ->
                val characteristics = manager.getCameraCharacteristics(id)
                characteristics.get(CameraCharacteristics.LENS_FACING) == CameraCharacteristics.LENS_FACING_BACK
            } ?: manager.cameraIdList[0]
            
            if (!cameraOpenCloseLock.tryAcquire(2500, TimeUnit.MILLISECONDS)) {
                throw RuntimeException("Timeout esperando lock de cámara")
            }
            
            manager.openCamera(cameraId, stateCallback, backgroundHandler)
            
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error abriendo cámara para preview")
            throw e
        }
    }
    
    /**
     * Abre la cámara trasera para captura y codificación.
     */
    /**
     * Abre la cámara para captura con preview opcional.
     */
    fun openCamera(onFrameAvailable: (ByteArray, Long) -> Unit, previewSurface: Surface? = null) {
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA)
            != PackageManager.PERMISSION_GRANTED) {
            Timber.e("$TAG: Permiso de cámara no otorgado")
            return
        }
        
        this.previewSurface = previewSurface
        frameCallback = onFrameAvailable
        previewOnly = false
        startBackgroundThread()
        
        val manager = context.getSystemService(Context.CAMERA_SERVICE) as android.hardware.camera2.CameraManager
        
        try {
            // Buscar cámara trasera
            val cameraId = manager.cameraIdList.firstOrNull { id ->
                val characteristics = manager.getCameraCharacteristics(id)
                characteristics.get(CameraCharacteristics.LENS_FACING) == CameraCharacteristics.LENS_FACING_BACK
            } ?: manager.cameraIdList[0]
            
            if (!cameraOpenCloseLock.tryAcquire(2500, TimeUnit.MILLISECONDS)) {
                throw RuntimeException("Timeout esperando lock de cámara")
            }
            
            manager.openCamera(cameraId, stateCallback, backgroundHandler)
            
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error abriendo cámara")
            throw e
        }
    }
    
    /**
     * Callback de estado de la cámara.
     */
    private val stateCallback = object : CameraDevice.StateCallback() {
        override fun onOpened(camera: CameraDevice) {
            cameraOpenCloseLock.release()
            cameraDevice = camera
            createCaptureSession()
            Timber.d("$TAG: Cámara abierta")
        }
        
        override fun onDisconnected(camera: CameraDevice) {
            cameraOpenCloseLock.release()
            camera.close()
            cameraDevice = null
            Timber.d("$TAG: Cámara desconectada")
        }
        
        override fun onError(camera: CameraDevice, error: Int) {
            cameraOpenCloseLock.release()
            camera.close()
            cameraDevice = null
            Timber.e("$TAG: Error de cámara: $error")
        }
    }
    
    /**
     * Crea la sesión de captura.
     */
    private fun createCaptureSession() {
        try {
            val surfaces = mutableListOf<Surface>()
            
            if (previewOnly && previewSurface != null) {
                // Modo preview: solo mostrar en SurfaceView
                surfaces.add(previewSurface!!)
                
            } else {
                // Modo captura: ImageReader para obtener frames
                imageReader = ImageReader.newInstance(width, height, ImageFormat.YUV_420_888, 2).apply {
                    setOnImageAvailableListener({ reader ->
                        val image = reader.acquireLatestImage()
                        image?.let {
                            try {
                                // Convertir YUV a NV21 (formato que MediaCodec acepta)
                                val nv21 = yuv420ToNv21(it)
                                val timestamp = it.timestamp
                                frameCallback?.invoke(nv21, timestamp)
                            } finally {
                                it.close()
                            }
                        }
                    }, backgroundHandler)
                }
                surfaces.add(imageReader!!.surface)
                
                // Si también hay preview surface, agregarlo
                previewSurface?.let { surfaces.add(it) }
            }
            
            cameraDevice?.createCaptureSession(surfaces, object : CameraCaptureSession.StateCallback() {
                override fun onConfigured(session: CameraCaptureSession) {
                    if (cameraDevice == null) return
                    
                    captureSession = session
                    
                    try {
                        // Crear request de captura continua
                        val template = if (previewOnly) CameraDevice.TEMPLATE_PREVIEW else CameraDevice.TEMPLATE_RECORD
                        val captureRequest = cameraDevice!!.createCaptureRequest(template).apply {
                            surfaces.forEach { addTarget(it) }
                            set(CaptureRequest.CONTROL_MODE, CameraMetadata.CONTROL_MODE_AUTO)
                            set(CaptureRequest.CONTROL_AE_TARGET_FPS_RANGE, android.util.Range(fps, fps))
                        }
                        
                        session.setRepeatingRequest(captureRequest.build(), null, backgroundHandler)
                        Timber.d("$TAG: Sesión de captura iniciada (previewOnly=$previewOnly)")
                        
                    } catch (e: Exception) {
                        Timber.e(e, "$TAG: Error iniciando captura")
                    }
                }
                
                override fun onConfigureFailed(session: CameraCaptureSession) {
                    Timber.e("$TAG: Configuración de sesión falló")
                }
            }, backgroundHandler)
            
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error creando sesión de captura")
        }
    }
    
    /**
     * Convierte imagen YUV_420_888 a NV21.
     */
    private fun yuv420ToNv21(image: android.media.Image): ByteArray {
        val ySize = image.width * image.height
        val uvSize = image.width * image.height / 2
        
        val nv21 = ByteArray(ySize + uvSize)
        
        val yBuffer = image.planes[0].buffer
        val uBuffer = image.planes[1].buffer
        val vBuffer = image.planes[2].buffer
        
        // Copiar Y
        yBuffer.get(nv21, 0, ySize)
        
        // Intercalar U y V para NV21 (formato: V, U, V, U, ...)
        val uvPixelStride = image.planes[1].pixelStride
        val uvRowStride = image.planes[1].rowStride
        
        if (uvPixelStride == 1) {
            // Caso simple: datos contiguos
            vBuffer.get(nv21, ySize, uvSize / 2)
            uBuffer.get(nv21, ySize + uvSize / 2, uvSize / 2)
        } else {
            // Caso complejo: intercalar manualmente
            var pos = ySize
            val uvWidth = image.width / 2
            val uvHeight = image.height / 2
            
            for (y in 0 until uvHeight) {
                for (x in 0 until uvWidth) {
                    val vIndex = y * uvRowStride + x * uvPixelStride
                    val uIndex = y * uvRowStride + x * uvPixelStride
                    nv21[pos++] = vBuffer.get(vIndex)
                    nv21[pos++] = uBuffer.get(uIndex)
                }
            }
        }
        
        return nv21
    }
    
    /**
     * Cierra la cámara.
     */
    fun closeCamera() {
        try {
            cameraOpenCloseLock.acquire()
            
            captureSession?.close()
            captureSession = null
            
            cameraDevice?.close()
            cameraDevice = null
            
            imageReader?.close()
            imageReader = null
            
            Timber.d("$TAG: Cámara cerrada")
            
        } catch (e: InterruptedException) {
            Timber.e(e, "$TAG: Error cerrando cámara")
        } finally {
            cameraOpenCloseLock.release()
            stopBackgroundThread()
        }
    }
}
