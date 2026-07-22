package com.mentor.mobile.camera

import android.media.MediaCodec
import android.media.MediaCodecInfo
import android.media.MediaFormat
import timber.log.Timber
import java.nio.ByteBuffer

/**
 * Codificador de video usando MediaCodec.
 * Codifica frames YUV a H.264.
 */
class VideoEncoder(
    private val width: Int = 1280,
    private val height: Int = 720,
    private val bitrate: Int = 2500000, // 2.5 Mbps
    private val fps: Int = 30
) {
    
    private var encoder: MediaCodec? = null
    private var isEncoding = false
    
    private var encodedDataCallback: ((ByteBuffer, MediaCodec.BufferInfo) -> Unit)? = null
    
    companion object {
        private const val TAG = "VideoEncoder"
        private const val MIME_TYPE = "video/avc" // H.264
        private const val I_FRAME_INTERVAL = 1 // 1 segundo entre keyframes
    }
    
    /**
     * Inicia el codificador.
     */
    fun start(onEncodedData: (ByteBuffer, MediaCodec.BufferInfo) -> Unit) {
        encodedDataCallback = onEncodedData
        
        try {
            val format = MediaFormat.createVideoFormat(MIME_TYPE, width, height).apply {
                setInteger(MediaFormat.KEY_COLOR_FORMAT, MediaCodecInfo.CodecCapabilities.COLOR_FormatYUV420SemiPlanar)
                setInteger(MediaFormat.KEY_BIT_RATE, bitrate)
                setInteger(MediaFormat.KEY_FRAME_RATE, fps)
                setInteger(MediaFormat.KEY_I_FRAME_INTERVAL, I_FRAME_INTERVAL)
                
                // Configuración para baja latencia
                setInteger(MediaFormat.KEY_LATENCY, 0)
                setInteger(MediaFormat.KEY_PRIORITY, 0) // Realtime priority
            }
            
            encoder = MediaCodec.createEncoderByType(MIME_TYPE).apply {
                configure(format, null, null, MediaCodec.CONFIGURE_FLAG_ENCODE)
                start()
            }
            
            isEncoding = true
            Timber.d("$TAG: Codificador iniciado ($width x $height @ ${fps}fps, ${bitrate/1000}kbps)")
            
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error iniciando codificador")
            throw e
        }
    }
    
    /**
     * Codifica un frame YUV.
     */
    fun encodeFrame(frameData: ByteArray, timestamp: Long) {
        if (!isEncoding || encoder == null) {
            return
        }
        
        try {
            // Obtener buffer de entrada
            val inputBufferIndex = encoder!!.dequeueInputBuffer(10000)
            if (inputBufferIndex >= 0) {
                val inputBuffer = encoder!!.getInputBuffer(inputBufferIndex)
                inputBuffer?.clear()
                inputBuffer?.put(frameData)
                
                encoder!!.queueInputBuffer(
                    inputBufferIndex,
                    0,
                    frameData.size,
                    timestamp / 1000, // Convertir a microsegundos
                    0
                )
            }
            
            // Obtener datos codificados
            val bufferInfo = MediaCodec.BufferInfo()
            var outputBufferIndex = encoder!!.dequeueOutputBuffer(bufferInfo, 0)
            
            while (outputBufferIndex >= 0) {
                val outputBuffer = encoder!!.getOutputBuffer(outputBufferIndex)
                
                if (outputBuffer != null && bufferInfo.size > 0) {
                    // Copiar datos antes de liberar el buffer
                    val encodedData = ByteArray(bufferInfo.size)
                    outputBuffer.position(bufferInfo.offset)
                    outputBuffer.get(encodedData)
                    
                    // Crear un nuevo ByteBuffer con los datos copiados
                    val copiedBuffer = java.nio.ByteBuffer.wrap(encodedData)
                    val copiedBufferInfo = MediaCodec.BufferInfo().apply {
                        offset = 0
                        size = bufferInfo.size
                        presentationTimeUs = bufferInfo.presentationTimeUs
                        flags = bufferInfo.flags
                    }
                    
                    // Callback con datos H.264
                    encodedDataCallback?.invoke(copiedBuffer, copiedBufferInfo)
                }
                
                encoder!!.releaseOutputBuffer(outputBufferIndex, false)
                outputBufferIndex = encoder!!.dequeueOutputBuffer(bufferInfo, 0)
            }
            
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error codificando frame")
        }
    }
    
    /**
     * Detiene el codificador.
     */
    fun stop() {
        try {
            isEncoding = false
            
            encoder?.stop()
            encoder?.release()
            encoder = null
            
            Timber.d("$TAG: Codificador detenido")
            
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error deteniendo codificador")
        }
    }
}
