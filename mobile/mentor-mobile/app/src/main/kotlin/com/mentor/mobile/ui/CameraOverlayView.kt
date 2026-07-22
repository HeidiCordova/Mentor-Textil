package com.mentor.mobile.ui

import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.RectF
import android.util.AttributeSet
import android.view.View

/**
 * Vista de overlay para mostrar guía visual (bounding box) sobre el preview de cámara.
 * Ayuda al operario a encuadrar correctamente los pallets.
 */
class CameraOverlayView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0
) : View(context, attrs, defStyleAttr) {

    private val boundingBoxPaint = Paint().apply {
        color = Color.GREEN
        style = Paint.Style.STROKE
        strokeWidth = 6f
        isAntiAlias = true
    }

    private val cornerPaint = Paint().apply {
        color = Color.GREEN
        style = Paint.Style.STROKE
        strokeWidth = 8f
        isAntiAlias = true
    }

    private val textPaint = Paint().apply {
        color = Color.GREEN
        textSize = 40f
        textAlign = Paint.Align.CENTER
        isAntiAlias = true
    }

    private val backgroundPaint = Paint().apply {
        color = Color.argb(100, 0, 0, 0)
        style = Paint.Style.FILL
    }

    private var boundingBox: RectF? = null
    private val cornerLength = 60f

    init {
        // Vista transparente
        setBackgroundColor(Color.TRANSPARENT)
    }

    override fun onSizeChanged(w: Int, h: Int, oldw: Int, oldh: Int) {
        super.onSizeChanged(w, h, oldw, oldh)
        calculateBoundingBox()
    }

    /**
     * Calcula el bounding box central basado en el tamaño de la vista.
     */
    private fun calculateBoundingBox() {
        val centerX = width / 2f
        val centerY = height / 2f
        
        // Bounding box ocupa 70% del ancho y 50% del alto
        val boxWidth = width * 0.7f
        val boxHeight = height * 0.5f

        boundingBox = RectF(
            centerX - boxWidth / 2,
            centerY - boxHeight / 2,
            centerX + boxWidth / 2,
            centerY + boxHeight / 2
        )
    }

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)

        boundingBox?.let { box ->
            // Dibujar área oscurecida fuera del bounding box
            drawDimmedArea(canvas, box)

            // Dibujar bounding box principal
            canvas.drawRect(box, boundingBoxPaint)

            // Dibujar esquinas decorativas
            drawCorners(canvas, box)

            // Dibujar texto de instrucción
            val instructionText = "Encuadre el pallet aquí"
            canvas.drawText(
                instructionText,
                width / 2f,
                box.top - 30f,
                textPaint
            )

            // Dibujar línea central horizontal
            val centerY = box.centerY()
            canvas.drawLine(
                box.left,
                centerY,
                box.right,
                centerY,
                boundingBoxPaint.apply { alpha = 100 }
            )

            // Dibujar línea central vertical
            val centerX = box.centerX()
            canvas.drawLine(
                centerX,
                box.top,
                centerX,
                box.bottom,
                boundingBoxPaint.apply { alpha = 100 }
            )

            // Restaurar alpha
            boundingBoxPaint.alpha = 255
        }
    }

    /**
     * Dibuja área oscurecida fuera del bounding box.
     */
    private fun drawDimmedArea(canvas: Canvas, box: RectF) {
        // Área superior
        canvas.drawRect(0f, 0f, width.toFloat(), box.top, backgroundPaint)
        
        // Área inferior
        canvas.drawRect(0f, box.bottom, width.toFloat(), height.toFloat(), backgroundPaint)
        
        // Área izquierda
        canvas.drawRect(0f, box.top, box.left, box.bottom, backgroundPaint)
        
        // Área derecha
        canvas.drawRect(box.right, box.top, width.toFloat(), box.bottom, backgroundPaint)
    }

    /**
     * Dibuja esquinas decorativas en el bounding box.
     */
    private fun drawCorners(canvas: Canvas, box: RectF) {
        // Esquina superior izquierda
        canvas.drawLine(box.left, box.top, box.left + cornerLength, box.top, cornerPaint)
        canvas.drawLine(box.left, box.top, box.left, box.top + cornerLength, cornerPaint)

        // Esquina superior derecha
        canvas.drawLine(box.right - cornerLength, box.top, box.right, box.top, cornerPaint)
        canvas.drawLine(box.right, box.top, box.right, box.top + cornerLength, cornerPaint)

        // Esquina inferior izquierda
        canvas.drawLine(box.left, box.bottom - cornerLength, box.left, box.bottom, cornerPaint)
        canvas.drawLine(box.left, box.bottom, box.left + cornerLength, box.bottom, cornerPaint)

        // Esquina inferior derecha
        canvas.drawLine(box.right, box.bottom - cornerLength, box.right, box.bottom, cornerPaint)
        canvas.drawLine(box.right - cornerLength, box.bottom, box.right, box.bottom, cornerPaint)
    }

    /**
     * Actualiza el color del bounding box (útil para feedback visual).
     */
    fun setBoundingBoxColor(color: Int) {
        boundingBoxPaint.color = color
        cornerPaint.color = color
        textPaint.color = color
        invalidate()
    }

    /**
     * Muestra/oculta el overlay.
     */
    fun setOverlayVisible(visible: Boolean) {
        visibility = if (visible) VISIBLE else GONE
    }
}
