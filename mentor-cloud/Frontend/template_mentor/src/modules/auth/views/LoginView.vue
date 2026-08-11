<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/shared/composables/useAuth'

const router = useRouter()
const { login } = useAuth()

const form = ref({
  username: '',
  password: ''
})

const rememberMe = ref(false)
const loading = ref(false)
const error = ref('')
const showPassword = ref(false)

async function handleSubmit() {
  error.value = ''
  loading.value = true

  try {
    const result = await login(form.value)

    if (result.success) {
      router.push('/dashboard')
    } else {
      error.value = result.error || 'Credenciales inválidas'
    }
  } catch (err) {
    error.value = 'Error al iniciar sesión'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrapper">
    <!-- Panel Izquierdo - Información y Animación -->
    <div class="left-panel">
      <div class="factory-animation">
        <!-- Elementos animados de fábrica industrial -->
        <div class="gear gear-1">
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
        </div>
        <div class="gear gear-2">
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
        </div>
        <div class="gear gear-3">
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
          <div class="gear-tooth"></div>
        </div>
        
        <!-- Banda transportadora con productos -->
        <div class="conveyor-belt">
          <div class="product-box"></div>
          <div class="product-box"></div>
          <div class="product-box"></div>
        </div>
        
        <!-- Brazos robóticos -->
        <div class="robot-arm robot-1">
          <div class="arm-base"></div>
          <div class="arm-joint"></div>
          <div class="arm-end"></div>
        </div>
        <div class="robot-arm robot-2">
          <div class="arm-base"></div>
          <div class="arm-joint"></div>
          <div class="arm-end"></div>
        </div>
        
        <!-- Líneas de producción -->
        <div class="production-line line-1"></div>
        <div class="production-line line-2"></div>
        
        <!-- Indicadores de proceso -->
        <div class="process-indicator indicator-1">
          <div class="indicator-pulse"></div>
        </div>
        <div class="process-indicator indicator-2">
          <div class="indicator-pulse"></div>
        </div>
        <div class="process-indicator indicator-3">
          <div class="indicator-pulse"></div>
        </div>
        
        <!-- Estructuras de fábrica -->
        <div class="factory-structure structure-1"></div>
        <div class="factory-structure structure-2"></div>
        <div class="factory-structure structure-3"></div>
        
        <!-- Partículas flotantes -->
        <div class="particle particle-1"></div>
        <div class="particle particle-2"></div>
        <div class="particle particle-3"></div>
        <div class="particle particle-4"></div>
        <div class="particle particle-5"></div>
      </div>
      
      <div class="content-overlay">
        <div class="logo-section">
          <img src="/mentor logo.png" alt="Mentor Textil" class="brand-logo">
        </div>
        
        <div class="welcome-content">
          <h2 class="welcome-title">Excelencia en Monitoreo Industrial</h2>
          <p class="welcome-subtitle">
            Optimiza tus procesos de producción con monitoreo en tiempo real<br>
            y análisis avanzado para la eficiencia industrial
          </p>
          
          <div class="features">
            <div class="feature-item">
              <svg class="feature-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
              </svg>
              <span>Monitoreo en Tiempo Real</span>
            </div>
            <div class="feature-item">
              <svg class="feature-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zM9 17H7v-7h2v7zm4 0h-2V7h2v10zm4 0h-2v-4h2v4z"/>
              </svg>
              <span>Analítica OEE</span>
            </div>
            <div class="feature-item">
              <svg class="feature-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                <path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z"/>
              </svg>
              <span>Seguimiento de Producción</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Panel Derecho - Formulario de Login -->
    <div class="right-panel">
      <div class="login-container">
        <div class="login-header">
          <img src="/mentor logo.png" alt="Mentor Textil" class="login-logo">
          <h2 class="welcome-back">Bienvenido de Nuevo</h2>
          <p class="signin-prompt">Inicia sesión en tu cuenta para continuar</p>
        </div>

        <form @submit.prevent="handleSubmit" class="login-form">
          <!-- Email/Username Input -->
          <div class="form-group">
            <label for="username" class="form-label">Usuario</label>
            <div class="input-wrapper">
              <svg class="input-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"/>
              </svg>
              <input
                id="username"
                v-model="form.username"
                type="text"
                class="form-input"
                placeholder="admin"
                required
                autocomplete="username"
              >
            </div>
          </div>

          <!-- Password Input -->
          <div class="form-group">
            <label for="password" class="form-label">Contraseña</label>
            <div class="input-wrapper">
              <svg class="input-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/>
              </svg>
              <input
                id="password"
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                class="form-input"
                placeholder="••••••"
                required
                autocomplete="current-password"
              >
              <button
                type="button"
                class="password-toggle"
                @click="showPassword = !showPassword"
              >
                <svg v-if="!showPassword" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 7c2.76 0 5 2.24 5 5 0 .65-.13 1.26-.36 1.83l2.92 2.92c1.51-1.26 2.7-2.89 3.43-4.75-1.73-4.39-6-7.5-11-7.5-1.4 0-2.74.25-3.98.7l2.16 2.16C10.74 7.13 11.35 7 12 7zM2 4.27l2.28 2.28.46.46C3.08 8.3 1.78 10.02 1 12c1.73 4.39 6 7.5 11 7.5 1.55 0 3.03-.3 4.38-.84l.42.42L19.73 22 21 20.73 3.27 3 2 4.27zM7.53 9.8l1.55 1.55c-.05.21-.08.43-.08.65 0 1.66 1.34 3 3 3 .22 0 .44-.03.65-.08l1.55 1.55c-.67.33-1.41.53-2.2.53-2.76 0-5-2.24-5-5 0-.79.2-1.53.53-2.2zm4.31-.78l3.15 3.15.02-.16c0-1.66-1.34-3-3-3l-.17.01z"/>
                </svg>
              </button>
            </div>
          </div>

          <!-- Remember -->
          <div class="form-options">
            <label class="remember-me">
              <input
                v-model="rememberMe"
                type="checkbox"
                class="checkbox-input"
              >
              <span class="checkbox-label">Recordarme</span>
            </label>
          </div>

          <!-- Error Message -->
          <div v-if="error" class="error-message">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
            </svg>
            {{ error }}
          </div>

          <!-- Submit Button -->
          <button
            type="submit"
            class="submit-button"
            :disabled="loading"
          >
            <span v-if="!loading" class="button-content">
              <span>Iniciar Sesión</span>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 4l-1.41 1.41L16.17 11H4v2h12.17l-5.58 5.59L12 20l8-8z"/>
              </svg>
            </span>
            <span v-else class="loading-spinner"></span>
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-wrapper {
  display: flex;
  min-height: 100vh;
  background: #1a1d29;
}

/* ===== PANEL IZQUIERDO ===== */
.left-panel {
  position: relative;
  flex: 1;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  overflow: hidden;
  display: none;
}

@media (min-width: 1024px) {
  .left-panel {
    display: block;
  }
}

/* Animación de Fábrica */
.factory-animation {
  position: absolute;
  width: 100%;
  height: 100%;
  opacity: 0.15;
  overflow: hidden;
}

/* Engranajes con dientes */
.gear {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
}

.gear::before {
  content: '';
  position: absolute;
  width: 60%;
  height: 60%;
  border: 3px solid rgba(255, 255, 255, 0.8);
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.05);
}

.gear-tooth {
  position: absolute;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 2px;
}

.gear-1 {
  width: 200px;
  height: 200px;
  top: 15%;
  left: 10%;
  animation: rotateGear 8s linear infinite;
}

.gear-1 .gear-tooth {
  width: 20px;
  height: 40px;
  transform-origin: center 100px;
}

.gear-1 .gear-tooth:nth-child(1) { transform: rotate(0deg) translateY(-90px); }
.gear-1 .gear-tooth:nth-child(2) { transform: rotate(45deg) translateY(-90px); }
.gear-1 .gear-tooth:nth-child(3) { transform: rotate(90deg) translateY(-90px); }
.gear-1 .gear-tooth:nth-child(4) { transform: rotate(135deg) translateY(-90px); }
.gear-1 .gear-tooth:nth-child(5) { transform: rotate(180deg) translateY(-90px); }
.gear-1 .gear-tooth:nth-child(6) { transform: rotate(225deg) translateY(-90px); }
.gear-1 .gear-tooth:nth-child(7) { transform: rotate(270deg) translateY(-90px); }
.gear-1 .gear-tooth:nth-child(8) { transform: rotate(315deg) translateY(-90px); }

.gear-2 {
  width: 140px;
  height: 140px;
  top: 35%;
  right: 15%;
  animation: rotateGear 6s linear infinite reverse;
}

.gear-2 .gear-tooth {
  width: 16px;
  height: 30px;
  transform-origin: center 70px;
}

.gear-2 .gear-tooth:nth-child(1) { transform: rotate(0deg) translateY(-63px); }
.gear-2 .gear-tooth:nth-child(2) { transform: rotate(60deg) translateY(-63px); }
.gear-2 .gear-tooth:nth-child(3) { transform: rotate(120deg) translateY(-63px); }
.gear-2 .gear-tooth:nth-child(4) { transform: rotate(180deg) translateY(-63px); }
.gear-2 .gear-tooth:nth-child(5) { transform: rotate(240deg) translateY(-63px); }
.gear-2 .gear-tooth:nth-child(6) { transform: rotate(300deg) translateY(-63px); }

.gear-3 {
  width: 100px;
  height: 100px;
  bottom: 25%;
  left: 35%;
  animation: rotateGear 5s linear infinite;
}

.gear-3 .gear-tooth {
  width: 14px;
  height: 24px;
  transform-origin: center 50px;
}

.gear-3 .gear-tooth:nth-child(1) { transform: rotate(0deg) translateY(-45px); }
.gear-3 .gear-tooth:nth-child(2) { transform: rotate(72deg) translateY(-45px); }
.gear-3 .gear-tooth:nth-child(3) { transform: rotate(144deg) translateY(-45px); }
.gear-3 .gear-tooth:nth-child(4) { transform: rotate(216deg) translateY(-45px); }
.gear-3 .gear-tooth:nth-child(5) { transform: rotate(288deg) translateY(-45px); }

@keyframes rotateGear {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Banda transportadora con cajas */
.conveyor-belt {
  position: absolute;
  bottom: 15%;
  left: 0;
  right: 0;
  height: 60px;
  background: linear-gradient(180deg, 
    rgba(255, 255, 255, 0.1) 0%,
    rgba(255, 255, 255, 0.2) 50%,
    rgba(255, 255, 255, 0.1) 100%
  );
  border-top: 2px solid rgba(255, 255, 255, 0.3);
  border-bottom: 2px solid rgba(255, 255, 255, 0.3);
  overflow: hidden;
}

.conveyor-belt::before {
  content: '';
  position: absolute;
  width: 200%;
  height: 100%;
  background: repeating-linear-gradient(
    90deg,
    transparent,
    transparent 30px,
    rgba(255, 255, 255, 0.1) 30px,
    rgba(255, 255, 255, 0.1) 32px
  );
  animation: conveyorMove 3s linear infinite;
}

@keyframes conveyorMove {
  from { transform: translateX(0); }
  to { transform: translateX(-60px); }
}

.product-box {
  position: absolute;
  width: 50px;
  height: 45px;
  background: rgba(255, 255, 255, 0.25);
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-radius: 4px;
  top: 50%;
  transform: translateY(-50%);
  animation: productMove 8s linear infinite;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.product-box:nth-child(2) {
  animation-delay: -2.6s;
}

.product-box:nth-child(3) {
  animation-delay: -5.2s;
}

.product-box::before {
  content: '';
  position: absolute;
  top: 8px;
  left: 8px;
  right: 8px;
  bottom: 8px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 2px;
}

@keyframes productMove {
  0% { left: -80px; }
  100% { left: calc(100% + 80px); }
}

/* Brazos robóticos */
.robot-arm {
  position: absolute;
  width: 120px;
  height: 120px;
}

.robot-1 {
  top: 45%;
  left: 25%;
  animation: robotWork1 4s ease-in-out infinite;
}

.robot-2 {
  top: 50%;
  right: 25%;
  animation: robotWork2 5s ease-in-out infinite;
}

.arm-base {
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 30px;
  height: 20px;
  background: rgba(255, 255, 255, 0.3);
  border-radius: 4px;
}

.arm-joint {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  width: 8px;
  height: 50px;
  background: rgba(255, 255, 255, 0.4);
  border-radius: 4px;
  transform-origin: bottom center;
  animation: armRotate 4s ease-in-out infinite;
}

.arm-end {
  position: absolute;
  bottom: 70px;
  left: 50%;
  transform: translateX(-50%);
  width: 20px;
  height: 20px;
  background: rgba(255, 255, 255, 0.5);
  border-radius: 50%;
  box-shadow: 0 0 10px rgba(255, 255, 255, 0.4);
}

@keyframes robotWork1 {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  50% { transform: translateY(-20px) rotate(15deg); }
}

@keyframes robotWork2 {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  50% { transform: translateY(-15px) rotate(-10deg); }
}

@keyframes armRotate {
  0%, 100% { transform: translateX(-50%) rotate(-10deg); }
  50% { transform: translateX(-50%) rotate(20deg); }
}

/* Líneas de producción */
.production-line {
  position: absolute;
  height: 3px;
  background: rgba(255, 255, 255, 0.2);
  left: 0;
  right: 0;
}

.line-1 {
  top: 35%;
  animation: pulse 2s ease-in-out infinite;
}

.line-2 {
  top: 60%;
  animation: pulse 2s ease-in-out infinite;
  animation-delay: 1s;
}

@keyframes pulse {
  0%, 100% { opacity: 0.2; }
  50% { opacity: 0.5; }
}

/* Indicadores de proceso */
.process-indicator {
  position: absolute;
  width: 30px;
  height: 30px;
  border: 3px solid rgba(255, 255, 255, 0.4);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.indicator-1 {
  top: 25%;
  right: 30%;
  animation: indicatorBlink 2s ease-in-out infinite;
}

.indicator-2 {
  top: 55%;
  left: 20%;
  animation: indicatorBlink 2s ease-in-out infinite;
  animation-delay: 0.6s;
}

.indicator-3 {
  bottom: 35%;
  right: 20%;
  animation: indicatorBlink 2s ease-in-out infinite;
  animation-delay: 1.2s;
}

.indicator-pulse {
  width: 12px;
  height: 12px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 50%;
  animation: pulseGlow 2s ease-in-out infinite;
}

@keyframes indicatorBlink {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}

@keyframes pulseGlow {
  0%, 100% { transform: scale(1); opacity: 0.8; }
  50% { transform: scale(1.3); opacity: 1; }
}

/* Estructuras de fábrica */
.factory-structure {
  position: absolute;
  background: rgba(255, 255, 255, 0.08);
  border: 2px solid rgba(255, 255, 255, 0.15);
  border-radius: 4px;
}

.structure-1 {
  width: 80px;
  height: 120px;
  top: 10%;
  right: 5%;
  animation: structureFloat 6s ease-in-out infinite;
}

.structure-2 {
  width: 60px;
  height: 100px;
  bottom: 20%;
  left: 5%;
  animation: structureFloat 7s ease-in-out infinite;
  animation-delay: -2s;
}

.structure-3 {
  width: 70px;
  height: 90px;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  animation: structureFloat 8s ease-in-out infinite;
  animation-delay: -4s;
}

@keyframes structureFloat {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

/* Partículas flotantes */
.particle {
  position: absolute;
  width: 4px;
  height: 4px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 50%;
  animation: particleFloat 15s linear infinite;
}

.particle-1 {
  top: 10%;
  left: 20%;
  animation-duration: 12s;
}

.particle-2 {
  top: 30%;
  left: 60%;
  animation-duration: 15s;
  animation-delay: -3s;
}

.particle-3 {
  top: 60%;
  left: 40%;
  animation-duration: 18s;
  animation-delay: -6s;
}

.particle-4 {
  top: 80%;
  left: 70%;
  animation-duration: 14s;
  animation-delay: -9s;
}

.particle-5 {
  top: 40%;
  left: 10%;
  animation-duration: 16s;
  animation-delay: -12s;
}

@keyframes particleFloat {
  0% { 
    transform: translate(0, 0) scale(1);
    opacity: 0;
  }
  10% {
    opacity: 0.6;
  }
  90% {
    opacity: 0.6;
  }
  100% { 
    transform: translate(100px, -100px) scale(0.5);
    opacity: 0;
  }
}

/* Contenido del Panel Izquierdo */
.content-overlay {
  position: relative;
  z-index: 1;
  padding: 3rem;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  color: white;
}

.logo-section {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 2rem;
}

.brand-logo {
  height: 60px;
  width: auto;
}

.logo-icon {
  width: 40px;
  height: 40px;
  color: #fbbf24;
}

.logo-icon svg {
  width: 100%;
  height: 100%;
}

.brand-name {
  font-size: 1.5rem;
  font-weight: 700;
  letter-spacing: 0.05em;
}

.welcome-content {
  margin-top: auto;
  margin-bottom: 4rem;
}

.welcome-title {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 1rem;
  line-height: 1.2;
}

.welcome-subtitle {
  font-size: 1.125rem;
  opacity: 0.9;
  line-height: 1.6;
  margin-bottom: 2rem;
}

.features {
  display: flex;
  gap: 2rem;
  flex-wrap: wrap;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  opacity: 0.9;
}

.feature-icon {
  width: 20px;
  height: 20px;
}

/* ===== PANEL DERECHO ===== */
.right-panel {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background: #2a2d3a;
}

.login-container {
  width: 100%;
  max-width: min(440px, 92vw);
}

.login-header {
  text-align: center;
  margin-bottom: 2rem;
}

.login-logo {
  height: 50px;
  width: auto;
  margin-bottom: 2rem;
}

.header-logo {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  color: #fbbf24;
  font-size: 1.25rem;
  font-weight: 700;
  margin-bottom: 2rem;
}

.header-logo svg {
  width: 28px;
  height: 28px;
}

.welcome-back {
  font-size: 2rem;
  font-weight: 700;
  color: white;
  margin-bottom: 0.5rem;
}

.signin-prompt {
  color: #9ca3af;
  font-size: 0.875rem;
}

/* Formulario */
.login-form {
  margin-top: 2rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-label {
  display: block;
  color: white;
  font-size: 0.875rem;
  font-weight: 500;
  margin-bottom: 0.5rem;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 1rem;
  width: 20px;
  height: 20px;
  color: #6b7280;
  pointer-events: none;
}

.form-input {
  width: 100%;
  padding: 0.875rem 1rem 0.875rem 3rem;
  background: #1a1d29;
  border: 1px solid #374151;
  border-radius: 0.5rem;
  color: white;
  font-size: 0.875rem;
  transition: all 0.2s;
  outline: none;
}

.form-input:focus {
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.form-input::placeholder {
  color: #6b7280;
}

.password-toggle {
  position: absolute;
  right: 1rem;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.25rem;
  color: #6b7280;
  transition: color 0.2s;
}

.password-toggle:hover {
  color: #9ca3af;
}

.password-toggle svg {
  width: 20px;
  height: 20px;
}

.form-options {
  display: flex;
  align-items: center;
  margin-bottom: 1.5rem;
}

.remember-me {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  color: #d1d5db;
  font-size: 0.875rem;
}

.checkbox-input {
  width: 1rem;
  height: 1rem;
  border-radius: 0.25rem;
  cursor: pointer;
  accent-color: #667eea;
}

.checkbox-label {
  user-select: none;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 0.5rem;
  color: #fca5a5;
  font-size: 0.875rem;
  margin-bottom: 1.5rem;
}

.error-message svg {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.submit-button {
  width: 100%;
  padding: 0.875rem 1.5rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  border-radius: 0.5rem;
  color: white;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.submit-button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
}

.submit-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.button-content {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.button-content svg {
  width: 20px;
  height: 20px;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Responsive */
@media (max-width: 1023px) {
  .right-panel {
    background: linear-gradient(135deg, rgba(102, 126, 234, 0.05) 0%, rgba(118, 75, 162, 0.05) 100%), #2a2d3a;
  }
}

@media (max-width: 640px) {
  .right-panel {
    padding: 1.5rem;
  }

  .welcome-back {
    font-size: 1.5rem;
  }
}
</style>
