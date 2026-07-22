package com.mentor.mobile.ui

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.os.PowerManager
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.mentor.mobile.BuildConfig
import com.mentor.mobile.R
import com.mentor.mobile.databinding.ActivityMainBinding
import com.mentor.mobile.gstreamer.GStreamerManager
import com.mentor.mobile.network.EdgeConnectionManager
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import timber.log.Timber

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var gstreamerManager: GStreamerManager
    private lateinit var edgeConnectionManager: EdgeConnectionManager
    private var wakeLock: PowerManager.WakeLock? = null

    private val requiredPermissions = arrayOf(
        Manifest.permission.CAMERA,
        Manifest.permission.INTERNET,
        Manifest.permission.ACCESS_NETWORK_STATE,
        Manifest.permission.RECORD_AUDIO
    )

    companion object {
        private const val PERMISSION_REQUEST_CODE = 100
        private const val TAG = "MainActivity"
        private const val STATUS_UPDATE_INTERVAL = 500L // ms
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        // Inicializar View Binding
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        // Inicializar Timber para logging
        if (BuildConfig.DEBUG) {
            Timber.plant(Timber.DebugTree())
        }

        Timber.d("$TAG: Iniciando aplicación")

        // Adquirir WakeLock para mantener pantalla encendida
        acquireWakeLock()

        // Inicializar managers
        edgeConnectionManager = EdgeConnectionManager(this)
        gstreamerManager = GStreamerManager(this, binding.cameraPreview)

        // Configurar callback de conteo
        edgeConnectionManager.setCountUpdateCallback { count ->
            runOnUiThread {
                updatePalletCount(count)
            }
        }

        // Verificar permisos
        if (allPermissionsGranted()) {
            initializeApp()
        } else {
            requestPermissions()
        }

        // Configurar listeners de UI
        setupUIListeners()
        
        // Iniciar actualización periódica de estado
        startStatusUpdates()
    }

    private fun acquireWakeLock() {
        val powerManager = getSystemService(POWER_SERVICE) as PowerManager
        wakeLock = powerManager.newWakeLock(
            PowerManager.SCREEN_BRIGHT_WAKE_LOCK or PowerManager.ACQUIRE_CAUSES_WAKEUP,
            "MentorMobile::StreamingWakeLock"
        )
        wakeLock?.acquire(4 * 60 * 60 * 1000L) // 4 horas
        Timber.d("$TAG: WakeLock adquirido")
    }

    private fun allPermissionsGranted(): Boolean {
        return requiredPermissions.all { permission ->
            ContextCompat.checkSelfPermission(this, permission) == PackageManager.PERMISSION_GRANTED
        }
    }

    private fun requestPermissions() {
        ActivityCompat.requestPermissions(
            this,
            requiredPermissions,
            PERMISSION_REQUEST_CODE
        )
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)

        if (requestCode == PERMISSION_REQUEST_CODE) {
            if (grantResults.isNotEmpty() && grantResults.all { it == PackageManager.PERMISSION_GRANTED }) {
                Timber.d("$TAG: Permisos otorgados")
                initializeApp()
            } else {
                Timber.e("$TAG: Permisos denegados")
                showError("Se requieren permisos de cámara y red para funcionar")
            }
        }
    }

    private fun initializeApp() {
        lifecycleScope.launch {
            try {
                // Inicializar GStreamer
                gstreamerManager.initialize()
                Timber.d("$TAG: GStreamer inicializado")

                // Obtener IP y puerto de los campos de entrada
                val serverIp = binding.etServerIp.text.toString().trim()
                val serverPort = binding.etServerPort.text.toString().toIntOrNull() ?: 5000

                // Conectar al nodo Edge
                edgeConnectionManager.connect(
                    serverIp = serverIp,
                    serverPort = serverPort,
                    protocol = EdgeConnectionManager.Protocol.UDP
                )

                // NUEVO: Iniciar preview de cámara inmediatamente
                gstreamerManager.startCameraPreview()
                Timber.d("$TAG: Preview de cámara iniciado")
                
                // Mostrar overlay de guía desde el inicio
                binding.cameraOverlay.setOverlayVisible(true)
                binding.cameraOverlay.setBoundingBoxColor(android.graphics.Color.YELLOW)

                // Actualizar UI con estado
                updateStatusPanel()
                
                showSuccess("Sistema inicializado correctamente")

            } catch (e: Exception) {
                Timber.e(e, "$TAG: Error durante inicialización")
                showError("Error: ${e.message}")
            }
        }
    }

    private fun setupUIListeners() {
        binding.btnStartStream.setOnClickListener {
            lifecycleScope.launch {
                try {
                    // Obtener IP y puerto actualizados
                    val serverIp = binding.etServerIp.text.toString().trim()
                    val serverPort = binding.etServerPort.text.toString().toIntOrNull() ?: 5000
                    
                    // Reconectar si cambió la configuración
                    if (serverIp != edgeConnectionManager.getServerIp() || 
                        serverPort != edgeConnectionManager.getServerPort()) {
                        edgeConnectionManager.disconnect()
                        delay(500)
                        edgeConnectionManager.connect(serverIp, serverPort)
                    }
                    
                    // Iniciar transmisión (la cámara ya está en preview)
                    gstreamerManager.startStreaming(serverIp, serverPort)
                    Timber.i("$TAG: Transmisión iniciada")
                    
                    // Cambiar color del overlay a verde (transmitiendo)
                    binding.cameraOverlay.setBoundingBoxColor(android.graphics.Color.GREEN)
                    
                    updateStatusPanel()
                    showSuccess("✅ Transmisión iniciada")
                    
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error al iniciar transmisión")
                    showError("Error: ${e.message}")
                }
            }
        }

        binding.btnStopStream.setOnClickListener {
            lifecycleScope.launch {
                try {
                    gstreamerManager.stopStreaming()
                    Timber.i("$TAG: Transmisión detenida")
                    
                    // Cambiar color del overlay a amarillo (solo preview)
                    binding.cameraOverlay.setBoundingBoxColor(android.graphics.Color.YELLOW)
                    
                    updateStatusPanel()
                    showSuccess("⏹ Transmisión detenida")
                    
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error al detener transmisión")
                    showError("Error: ${e.message}")
                }
            }
        }

        binding.btnDisconnect.setOnClickListener {
            lifecycleScope.launch {
                try {
                    // Detener transmisión si está activa
                    if (gstreamerManager.isStreaming()) {
                        gstreamerManager.stopStreaming()
                        binding.cameraOverlay.setBoundingBoxColor(android.graphics.Color.YELLOW)
                    }
                    
                    // Desconectar del servidor
                    edgeConnectionManager.disconnect()
                    Timber.i("$TAG: Desconectado del servidor")
                    
                    updateStatusPanel()
                    showSuccess("🔌 Desconectado del servidor Edge")
                    
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error al desconectar")
                    showError("Error: ${e.message}")
                }
            }
        }

        binding.btnReconnect.setOnClickListener {
            lifecycleScope.launch {
                try {
                    // Obtener IP y puerto de los campos
                    val serverIp = binding.etServerIp.text.toString().trim()
                    val serverPort = binding.etServerPort.text.toString().toIntOrNull() ?: 5000
                    
                    // Desconectar primero
                    edgeConnectionManager.disconnect()
                    delay(500)
                    
                    // Reconectar con nueva configuración
                    edgeConnectionManager.connect(serverIp, serverPort)
                    Timber.i("$TAG: Reconectado a $serverIp:$serverPort")
                    
                    updateStatusPanel()
                    showSuccess("🔄 Reconectado a $serverIp:$serverPort")
                    
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error al reconectar")
                    showError("Error: ${e.message}")
                }
            }
        }
    }

    private fun startStatusUpdates() {
        lifecycleScope.launch {
            while (true) {
                try {
                    updateStatusPanel()
                    delay(STATUS_UPDATE_INTERVAL)
                } catch (e: Exception) {
                    Timber.e(e, "$TAG: Error en actualización de estado")
                }
            }
        }
    }

    private fun updateStatusPanel() {
        try {
            val serverIp = edgeConnectionManager.getServerIp()
            val serverPort = edgeConnectionManager.getServerPort()
            val connectionStatus = edgeConnectionManager.getConnectionStatus()
            val latency = edgeConnectionManager.getLatency()
            val streamingStatus = gstreamerManager.isStreaming()
            val palletCount = edgeConnectionManager.getPalletCount()

            binding.tvServerIp.text = "Servidor: $serverIp:$serverPort"
            binding.tvConnectionStatus.text = "Estado: $connectionStatus"
            binding.tvLatency.text = "Latencia: ${latency}ms"
            
            // Actualizar indicador de transmisión con emoji y color
            if (streamingStatus) {
                binding.tvStreamingStatus.text = "🔴 TRANSMITIENDO"
                binding.tvStreamingStatus.setTextColor(android.graphics.Color.rgb(255, 50, 50))
                binding.streamingIndicator.setBackgroundColor(android.graphics.Color.RED)
            } else {
                binding.tvStreamingStatus.text = "📹 Preview"
                binding.tvStreamingStatus.setTextColor(android.graphics.Color.rgb(255, 200, 0))
                binding.streamingIndicator.setBackgroundColor(android.graphics.Color.YELLOW)
            }
            
            binding.tvPalletCount.text = "Conteo: $palletCount"

            // Cambiar color según estado de conexión
            val statusColor = if (connectionStatus == "Conectado") {
                android.graphics.Color.GREEN
            } else {
                android.graphics.Color.RED
            }
            binding.tvConnectionStatus.setTextColor(statusColor)
            
            // Cambiar color del overlay según estado
            if (streamingStatus) {
                binding.cameraOverlay.setBoundingBoxColor(android.graphics.Color.GREEN)
            } else {
                binding.cameraOverlay.setBoundingBoxColor(android.graphics.Color.YELLOW)
            }
            
        } catch (e: Exception) {
            Timber.e(e, "$TAG: Error actualizando panel de estado")
        }
    }

    private fun updatePalletCount(count: Int) {
        binding.tvPalletCount.text = "Conteo: $count"
        
        // Feedback visual: parpadeo verde
        binding.tvPalletCount.setTextColor(android.graphics.Color.rgb(0, 255, 0))
        lifecycleScope.launch {
            delay(200)
            binding.tvPalletCount.setTextColor(android.graphics.Color.GREEN)
        }
    }

    private fun showError(message: String) {
        binding.tvError.text = message
        binding.tvError.setTextColor(android.graphics.Color.RED)
        binding.tvError.visibility = android.view.View.VISIBLE
        
        lifecycleScope.launch {
            delay(5000)
            binding.tvError.visibility = android.view.View.GONE
        }
    }

    private fun showSuccess(message: String) {
        binding.tvError.text = message
        binding.tvError.setTextColor(android.graphics.Color.GREEN)
        binding.tvError.visibility = android.view.View.VISIBLE
        
        lifecycleScope.launch {
            delay(3000)
            binding.tvError.visibility = android.view.View.GONE
        }
    }

    override fun onResume() {
        super.onResume()
        gstreamerManager.onResume()
    }

    override fun onPause() {
        gstreamerManager.onPause()
        super.onPause()
    }

    override fun onDestroy() {
        gstreamerManager.cleanup()
        edgeConnectionManager.disconnect()
        
        // Liberar WakeLock
        wakeLock?.let {
            if (it.isHeld) {
                it.release()
                Timber.d("$TAG: WakeLock liberado")
            }
        }
        
        super.onDestroy()
    }
}
