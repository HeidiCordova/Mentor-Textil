package com.mentor.mobile.network

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import timber.log.Timber
import java.net.DatagramPacket
import java.net.DatagramSocket
import java.net.InetAddress
import java.net.SocketTimeoutException
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicLong

/**
 * Gestor de conexión con el nodo Edge.
 * Maneja UDP, WebRTC, monitoreo de latencia, reconexión automática y recepción de conteo.
 */
class EdgeConnectionManager(private val context: Context) {

    private var serverIp: String = ""
    private var serverPort: Int = 0
    private var protocol: Protocol = Protocol.UDP
    private val isConnected = AtomicBoolean(false)
    private val shouldReconnect = AtomicBoolean(true)

    private var udpSocket: DatagramSocket? = null
    private var lastLatency: AtomicLong = AtomicLong(0)
    private var palletCount: AtomicInteger = AtomicInteger(0)
    
    private var heartbeatJob: Job? = null
    private var reconnectJob: Job? = null
    private var receiveJob: Job? = null
    
    private var countUpdateCallback: ((Int) -> Unit)? = null

    enum class Protocol {
        UDP, WEBRTC, RTSP
    }

    companion object {
        private const val TAG = "EdgeConnectionManager"
        private const val HEARTBEAT_INTERVAL = 1000L // ms
        private const val HEARTBEAT_TIMEOUT = 5000L // ms
        private const val RECONNECT_INTERVAL = 5000L // ms
        private const val MAX_RECONNECT_ATTEMPTS = 10
    }

    /**
     * Conecta con el servidor Edge.
     */
    suspend fun connect(
        serverIp: String,
        serverPort: Int,
        protocol: Protocol = Protocol.UDP
    ) = withContext(Dispatchers.IO) {
        try {
            this@EdgeConnectionManager.serverIp = serverIp
            this@EdgeConnectionManager.serverPort = serverPort
            this@EdgeConnectionManager.protocol = protocol

            // Verificar conectividad de red
            if (!isNetworkAvailable()) {
                throw RuntimeException("Red no disponible")
            }

            when (protocol) {
                Protocol.UDP -> connectUDP()
                Protocol.WEBRTC -> connectWebRTC()
                Protocol.RTSP -> connectRTSP()
            }

            isConnected.set(true)
            
            // Iniciar heartbeat periódico
            startHeartbeat()
            
            // Iniciar recepción de mensajes
            startReceiving()
            
            // Iniciar reconexión automática
            startAutoReconnect()
            
            Timber.i("$TAG: Conectado a $serverIp:$serverPort ($protocol)")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error conectando")
            isConnected.set(false)
            throw e
        }
    }

    /**
     * Conecta usando UDP.
     */
    private fun connectUDP() {
        try {
            udpSocket?.close()
            udpSocket = DatagramSocket().apply {
                broadcast = true
                soTimeout = 2000 // 2 segundos de timeout para recepción
            }

            Timber.d("$TAG: UDP socket creado")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error en UDP")
            throw e
        }
    }

    /**
     * Conecta usando WebRTC.
     */
    private fun connectWebRTC() {
        Timber.d("$TAG: WebRTC connection (implementación futura)")
        // TODO: Implementar WebRTC
        throw NotImplementedError("WebRTC no implementado aún")
    }

    /**
     * Conecta usando RTSP.
     */
    private fun connectRTSP() {
        Timber.d("$TAG: RTSP connection (implementación futura)")
        // TODO: Implementar RTSP
        throw NotImplementedError("RTSP no implementado aún")
    }

    /**
     * Envía heartbeat para medir latencia.
     */
    private suspend fun sendHeartbeat() = withContext(Dispatchers.IO) {
        try {
            if (!isConnected.get()) return@withContext
            
            val timestamp = System.currentTimeMillis()
            val message = "HEARTBEAT:$timestamp".toByteArray()

            val packet = DatagramPacket(
                message,
                message.size,
                InetAddress.getByName(serverIp),
                serverPort
            )

            val startTime = System.currentTimeMillis()
            udpSocket?.send(packet)
            val latency = System.currentTimeMillis() - startTime

            lastLatency.set(latency)
            Timber.d("$TAG: Heartbeat enviado, latencia: ${latency}ms")

        } catch (e: SocketTimeoutException) {
            Timber.w("$TAG: Timeout en heartbeat")
            handleConnectionLost()
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error enviando heartbeat")
            handleConnectionLost()
        }
    }

    /**
     * Inicia la recepción de mensajes del servidor.
     */
    private fun startReceiving() {
        receiveJob?.cancel()
        receiveJob = GlobalScope.launch(Dispatchers.IO) {
            val buffer = ByteArray(1024)
            
            while (isConnected.get()) {
                try {
                    val packet = DatagramPacket(buffer, buffer.size)
                    udpSocket?.receive(packet)
                    
                    val message = String(packet.data, 0, packet.length)
                    processReceivedMessage(message)
                    
                } catch (e: SocketTimeoutException) {
                    // Timeout normal, continuar
                    continue
                } catch (e: Exception) {
                    if (isConnected.get()) {
                        Timber.e(e, "$TAG: Error recibiendo datos")
                    }
                }
            }
        }
    }

    /**
     * Procesa mensajes recibidos del servidor.
     */
    private fun processReceivedMessage(message: String) {
        try {
            when {
                message.startsWith("COUNT:") -> {
                    val count = message.substringAfter("COUNT:").trim().toIntOrNull()
                    if (count != null) {
                        palletCount.set(count)
                        countUpdateCallback?.invoke(count)
                        Timber.d("$TAG: Conteo actualizado: $count")
                    }
                }
                message.startsWith("ACK") -> {
                    Timber.d("$TAG: ACK recibido")
                }
                message.startsWith("PONG") -> {
                    Timber.d("$TAG: PONG recibido")
                }
                else -> {
                    Timber.d("$TAG: Mensaje recibido: $message")
                }
            }
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error procesando mensaje: $message")
        }
    }

    /**
     * Maneja pérdida de conexión.
     */
    private fun handleConnectionLost() {
        if (isConnected.get()) {
            Timber.w("$TAG: Conexión perdida")
            isConnected.set(false)
        }
    }

    /**
     * Inicia reconexión automática.
     */
    private fun startAutoReconnect() {
        reconnectJob?.cancel()
        reconnectJob = GlobalScope.launch(Dispatchers.IO) {
            var attempts = 0
            
            while (shouldReconnect.get()) {
                delay(RECONNECT_INTERVAL)
                
                if (!isConnected.get() && isNetworkAvailable()) {
                    attempts++
                    Timber.i("$TAG: Intentando reconexión (intento $attempts/$MAX_RECONNECT_ATTEMPTS)")
                    
                    try {
                        connectUDP()
                        isConnected.set(true)
                        
                        // Reiniciar heartbeat y recepción
                        startHeartbeat()
                        startReceiving()
                        
                        Timber.i("$TAG: Reconexión exitosa")
                        attempts = 0
                        
                    } catch (e: Exception) {
                        Timber.e(e, "$TAG: Error en reconexión")
                        
                        if (attempts >= MAX_RECONNECT_ATTEMPTS) {
                            Timber.e("$TAG: Máximo de intentos alcanzado")
                            attempts = 0
                            delay(RECONNECT_INTERVAL * 3) // Esperar más tiempo
                        }
                    }
                }
            }
        }
    }

    /**
     * Desconecta del servidor Edge.
     */
    fun disconnect() {
        try {
            shouldReconnect.set(false)
            isConnected.set(false)
            
            // Detener todos los jobs
            heartbeatJob?.cancel()
            heartbeatJob = null
            
            reconnectJob?.cancel()
            reconnectJob = null
            
            receiveJob?.cancel()
            receiveJob = null
            
            udpSocket?.close()
            udpSocket = null
            
            Timber.i("$TAG: Desconectado")

        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error desconectando")
        }
    }

    /**
     * Registra callback para actualizaciones de conteo.
     */
    fun setCountUpdateCallback(callback: (Int) -> Unit) {
        countUpdateCallback = callback
    }

    /**
     * Obtiene la IP del servidor.
     */
    fun getServerIp(): String = serverIp

    /**
     * Obtiene el puerto del servidor.
     */
    fun getServerPort(): Int = serverPort

    /**
     * Obtiene el estado de conexión.
     */
    fun getConnectionStatus(): String {
        return if (isConnected.get()) "Conectado" else "Desconectado"
    }

    /**
     * Verifica si está conectado.
     */
    fun isConnected(): Boolean = isConnected.get()

    /**
     * Obtiene la latencia estimada en ms.
     */
    fun getLatency(): Long = lastLatency.get()

    /**
     * Obtiene el conteo actual de pallets.
     */
    fun getPalletCount(): Int = palletCount.get()

    /**
     * Verifica si hay conectividad de red disponible.
     */
    private fun isNetworkAvailable(): Boolean {
        val connectivityManager = context.getSystemService(Context.CONNECTIVITY_SERVICE)
                as ConnectivityManager

        val network = connectivityManager.activeNetwork ?: return false
        val capabilities = connectivityManager.getNetworkCapabilities(network) ?: return false

        return capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
    }

    /**
     * Obtiene información de la red actual.
     */
    fun getNetworkInfo(): NetworkInfo {
        val connectivityManager = context.getSystemService(Context.CONNECTIVITY_SERVICE)
                as ConnectivityManager

        val network = connectivityManager.activeNetwork
        val capabilities = connectivityManager.getNetworkCapabilities(network)

        val type = when {
            capabilities?.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) == true -> "WiFi"
            capabilities?.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) == true -> "Cellular"
            capabilities?.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) == true -> "Ethernet"
            else -> "Unknown"
        }

        return NetworkInfo(
            type = type,
            isConnected = network != null,
            serverIp = serverIp,
            serverPort = serverPort
        )
    }

    /**
     * Inicia el heartbeat periódico.
     */
    private fun startHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = GlobalScope.launch(Dispatchers.IO) {
            while (isConnected.get()) {
                try {
                    sendHeartbeat()
                    delay(HEARTBEAT_INTERVAL)
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error en heartbeat loop")
                }
            }
        }
    }
}

/**
 * Información de la red actual.
 */
data class NetworkInfo(
    val type: String,
    val isConnected: Boolean,
    val serverIp: String,
    val serverPort: Int
)
