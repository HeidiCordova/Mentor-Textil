<template>
  <div class="flex gap-6 min-h-[calc(100vh-7rem)]">

    <aside class="w-52 shrink-0 hidden lg:flex lg:flex-col fixed top-16 bottom-0 left-0 z-30 bg-slate-950 border-r border-slate-800 px-3 py-6">
      <!-- Botón volver -->
      <button @click="router.push('/config')"
        class="flex items-center gap-1.5 px-3 py-2 mb-4 rounded-lg text-xs font-medium text-gray-400 hover:text-white hover:bg-slate-800 transition-colors">
        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
        </svg>
        Todas las líneas
      </button>
      <!-- Line badge -->
      <div class="px-3 py-2 mb-3 rounded-lg bg-blue-900/20 border border-blue-700/30">
        <p class="text-[10px] text-blue-400 uppercase tracking-wider mb-0.5">Editando</p>
        <p class="font-mono text-xs text-blue-300 font-semibold truncate">Línea {{ lineaId || 'nueva' }}</p>
      </div>
      <nav class="flex-1 space-y-1 overflow-y-auto">
        <button
          v-for="s in sections" :key="s.id"
          @click="activeSection = s.id"
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors text-left"
          :class="activeSection === s.id
            ? 'bg-blue-600/20 text-blue-400 border-l-2 border-blue-500'
            : 'text-gray-400 hover:bg-slate-800 hover:text-gray-200'"
        >
          <span class="w-4 h-4 shrink-0" v-html="s.svg"></span>
          {{ s.label }}
        </button>
      </nav>

      <div class="pt-4 space-y-2 border-t border-slate-800">
        <div class="flex items-center gap-2 px-3 text-xs">
          <span class="w-1.5 h-1.5 rounded-full" :class="savedDot"></span>
          <span :class="lastSaved ? 'text-green-400' : 'text-gray-500'">
            {{ lastSaved ? `Guardado ${lastSaved}` : `v${config.config_version}` }}
          </span>
        </div>
        <button @click="saveConfig" :disabled="saving"
          class="w-full btn-primary flex items-center justify-center gap-2 text-sm">
          <svg v-if="!saving" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
          </svg>
          <svg v-else class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          {{ saving ? 'Guardando...' : 'Guardar' }}
        </button>
      </div>
    </aside>

    <main class="flex-1 min-w-0 space-y-6 lg:pl-56">

      <div class="lg:hidden flex overflow-x-auto gap-1 pb-2 border-b border-slate-700">
        <button v-for="s in sections" :key="s.id"
          @click="activeSection = s.id"
          class="px-3 py-1.5 text-xs font-medium rounded whitespace-nowrap transition-colors"
          :class="activeSection === s.id ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'">
          {{ s.label }}
        </button>
      </div>

      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-white">{{ currentSection.label }}</h1>
          <p class="mt-0.5 text-sm text-gray-400">{{ currentSection.desc }}</p>
        </div>
        <span class="hidden lg:inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium"
          :class="lastSaved ? 'bg-green-900 text-green-300' : 'bg-slate-700 text-gray-400'">
          <span class="w-1.5 h-1.5 rounded-full mr-1.5" :class="savedDot"></span>
          {{ lastSaved ? `Guardado ${lastSaved}` : 'Sin cambios' }}
        </span>
      </div>

      <!-- DISPOSITIVO -->
      <template v-if="activeSection === 'device'">

        <!-- Header: info línea -->
        <div class="flex items-center gap-2 text-xs text-gray-500">
          Editando:
          <span class="font-mono text-blue-400 bg-blue-900/30 px-2 py-0.5 rounded">Línea {{ lineaId || '(sin seleccionar)' }}</span>
        </div>

        <div class="card">
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label class="label-field">Modo de Deteccion <HelpTip :text="tips.mode" /></label>
              <select v-model="config.mode" class="mt-1 input-field">
                <option value="textil">Textil</option>
              </select>
            </div>
            <div>
              <label class="label-field">Nombre de Linea (OEE) <HelpTip :text="tips.lineName" /></label>
              <input v-model="oee.line_name" type="text" class="mt-1 input-field font-mono" placeholder="MENTOR_PLANTA01_L1">
            </div>
            <div>
              <label class="label-field">Backend de Captura <HelpTip :text="tips.frameBackend" /></label>
              <select v-model="frameBackend" class="mt-1 input-field">
                <option value="opencv">OpenCV (siempre funciona)</option>
                <option value="gstreamer">GStreamer NVDEC (GPU Jetson)</option>
              </select>
            </div>
          </div>
        </div>

        <!-- Líneas registradas: tabla compacta -->
        <div class="card space-y-3">
          <div class="flex items-center justify-between gap-2">
            <h3 class="section-title mb-0">Líneas registradas</h3>
            <div class="flex gap-1.5">
              <button @click="loadLines" :disabled="linesLoading"
                class="flex items-center gap-1.5 px-2.5 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-white disabled:opacity-50 transition-colors">
                <svg class="w-3 h-3" :class="linesLoading ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
                Recargar
              </button>
              <button @click="router.push('/config')"
                class="flex items-center gap-1.5 px-2.5 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-gray-300 hover:text-white transition-colors">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16"/>
                </svg>
                Ver todas
              </button>
            </div>
          </div>

          <!-- Cabecera tabla -->
          <div v-if="linesList.length > 0" class="rounded-lg bg-slate-900 overflow-hidden">
            <div class="grid grid-cols-12 px-4 py-2 text-[10px] font-medium text-gray-500 uppercase tracking-wider border-b border-slate-700/60">
              <span class="col-span-3">Línea</span>
              <span class="col-span-4">Nombre (OEE)</span>
              <span class="col-span-2 text-center">Versión</span>
              <span class="col-span-3 text-right">Acciones</span>
            </div>
            <div v-for="l in linesList" :key="l"
              class="grid grid-cols-12 items-center px-4 py-3 border-b border-slate-700/30 last:border-0 transition-colors"
              :class="String(l) === String(lineaId) ? 'bg-blue-900/20 border-l-2 border-l-blue-500' : 'hover:bg-slate-800/50'">

              <!-- Linea ID -->
              <div class="col-span-3 flex items-center gap-2 min-w-0">
                <svg class="w-3.5 h-3.5 shrink-0" :class="String(l) === String(lineaId) ? 'text-blue-400' : 'text-gray-500'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"/>
                </svg>
                <span class="text-xs font-mono truncate" :class="String(l) === String(lineaId) ? 'text-blue-300 font-semibold' : 'text-gray-300'">{{ l }}</span>
                <span v-if="String(l) === String(lineaId)" class="shrink-0 text-[9px] px-1.5 py-0.5 rounded-full bg-blue-600/40 text-blue-300">activa</span>
              </div>

              <!-- Line name -->
              <div class="col-span-4 min-w-0">
                <span class="text-xs font-mono truncate" :class="lineMeta[l]?.line_name ? 'text-gray-300' : 'text-gray-600'">
                  {{ lineMeta[l]?.line_name || '—' }}
                </span>
              </div>

              <!-- Config version -->
              <div class="col-span-2 text-center">
                <span class="text-[10px] font-mono px-1.5 py-0.5 rounded" :class="lineMeta[l]?.config_version ? 'bg-slate-700 text-gray-300' : 'text-gray-600'">
                  {{ lineMeta[l]?.config_version != null ? 'v' + lineMeta[l].config_version : '—' }}
                </span>
              </div>

              <!-- Acciones -->
              <div class="col-span-3 flex items-center justify-end gap-1.5">
                <button @click.stop="selectLine(l)" :disabled="String(l) === String(lineaId)"
                  title="Cargar configuración de esta línea en el formulario"
                  class="flex items-center gap-1 px-2 py-1 rounded text-[10px] font-medium transition-colors disabled:opacity-30 disabled:cursor-default"
                  :class="String(l) === String(lineaId) ? 'bg-slate-700 text-gray-500' : 'bg-slate-700 hover:bg-blue-700 text-gray-300 hover:text-white'">
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/>
                  </svg>
                  Cargar
                </button>
                <button @click.stop="updateLineConfig(l)" :disabled="lineUpdating === l"
                  title="Guardar la configuración actual del formulario en esta línea"
                  class="flex items-center gap-1 px-2 py-1 rounded text-[10px] font-medium bg-emerald-800 hover:bg-emerald-700 text-emerald-200 transition-colors disabled:opacity-50">
                  <svg class="w-3 h-3" :class="lineUpdating === l ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
                  </svg>
                  {{ lineUpdating === l ? 'Guardando...' : 'Actualizar' }}
                </button>
              </div>
            </div>
          </div>

          <div v-else class="px-4 py-6 text-center text-xs text-gray-500 bg-slate-900 rounded-lg">
            {{ linesLoading ? 'Cargando líneas...' : 'No hay líneas registradas.' }}
          </div>
          <p class="text-[10px] text-gray-600">
            <span class="text-gray-500 font-medium">Cargar</span> carga la config de esa línea en el formulario. 
            <span class="text-gray-500 font-medium">Actualizar</span> escribe el formulario actual en esa línea directamente en la BD.
          </p>
        </div>

      </template>

      <!-- CAMARA -->
      <template v-if="activeSection === 'camera'">
        <div class="card">
          <div class="flex items-center justify-between mb-4">
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs" :class="cameraStatusClass">{{ cameraStatusLabel }}</span>
          </div>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div class="sm:col-span-2">
              <label class="label-field">URL de Camara (RTSP / USB) <HelpTip :text="tips.cameraUrl" /></label>
              <div class="flex gap-2 mt-1">
                <input v-model="camera.url" type="text" class="flex-1 input-field font-mono text-xs"
                  placeholder="rtsp://usuario:pass@192.168.1.100:554/stream" @change="cameraHealth = null">
                <button type="button" :disabled="!camera.url || cameraChecking" @click="checkCameraHealth"
                  class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-blue-700 hover:bg-blue-600 text-white disabled:opacity-40 disabled:cursor-not-allowed transition-colors">
                  <svg v-if="!cameraChecking" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                  </svg>
                  <svg v-else class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                  </svg>
                  {{ cameraChecking ? 'Verificando...' : 'Verificar' }}
                </button>
              </div>
            </div>
            <div>
              <label class="label-field">FPS objetivo <HelpTip :text="tips.fps" /></label>
              <input v-model.number="camera.fps" type="number" min="1" max="60" class="mt-1 input-field">
            </div>
          </div>

          <transition name="fade">
            <div v-if="cameraHealth" class="mt-4 rounded-lg border p-4 text-xs font-mono space-y-1"
              :class="{
                'border-green-700 bg-green-950/40 text-green-300': cameraHealth.status === 'ok',
                'border-yellow-700 bg-yellow-950/40 text-yellow-300': ['tcp_ok_no_ffprobe','ok_busy'].includes(cameraHealth.status),
                'border-red-700 bg-red-950/40 text-red-300': !['ok','tcp_ok_no_ffprobe','ok_busy'].includes(cameraHealth.status)
              }">
              <div class="flex items-center gap-2 font-semibold text-sm mb-2">
                <svg v-if="['ok','ok_busy'].includes(cameraHealth.status)" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
                <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M12 3a9 9 0 100 18A9 9 0 0012 3z"/>
                </svg>
                {{ { ok:'Camara OK', ok_busy:'Cámara activa (en uso)', tcp_ok_no_ffprobe:'TCP OK - sin ffprobe', tcp_unreachable:'Sin conexion TCP', rtsp_error:'Error RTSP', rtsp_no_streams:'Sin streams' }[cameraHealth.status] || cameraHealth.status }}
              </div>
              <div class="grid grid-cols-2 gap-x-6 gap-y-0.5">
                <span class="text-gray-400">URL:</span><span class="truncate">{{ cameraHealth.url_masked }}</span>
                <span class="text-gray-400">TCP:</span><span>{{ cameraHealth.tcp_reachable ? `OK (${cameraHealth.tcp_latency_ms} ms)` : 'no alcanzable' }}</span>
                <template v-if="cameraHealth.ffprobe_available !== undefined">
                  <span class="text-gray-400">ffprobe:</span><span>{{ cameraHealth.ffprobe_available ? 'disponible' : 'no disponible' }}</span>
                </template>
                <template v-if="cameraHealth.rtsp_ok !== undefined">
                  <span class="text-gray-400">RTSP:</span><span>{{ cameraHealth.rtsp_ok ? 'stream OK' : 'error' }}</span>
                </template>
                <template v-if="cameraHealth.rtsp_codec">
                  <span class="text-gray-400">Codec:</span><span>{{ cameraHealth.rtsp_codec }}</span>
                </template>
                <template v-if="cameraHealth.rtsp_message">
                  <span class="text-gray-400">Info:</span><span class="break-all">{{ cameraHealth.rtsp_message }}</span>
                </template>
                <span class="text-gray-400">Verificado:</span><span>{{ new Date(cameraHealth.checked_at).toLocaleTimeString() }}</span>
              </div>
            </div>
          </transition>
        </div>
      </template>

      <!-- DETECCION -->
      <template v-if="activeSection === 'detection'">

        <div class="flex items-center justify-between mb-2">
          <div></div>
          <button
            @click="openLab"
            :disabled="!camera.url"
            class="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold transition-colors"
            :class="camera.url ? 'bg-indigo-600 hover:bg-indigo-500 text-white' : 'bg-slate-700 text-gray-500 cursor-not-allowed'"
            title="Abrir Modo Lab — configuración con vista en vivo de cámara">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M9 3H5a2 2 0 00-2 2v4m0 0h18M3 9v10a2 2 0 002 2h5m4 0h5a2 2 0 002-2V9M9 21v-6a2 2 0 012-2h2a2 2 0 012 2v6"/>
            </svg>
            Modo Lab
            <span v-if="!camera.url" class="text-[10px] font-normal text-gray-500">Requiere URL de cámara</span>
          </button>
        </div>

        <div class="card">
          <h3 class="section-title">Region de Interes (ROI) <HelpTip :text="tips.roi" /></h3>
          <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div>
              <label class="label-field">X (origen) <HelpTip :text="tips.roiX" /></label>
              <input v-model.number="roi[0]" type="number" min="0" class="mt-1 input-field">
            </div>
            <div>
              <label class="label-field">Y (origen) <HelpTip :text="tips.roiY" /></label>
              <input v-model.number="roi[1]" type="number" min="0" class="mt-1 input-field">
            </div>
            <div>
              <label class="label-field">Ancho (px) <HelpTip :text="tips.roiW" /></label>
              <input v-model.number="roi[2]" type="number" min="1" class="mt-1 input-field">
            </div>
            <div>
              <label class="label-field">Alto (px) <HelpTip :text="tips.roiH" /></label>
              <input v-model.number="roi[3]" type="number" min="1" class="mt-1 input-field">
            </div>
          </div>
          <div class="mt-3">
            <label class="label-field">Margen inferior excluido (px) <HelpTip :text="tips.roiMargin" /></label>
            <div class="flex items-center space-x-2 mt-1">
              <input v-model.number="roi[4]" type="range" min="0" :max="Math.floor(roi[3] * 0.4)" step="1" class="flex-1 accent-orange-500">
              <span class="w-10 text-right text-sm font-mono text-white">{{ roi[4] ?? 0 }}px</span>
            </div>
          </div>
          <div class="mt-4 relative bg-slate-900 rounded overflow-hidden" style="height: 120px;">
            <svg class="w-full h-full" viewBox="0 0 640 360" preserveAspectRatio="none">
              <rect x="0" y="0" width="640" height="360" fill="#0f172a"/>
              <rect :x="roi[0]" :y="roi[1]" :width="roi[2]" :height="roi[3]" fill="none" stroke="#3b82f6" stroke-width="2" stroke-dasharray="6 3"/>
              <rect v-if="roi[4] > 0" :x="roi[0]" :y="roi[1] + roi[3] - roi[4]" :width="roi[2]" :height="roi[4]" fill="#f97316" opacity="0.25"/>
              <text x="4" y="14" fill="#64748b" font-size="12" font-family="monospace">640 x 360</text>
              <text :x="roi[0] + 4" :y="roi[1] + 14" fill="#93c5fd" font-size="10" font-family="monospace">{{ roi[2] }}x{{ roi[3] }}</text>
            </svg>
          </div>
        </div>

        <!-- Detección textil -->
        <div class="card">
          <h3 class="section-title">Umbrales de Deteccion <HelpTip :text="tips.thresholds" /></h3>
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
            <div v-for="field in thresholdFields" :key="field.key">
              <label class="label-field">{{ field.label }} <HelpTip :text="field.tip" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.thresholds[field.key]" type="range" :min="field.min" :max="field.max" :step="field.step" class="flex-1 accent-blue-500">
                <span class="w-12 text-right text-sm font-mono text-white">{{ (config.thresholds[field.key] ?? field.default).toFixed(2) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <h3 class="section-title">Histeresis <HelpTip :text="tips.hysteresis" /></h3>
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <div>
              <label class="label-field">Umbral Alto (activacion) <HelpTip :text="tips.hystHigh" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.thresholds.high" type="range" min="0" max="1" step="0.05" class="flex-1 accent-green-500">
                <span class="w-12 text-right text-sm font-mono text-white">{{ (config.thresholds.high ?? 0.7).toFixed(2) }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Umbral Bajo (cancelacion) <HelpTip :text="tips.hystLow" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.thresholds.low" type="range" min="0" max="1" step="0.05" class="flex-1 accent-red-500">
                <span class="w-12 text-right text-sm font-mono text-white">{{ (config.thresholds.low ?? 0.3).toFixed(2) }}</span>
              </div>
            </div>
          </div>
          <div class="mt-4 relative bg-slate-900 rounded p-3">
            <svg viewBox="0 0 300 60" class="w-full" style="height:56px">
              <line x1="0" y1="30" x2="300" y2="30" stroke="#334155" stroke-width="1"/>
              <rect :x="(config.thresholds.low ?? 0.3) * 300" y="10" :width="((config.thresholds.high ?? 0.7) - (config.thresholds.low ?? 0.3)) * 300" height="40" fill="#1e3a5f" opacity="0.6"/>
              <line :x1="(config.thresholds.high ?? 0.7) * 300" y1="0" :x2="(config.thresholds.high ?? 0.7) * 300" y2="60" stroke="#22c55e" stroke-width="1.5" stroke-dasharray="3 2"/>
              <line :x1="(config.thresholds.low ?? 0.3) * 300" y1="0" :x2="(config.thresholds.low ?? 0.3) * 300" y2="60" stroke="#ef4444" stroke-width="1.5" stroke-dasharray="3 2"/>
              <text :x="(config.thresholds.high ?? 0.7) * 300 + 3" y="10" fill="#86efac" font-size="9" font-family="monospace">HIGH</text>
              <text :x="(config.thresholds.low ?? 0.3) * 300 + 3" y="10" fill="#fca5a5" font-size="9" font-family="monospace">LOW</text>
            </svg>
          </div>
        </div>

        <div class="card">
          <h3 class="section-title">Maquina de Estados (FSM) <HelpTip :text="tips.fsm" /></h3>
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <div>
              <label class="label-field">Frames de confirmacion <HelpTip :text="tips.fsmFrames" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.fsm.n_frames" type="range" min="1" max="30" step="1" class="flex-1 accent-blue-500">
                <span class="w-10 text-right text-sm font-mono text-white">{{ config.fsm.n_frames ?? 3 }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Cooldown (frames) <HelpTip :text="tips.fsmCooldown" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.fsm.cooldown" type="range" min="0" max="60" step="1" class="flex-1 accent-purple-500">
                <span class="w-10 text-right text-sm font-mono text-white">{{ config.fsm.cooldown ?? 8 }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Frames salida (exit_frames) <HelpTip text="Frames consecutivos con score bajo necesarios para confirmar que el objeto salio del ROI y cerrar el evento WAIT_EXIT." /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.fsm.exit_frames" type="range" min="1" max="100" step="1" class="flex-1 accent-orange-500">
                <span class="w-10 text-right text-sm font-mono text-white">{{ config.fsm.exit_frames ?? 5 }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Timeout salida (max_wait) <HelpTip text="Maximo de frames esperando salida del objeto antes de forzar el cierre del evento. Evita que la FSM quede bloqueada en WAIT_EXIT. A 8fps: 50 frames = ~6s." /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.fsm.max_wait_exit_frames" type="range" min="1" max="200" step="1" class="flex-1 accent-rose-500">
                <span class="w-10 text-right text-sm font-mono text-white">{{ config.fsm.max_wait_exit_frames ?? 50 }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Min. entre conteos, s (min_rearm_s) <HelpTip text="Segundos minimos entre dos CORTEs consecutivos. Elimina falsos positivos del ciclo de timeout. Valor recomendado: un poco menos que el tiempo minimo entre piezas." /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="config.fsm.min_rearm_s" type="range" min="0" max="60" step="0.5" class="flex-1 accent-violet-500">
                <span class="w-10 text-right text-sm font-mono text-white">{{ config.fsm.min_rearm_s ?? 6 }}s</span>
              </div>
            </div>
          </div>
          <div class="mt-4 bg-slate-900 rounded p-3">
            <svg viewBox="0 0 400 50" class="w-full" style="height:48px">
              <rect x="2" y="10" width="72" height="30" rx="4" fill="#1e293b" stroke="#475569" stroke-width="1.5"/>
              <text x="38" y="30" text-anchor="middle" fill="#94a3b8" font-size="10">IDLE</text>
              <rect x="110" y="10" width="72" height="30" rx="4" fill="#1e293b" stroke="#3b82f6" stroke-width="1.5"/>
              <text x="146" y="30" text-anchor="middle" fill="#93c5fd" font-size="10">DETECTING</text>
              <rect x="218" y="10" width="80" height="30" rx="4" fill="#1e293b" stroke="#22c55e" stroke-width="1.5"/>
              <text x="258" y="30" text-anchor="middle" fill="#86efac" font-size="10">WAIT_EXIT</text>
              <rect x="324" y="10" width="72" height="30" rx="4" fill="#1e293b" stroke="#a855f7" stroke-width="1.5"/>
              <text x="360" y="30" text-anchor="middle" fill="#d8b4fe" font-size="10">COOLDOWN</text>
              <line x1="74" y1="25" x2="110" y2="25" stroke="#3b82f6" stroke-width="1" marker-end="url(#arr)"/>
              <line x1="182" y1="25" x2="218" y2="25" stroke="#22c55e" stroke-width="1" marker-end="url(#arr)"/>
              <line x1="298" y1="25" x2="324" y2="25" stroke="#a855f7" stroke-width="1" marker-end="url(#arr)"/>
              <defs><marker id="arr" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0,0 L0,6 L6,3 z" fill="#64748b"/></marker></defs>
            </svg>
          </div>
        </div>
      </template><!-- end detection section -->

      <!-- OEE -->
      <template v-if="activeSection === 'oee'">
        <div class="flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-mono bg-indigo-900/30 border border-indigo-700/40 text-indigo-300">
          <span>Modo activo:</span>
          <span class="font-semibold">Textil — umbral micro 2min, snapshot 30min</span>
          <span class="ml-auto text-gray-500">Configuración OEE de la línea textil.</span>
        </div>

        <div class="card">
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <div>
              <label class="label-field">Límite micro-parada <HelpTip :text="tips.microStop" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="oee.micro_stop_max_s" type="range" min="10" max="3600" step="30" class="flex-1 accent-yellow-500">
                <span class="w-20 text-right text-sm font-mono text-white">{{ fmtSeg(oee.micro_stop_max_s) }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Límite parada <HelpTip :text="tips.stopMax" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="oee.stop_max_s" type="range" min="30" max="86400" step="30" class="flex-1 accent-red-500">
                <span class="w-20 text-right text-sm font-mono text-white">{{ fmtSeg(oee.stop_max_s) }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Intervalo de snapshot <HelpTip :text="tips.snapshotInterval" /></label>
              <div class="flex items-center space-x-2 mt-1">
                <input v-model.number="oee.snapshot_interval_s" type="range" min="10" max="3600" step="30" class="flex-1 accent-blue-500">
                <span class="w-20 text-right text-sm font-mono text-white">{{ fmtSeg(oee.snapshot_interval_s) }}</span>
              </div>
            </div>
            <div>
              <label class="label-field">Velocidad nominal <HelpTip text="Velocidad de producción de la línea en condiciones óptimas. Internamente siempre en u/s — la UI convierte automáticamente según la unidad seleccionada." /></label>
              <div class="flex items-center gap-2 mt-1">
                <input v-model.number="velNominalDisplay" type="number" min="0.0001" step="0.1" class="input-field flex-1">
                <select v-model="oee.vel_unit" class="input-field w-28 shrink-0">
                  <option value="us">u / seg</option>
                  <option value="uh">u / hora</option>
                </select>
              </div>
              <p class="mt-1 text-[10px] text-gray-600 font-mono">
                Almacenado: {{ (oee.vel_nominal_us ?? 0).toFixed(6) }} u/s
              </p>
            </div>
          </div>
          <div class="mt-4 bg-slate-900 rounded p-3">
            <svg viewBox="0 0 400 40" class="w-full" style="height:38px">
              <rect x="0" y="8" :width="microBarW" height="24" rx="3" fill="#eab308" opacity="0.3"/>
              <rect :x="microBarW" y="8" :width="stopBarW" height="24" rx="3" fill="#ef4444" opacity="0.3"/>
              <rect :x="microBarW + stopBarW" y="8" :width="Math.max(0, 400 - microBarW - stopBarW)" height="24" rx="3" fill="#7f1d1d" opacity="0.2"/>
              <line :x1="microBarW" y1="4" :x2="microBarW" y2="36" stroke="#eab308" stroke-width="1.5" stroke-dasharray="3 2"/>
              <line :x1="microBarW + stopBarW" y1="4" :x2="microBarW + stopBarW" y2="36" stroke="#ef4444" stroke-width="1.5" stroke-dasharray="3 2"/>
              <text x="4" y="24" fill="#fbbf24" font-size="9" font-family="monospace">MICRO-PARADA</text>
              <text :x="microBarW + 4" y="24" fill="#fca5a5" font-size="9" font-family="monospace">PARADA</text>
              <text :x="microBarW + stopBarW + 4" y="24" fill="#991b1b" font-size="9" font-family="monospace">PARADA MAYOR</text>
            </svg>
          </div>
        </div>
      </template>


      <!-- CLOUD -->
      <template v-if="activeSection === 'cloud'">
        <div class="card space-y-5">

          <!-- Paso 1: credenciales -->
          <div class="flex items-start justify-between gap-3 flex-wrap">
            <div>
              <p class="text-xs font-semibold text-cyan-400 uppercase tracking-widest mb-0.5">Paso 1 — Credenciales</p>
              <p class="text-xs text-gray-500">Ingresa la URL y la API Key antes de continuar con el resto de la configuracion.</p>
            </div>
            <!-- badge estado operacional -->
            <span class="text-xs font-mono px-2 py-1 rounded shrink-0"
              :class="cloudHealth?.cloud_status === 'ok' ? 'bg-green-900/40 text-green-400'
                     : cloudHealth?.cloud_status === 'degraded' ? 'bg-yellow-900/40 text-yellow-400'
                     : 'bg-slate-700 text-gray-500'">
              {{ cloudHealth ? (cloudHealth.cloud_status === 'ok' ? 'Nube conectada' : cloudHealth.cloud_status === 'degraded' ? 'Nube degradada' : 'Nube sin conexion') : 'Sin estado' }}
            </span>
          </div>

          <!-- Campos principales: URL + API Key -->
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <div>
              <label class="label-field">URL del servidor cloud <HelpTip :text="tips.cloudUrl" /></label>
              <input v-model="cloud.cloud_url" type="url" placeholder="https://api.mentoredge.io" class="mt-1 input-field font-mono text-xs">
              <p class="mt-1 text-[10px] text-gray-600">POST {{ cloud.cloud_url }}/api/v1/edge/oee</p>
            </div>
            <div>
              <label class="label-field">
                API Key
                <HelpTip :text="tips.cloudApiKey" />
                <span v-if="cloud.cloud_api_key" class="ml-2 text-[10px] text-green-500 font-mono">&#10003; configurada</span>
                <span v-else class="ml-2 text-[10px] text-yellow-500 font-mono">sin clave</span>
              </label>
              <div class="relative mt-1">
                <input v-model="cloud.cloud_api_key" :type="showApiKey ? 'text' : 'password'" placeholder="Pega aqui la API Key del dispositivo..." class="input-field font-mono text-xs pr-10">
                <button type="button" @click="showApiKey = !showApiKey" class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path v-if="!showApiKey" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                    <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Paso 2: validar conexion -->
          <div class="border-t border-slate-700/60 pt-4">
            <p class="text-xs font-semibold text-cyan-400 uppercase tracking-widest mb-3">Paso 2 — Validar conexion</p>
            <div class="flex gap-2 flex-wrap">
              <button type="button" @click="pingCloud" :disabled="cloudPinging"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-violet-700 hover:bg-violet-600 text-white transition-colors disabled:opacity-50">
                <svg class="w-3.5 h-3.5" :class="cloudPinging ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                </svg>
                {{ cloudPinging ? 'Probando...' : 'Probar conexion' }}
              </button>
              <button type="button" @click="checkCloudHealth" :disabled="cloudChecking"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-gray-300 transition-colors disabled:opacity-50">
                <svg class="w-3.5 h-3.5" :class="cloudChecking ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
                Estado del enviador
              </button>
            </div>

            <!-- Resultado del ping -->
            <div v-if="cloudPing !== null" class="mt-3 rounded-lg border overflow-hidden"
              :class="cloudPing.reachable && cloudPing.status_code >= 200 && cloudPing.status_code < 300
                ? 'border-green-800/40 bg-green-950/20'
                : cloudPing.reachable
                  ? 'border-yellow-800/40 bg-yellow-950/20'
                  : 'border-red-800/40 bg-red-950/20'">
              <div class="flex items-center gap-2 px-4 py-2.5 border-b"
                :class="cloudPing.reachable && cloudPing.status_code >= 200 && cloudPing.status_code < 300
                  ? 'border-green-800/40'
                  : cloudPing.reachable ? 'border-yellow-800/40' : 'border-red-800/40'">
                <svg v-if="cloudPing.reachable && cloudPing.status_code >= 200 && cloudPing.status_code < 300"
                  class="w-4 h-4 text-green-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                </svg>
                <svg v-else-if="cloudPing.reachable" class="w-4 h-4 text-yellow-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
                </svg>
                <svg v-else class="w-4 h-4 text-red-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                </svg>
                <p class="text-xs font-medium"
                  :class="cloudPing.reachable && cloudPing.status_code >= 200 && cloudPing.status_code < 300
                    ? 'text-green-300'
                    : cloudPing.reachable ? 'text-yellow-300' : 'text-red-300'">
                  {{
                    cloudPing.reachable && cloudPing.status_code >= 200 && cloudPing.status_code < 300
                      ? 'Conexion exitosa — servidor alcanzable y autenticado'
                      : cloudPing.reachable && (cloudPing.status_code === 401 || cloudPing.status_code === 403)
                        ? 'Servidor alcanzable pero API Key incorrecta o sin permisos'
                        : cloudPing.reachable
                          ? 'Servidor alcanzable pero responde con error ' + cloudPing.status_code
                          : cloudPing.error?.includes('URL') ? cloudPing.error : 'No se pudo alcanzar el servidor'
                  }}
                </p>
              </div>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 px-4 py-3 text-xs">
                <div>
                  <p class="text-gray-500">Latencia</p>
                  <p class="font-mono" :class="cloudPing.reachable ? 'text-white' : 'text-gray-500'">{{ cloudPing.reachable ? cloudPing.latency_ms + ' ms' : '---' }}</p>
                  <div v-if="cloudPing.reachable" class="mt-1 h-1.5 rounded-full bg-slate-700 overflow-hidden">
                    <div class="h-full rounded-full transition-all"
                      :class="cloudPing.latency_ms < 200 ? 'bg-green-500' : cloudPing.latency_ms < 600 ? 'bg-yellow-500' : 'bg-red-500'"
                      :style="{ width: Math.min(cloudPing.latency_ms / 10, 100) + '%' }">
                    </div>
                  </div>
                </div>
                <div>
                  <p class="text-gray-500">Codigo HTTP</p>
                  <p class="font-mono font-bold"
                    :class="cloudPing.status_code >= 200 && cloudPing.status_code < 300 ? 'text-green-400'
                           : cloudPing.status_code === 401 || cloudPing.status_code === 403 ? 'text-yellow-400'
                           : cloudPing.status_code >= 400 ? 'text-red-400' : 'text-gray-400'">
                    {{ cloudPing.status_code || '---' }}
                    <span class="font-normal text-gray-500 ml-1 text-[10px]">{{
                      cloudPing.status_code === 200 ? 'OK'
                      : cloudPing.status_code === 401 ? 'Unauthorized'
                      : cloudPing.status_code === 403 ? 'Forbidden'
                      : cloudPing.status_code === 404 ? 'Not Found'
                      : cloudPing.status_code >= 500 ? 'Server Error' : ''
                    }}</span>
                  </p>
                </div>
                <div>
                  <p class="text-gray-500">URL probada</p>
                  <p class="font-mono text-gray-400 truncate text-[10px]" :title="cloudPing.tested_url">{{ cloudPing.tested_url || '(no configurada)' }}</p>
                </div>
                <div>
                  <p class="text-gray-500">Probado a las</p>
                  <p class="font-mono text-gray-300">{{ cloudPing.tested_at ? new Date(cloudPing.tested_at).toLocaleTimeString('es-MX') : '---' }}</p>
                </div>
              </div>
              <div v-if="cloudPing.error && !cloudPing.reachable && !cloudPing.error.includes('URL')" class="mx-4 mb-3 p-2 rounded bg-slate-900/60 border border-red-900/30">
                <p class="text-[11px] text-red-400 font-mono break-all">{{ cloudPing.error }}</p>
              </div>
            </div>
          </div>

          <!-- Paso 3: frecuencia de envio -->
          <div class="border-t border-slate-700/60 pt-4">
            <p class="text-xs font-semibold text-cyan-400 uppercase tracking-widest mb-3">Paso 3 — Frecuencia de envio</p>
            <label class="label-field">Intervalo de sincronizacion <HelpTip :text="tips.syncInterval" /></label>
            <div class="flex items-center space-x-2 mt-1">
              <input v-model.number="cloud.sync_interval_s" type="range" min="30" max="1800" step="30" class="flex-1 accent-cyan-500">
              <span class="w-20 text-right text-sm font-mono text-white">{{ formatInterval(cloud.sync_interval_s) }}</span>
            </div>
            <div class="mt-3 bg-slate-900 rounded p-3">
              <svg viewBox="0 0 400 50" class="w-full" style="height:48px">
                <rect x="2" y="10" width="80" height="30" rx="4" fill="#1e293b" stroke="#475569" stroke-width="1.5"/>
                <text x="42" y="24" text-anchor="middle" fill="#94a3b8" font-size="8" font-family="monospace">OEE cada</text>
                <text x="42" y="34" text-anchor="middle" fill="#fbbf24" font-size="9" font-family="monospace">{{ oee.snapshot_interval_s }}s</text>
                <line x1="82" y1="25" x2="118" y2="25" stroke="#475569" stroke-width="1" marker-end="url(#arrC)"/>
                <rect x="118" y="10" width="80" height="30" rx="4" fill="#1e293b" stroke="#06b6d4" stroke-width="1.5"/>
                <text x="158" y="28" text-anchor="middle" fill="#67e8f9" font-size="9" font-family="monospace">PostgreSQL</text>
                <line x1="198" y1="25" x2="234" y2="25" stroke="#06b6d4" stroke-width="1" marker-end="url(#arrC)"/>
                <rect x="234" y="10" width="90" height="30" rx="4" fill="#1e293b" stroke="#06b6d4" stroke-width="1.5"/>
                <text x="279" y="24" text-anchor="middle" fill="#67e8f9" font-size="8" font-family="monospace">Envio cada</text>
                <text x="279" y="34" text-anchor="middle" fill="#22d3ee" font-size="9" font-family="monospace">{{ formatInterval(cloud.sync_interval_s) }}</text>
                <line x1="324" y1="25" x2="352" y2="25" stroke="#22c55e" stroke-width="1" marker-end="url(#arrC)"/>
                <rect x="352" y="10" width="46" height="30" rx="4" fill="#1e293b" stroke="#22c55e" stroke-width="1.5"/>
                <text x="375" y="28" text-anchor="middle" fill="#86efac" font-size="9" font-family="monospace">Nube</text>
                <defs><marker id="arrC" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0,0 L0,6 L6,3 z" fill="#64748b"/></marker></defs>
              </svg>
              <p class="mt-2 text-xs text-gray-500 text-center">~{{ snapshotsPerBatch }} snapshots por envio | {{ Math.round(3600 / cloud.sync_interval_s) }} envios/hora</p>
            </div>
          </div>

          <!-- Estado operacional (solo si hay datos) -->
          <div v-if="cloudHealth" class="border-t border-slate-700/60 pt-4">
            <p class="text-xs font-semibold text-cyan-400 uppercase tracking-widest mb-3">Estado operacional</p>
            <div class="bg-slate-900 rounded-lg p-3 grid grid-cols-3 sm:grid-cols-6 gap-3 text-xs">
              <div>
                <p class="text-gray-500">Servicio</p>
                <p class="font-mono" :class="cloudHealth.status === 'ok' ? 'text-green-400' : 'text-yellow-400'">{{ cloudHealth.status || '---' }}</p>
              </div>
              <div>
                <p class="text-gray-500">Cloud</p>
                <p class="font-mono" :class="cloudHealth.cloud_status === 'ok' ? 'text-green-400' : cloudHealth.cloud_status === 'degraded' ? 'text-yellow-400' : 'text-red-400'">{{ cloudHealth.cloud_status || '---' }}</p>
              </div>
              <div>
                <p class="text-gray-500">Ultimo OK</p>
                <p class="font-mono text-gray-300">{{ cloudHealth.cloud_last_ok ? new Date(cloudHealth.cloud_last_ok).toLocaleTimeString('es-MX') : 'Nunca' }}</p>
              </div>
              <div>
                <p class="text-gray-500">Errores consec.</p>
                <p class="font-mono" :class="(cloudHealth.consecutive_errors ?? 0) > 0 ? 'text-red-400' : 'text-green-400'">
                  {{ cloudHealth.consecutive_errors ?? '---' }}
                </p>
              </div>
              <div>
                <p class="text-gray-500">Total sincroniz.</p>
                <p class="font-mono text-cyan-300">{{ cloudHealth.total_synced ?? '---' }}</p>
              </div>
              <div>
                <p class="text-gray-500">Buffer pendiente</p>
                <p class="font-mono" :class="(cloudBufferPending ?? 0) > 100 ? 'text-yellow-400' : 'text-gray-300'">
                  {{ cloudBufferPending !== null ? cloudBufferPending : '---' }}
                </p>
              </div>
            </div>
          </div>

          <!-- Contrato de datos (referencia tecnica) -->
          <div class="border-t border-slate-700/60 pt-4">
            <div class="bg-slate-900 rounded-lg p-4">
              <p class="text-xs text-gray-400 mb-2">Contrato de datos</p>
              <div class="space-y-1 text-xs">
                <div class="flex justify-between"><span class="text-gray-500">Endpoint</span><code class="text-cyan-300 font-mono">POST /api/v1/edge/oee</code></div>
                <div class="flex justify-between"><span class="text-gray-500">Auth</span><code class="text-cyan-300 font-mono">Bearer {{ cloud.cloud_api_key ? '****' + cloud.cloud_api_key.slice(-4) : '(sin clave)' }}</code></div>
                <div class="flex justify-between"><span class="text-gray-500">Headers</span><code class="text-cyan-300 font-mono">X-Device-ID: {{ deviceId }}</code></div>
                <div class="flex justify-between"><span class="text-gray-500">Payload</span><code class="text-cyan-300 font-mono">OEERecord[] (columnar head/data)</code></div>
                <div class="flex justify-between"><span class="text-gray-500">Retry</span><code class="text-cyan-300 font-mono">3 intentos, backoff exponencial 2s-2min</code></div>
                <div class="flex justify-between"><span class="text-gray-500">Buffer max</span><code class="text-cyan-300 font-mono">2880 reintentos antes de descarte</code></div>
              </div>
            </div>
          </div>

        </div>
      </template>

      <!-- TABLET -->
      <template v-if="activeSection === 'tablet'">
        <div class="card">
          <div class="flex items-center justify-between mb-4">
            <span class="text-xs font-mono px-2 py-1 rounded" :class="(tablet.host || detectedHost) ? 'bg-green-900/40 text-green-400' : 'bg-slate-700 text-gray-500'">
              {{ (tablet.host || detectedHost) ? `${tablet.host || detectedHost} — App:${tablet.port} / API:${tablet.gatewayPort}` : 'Sin host configurado' }}
            </span>
            <button type="button" @click="autoDetectHost"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-blue-700 hover:bg-blue-600 text-white transition-colors">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9"/>
              </svg>
              Auto-detectar
            </button>
          </div>
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
            <div>
              <label class="label-field">IP / Hostname del Edge <HelpTip :text="tips.tabletHost" /></label>
              <input v-model="tablet.host" type="text" :placeholder="detectedHost || '192.168.x.x'" class="mt-1 input-field font-mono text-xs" @input="onTabletChanged">
              <p v-if="detectedHost && !tablet.host" class="mt-1 text-xs text-blue-400">Se usara {{ detectedHost }} (detectado automaticamente)</p>
            </div>
            <div>
              <label class="label-field">Puerto App (web) <HelpTip :text="tips.tabletPort" /></label>
              <input v-model.number="tablet.port" type="number" min="1" max="65535" class="mt-1 input-field font-mono text-xs" @input="onTabletChanged">
            </div>
            <div>
              <label class="label-field">Puerto Gateway API <HelpTip :text="tips.gatewayPort" /></label>
              <input v-model.number="tablet.gatewayPort" type="number" min="1" max="65535" class="mt-1 input-field font-mono text-xs" @input="onTabletChanged">
            </div>
          </div>
          <div class="mt-4 p-4 bg-slate-900 rounded-lg space-y-3">
            <div>
              <label class="label-field mb-1">Deep link QR (auto-conecta app + gateway) <HelpTip :text="tips.tabletUrl" /></label>
              <div class="flex items-center space-x-2">
                <code class="flex-1 px-3 py-2 bg-slate-700 rounded text-xs font-mono text-emerald-400 select-all break-all">{{ deepLinkURL }}</code>
                <button type="button" @click="copyTabletURL" class="shrink-0 px-3 py-2 bg-slate-600 text-white rounded hover:bg-slate-500 transition-colors">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"/>
                  </svg>
                </button>
              </div>
              <p v-if="tabletCopied" class="mt-1 text-xs text-emerald-400">Deep link copiado</p>
            </div>
            <div class="grid grid-cols-2 gap-3 text-xs">
              <div>
                <p class="text-gray-500 mb-1">App web (navegador)</p>
                <code class="text-blue-300 font-mono">{{ tabletURL }}</code>
              </div>
              <div>
                <p class="text-gray-500 mb-1">Gateway API (datos)</p>
                <code class="text-orange-300 font-mono">{{ gatewayURL }}</code>
              </div>
            </div>
          </div>
          <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div class="p-4 bg-slate-900 rounded-lg flex flex-col items-center">
              <p class="text-xs text-gray-400 mb-3">Escanee el codigo QR desde la tablet</p>
              <svg :viewBox="`0 0 ${qrSize} ${qrSize}`" class="w-48 h-48" v-if="qrModules.length > 0">
                <rect :width="qrSize" :height="qrSize" fill="#ffffff"/>
                <rect v-for="(cell, i) in qrDarkCells" :key="i" :x="cell.x" :y="cell.y" width="1" height="1" fill="#0d1b2a"/>
              </svg>
              <p class="mt-2 text-[10px] text-gray-500 font-mono break-all text-center max-w-[200px]">{{ deepLinkURL }}</p>
            </div>
            <div class="p-4 bg-slate-900 rounded-lg">
              <p class="text-xs text-gray-400 mb-3">Instrucciones</p>
              <ol class="list-decimal list-inside space-y-2 text-xs text-gray-300">
                <li>Conecte la tablet a la misma red LAN</li>
                <li>Abra el navegador en la tablet</li>
                <li>Escanee el QR — la app se abre y conecta automaticamente al gateway</li>
                <li>Sin configuracion adicional necesaria</li>
              </ol>
            </div>
          </div>
        </div>
      </template>

      <!-- DESPLIEGUE -->
      <template v-if="activeSection === 'deploy'">
        <div class="card space-y-4">
          <div class="flex items-center justify-between">
            <h3 class="section-title mb-0">Estado de los servicios</h3>
            <button type="button" @click="checkAllServices" :disabled="servicesChecking"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-blue-700 hover:bg-blue-600 text-white disabled:opacity-40 disabled:cursor-not-allowed transition-colors">
              <svg v-if="!servicesChecking" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              <svg v-else class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              {{ servicesChecking ? 'Verificando...' : 'Verificar todos' }}
            </button>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <div v-for="svc in serviceChecks" :key="svc.id"
              class="flex items-center gap-3 bg-slate-900 rounded-lg px-4 py-3">
              <span class="w-2.5 h-2.5 rounded-full shrink-0" :class="{
                'bg-emerald-400': svc.status === 'ok',
                'bg-red-400': svc.status === 'error',
                'bg-gray-600': svc.status === null
              }"></span>
              <div class="min-w-0">
                <p class="text-sm text-white font-medium truncate">{{ svc.name }}</p>
                <p class="text-[10px] font-mono" :class="{
                  'text-emerald-500': svc.status === 'ok',
                  'text-red-500': svc.status === 'error',
                  'text-gray-600': svc.status === null
                }">:{{ svc.port }} {{ svc.status === 'ok' ? 'activo' : svc.status === 'error' ? 'sin respuesta' : 'sin verificar' }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Nota arquitectura -->
        <div class="bg-blue-900/20 border border-blue-700/30 rounded-lg px-4 py-3">
          <div class="flex items-start gap-2">
            <svg class="w-4 h-4 text-blue-400 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <div class="text-xs text-blue-300 space-y-1">
              <p class="font-semibold">Toda la configuración de negocio se lee de la base de datos</p>
              <p class="text-blue-400/80">Device ID, cámara, OEE, cloud y credenciales se configuran desde esta UI y se guardan en PostgreSQL.
              El <code class="bg-blue-900/50 px-1 rounded">.env</code> solo contiene credenciales de infraestructura (PostgreSQL, zona horaria) que ya vienen pre-configuradas.</p>
            </div>
          </div>
        </div>

        <div class="card space-y-4">
          <div class="flex items-center justify-between">
            <h3 class="section-title mb-0">Variables de entorno</h3>
            <span class="text-[10px] text-gray-500 font-mono">infrastructure/docker/.env</span>
          </div>
          <div v-if="envVars.length > 0" class="bg-slate-900 rounded-lg divide-y divide-slate-700/50">
            <div v-for="v in envVars" :key="v.key" class="flex items-center justify-between px-4 py-2.5">
              <span class="text-xs font-mono text-gray-400 shrink-0">{{ v.key }}</span>
              <div class="flex items-center gap-2">
                <span class="text-xs font-mono truncate max-w-[220px]" :class="v.set ? 'text-emerald-400' : 'text-red-400'">{{ v.display }}</span>
                <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="v.set ? 'bg-emerald-400' : 'bg-red-400'"></span>
              </div>
            </div>
          </div>
          <div v-else class="bg-slate-900 rounded-lg px-4 py-8 text-center">
            <p class="text-xs text-gray-500">No se encontro archivo .env en el Jetson</p>
          </div>
        </div>

        <div class="card space-y-4">
          <h3 class="section-title">Credenciales de entorno <HelpTip :text="tips.envFile" /></h3>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label class="label-field">Password PostgreSQL <HelpTip :text="tips.pgPassword" /></label>
              <div class="relative mt-1">
                <input v-model="deploy.pg_password" :type="showPgPass ? 'text' : 'password'" placeholder="Contrasena segura" class="input-field font-mono text-xs pr-10">
                <button type="button" @click="showPgPass = !showPgPass" class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path v-if="!showPgPass" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                    <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                  </svg>
                </button>
              </div>
            </div>
            <div>
              <label class="label-field">Zona horaria <HelpTip :text="tips.timezone" /></label>
              <select v-model="deploy.tz" class="mt-1 input-field text-xs">
                <option value="America/Lima">America/Lima (PET)</option>
                <option value="America/Bogota">America/Bogota (COT)</option>
                <option value="America/Mexico_City">America/Mexico_City (CST)</option>
                <option value="America/Monterrey">America/Monterrey (CST)</option>
                <option value="America/Tijuana">America/Tijuana (PST)</option>
                <option value="America/Cancun">America/Cancun (EST)</option>
                <option value="America/Santiago">America/Santiago (CLT)</option>
                <option value="America/Argentina/Buenos_Aires">America/Buenos_Aires (ART)</option>
                <option value="UTC">UTC</option>
              </select>
            </div>
            <div>
              <label class="label-field">Linea ID (cloud)</label>
              <input v-model="deploy.linea_id" type="text" inputmode="numeric" pattern="[0-9]*" placeholder="Ej: 7" class="mt-1 input-field font-mono text-xs">
              <p class="mt-1 text-[10px] text-gray-600">ID de la linea asignada en el cloud (gateway.device_registry).</p>
            </div>
          </div>
          <div v-if="envPendingReload" class="flex items-center gap-2 mt-1 mb-1 px-3 py-2 bg-amber-900/50 border border-amber-500/60 rounded text-amber-300 text-xs">
            <svg class="w-4 h-4 shrink-0 animate-bounce" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01M12 3a9 9 0 100 18A9 9 0 0012 3z"/>
            </svg>
            <span>Guardaste cambios — presiona <strong>↺ Actualizar</strong> (botón de la derecha) para confirmar los valores del .env.</span>
          </div>
          <div class="flex items-center gap-3">
            <div class="flex-1"></div>
            <button type="button" @click="reloadEnvFromJetson" :disabled="envReloading"
              title="Descartar cambios y recargar el .env actual del Jetson"
              :class="[
                'flex items-center gap-1.5 px-4 py-3 text-white text-sm rounded transition-all disabled:opacity-40',
                envPendingReload
                  ? 'bg-amber-600 hover:bg-amber-500 ring-2 ring-amber-400 ring-offset-1 ring-offset-slate-800 animate-pulse'
                  : 'bg-slate-700 hover:bg-slate-600'
              ]">
              <svg v-if="!envReloading" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              <svg v-else class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              Actualizar
            </button>
            <button type="button" @click="copyEnv"
              class="px-4 py-3 bg-slate-600 hover:bg-slate-500 text-white text-sm rounded transition-colors">
              {{ envCopied ? 'Copiado' : 'Copiar' }}
            </button>
          </div>
          <p v-if="envSaved" class="text-xs text-emerald-400 text-center">Archivo .env guardado. Recrea los contenedores para aplicar los cambios.</p>
        </div>
      </template>

      <!-- VISTA PREVIA -->
      <template v-if="activeSection === 'preview'">

        <!-- Snapshot de cámara -->
        <div class="card">
          <div class="flex items-center justify-between mb-3 flex-wrap gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="text-sm font-medium text-gray-200">Vista de Cámara</span>
              <span class="text-xs text-gray-500 truncate max-w-xs font-mono">
                {{ camera.url ? camera.url.replace(/:\/\/[^@]+@/, '://****@') : 'Sin URL configurada' }}
              </span>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <span v-if="snapshotTimestamp && !liveMode" class="text-xs text-gray-500">{{ snapshotTimestamp }}</span>
              <!-- Indicador LIVE -->
              <span v-if="liveMode" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-semibold bg-red-900/60 text-red-300">
                <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                EN VIVO
              </span>
              <!-- Botón capturar (solo si no está en live) -->
              <button v-if="!liveMode" @click="captureSnapshot" :disabled="snapshotLoading || !camera.url"
                class="px-3 py-1.5 text-xs font-medium rounded flex items-center gap-1.5 transition-colors"
                :class="camera.url ? 'bg-slate-600 hover:bg-slate-500 text-white' : 'bg-slate-700 text-gray-500 cursor-not-allowed'">
                <svg class="w-3.5 h-3.5" :class="snapshotLoading ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
                {{ snapshotLoading ? 'Capturando...' : 'Capturar frame' }}
              </button>
              <!-- Botón En Vivo / Parar -->
              <button @click="toggleLive" :disabled="!camera.url"
                class="px-3 py-1.5 text-xs font-medium rounded flex items-center gap-1.5 transition-colors"
                :class="!camera.url
                  ? 'bg-slate-700 text-gray-500 cursor-not-allowed'
                  : liveMode
                    ? 'bg-red-700 hover:bg-red-600 text-white'
                    : 'bg-blue-600 hover:bg-blue-500 text-white'">
                <svg v-if="!liveMode" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
                <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"/>
                </svg>
                {{ liveMode ? 'Parar stream' : 'En vivo' }}
              </button>
            </div>
          </div>

          <!-- Visor: 16:9 con overlay de canvas -->
          <div class="relative bg-slate-900 rounded overflow-hidden w-full" style="aspect-ratio:16/9;">

            <!-- Cargando snapshot -->
            <div v-if="snapshotLoading && !liveMode" class="absolute inset-0 flex flex-col items-center justify-center gap-2">
              <svg class="w-8 h-8 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              <p class="text-sm text-gray-400">Capturando frame desde RTSP...</p>
            </div>

            <!-- Sin URL -->
            <div v-if="!camera.url && !snapshotLoading" class="absolute inset-0 flex flex-col items-center justify-center gap-2 text-gray-500">
              <svg class="w-12 h-12 opacity-40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 10l4.553-2.069A1 1 0 0121 8.876v6.248a1 1 0 01-1.447.894L15 14M3 8a2 2 0 012-2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8z"/>
              </svg>
              <p class="text-sm">Sin URL de cámara configurada</p>
              <button @click="activeSection = 'camera'" class="text-xs text-blue-400 hover:underline">Ir a Cámara</button>
            </div>

            <!-- Error -->
            <div v-if="snapshotError && !previewHasFrame" class="absolute inset-0 flex flex-col items-center justify-center gap-2 text-gray-500 px-6 text-center">
              <svg class="w-10 h-10 text-red-500 opacity-70" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01M5.07 19H19a2 2 0 001.75-2.98l-6.97-12a2 2 0 00-3.54 0L3.32 16.02A2 2 0 005.07 19z"/>
              </svg>
              <p class="text-sm text-red-400">{{ snapshotError }}</p>
              <button @click="captureSnapshot" class="text-xs text-blue-400 hover:underline">Reintentar</button>
            </div>

            <!-- Canvas de imagen (frame actual) -->
            <canvas ref="frameCanvasRef" v-show="previewHasFrame"
              class="absolute inset-0 w-full h-full" />
            <!-- Canvas de overlay ROI -->
            <canvas ref="overlayCanvasRef" v-show="previewHasFrame"
              class="absolute inset-0 w-full h-full pointer-events-none" />

            <!-- Conectando al stream -->
            <div v-if="liveMode && !previewHasFrame && !snapshotError" class="absolute inset-0 flex flex-col items-center justify-center gap-2">
              <svg class="w-8 h-8 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              <p class="text-sm text-gray-400">Conectando al stream...</p>
            </div>

            <!-- Placeholder inicial -->
            <div v-if="!previewHasFrame && !snapshotLoading && !liveMode && !snapshotError && camera.url" class="absolute inset-0 flex flex-col items-center justify-center gap-2 text-gray-500">
              <svg class="w-12 h-12 opacity-40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"/>
                <circle cx="12" cy="13" r="3" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"/>
              </svg>
              <p class="text-sm">Presiona "Capturar frame" o "En vivo"</p>
            </div>
          </div>

          <!-- Barra de info del ROI -->
          <div class="mt-2 flex items-center flex-wrap gap-3 text-xs text-gray-500">
            <span class="inline-flex items-center gap-1.5">
              <span class="inline-block w-4 h-1 bg-yellow-400 rounded opacity-90"></span>
              ROI {{ roi[2] }}×{{ roi[3] }} @ ({{ roi[0] }},{{ roi[1] }})
              <span v-if="roi[4] > 0" class="text-gray-600">· margen {{ roi[4] }}px</span>
            </span>
            <button @click="activeSection = 'detection'"
              class="ml-auto text-blue-400 hover:text-blue-300 hover:underline transition-colors">
              Ajustar ROI / Umbrales →
            </button>
          </div>
        </div>

        <!-- Calibración -->
        <div class="card space-y-3">
          <div>
            <label class="label-field">Calibracion automatica <HelpTip :text="tips.calibration" /></label>
            <p class="mt-1 text-xs text-gray-400">Captura 30 muestras del fondo actual para establecer la linea base. Ejecuta esto con la maquina parada y sin producto en el ROI.</p>
          </div>
          <button type="button" @click="startCalibration" :disabled="calibrating"
            class="w-full sm:w-auto btn-secondary flex items-center gap-2">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
              <circle cx="12" cy="12" r="3" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"/>
            </svg>
            {{ calibrating ? 'Calibrando...' : 'Iniciar Calibracion' }}
          </button>
        </div>
      </template>

      <template v-if="activeSection === 'produccion'">
        <div class="-mx-0 -mt-6" style="height:calc(100vh - 8rem)">
          <Produccion :deviceIdProp="lineaId" :isNew="!lineaId" />
        </div>
      </template>

      <teleport to="body">
        <transition name="fade">
          <div v-if="message"
            class="fixed top-6 left-1/2 -translate-x-1/2 z-[9999] px-5 py-3 rounded-lg shadow-xl text-sm font-medium backdrop-blur-sm border border-white/10"
            :class="messageClass">
            {{ message }}
          </div>
        </transition>
      </teleport>

      <div class="lg:hidden fixed bottom-4 right-4 z-50">
        <button @click="saveConfig" :disabled="saving"
          class="btn-primary rounded-full shadow-lg px-4 py-3 flex items-center gap-2">
          <svg v-if="!saving" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
          </svg>
          Guardar
        </button>
      </div>
    </main>
  </div>

  <!-- MODO LAB -->
  <transition name="lab-fade">
    <div v-if="labOpen" class="fixed inset-0 z-[9999] flex flex-col bg-slate-950 overflow-hidden">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3 border-b border-slate-700/70 shrink-0 bg-slate-900">
        <div class="flex items-center gap-3">
          <svg class="w-5 h-5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 3H5a2 2 0 00-2 2v4m0 0h18M3 9v10a2 2 0 002 2h5m4 0h5a2 2 0 002-2V9M9 21v-6a2 2 0 012-2h2a2 2 0 012 2v6"/>
          </svg>
          <span class="text-sm font-semibold text-white tracking-wide">Modo Lab</span>
          <span class="text-xs text-gray-500 font-mono">{{ deviceId }}</span>
        </div>
        <div class="flex items-center gap-3">
          <span v-if="labSaved" class="text-xs text-green-400 font-mono transition-opacity">Guardado</span>
          <button @click="labGuideOpen = !labGuideOpen"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
            :class="labGuideOpen ? 'bg-indigo-700 text-white' : 'bg-slate-700 text-gray-300 hover:text-white hover:bg-slate-600'"
            title="Guia de parametros">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            Guia
          </button>
          <button @click="labConfirmOpen = true" :disabled="labSaving"
            class="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-colors">
            <svg class="w-4 h-4" :class="labSaving ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path v-if="!labSaving" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
              <circle v-else class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            </svg>
            {{ labSaving ? 'Guardando...' : 'Guardar' }}
          </button>
          <button @click="closeLab"
            class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-slate-700 transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
            Cerrar
          </button>
        </div>
      </div>

      <!-- Split body -->
      <div class="flex flex-1 min-h-0 relative">

        <!-- Panel izquierdo: controles -->
        <div class="w-[420px] shrink-0 flex flex-col border-r border-slate-700/60 bg-slate-900 overflow-y-auto">
          <div class="p-5 space-y-6">

            <!-- ROI numérico -->
            <section>
              <div class="flex items-center justify-between mb-3">
                <h3 class="text-xs font-semibold text-gray-300 uppercase tracking-widest">Region de Interes <HelpTip :text="tips.roi" /></h3>
                <span class="text-[10px] font-mono text-indigo-400 bg-indigo-900/30 px-2 py-0.5 rounded">Arrastrar en camara</span>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="label-field text-xs">X (origen) <HelpTip :text="tips.roiX" /></label>
                  <input v-model.number="labRoi[0]" type="number" min="0" class="mt-1 input-field text-sm" @input="drawLabOverlay">
                </div>
                <div>
                  <label class="label-field text-xs">Y (origen) <HelpTip :text="tips.roiY" /></label>
                  <input v-model.number="labRoi[1]" type="number" min="0" class="mt-1 input-field text-sm" @input="drawLabOverlay">
                </div>
                <div>
                  <label class="label-field text-xs">Ancho (px) <HelpTip :text="tips.roiW" /></label>
                  <input v-model.number="labRoi[2]" type="number" min="1" class="mt-1 input-field text-sm" @input="drawLabOverlay">
                </div>
                <div>
                  <label class="label-field text-xs">Alto (px) <HelpTip :text="tips.roiH" /></label>
                  <input v-model.number="labRoi[3]" type="number" min="1" class="mt-1 input-field text-sm" @input="drawLabOverlay">
                </div>
              </div>
              <div class="mt-3">
                <label class="label-field text-xs">Margen inferior excluido <HelpTip :text="tips.roiMargin" /></label>
                <div class="flex items-center gap-2 mt-1">
                  <input v-model.number="labRoi[4]" type="range" min="0"
                    :max="Math.floor((labRoi[3] || 1) * 0.4)" step="1"
                    class="flex-1 accent-orange-500" @input="drawLabOverlay">
                  <span class="w-12 text-right text-sm font-mono text-white">{{ labRoi[4] ?? 0 }}px</span>
                </div>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Histeresis -->
            <section>
              <h3 class="text-xs font-semibold text-gray-300 uppercase tracking-widest mb-3">Histeresis <HelpTip :text="tips.hysteresis" /></h3>
              <div class="space-y-4">
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Umbral Alto (activacion) <HelpTip :text="tips.hystHigh" /></label>
                    <span class="text-xs font-mono text-white">{{ labThresh.high.toFixed(2) }}</span>
                  </div>
                  <input v-model.number="labThresh.high" type="range" min="0" max="1" step="0.01" class="w-full accent-green-500">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Umbral Bajo (cancelacion) <HelpTip :text="tips.hystLow" /></label>
                    <span class="text-xs font-mono text-white">{{ labThresh.low.toFixed(2) }}</span>
                  </div>
                  <input v-model.number="labThresh.low" type="range" min="0" max="1" step="0.01" class="w-full accent-red-500">
                </div>
                <div class="bg-slate-800 rounded p-2">
                  <svg viewBox="0 0 280 44" class="w-full" style="height:42px">
                    <line x1="0" y1="22" x2="280" y2="22" stroke="#334155" stroke-width="1"/>
                    <rect :x="labThresh.low * 280" y="6"
                      :width="Math.max(0, (labThresh.high - labThresh.low) * 280)" height="32"
                      fill="#1e3a5f" opacity="0.6"/>
                    <line :x1="labThresh.high * 280" y1="0" :x2="labThresh.high * 280" y2="44"
                      stroke="#22c55e" stroke-width="1.5" stroke-dasharray="3 2"/>
                    <line :x1="labThresh.low * 280" y1="0" :x2="labThresh.low * 280" y2="44"
                      stroke="#ef4444" stroke-width="1.5" stroke-dasharray="3 2"/>
                    <text :x="labThresh.high * 280 + 3" y="11" fill="#86efac" font-size="8" font-family="monospace">HIGH</text>
                    <text :x="labThresh.low * 280 + 3" y="11" fill="#fca5a5" font-size="8" font-family="monospace">LOW</text>
                  </svg>
                </div>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Umbrales de señal -->
            <section>
              <h3 class="text-xs font-semibold text-gray-300 uppercase tracking-widest mb-3">Umbrales de Senal <HelpTip :text="tips.thresholds" /></h3>
              <div class="space-y-4">
                <div v-for="field in thresholdFields" :key="field.key">
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">{{ field.label }} <HelpTip :text="field.tip" /></label>
                    <span class="text-xs font-mono text-white">{{ (labThresh[field.key] ?? field.default).toFixed(field.step < 1 ? 2 : 1) }}</span>
                  </div>
                  <input v-model.number="labThresh[field.key]" type="range"
                    :min="field.min" :max="field.max" :step="field.step"
                    class="w-full accent-blue-500">
                </div>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Rango HSV de deteccion beige (solo visual Lab) -->
            <section v-if="labDebugMode && (labDebugChannel === 'beige' || labDebugChannel === 'combined')">
              <div class="flex items-center gap-2 mb-3">
                <span class="w-3 h-3 rounded-sm bg-amber-400 shrink-0"></span>
                <h3 class="text-xs font-semibold text-amber-300 uppercase tracking-widest">Rango de Deteccion Beige <HelpTip text="Ajusta los rangos HSV del analisis visual en tiempo real. Mueve los sliders hasta que la mascara naranja cubra exactamente tu tela y no el fondo." /></h3>
              </div>
              <div class="space-y-3">
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Tono min (H) <HelpTip text="Tono minimo (0=rojo, 30=naranja/beige, 60=amarillo). Beige claro suele estar entre 20-50." /></label>
                    <span class="text-xs font-mono text-amber-300">{{ labBeigeRange.hMin }}&deg;</span>
                  </div>
                  <input v-model.number="labBeigeRange.hMin" type="range" min="0" max="60" step="1" class="w-full accent-amber-500">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Tono max (H) <HelpTip text="Tono maximo. Para beige/crema claro subir hasta 55-65. Para beige oscuro/marron mantener en 35-45." /></label>
                    <span class="text-xs font-mono text-amber-300">{{ labBeigeRange.hMax }}&deg;</span>
                  </div>
                  <input v-model.number="labBeigeRange.hMax" type="range" min="10" max="80" step="1" class="w-full accent-amber-500">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Saturacion min (S) <HelpTip text="Telas claras/blancas tienen saturacion muy baja (2-15). Baja este valor para detectar colores mas palidos." /></label>
                    <span class="text-xs font-mono text-amber-300">{{ labBeigeRange.sMin }}</span>
                  </div>
                  <input v-model.number="labBeigeRange.sMin" type="range" min="0" max="80" step="1" class="w-full accent-amber-400">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Saturacion max (S) <HelpTip text="Reduce para excluir colores muy saturados/vivos que no son tu tela." /></label>
                    <span class="text-xs font-mono text-amber-300">{{ labBeigeRange.sMax }}</span>
                  </div>
                  <input v-model.number="labBeigeRange.sMax" type="range" min="50" max="255" step="1" class="w-full accent-amber-400">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Brillo min (V) <HelpTip text="Telas claras tienen brillo alto (150+). Sube para filtrar pixels muy oscuros que no son tu tela." /></label>
                    <span class="text-xs font-mono text-amber-300">{{ labBeigeRange.vMin2 }}</span>
                  </div>
                  <input v-model.number="labBeigeRange.vMin2" type="range" min="0" max="255" step="1" class="w-full accent-yellow-500">
                </div>
                <!-- Barra de tono -->
                <div class="mt-2 rounded overflow-hidden" style="height:12px">
                  <svg viewBox="0 0 360 12" class="w-full h-full" preserveAspectRatio="none">
                    <defs>
                      <linearGradient id="hueGrad" x1="0" x2="1" y1="0" y2="0">
                        <stop offset="0%"   stop-color="hsl(0,80%,55%)"/>
                        <stop offset="14%"  stop-color="hsl(50,80%,55%)"/>
                        <stop offset="28%"  stop-color="hsl(100,80%,55%)"/>
                        <stop offset="56%"  stop-color="hsl(200,80%,55%)"/>
                        <stop offset="84%"  stop-color="hsl(300,80%,55%)"/>
                        <stop offset="100%" stop-color="hsl(360,80%,55%)"/>
                      </linearGradient>
                    </defs>
                    <rect x="0" y="0" width="360" height="12" fill="url(#hueGrad)" opacity="0.35"/>
                    <rect :x="labBeigeRange.hMin * 4.5" y="0" :width="Math.max(0, (labBeigeRange.hMax - labBeigeRange.hMin) * 4.5)" height="12" fill="rgba(251,146,36,0.55)" rx="1"/>
                    <line :x1="labBeigeRange.hMin * 4.5" y1="0" :x2="labBeigeRange.hMin * 4.5" y2="12" stroke="#fbbf24" stroke-width="1.5"/>
                    <line :x1="labBeigeRange.hMax * 4.5" y1="0" :x2="labBeigeRange.hMax * 4.5" y2="12" stroke="#fbbf24" stroke-width="1.5"/>
                  </svg>
                </div>
                <p class="text-[10px] text-gray-600">H {{ labBeigeRange.hMin }}&ndash;{{ labBeigeRange.hMax }}&deg; &middot; S {{ labBeigeRange.sMin }}&ndash;{{ labBeigeRange.sMax }} &middot; V &ge;{{ labBeigeRange.vMin2 }}</p>
              </div>
            </section>

            <div v-if="labDebugMode && (labDebugChannel === 'beige' || labDebugChannel === 'combined')" class="border-t border-slate-700/50"></div>

            <!-- Sensibilidad ROI de Presencia -->
            <template v-if="labRoiPresencia[2] > 0">
              <div class="border-t border-slate-700/50"></div>
              <section>
                <div class="flex items-center gap-2 mb-3">
                  <span class="w-3 h-3 rounded-sm bg-emerald-400 shrink-0"></span>
                  <h3 class="text-xs font-semibold text-emerald-300 uppercase tracking-widest">Sensibilidad ROI Presencia</h3>
                </div>
                <!-- Indicador en vivo del score de movimiento -->
                <div v-if="labHasFrame" class="mb-3 p-2 rounded bg-slate-900/60 flex items-center gap-2">
                  <span class="w-2 h-2 rounded-full shrink-0" :class="labPresenceMotion ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'"></span>
                  <span class="text-xs font-mono" :class="labPresenceMotion ? 'text-emerald-300' : 'text-red-400'">
                    {{ labPresenceMotion ? 'En movimiento' : 'Detenida' }}
                  </span>
                  <span class="ml-auto text-[10px] font-mono text-gray-400">Δmv {{ labMotionScore.toFixed(2) }}</span>
                </div>
                <div class="space-y-4">
                  <!-- Ventana de comparación -->
                  <div>
                    <div class="flex justify-between mb-1">
                      <label class="label-field text-xs">Ventana de comparación (s)
                        <HelpTip text="Segundos hacia atrás para comparar el frame actual. Máquinas lentas (40 min/prenda) necesitan ventanas largas (20-40 s) para que el desplazamiento de la tela sea visible. Ventana corta = más reactivo pero puede ignorar movimiento lento." />
                      </label>
                      <span class="text-xs font-mono text-white">{{ ((labThresh.presencia_window ?? 375) / 12.5).toFixed(0) }}s</span>
                    </div>
                    <input v-model.number="labThresh.presencia_window" type="range" min="25" max="750" step="25" class="w-full accent-emerald-500">
                    <div class="flex justify-between mt-0.5">
                      <span class="text-[10px] text-emerald-500">← reactivo (2 s)</span>
                      <span class="text-[10px] text-gray-600">ciclos lentos (60 s) →</span>
                    </div>
                  </div>
                  <!-- Sensibilidad pixel -->
                  <div>
                    <div class="flex justify-between mb-1">
                      <label class="label-field text-xs">Sensibilidad de píxel
                        <HelpTip text="Cambio mínimo de intensidad (0-255) para que un píxel cuente como 'movido'. Valor bajo = más sensible al ruido y movimientos pequeños. Valor alto = solo cambios bruscos. Default 8." />
                      </label>
                      <span class="text-xs font-mono text-white">{{ labThresh.presencia_pixel_threshold ?? 8 }}</span>
                    </div>
                    <input v-model.number="labThresh.presencia_pixel_threshold" type="range" min="2" max="40" step="1" class="w-full accent-teal-500">
                    <div class="flex justify-between mt-0.5">
                      <span class="text-[10px] text-teal-400">← más sensible</span>
                      <span class="text-[10px] text-gray-600">menos ruido →</span>
                    </div>
                  </div>
                  <!-- Área mínima de movimiento -->
                  <div>
                    <div class="flex justify-between mb-1">
                      <label class="label-field text-xs">Área mínima con movimiento
                        <HelpTip text="Fracción mínima del ROI que debe mostrar píxeles cambiados para declarar movimiento. 0.5% = muy sensible. 5% = exige cambio amplio. Default 0.5%." />
                      </label>
                      <span class="text-xs font-mono text-white">{{ ((labThresh.presencia_scale ?? 0.005) * 100).toFixed(1) }}%</span>
                    </div>
                    <input v-model.number="labThresh.presencia_scale" type="range" min="0.001" max="0.1" step="0.001" class="w-full accent-cyan-500">
                    <div class="flex justify-between mt-0.5">
                      <span class="text-[10px] text-cyan-400">← 0.1% (muy sensible)</span>
                      <span class="text-[10px] text-gray-600">10% (exige mucho) →</span>
                    </div>
                  </div>
                  <!-- Hold de parada -->
                  <div>
                    <div class="flex justify-between mb-1">
                      <label class="label-field text-xs">Tiempo para declarar parada
                        <HelpTip text="Segundos consecutivos sin movimiento detectado para pasar a 'Detenida'. Si la tela se mueve lento, usar valores altos (60-120 s) para no generar falsas paradas. Default 60 s." />
                      </label>
                      <span class="text-xs font-mono text-white">{{ ((labThresh.presencia_hold ?? 750) / 12.5).toFixed(0) }}s</span>
                    </div>
                    <input v-model.number="labThresh.presencia_hold" type="range" min="125" max="3750" step="125" class="w-full accent-orange-500">
                    <div class="flex justify-between mt-0.5">
                      <span class="text-[10px] text-orange-400">← 10 s (rápido)</span>
                      <span class="text-[10px] text-gray-600">300 s (5 min) →</span>
                    </div>
                  </div>
                  <!-- Botón reiniciar buffer -->
                  <button @click="labRecalibrar" class="w-full py-1.5 rounded bg-emerald-700/50 hover:bg-emerald-700 text-emerald-200 text-xs font-semibold transition-colors">
                    ↺ Reiniciar detector (nuevo ROI o cambio de escena)
                  </button>
                </div>
                <div class="mt-3 p-2 rounded bg-slate-900/60 text-[10px] text-gray-500">
                  Detección por movimiento (slow-window diff) · independiente del color de tela ·
                  compara frame actual vs hace <span class="text-emerald-400 font-mono">{{ ((labThresh.presencia_window ?? 375) / 12.5).toFixed(0) }}s</span>
                </div>
              </section>
            </template>

            <!-- FSM -->
            <section>
              <h3 class="text-xs font-semibold text-gray-300 uppercase tracking-widest mb-3">Maquina de Estados (FSM) <HelpTip :text="tips.fsm" /></h3>
              <div class="space-y-4">
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Frames de confirmacion <HelpTip :text="tips.fsmFrames" /></label>
                    <span class="text-xs font-mono text-white">{{ labFsm.n_frames }}</span>
                  </div>
                  <input v-model.number="labFsm.n_frames" type="range" min="1" max="30" step="1" class="w-full accent-blue-500">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Cooldown (frames) <HelpTip :text="tips.fsmCooldown" /></label>
                    <span class="text-xs font-mono text-white">{{ labFsm.cooldown }}</span>
                  </div>
                  <input v-model.number="labFsm.cooldown" type="range" min="0" max="60" step="1" class="w-full accent-purple-500">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Frames de salida (exit_frames) <HelpTip text="Frames consecutivos con score bajo para confirmar que la tela salio del ROI y cerrar el evento." /></label>
                    <span class="text-xs font-mono text-white">{{ labFsm.exit_frames }}</span>
                  </div>
                  <input v-model.number="labFsm.exit_frames" type="range" min="1" max="100" step="1" class="w-full accent-orange-500">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Timeout salida (max_wait) <HelpTip text="Maximo de frames esperando salida antes de forzar el cierre del evento. Evita que la FSM quede bloqueada en WAIT_EXIT." /></label>
                    <span class="text-xs font-mono text-white">{{ labFsm.max_wait_exit_frames }}</span>
                  </div>
                  <input v-model.number="labFsm.max_wait_exit_frames" type="range" min="1" max="200" step="1" class="w-full accent-rose-500">
                </div>
                <div>
                  <div class="flex justify-between mb-1">
                    <label class="label-field text-xs">Min. entre conteos (min_rearm_s) <HelpTip text="Segundos minimos entre dos CORTEs. Bloquea falsos del timeout FSM. Recomendado: menor que el tiempo entre piezas." /></label>
                    <span class="text-xs font-mono text-white">{{ labFsm.min_rearm_s }}s</span>
                  </div>
                  <input v-model.number="labFsm.min_rearm_s" type="range" min="0" max="60" step="0.5" class="w-full accent-violet-500">
                </div>
              </div>
              <div class="mt-4 bg-slate-800 rounded p-2">
                <svg viewBox="0 0 380 44" class="w-full" style="height:42px">
                  <rect x="2" y="7" width="66" height="28" rx="3" fill="#1e293b" stroke="#475569" stroke-width="1.5"/>
                  <text x="35" y="25" text-anchor="middle" fill="#94a3b8" font-size="9">IDLE</text>
                  <rect x="96" y="7" width="72" height="28" rx="3" fill="#1e293b" stroke="#3b82f6" stroke-width="1.5"/>
                  <text x="132" y="25" text-anchor="middle" fill="#93c5fd" font-size="9">DETECTING</text>
                  <rect x="196" y="7" width="78" height="28" rx="3" fill="#1e293b" stroke="#22c55e" stroke-width="1.5"/>
                  <text x="235" y="25" text-anchor="middle" fill="#86efac" font-size="9">WAIT_EXIT</text>
                  <rect x="302" y="7" width="76" height="28" rx="3" fill="#1e293b" stroke="#a855f7" stroke-width="1.5"/>
                  <text x="340" y="25" text-anchor="middle" fill="#d8b4fe" font-size="9">COOLDOWN</text>
                  <line x1="68" y1="21" x2="96" y2="21" stroke="#3b82f6" stroke-width="1" marker-end="url(#labArr)"/>
                  <line x1="168" y1="21" x2="196" y2="21" stroke="#22c55e" stroke-width="1" marker-end="url(#labArr)"/>
                  <line x1="274" y1="21" x2="302" y2="21" stroke="#a855f7" stroke-width="1" marker-end="url(#labArr)"/>
                  <defs><marker id="labArr" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0,0 L0,6 L6,3 z" fill="#64748b"/></marker></defs>
                </svg>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Calibración -->
            <section class="pb-4">
              <h3 class="text-xs font-semibold text-gray-300 uppercase tracking-widest mb-3">Calibracion <HelpTip text="Captura ~30 frames del fondo vacio para establecer la referencia base de color. Ejecutar con el ROI completamente libre de tela." /></h3>
              <p class="text-xs text-gray-500 mb-3">Asegura que el ROI este vacio antes de calibrar. El sistema capturara 30 muestras del fondo.</p>
              <button @click="startCalibration" :disabled="calibrating"
                class="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium bg-slate-700 hover:bg-slate-600 text-white disabled:opacity-50 transition-colors">
                <svg class="w-4 h-4" :class="calibrating ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                  <circle cx="12" cy="12" r="3" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"/>
                </svg>
                {{ calibrating ? 'Calibrando...' : 'Iniciar Calibracion' }}
              </button>
            </section>

          </div>
        </div>

        <!-- Panel derecho: video con ROI interactivo -->
        <div class="flex-1 flex flex-col bg-slate-950 overflow-hidden">

          <!-- Toolbar de cámara -->
          <div class="flex items-center justify-between px-4 py-2.5 border-b border-slate-700/60 shrink-0 gap-3 flex-wrap">
            <div class="flex items-center gap-3">
              <span v-if="labLiveMode" class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-semibold bg-red-900/60 text-red-300">
                <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                EN VIVO
              </span>
              <span v-if="labFrameSize" class="text-[10px] font-mono text-gray-500">{{ labFrameSize }}</span>
              <!-- Indicador de detección de prenda + cronómetro -->
              <template v-if="labHasFrame">
                <!-- Produciendo -->
                <span v-if="labPresenceMotion"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold bg-emerald-900/70 text-emerald-300 border border-emerald-700/50"
                  title="Tiempo produciendo desde el último cambio de estado">
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                  Produciendo
                  <span class="font-mono tabular-nums text-emerald-200 ml-0.5">{{ labFmtTime(labElapsedSec) }}</span>
                </span>
                <!-- Micro-parada (< 2 min) -->
                <span v-else-if="labStopType === 'micro'"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold bg-amber-900/70 text-amber-300 border border-amber-600/50"
                  title="Parada menor a 2 minutos — micro-parada">
                  <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse"></span>
                  Micro-parada
                  <span class="font-mono tabular-nums text-amber-200 ml-0.5">{{ labFmtTime(labElapsedSec) }}</span>
                </span>
                <!-- Parada (≥ 2 min) -->
                <span v-else-if="labStopType === 'parada'"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold bg-red-900/70 text-red-300 border border-red-600/50"
                  title="Parada mayor a 2 minutos">
                  <span class="w-1.5 h-1.5 rounded-full bg-red-400"></span>
                  Parada
                  <span class="font-mono tabular-nums text-red-200 ml-0.5">{{ labFmtTime(labElapsedSec) }}</span>
                </span>
                <!-- Score de movimiento -->
                <span v-if="labMotionScore > 0"
                  class="text-[10px] font-mono tabular-nums"
                  :class="labMotionScore > 3 ? 'text-amber-400' : 'text-gray-600'"
                  title="Micromovimiento promedio entre frames (Δpx)">
                  Δ{{ labMotionScore.toFixed(1) }}px
                </span>
              </template>
              <!-- Contador de unidades (solo visible cuando el análisis está activo) -->
              <template v-if="labDebugMode && labHasFrame">
                <div class="h-4 w-px bg-slate-600 mx-0.5"></div>
                <span
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-semibold border"
                  :class="labFsmState === 'DETECTING'
                    ? 'bg-blue-900/70 text-blue-200 border-blue-600/60'
                    : labFsmState === 'WAIT_EXIT'
                      ? 'bg-orange-900/70 text-orange-200 border-orange-600/60'
                      : labFsmState === 'COOLDOWN'
                        ? 'bg-violet-900/70 text-violet-200 border-violet-600/60'
                        : 'bg-slate-800/80 text-slate-300 border-slate-600/40'"
                  title="Unidades detectadas por beige en esta sesión">
                  <svg class="w-3 h-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M20 7H4a2 2 0 00-2 2v6a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2zM9 12h6"/>
                  </svg>
                  <span class="font-mono tabular-nums text-base leading-none">{{ labUnitCount }}</span>
                  <span class="text-[10px] opacity-70">uds</span>
                  <span v-if="labFsmState !== 'IDLE'" class="text-[9px] opacity-70 ml-0.5"
                    :class="labFsmState === 'DETECTING' ? 'text-blue-300' : labFsmState === 'WAIT_EXIT' ? 'text-orange-300' : 'text-violet-300'">
                    {{ labFsmState === 'DETECTING' ? '●' : labFsmState === 'WAIT_EXIT' ? '⏹' : '⏳' }}
                  </span>
                </span>
                <button @click="resetLabCounter"
                  class="px-1.5 py-1 rounded text-[10px] text-gray-500 hover:text-red-400 hover:bg-red-900/30 transition-colors"
                  title="Reiniciar contador">
                  ↺
                </button>
              </template>
              <span v-if="!labLiveMode && labSnapshotTs" class="text-xs text-gray-500 font-mono">{{ labSnapshotTs }}</span>
            </div>

            <!-- Controles de análisis visual -->
            <div class="flex items-center gap-1.5">
              <button @click="labToggleDebug"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors"
                :class="labDebugMode ? 'bg-violet-700 text-white' : 'bg-slate-700 text-gray-400 hover:text-white'"
                :disabled="!labHasFrame">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
                </svg>
                Analisis
              </button>
              <template v-if="labDebugMode">
                <button @click="labDebugChannel = 'beige'"
                  class="px-2.5 py-1.5 rounded text-xs font-medium transition-colors"
                  :class="labDebugChannel === 'beige' ? 'bg-amber-600 text-white' : 'bg-slate-800 text-gray-400 hover:text-white'">
                  Beige
                </button>
                <button @click="labDebugChannel = 'edge'"
                  class="px-2.5 py-1.5 rounded text-xs font-medium transition-colors"
                  :class="labDebugChannel === 'edge' ? 'bg-cyan-700 text-white' : 'bg-slate-800 text-gray-400 hover:text-white'">
                  Bordes
                </button>
                <button @click="labDebugChannel = 'combined'"
                  class="px-2.5 py-1.5 rounded text-xs font-medium transition-colors"
                  :class="labDebugChannel === 'combined' ? 'bg-indigo-700 text-white' : 'bg-slate-800 text-gray-400 hover:text-white'">
                  Combinado
                </button>
              </template>
            </div>

            <div class="flex items-center gap-2">
              <!-- Toggle ROI objetivo -->
              <div class="flex items-center rounded overflow-hidden border border-slate-600">
                <button @click="labDragTarget = 'count'"
                  class="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium transition-colors"
                  :class="labDragTarget === 'count' ? 'bg-yellow-500/20 text-yellow-300' : 'bg-slate-800 text-gray-500 hover:text-gray-300'"
                  title="Dibujar ROI de conteo">
                  <span class="inline-block w-2.5 h-2.5 border-2 border-yellow-400 rounded-sm"></span>
                  Conteo
                </button>
                <button @click="labDragTarget = 'presence'"
                  class="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium transition-colors border-l border-slate-600"
                  :class="labDragTarget === 'presence' ? 'bg-emerald-500/20 text-emerald-300' : 'bg-slate-800 text-gray-500 hover:text-gray-300'"
                  title="Dibujar ROI de presencia">
                  <span class="inline-block w-2.5 h-2.5 border-2 border-emerald-400 rounded-sm"></span>
                  Presencia
                </button>
              </div>
              <button v-if="!labLiveMode" @click="labCaptureSnapshot" :disabled="labSnapshotLoading"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium bg-slate-700 hover:bg-slate-600 text-white disabled:opacity-50 transition-colors">
                <svg class="w-3.5 h-3.5" :class="labSnapshotLoading ? 'animate-spin' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"/>
                  <circle cx="12" cy="13" r="3" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"/>
                </svg>
                {{ labSnapshotLoading ? 'Capturando...' : 'Capturar frame' }}
              </button>
              <button @click="labToggleLive"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors"
                :class="labLiveMode ? 'bg-red-700 hover:bg-red-600 text-white' : 'bg-blue-600 hover:bg-blue-500 text-white'">
                <svg v-if="!labLiveMode" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                </svg>
                <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"/>
                </svg>
                {{ labLiveMode ? 'Parar' : 'En vivo' }}
              </button>
            </div>
          </div>

          <!-- Visor -->
          <div class="flex-1 flex items-center justify-center bg-slate-950 p-4 min-h-0 relative">

            <div v-if="!labHasFrame && !labSnapshotLoading && !labLiveMode"
              class="flex flex-col items-center gap-3 text-gray-600">
              <svg class="w-16 h-16 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1"
                  d="M15 10l4.553-2.069A1 1 0 0121 8.876v6.248a1 1 0 01-1.447.894L15 14M3 8a2 2 0 012-2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8z"/>
              </svg>
              <p class="text-sm">Inicia el stream o captura un frame</p>
            </div>

            <div v-if="labSnapshotLoading && !labLiveMode" class="flex flex-col items-center gap-2 text-gray-400">
              <svg class="w-8 h-8 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              <p class="text-sm">Capturando frame...</p>
            </div>

            <div v-if="labLiveMode && !labHasFrame" class="flex flex-col items-center gap-2 text-gray-400">
              <svg class="w-8 h-8 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              <p class="text-sm">Conectando al stream...</p>
            </div>

            <!-- Canvas de imagen + overlay ROI interactivo -->
            <div v-show="labHasFrame" class="relative max-w-full max-h-full" style="line-height:0">
              <canvas ref="labFrameRef" class="block max-w-full max-h-[calc(100vh-140px)]" style="image-rendering:auto"/>
              <canvas ref="labOverlayRef"
                class="absolute inset-0 w-full h-full"
                style="cursor:crosshair"
                @mousedown="labDragStart"
                @mousemove="labDragMove"
                @mouseup="labDragEnd"
                @mouseleave="labDragEnd"/>
            </div>

            <!-- Hint de arrastre -->
            <div v-if="labHasFrame && !labDragging"
              class="absolute bottom-6 left-1/2 -translate-x-1/2 pointer-events-none">
              <span class="px-3 py-1.5 rounded-full text-[10px] text-gray-400 bg-slate-900/80 font-mono border border-slate-700/50">
                Arrastra para redibujar el ROI de
                <span :class="labDragTarget === 'presence' ? 'text-emerald-400' : 'text-yellow-400'">
                  {{ labDragTarget === 'presence' ? 'presencia' : 'conteo' }}
                </span>
              </span>
            </div>

          </div>

          <!-- Info ROI actual -->
          <div class="shrink-0 flex items-center gap-4 px-4 py-2 border-t border-slate-700/60 text-xs text-gray-500 font-mono bg-slate-900 flex-wrap">
            <span class="inline-flex items-center gap-1.5">
              <span class="inline-block w-3 h-0.5 bg-yellow-400 rounded opacity-80"></span>
              Conteo {{ labRoi[2] }}&times;{{ labRoi[3] }} @ ({{ labRoi[0] }},{{ labRoi[1] }})
              <span v-if="labRoi[4] > 0" class="text-gray-600">· margen {{ labRoi[4] }}px</span>
            </span>
            <span v-if="labRoiPresencia[2] > 0" class="inline-flex items-center gap-1.5">
              <span class="inline-block w-3 h-0.5 bg-emerald-400 rounded opacity-80"></span>
              Presencia {{ labRoiPresencia[2] }}&times;{{ labRoiPresencia[3] }} @ ({{ labRoiPresencia[0] }},{{ labRoiPresencia[1] }})
            </span>
            <span v-if="labFrameSize" class="text-gray-600">{{ labFrameSize }}</span>
            <!-- Score de presencia: siempre visible cuando hay ROI de presencia configurado -->
            <span v-if="labHasFrame && labRoiPresencia[2] > 0" class="flex items-center gap-1.5 ml-auto">
              <span class="inline-flex items-center gap-1.5">
                <span class="inline-block w-2 h-2 rounded-full"
                  :class="labPresenceMotion ? 'bg-emerald-400' : 'bg-red-400'"></span>
                <span :class="labPresenceMotion ? 'text-emerald-400' : 'text-red-400'">
                  Prenda {{ labPresenceScore }}%
                </span>
                <span class="w-20 h-1.5 rounded-full bg-slate-700 overflow-hidden inline-block">
                  <span class="block h-full rounded-full transition-all"
                    :class="labPresenceMotion ? 'bg-emerald-400' : 'bg-red-400'"
                    :style="{ width: labPresenceScore + '%' }"></span>
                </span>
                <span class="text-gray-600">min {{ Math.round((labThresh.presencia_threshold ?? 0.05) * 100) }}%</span>
              </span>
            </span>
            <template v-if="labDebugMode">
              <span class="flex items-center gap-1.5 ml-auto">
                <span v-if="labDebugChannel !== 'edge'" class="inline-flex items-center gap-1.5">
                  <span class="inline-block w-2 h-2 rounded-full bg-amber-400"></span>
                  <span class="text-amber-400">Beige {{ labScores.beige }}%</span>
                  <span class="w-20 h-1.5 rounded-full bg-slate-700 overflow-hidden inline-block">
                    <span class="block h-full rounded-full bg-amber-400 transition-all"
                      :style="{ width: labScores.beige + '%' }"></span>
                  </span>
                  <span class="text-gray-600">min {{ Math.round(labThresh.beige * 100) }}%</span>
                </span>
                <span v-if="labDebugChannel !== 'beige'" class="inline-flex items-center gap-1.5 ml-3">
                  <span class="inline-block w-2 h-2 rounded-full bg-cyan-400"></span>
                  <span class="text-cyan-400">Bordes {{ labScores.edge }}%</span>
                  <span class="w-20 h-1.5 rounded-full bg-slate-700 overflow-hidden inline-block">
                    <span class="block h-full rounded-full bg-cyan-400 transition-all"
                      :style="{ width: Math.min(labScores.edge * 5, 100) + '%' }"></span>
                  </span>
                </span>
              </span>
            </template>
          </div>
        </div>
      </div>

      <!-- Modal de confirmacion de guardado -->
      <transition name="lab-fade">
        <div v-if="labConfirmOpen"
          class="absolute inset-0 z-20 flex items-center justify-center bg-black/60 backdrop-blur-sm"
          @click.self="labConfirmOpen = false">
          <div class="w-[480px] bg-slate-800 rounded-xl shadow-2xl border border-slate-700/70 overflow-hidden">

            <!-- Header -->
            <div class="flex items-center gap-3 px-5 py-4 border-b border-slate-700/60 bg-slate-900">
              <svg class="w-5 h-5 text-blue-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
              </svg>
              <span class="text-sm font-semibold text-white">Confirmar guardado</span>
              <span class="text-xs text-gray-500 font-mono ml-auto">{{ deviceId }}</span>
            </div>

            <!-- Resumen de cambios -->
            <div class="px-5 py-4 space-y-4">
              <p class="text-xs text-gray-400">Se aplicaran los siguientes parametros al dispositivo:</p>

              <div class="grid grid-cols-2 gap-3 text-xs">
                <!-- ROI -->
                <div class="bg-slate-900 rounded-lg p-3">
                  <p class="text-[10px] font-semibold text-indigo-400 uppercase tracking-widest mb-2">Region de Interes</p>
                  <p class="font-mono text-white">{{ labRoi[2] }}&times;{{ labRoi[3] }} @ ({{ labRoi[0] }}, {{ labRoi[1] }})</p>
                  <p v-if="labRoi[4] > 0" class="text-gray-500 mt-0.5">Margen inferior: {{ labRoi[4] }}px</p>
                </div>
                <!-- Histeresis -->
                <div class="bg-slate-900 rounded-lg p-3">
                  <p class="text-[10px] font-semibold text-indigo-400 uppercase tracking-widest mb-2">Histeresis</p>
                  <div class="flex gap-3">
                    <span class="text-green-300 font-mono">Alto {{ labThresh.high.toFixed(2) }}</span>
                    <span class="text-red-300 font-mono">Bajo {{ labThresh.low.toFixed(2) }}</span>
                  </div>
                </div>
                <!-- Umbrales -->
                <div class="bg-slate-900 rounded-lg p-3 col-span-2">
                  <p class="text-[10px] font-semibold text-indigo-400 uppercase tracking-widest mb-2">Umbrales de Senal</p>
                  <div class="grid grid-cols-5 gap-x-3 gap-y-1 font-mono">
                    <div v-for="f in thresholdFields" :key="f.key">
                      <p class="text-[10px] text-gray-500">{{ f.label.replace('Umbral ','').replace('Gate ','') }}</p>
                      <p class="text-white">{{ (labThresh[f.key] ?? f.default).toFixed(2) }}</p>
                    </div>
                  </div>
                </div>
                <!-- FSM -->
                <div class="bg-slate-900 rounded-lg p-3 col-span-2">
                  <p class="text-[10px] font-semibold text-indigo-400 uppercase tracking-widest mb-2">FSM</p>
                  <div class="grid grid-cols-5 gap-x-3 font-mono">
                    <div><p class="text-[10px] text-gray-500">Confirm.</p><p class="text-white">{{ labFsm.n_frames }}</p></div>
                    <div><p class="text-[10px] text-gray-500">Cooldown</p><p class="text-white">{{ labFsm.cooldown }}</p></div>
                    <div><p class="text-[10px] text-gray-500">Salida</p><p class="text-white">{{ labFsm.exit_frames }}</p></div>
                    <div><p class="text-[10px] text-gray-500">Timeout</p><p class="text-white">{{ labFsm.max_wait_exit_frames }}</p></div>
                    <div><p class="text-[10px] text-gray-500">Rearm</p><p class="text-violet-300">{{ labFsm.min_rearm_s }}s</p></div>
                  </div>
                </div>
                <!-- Rango Beige HSV -->
                <div class="bg-slate-900 rounded-lg p-3 col-span-2">
                  <p class="text-[10px] font-semibold text-amber-400 uppercase tracking-widest mb-2">Rango Beige (HSV)</p>
                  <div class="grid grid-cols-5 gap-x-3 font-mono">
                    <div><p class="text-[10px] text-gray-500">H min</p><p class="text-amber-300">{{ labBeigeRange.hMin }}</p></div>
                    <div><p class="text-[10px] text-gray-500">H max</p><p class="text-amber-300">{{ labBeigeRange.hMax }}</p></div>
                    <div><p class="text-[10px] text-gray-500">S min</p><p class="text-amber-300">{{ labBeigeRange.sMin }}</p></div>
                    <div><p class="text-[10px] text-gray-500">S max</p><p class="text-amber-300">{{ labBeigeRange.sMax }}</p></div>
                    <div><p class="text-[10px] text-gray-500">V min</p><p class="text-amber-300">{{ labBeigeRange.vMin2 }}</p></div>
                  </div>
                </div>
              </div>

              <p class="text-[10px] text-yellow-500/80">Esta accion sobreescribe la configuracion activa del detector. Los cambios tienen efecto inmediato.</p>
            </div>

            <!-- Botones -->
            <div class="flex items-center justify-end gap-3 px-5 py-3.5 border-t border-slate-700/60 bg-slate-900">
              <button @click="labConfirmOpen = false"
                class="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-slate-700 transition-colors">
                Cancelar
              </button>
              <button @click="labConfirmOpen = false; saveLabConfig()"
                class="flex items-center gap-2 px-5 py-2 rounded-lg text-sm font-semibold bg-blue-600 hover:bg-blue-500 text-white transition-colors">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"/>
                </svg>
                Si, guardar
              </button>
            </div>

          </div>
        </div>
      </transition>

      <!-- Drawer de Guia de parametros -->
      <transition name="guide-slide">
        <div v-if="labGuideOpen"
          class="absolute inset-y-0 right-0 w-[400px] flex flex-col bg-slate-900 border-l border-indigo-700/50 shadow-2xl z-10 overflow-hidden">

          <!-- Header del drawer -->
          <div class="flex items-center justify-between px-5 py-3.5 border-b border-slate-700/70 shrink-0 bg-slate-800">
            <div class="flex items-center gap-2">
              <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/>
              </svg>
              <span class="text-sm font-semibold text-white">Guia de Parametros</span>
            </div>
            <button @click="labGuideOpen = false" class="text-gray-500 hover:text-white transition-colors">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <!-- Contenido scrollable -->
          <div class="flex-1 overflow-y-auto p-5 space-y-6 text-xs text-gray-300">

            <!-- Colores del analisis -->
            <section>
              <h4 class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest mb-3">Colores del Analisis Visual</h4>
              <div class="space-y-3">
                <div class="flex gap-3">
                  <span class="mt-0.5 shrink-0 w-3.5 h-3.5 rounded-sm bg-amber-400 opacity-90"></span>
                  <div>
                    <p class="font-medium text-amber-300">Naranja — Tela beige detectada</p>
                    <p class="text-gray-500 mt-0.5">Pixeles cuyo tono HSV coincide con beige/crema. El slider <span class="text-white font-medium">Umbral Beige</span> controla el brillo minimo: moverlo a la izquierda detecta tonos mas oscuros (mas area naranja); a la derecha solo beige muy claro (menos naranja).</p>
                  </div>
                </div>
                <div class="flex gap-3">
                  <span class="mt-0.5 shrink-0 w-3.5 h-3.5 rounded-sm bg-cyan-400 opacity-90"></span>
                  <div>
                    <p class="font-medium text-cyan-300">Cyan — Bordes Sobel</p>
                    <p class="text-gray-500 mt-0.5">Pixeles donde el algoritmo Sobel detecta un gradiente de intensidad. El slider <span class="text-white font-medium">Umbral Edge</span> ajusta la sensibilidad: izquierda = bordes suaves tambien se marcan; derecha = solo bordes muy fuertes.</p>
                  </div>
                </div>
                <div class="flex gap-3">
                  <span class="mt-0.5 shrink-0 w-3.5 h-3.5 rounded-sm bg-green-500 opacity-90"></span>
                  <div>
                    <p class="font-medium text-green-300">Borde verde del ROI</p>
                    <p class="text-gray-500 mt-0.5">El score de beige actual es mayor o igual al umbral configurado. Significa que hay suficiente tela en el ROI para activar la deteccion.</p>
                  </div>
                </div>
                <div class="flex gap-3">
                  <span class="mt-0.5 shrink-0 w-3.5 h-3.5 rounded-sm bg-red-500 opacity-90"></span>
                  <div>
                    <p class="font-medium text-red-300">Borde rojo del ROI</p>
                    <p class="text-gray-500 mt-0.5">El score de beige no llega al umbral. La tela no seria detectada con la configuracion actual.</p>
                  </div>
                </div>
                <div class="flex gap-3">
                  <span class="mt-0.5 shrink-0 w-3.5 h-1.5 bg-white opacity-60 self-center"></span>
                  <div>
                    <p class="font-medium text-white">Linea blanca (barra superior del ROI)</p>
                    <p class="text-gray-500 mt-0.5">Marca el umbral actual sobre la barra de progreso. La barra de color a su izquierda es el score en tiempo real.</p>
                  </div>
                </div>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- ROI -->
            <section>
              <h4 class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest mb-3">Region de Interes (ROI)</h4>
              <div class="space-y-2.5 text-gray-400">
                <p><span class="text-white font-medium">X, Y (origen):</span> Coordenada de la esquina superior-izquierda del ROI en pixeles desde la esquina superior-izquierda de la imagen.</p>
                <p><span class="text-white font-medium">Ancho / Alto:</span> Dimension del rectangulo en pixeles. Ajustar para cubrir exactamente la zona donde cae la tela.</p>
                <p><span class="text-white font-medium">Margen inferior excluido:</span> Ignora N pixeles del borde inferior del ROI. Util para eliminar el borde del rodillo o la mesa de la deteccion.</p>
                <p><span class="text-white font-medium">Arrastrar en camara:</span> Con la camara activa, arrastra el mouse sobre la imagen para redibujar el ROI directamente.</p>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Histeresis -->
            <section>
              <h4 class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest mb-3">Histeresis</h4>
              <p class="text-gray-400 mb-2.5">Mecanismo anti-ruido: el sistema activa un evento solo cuando el score sube por encima del umbral alto, y lo cancela solo cuando baja del umbral bajo. Evita activaciones intermitentes.</p>
              <div class="space-y-2 text-gray-400">
                <p><span class="text-green-300 font-medium">Umbral Alto (activacion):</span> Score minimo para iniciar la deteccion. Valores altos = mas exigente. Rango 0-1.</p>
                <p><span class="text-red-300 font-medium">Umbral Bajo (cancelacion):</span> Si el score cae por debajo, el sistema entra en WAIT_EXIT esperando que la tela salga. Debe ser menor que el umbral alto.</p>
                <p class="text-gray-600 italic">Ejemplo: Alto=0.7, Bajo=0.3 — activa cuando score llega a 0.7, pero solo cancela si baja a 0.3.</p>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Umbrales de senal -->
            <section>
              <h4 class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest mb-3">Umbrales de Senal</h4>
              <div class="space-y-2.5 text-gray-400">
                <p><span class="text-white font-medium">Umbral Edge:</span> Densidad minima de bordes Canny en el ROI. Valores bajos = cualquier textura activa la senal. Valores altos = solo con bordes muy marcados.</p>
                <p><span class="text-white font-medium">Umbral Color:</span> Diferencia minima entre el histograma actual y la referencia del fondo. Sube si hay muchos falsos positivos por cambios de luz.</p>
                <p><span class="text-white font-medium">Gate Flujo Optico:</span> Interruptor principal del detector. Si el movimiento (flujo optico) es menor que este valor, el score se fuerza a cero. Sube para ignorar vibraciones de la maquina.</p>
                <p><span class="text-amber-300 font-medium">Umbral Beige:</span> Cobertura minima de color beige/crema en el ROI (fraccion 0-1). La linea blanca en la barra del ROI marca este valor. El borde del ROI se vuelve verde cuando el score actual lo supera.</p>
                <p><span class="text-white font-medium">Umbral Caida (dy):</span> Velocidad vertical minima en px/frame para detectar un evento de caida. Ignorar si el producto no cae en la camara.</p>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Rango HSV en guia -->
            <section>
              <div class="flex items-center gap-2 mb-3">
                <span class="w-3 h-3 rounded-sm bg-amber-400 shrink-0"></span>
                <h4 class="text-[10px] font-bold text-amber-400 uppercase tracking-widest">Rango de Deteccion Beige (HSV)</h4>
              </div>
              <p class="text-gray-400 mb-2.5">Estos controles solo afectan el <span class="text-white">analisis visual del Lab</span> — no modifican el backend de deteccion. Sirven para encontrar el rango de color correcto para tu tela antes de ajustar los umbrales de senal.</p>
              <p class="text-gray-500 text-[10px] mb-3 italic">Aparecen en el panel izquierdo cuando el modo Analisis esta activo en Beige o Combinado.</p>
              <div class="space-y-2 text-gray-400">
                <p><span class="text-amber-300 font-medium">Tono min/max (H):</span> Define el rango de tono en el espectro de color. Beige oscuro: H 12-38. Beige claro/crema: H 20-55. Amarillento: H 30-65. La barra de color al pie muestra el rango seleccionado.</p>
                <p><span class="text-amber-300 font-medium">Saturacion min (S):</span> Telas claras/blancas tienen saturacion muy baja (2-10). Telas de color vivo tienen saturacion alta (80+). <span class="text-white">Bajar este valor es clave para detectar telas palidas o crema.</span></p>
                <p><span class="text-amber-300 font-medium">Saturacion max (S):</span> Limita el rango superior para no detectar objetos de colores saturados como cables o maquinaria.</p>
                <p><span class="text-amber-300 font-medium">Brillo min (V):</span> Filtra pixels muy oscuros. Telas claras tienen brillo alto (150+). Sube este valor para que el fondo oscuro no se quede incluido en la mascara naranja.</p>
              </div>
              <div class="mt-3 bg-slate-800 rounded p-2.5 text-[10px] space-y-1 font-mono text-gray-500">
                <p class="text-gray-400 font-semibold text-[9px] uppercase tracking-widest mb-1.5">Valores sugeridos por tipo de tela</p>
                <p><span class="text-amber-300">Beige oscuro/marron:</span>  H 12-38 · S 20-145 · V 80+</p>
                <p><span class="text-yellow-200">Beige claro/crema:</span>    H 20-55 · S 5-100  · V 140+</p>
                <p><span class="text-white">Blanca/muy palida:</span>      H 0-60  · S 0-40   · V 180+</p>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- FSM -->
            <section>
              <h4 class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest mb-3">Maquina de Estados (FSM)</h4>
              <div class="mb-3 bg-slate-800 rounded p-2 space-y-1.5 text-[10px] font-mono">
                <div class="flex items-center gap-2"><span class="w-20 text-slate-400 text-center border border-slate-600 rounded px-1 py-0.5">IDLE</span><span class="text-gray-500">→ esperando score alto</span></div>
                <div class="flex items-center gap-2"><span class="w-20 text-blue-300 text-center border border-blue-800 rounded px-1 py-0.5">DETECTING</span><span class="text-gray-500">→ acumulando frames</span></div>
                <div class="flex items-center gap-2"><span class="w-20 text-green-300 text-center border border-green-800 rounded px-1 py-0.5">WAIT_EXIT</span><span class="text-gray-500">→ esperando que salga</span></div>
                <div class="flex items-center gap-2"><span class="w-20 text-purple-300 text-center border border-purple-800 rounded px-1 py-0.5">COOLDOWN</span><span class="text-gray-500">→ bloqueado temporalmente</span></div>
              </div>
              <div class="space-y-2 text-gray-400">
                <p><span class="text-white font-medium">Frames de confirmacion:</span> Cuantos frames consecutivos con score &gt; umbral alto se necesitan para registrar un evento. Sube para reducir falsos positivos.</p>
                <p><span class="text-white font-medium">Cooldown:</span> Frames de espera obligatoria despues de cada evento. Evita contar dos veces una misma tela.</p>
                <p><span class="text-white font-medium">Frames de salida:</span> Frames consecutivos con score bajo para confirmar que la tela salio del ROI.</p>
                <p><span class="text-white font-medium">Timeout salida:</span> Si la FSM lleva este numero de frames en WAIT_EXIT sin confirmacion, cierra el evento automaticamente. A 8fps: 50 frames = ~6s.</p>
              </div>
            </section>

            <div class="border-t border-slate-700/50"></div>

            <!-- Calibracion -->
            <section class="pb-2">
              <h4 class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest mb-3">Calibracion</h4>
              <div class="space-y-2 text-gray-400">
                <p>Captura ~30 frames del fondo vacio y establece la distribucion de color de referencia para el detector de cambio de histograma.</p>
                <p><span class="text-yellow-300 font-medium">Cuando usar:</span> tras cambiar la iluminacion, reposicionar la camara, o si hay muchos falsos positivos de color.</p>
                <p><span class="text-red-300 font-medium">Importante:</span> ejecutar con el ROI completamente libre de tela. Si hay tela durante la calibracion el detector quedara mal calibrado.</p>
              </div>
            </section>

          </div>
        </div>
      </transition>

    </div>
  </transition>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import QRCode from 'qrcode'
import { configService, gatewayService, cameraService } from '../services/api'
import HelpTip from '../components/HelpTip.vue'
import Produccion from './Produccion.vue'

const route  = useRoute()
const router = useRouter()
const GATEWAY_URL = import.meta.env.VITE_GATEWAY_URL || '/api/gateway'

const svgIcon = (d) => `<svg xmlns="http://www.w3.org/2000/svg" fill="none" stroke="currentColor" viewBox="0 0 24 24" class="w-4 h-4"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="${d}"/></svg>`

const sections = [
  { id: 'cloud',     label: 'Cloud',        desc: 'Conexion con la nube — configurar primero', svg: svgIcon('M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z') },
  { id: 'device',    label: 'Dispositivo',  desc: 'Identificacion y modo del Jetson',          svg: svgIcon('M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18') },
  { id: 'camera',    label: 'Camara',       desc: 'Configuracion RTSP y verificacion',         svg: svgIcon('M15 10l4.553-2.069A1 1 0 0121 8.876v6.248a1 1 0 01-1.447.894L15 14M3 8a2 2 0 012-2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8z') },
  { id: 'detection', label: 'Deteccion',    desc: 'ROI, umbrales, histeresis y FSM',           svg: svgIcon('M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4') },
  { id: 'oee',       label: 'OEE',          desc: 'Parametros de eficiencia de linea',         svg: svgIcon('M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z') },
  { id: 'produccion', label: 'Produccion',  desc: 'Monitor en vivo con camara y contadores', svg: svgIcon('M15 10l4.553-2.069A1 1 0 0121 8.82v6.36a1 1 0 01-1.447.894L15 14M3 8a2 2 0 012-2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V8z') },
  { id: 'tablet',    label: 'Tablet',       desc: 'Conexion de la tablet HMI',                 svg: svgIcon('M12 18h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z') },
  { id: 'deploy',    label: 'Despliegue',   desc: 'Servicios Docker y variables del gateway', svg: svgIcon('M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4') },
]

const tips = {
  deviceId:         'Identificador del dispositivo, guardado en la BD. Todos los servicios edge lo auto-detectan al iniciar. Puedes cambiarlo aqui y los servicios lo usaran tras reiniciar.',
  mode:             'El dispositivo opera con el detector textil de cortes, presencia y paradas.',
  lineName:         'Codigo de la linea de produccion que aparece en los reportes OEE y en la nube. Debe coincidir con el registro del sistema central.',
  frameBackend:     'Motor de decodificacion de video. GStreamer NVDEC aprovecha la GPU del Jetson para decodificar H.264/H.265, reduciendo la carga en CPU. Si falla, el sistema conmuta automaticamente a OpenCV.',
  cameraUrl:        'Direccion del flujo de video. Soporta RTSP para camaras IP, HTTP para MJPEG, o un indice numerico (0, 1, 2) para camaras USB conectadas al Jetson.',
  fps:              'Frames por segundo que el sistema intentara procesar. Valores altos mejoran la precision de deteccion pero consumen mas recursos de GPU y CPU.',
  roi:              'Rectangulo dentro del frame donde el sistema analiza movimiento. Solo los eventos que ocurran dentro de esta area generan detecciones validas.',
  roiX:             'Coordenada horizontal de la esquina superior izquierda del ROI, en pixeles desde el borde izquierdo del frame.',
  roiY:             'Coordenada vertical de la esquina superior izquierda del ROI, en pixeles desde el borde superior del frame.',
  roiW:             'Ancho del rectangulo ROI en pixeles. Junto con X, define el limite derecho de la zona de analisis.',
  roiH:             'Alto del rectangulo ROI en pixeles. Junto con Y, define el limite inferior de la zona de analisis.',
  roiMargin:        'Banda en la parte baja del ROI que se ignora durante el analisis. Util para excluir bordes de maquina o superficies estaticas que generarian falsos positivos.',
  thresholds:       'Sensibilidad de cada canal del algoritmo de fusion multi-senal. El sistema combina estas senales para generar un score unificado de deteccion.',
  hysteresis:       'Doble umbral que evita oscilaciones en la deteccion. Funciona como un termostato: se activa al superar el umbral alto y se desactiva solo al bajar del umbral bajo.',
  hystHigh:         'Score de fusion minimo para iniciar una deteccion. Cuando la senal combinada supera este valor, la FSM transiciona de IDLE a DETECTING.',
  hystLow:          'Score de fusion por debajo del cual se cancela una deteccion activa. Debe ser menor que el umbral alto para crear la zona muerta de histeresis.',
  fsm:              'Maquina de estados finitos que gobierna el flujo de deteccion: IDLE (espera) -> DETECTING (senal activa) -> CONFIRMING (verificacion N frames) -> COOLDOWN (anti-rebote).',
  fsmFrames:        'Cantidad de frames consecutivos con senal activa necesarios para confirmar un evento real. Valores altos reducen falsos positivos pero agregan latencia.',
  fsmCooldown:      'Frames de espera despues de confirmar un evento. Evita conteos duplicados causados por rebote o vibraciones residuales del producto.',
  microStop:        'Duracion maxima de inactividad que se clasifica como micro-parada. Son interrupciones breves y frecuentes en la produccion que afectan la eficiencia.',
  stopMax:          'Duracion maxima para clasificar como parada normal. Tiempos de inactividad superiores a este valor se registran como parada mayor.',
  snapshotInterval: 'Frecuencia con la que el sistema calcula y almacena un resumen OEE que incluye disponibilidad, rendimiento y calidad de la linea.',
  syncInterval:     'Frecuencia con la que los datos acumulados se envian al servidor central. Intervalos largos reducen el trafico de red pero aumentan la latencia de los reportes.',
  cloudUrl:         'Endpoint base de la API cloud donde el Enviador manda los snapshots OEE. Requiere HTTPS en produccion. El Enviador llama POST {url}/api/v1/edge/oee.',
  cloudApiKey:      'Clave secreta de autenticacion Bearer que el Enviador incluye en cada peticion al servidor central. Si esta vacia, las peticiones se envian sin autenticacion.',
  cloudStatus:      'Estado de la conexion en tiempo real entre el Enviador y la nube. Verde = ultimo envio exitoso hace menos de 5 minutos. Amarillo = degradado.',
  tabletHost:       'Direccion IP del Jetson en la red local. La tablet HMI se conectara a esta direccion para recibir datos de produccion en tiempo real.',
  tabletPort:       'Puerto donde corre la app web del tablet (nginx, por defecto 8090). El navegador carga la interfaz HMI desde este puerto.',
  gatewayPort:      'Puerto del Edge Gateway API (por defecto 8005). La app del tablet se conecta a este puerto para recibir datos de produccion, paradas y eventos en tiempo real.',
  tabletUrl:        'Deep link que abre la app del tablet y la conecta automaticamente al gateway. Al escanear el QR no es necesaria ninguna configuracion adicional en la tablet.',
  calibration:      'Ejecuta una calibracion automatica capturando 30 muestras del fondo de la escena. Establece la linea base para la deteccion de cambios por color y bordes.',
  save:             'Persiste toda la configuracion en la base de datos. Los servicios de deteccion y OEE recargan automaticamente los nuevos valores en aproximadamente 5 segundos.',
  pgPassword:       'Contrasena del usuario mentor en PostgreSQL. Solo aplica en un despliegue limpio desde cero (sin volumen existente). Si la base de datos ya esta inicializada, este valor debe coincidir con el que se uso al crearla — cambiarlo no modifica la contrasena en una BD existente.',
  timezone:         'Zona horaria del sistema operativo del Jetson. Afecta los timestamps de logs, eventos OEE y registros en la base de datos. Debe coincidir con la zona horaria real de la planta donde esta instalado el equipo.',
  envFile:          'El archivo .env solo contiene credenciales de infraestructura (PostgreSQL, timezone). Toda la configuracion de negocio (device_id, camara, OEE, cloud) se lee directamente de la base de datos.',
}

const activeSection  = ref('cloud')
const currentSection = computed(() => sections.find(s => s.id === activeSection.value) || sections[0])

const deviceId       = ref('jetson-default')  // unused legacy — kept for compatibility
const lineaId        = ref('')
const loadedLineaId  = ref('')   // linea_id que está realmente cargado en el formulario
const frameBackend   = ref('opencv')
const config = ref({ thresholds: { edge: 0.4, color: 0.6, flow: 0.5, dy: 5.0, beige: 0.35, high: 0.7, low: 0.3, presencia_threshold: 0.05, presencia_scale: 0.005, presencia_hold: 750, presencia_window: 375, presencia_pixel_threshold: 8 }, fsm: { n_frames: 3, cooldown: 8, exit_frames: 5, max_wait_exit_frames: 50, min_rearm_s: 6.0 }, mode: 'textil', config_version: 0 })
const roi    = ref([120, 60, 320, 200, 0])
const camera = ref({ url: '', fps: 25 })
const oee    = ref({ line_name: '', micro_stop_max_s: 120, stop_max_s: 86400, snapshot_interval_s: 1800, vel_unit: 'uh', vel_nominal_us: 0.008333333 })
const cloud  = ref({ sync_interval_s: 300, cloud_url: 'https://api.mentoredge.io', cloud_api_key: '' })
const tablet = ref({ host: '', port: 8090, gatewayPort: 8005 })
// Valor que el operador ve/escribe en la UI (en la unidad seleccionada)
const velNominalDisplay = ref(30)

// Sincronizar velNominalDisplay cuando cambia vel_unit
watch(() => oee.value.vel_unit, (newUnit, oldUnit) => {
  if (oldUnit === 'us' && newUnit === 'uh') {
    velNominalDisplay.value = parseFloat((oee.value.vel_nominal_us * 3600).toFixed(4))
  } else if (oldUnit === 'uh' && newUnit === 'us') {
    velNominalDisplay.value = parseFloat((oee.value.vel_nominal_us).toFixed(6))
  }
})

// Cuando el operador escribe en el campo, actualizar vel_nominal_us internamente
watch(velNominalDisplay, (v) => {
  const num = parseFloat(v) || 0
  oee.value.vel_nominal_us = oee.value.vel_unit === 'uh' ? num / 3600 : num
})

const linesList      = ref([])
const linesLoading   = ref(false)
const lineMeta       = ref({})   // { [linea_id]: { config_version, line_name } }
const lineUpdating   = ref(null) // linea_id que está siendo guardado

const cloudHealth        = ref(null)
const cloudChecking      = ref(false)
const cloudPing          = ref(null)   // { reachable, latency_ms, status_code, error, tested_at, tested_url }
const cloudPinging       = ref(false)
const cloudBufferPending = ref(null)   // número de eventos pendientes en buffer

const cameraChecking = ref(false)
const cameraHealth   = ref(null)
const showApiKey     = ref(false)
const tabletCopied   = ref(false)
const envCopied      = ref(false)
const message      = ref('')
const messageClass = ref('')
const saving       = ref(false)
const calibrating  = ref(false)
const lastSaved    = ref(null)
const deploy = ref({ pg_password: '', tz: 'America/Lima', linea_id: '' })
const showPgPass   = ref(false)
const envSaving        = ref(false)
const envSaved         = ref(false)
const envReloading     = ref(false)
const envPendingReload = ref(false)
const envRaw       = ref('')
const serviceChecks = ref([
  { id: 'gateway',     name: 'Edge Gateway',    port: 8005, endpoint: '/api/gateway/health', status: null },
  { id: 'detector',    name: 'Vision Detector', port: 8001, endpoint: '/api/detector/health', status: null },
  { id: 'resiliencia', name: 'Resiliencia',     port: 8002, endpoint: '/api/resiliencia/health', status: null },
  { id: 'config',      name: 'Config Service',  port: 8004, endpoint: '/api/config/health', status: null },
  { id: 'enviador',    name: 'Enviador',        port: 8003, endpoint: '/api/enviador/health', status: null },
  { id: 'postgres',    name: 'PostgreSQL',      port: 5432, endpoint: null, status: null },
  { id: 'ui',          name: 'UI Local',        port: 8080, endpoint: null, status: null },
  { id: 'tablet',      name: 'Tablet App',      port: 8090, endpoint: null, status: null },
])
const servicesChecking = ref(false)

// --- Vista Previa ---
const frameCanvasRef    = ref(null)
const overlayCanvasRef  = ref(null)
const previewHasFrame   = ref(false)
const snapshotLoading   = ref(false)
const snapshotError     = ref(null)
const snapshotTimestamp = ref(null)
const liveMode          = ref(false)
let   liveAbortCtrl     = null

const envVars = computed(() => {
  if (!envRaw.value) return []
  const placeholders = ['CHANGE_ME_STRONG_PASSWORD', 'CHANGE_ME_API_KEY', 'jetson-orin-XXXX', 'MENTOR_PLANTAXX_LX']
  return envRaw.value.split('\n')
    .filter(l => l.trim() && !l.startsWith('#') && l.includes('='))
    .map(l => {
      const eq = l.indexOf('=')
      const key = l.substring(0, eq).trim()
      const val = l.substring(eq + 1).trim()
      let display = val
      if (key === 'POSTGRES_PASSWORD' && val) display = '\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022'
      else if (key === 'CLOUD_API_KEY' && val.length > 4) display = '****' + val.slice(-4)
      else if (key === 'CAMERA_URL' && val) display = val.replace(/:\/\/[^@]+@/, '://****@')
      return { key, value: val, display: display || '(vacio)', set: !!val && !placeholders.includes(val) }
    })
})

const thresholdFields = [
  { key: 'edge',  label: 'Umbral Edge',          tip: 'Densidad minima de bordes (Canny) detectados en el ROI para considerar que hay actividad en la escena.',                                                          min: 0, max: 1,  step: 0.01, default: 0.4  },
  { key: 'color', label: 'Umbral Color',          tip: 'Diferencia minima entre el histograma de color actual y la referencia base para indicar un cambio significativo.',                                         min: 0, max: 1,  step: 0.01, default: 0.6  },
  { key: 'flow',  label: 'Gate Flujo optico',     tip: 'Umbral gate de fusion: si el flujo optico (Farneback CPU o chip OFA del Jetson) es menor que este valor, el score de deteccion se fuerza a cero. Actua como interruptor principal de la deteccion.',  min: 0, max: 1,  step: 0.01, default: 0.5  },
  { key: 'beige', label: 'Umbral Beige',          tip: 'Cobertura minima de color beige/crema en el ROI (fraccion 0-1) para que la senal de color de prenda contribuya a la deteccion.',                          min: 0, max: 1,  step: 0.01, default: 0.35 },
  { key: 'dy',    label: 'Umbral Caida (dy)',      tip: 'Velocidad vertical minima en pixeles por frame para detectar un evento de caida de producto en la zona de interes.',                                        min: 0, max: 50, step: 0.5,  default: 5.0  },
]

const savedDot = computed(() => lastSaved.value ? 'bg-green-400' : 'bg-gray-500')

const cameraStatusClass = computed(() => {
  if (!camera.value.url) return 'bg-slate-700 text-gray-500'
  if (!cameraHealth.value) return 'bg-blue-900 text-blue-300'
  const s = cameraHealth.value.status
  if (s === 'ok') return 'bg-green-900 text-green-300'
  if (s === 'tcp_ok_no_ffprobe') return 'bg-yellow-900 text-yellow-300'
  return 'bg-red-900 text-red-300'
})

const cameraStatusLabel = computed(() => {
  if (!camera.value.url) return 'Sin URL'
  if (!cameraHealth.value) return 'URL configurada'
  const s = cameraHealth.value.status
  if (s === 'ok') return `OK - ${cameraHealth.value.rtsp_codec || 'stream OK'}`
  if (s === 'tcp_ok_no_ffprobe') return 'TCP OK - sin ffprobe'
  if (s === 'tcp_unreachable') return 'Sin conexion TCP'
  if (s === 'rtsp_error') return 'Error RTSP'
  return s
})

function fmtSeg(s) {
  if (s < 60) return `${s}s`
  if (s >= 86400 && s % 86400 === 0) return `${s / 86400}d`
  if (s >= 3600) {
    const h = Math.floor(s / 3600)
    const m = Math.floor((s % 3600) / 60)
    return m === 0 ? `${h}h` : `${h}h ${m}min`
  }
  const m = Math.floor(s / 60)
  const r = s % 60
  return r === 0 ? `${m}min` : `${m}m ${r}s`
}

const microBarW = computed(() => { const t = Math.max(oee.value.stop_max_s * 1.5, oee.value.micro_stop_max_s + 100); return Math.round((oee.value.micro_stop_max_s / t) * 400) })
const stopBarW  = computed(() => { const t = Math.max(oee.value.stop_max_s * 1.5, oee.value.micro_stop_max_s + 100); return Math.round(((oee.value.stop_max_s - oee.value.micro_stop_max_s) / t) * 400) })
const snapshotsPerBatch = computed(() => oee.value.snapshot_interval_s <= 0 ? 0 : Math.round(cloud.value.sync_interval_s / oee.value.snapshot_interval_s))

const tabletURL   = computed(() => { const h = tablet.value.host || detectedHost.value || window.location.hostname || 'localhost'; return `http://${h}:${tablet.value.port || 8090}` })
const gatewayURL  = computed(() => { const h = tablet.value.host || detectedHost.value || window.location.hostname || 'localhost'; return `http://${h}:${tablet.value.gatewayPort || 8005}` })
const deepLinkURL = computed(() => `${tabletURL.value}?edge=${encodeURIComponent(gatewayURL.value)}`)
const qrModules = computed(() => {
  if (!tablet.value.host && !detectedHost.value) return []
  try {
    const qr = QRCode.create(deepLinkURL.value, { errorCorrectionLevel: 'M' })
    const s = qr.modules.size
    const d = qr.modules.data
    const m = []
    for (let r = 0; r < s; r++) { const row = []; for (let c = 0; c < s; c++) row.push(d[r * s + c] ? 1 : 0); m.push(row) }
    return m
  } catch { return [] }
})
const qrSize = computed(() => qrModules.value.length || 21)
const qrDarkCells = computed(() => {
  const c = []
  for (let y = 0; y < qrModules.value.length; y++)
    for (let x = 0; x < qrModules.value[y].length; x++)
      if (qrModules.value[y][x]) c.push({ x, y })
  return c
})

const generatedEnv = computed(() => `# Mentor Edge - generado desde UI ${new Date().toLocaleDateString('es-MX')}
# Infraestructura Docker (credenciales y timezone)
# Toda la config de negocio (device_id, camera, oee, cloud) se lee de la BD.
POSTGRES_DB=mentor_edge
POSTGRES_USER=mentor
POSTGRES_PASSWORD=${deploy.value.pg_password || 'mentor'}

TZ=${deploy.value.tz || 'America/Lima'}`)

const formatInterval = (s) => {
  const m = Math.floor(s / 60)
  const r = s % 60
  if (m === 0) return `${s}s`
  if (r === 0) return `${m}min`
  return `${m}m${r}s`
}

const detectedHost = ref('')

const autoDetectHost = () => {
  try {
    const h = window.location.hostname
    if (h && h !== 'localhost' && h !== '127.0.0.1') {
      tablet.value.host = h
      detectedHost.value = h
    }
  } catch { /* ignore */ }
  tabletCopied.value = false
}

const onTabletChanged = () => { tabletCopied.value = false }

const loadLines = async () => {
  linesLoading.value = true
  try {
    const data = await configService.getLines()
    const list = data?.lines ?? []
    linesList.value = list
    // Cargar metadata (version + line_name) de cada línea en paralelo
    const metas = await Promise.allSettled(
      list.map(id => configService.getConfig(id).then(cfg => ({ id, config_version: cfg.config_version ?? 0, line_name: cfg.oee?.line_name || '' })))
    )
    const m = {}
    metas.forEach(r => { if (r.status === 'fulfilled') m[r.value.id] = r.value })
    lineMeta.value = m
  } catch {
    linesList.value = []
  } finally {
    linesLoading.value = false
  }
}

const updateLineConfig = async (targetLineaId) => {
  lineUpdating.value = targetLineaId
  try {
    await configService.updateConfig({
      roi: roi.value,
      thresholds: config.value.thresholds,
      fsm: config.value.fsm,
      mode: config.value.mode,
      camera: camera.value.url ? camera.value : undefined,
      oee: oee.value,
      cloud: cloud.value,
      tablet: tablet.value,
    }, targetLineaId)
    // Refrescar metadata de esa línea
    try {
      const cfg = await configService.getConfig(targetLineaId)
      lineMeta.value = { ...lineMeta.value, [targetLineaId]: { id: targetLineaId, config_version: cfg.config_version ?? 0, line_name: cfg.oee?.line_name || '' } }
    } catch { /* no critico */ }
    showMessage(`Config guardada en Línea ${targetLineaId} (v${lineMeta.value[targetLineaId]?.config_version ?? '?'}).`, 'success')
  } catch {
    showMessage(`Error al guardar en Línea ${targetLineaId}`, 'error')
  } finally {
    lineUpdating.value = null
  }
}

const selectLine = async (id) => {
  if (String(id) === String(lineaId.value)) return
  lineaId.value = String(id)
  router.replace('/config/' + id)
  await loadConfig()
  showMessage(`Línea ${id} cargada.`, 'info')
}

const checkAllServices = async () => {
  servicesChecking.value = true
  serviceChecks.value.forEach(s => { if (s.id !== 'ui') s.status = null })
  const uiSvc = serviceChecks.value.find(s => s.id === 'ui')
  if (uiSvc) uiSvc.status = 'ok'
  const proxied = serviceChecks.value.filter(s => s.endpoint)
  await Promise.all(proxied.map(async (svc) => {
    try { await axios.get(svc.endpoint, { timeout: 5000 }); svc.status = 'ok' }
    catch { svc.status = 'error' }
  }))
  const pgSvc = serviceChecks.value.find(s => s.id === 'postgres')
  if (pgSvc) pgSvc.status = serviceChecks.value.find(s => s.id === 'config')?.status || null
  const tabSvc = serviceChecks.value.find(s => s.id === 'tablet')
  if (tabSvc) {
    try {
      const h = tablet.value.host || detectedHost.value || window.location.hostname
      await fetch(`http://${h}:${tablet.value.port || 8090}`, { mode: 'no-cors', signal: AbortSignal.timeout(3000) })
      tabSvc.status = 'ok'
    } catch { tabSvc.status = 'error' }
  }
  servicesChecking.value = false
}

const checkCloudHealth = async () => {
  cloudChecking.value = true
  cloudHealth.value = null
  cloudBufferPending.value = null
  try {
    const [envResp, bufResp] = await Promise.allSettled([
      axios.get('/api/enviador/health'),
      axios.get('/api/resiliencia/health/buffer'),
    ])
    cloudHealth.value = envResp.status === 'fulfilled'
      ? envResp.value.data
      : { status: 'error', cloud_status: 'unreachable' }
    if (bufResp.status === 'fulfilled') {
      cloudBufferPending.value = bufResp.value.data?.pending_events ?? null
    }
  } catch {
    cloudHealth.value = { status: 'error', cloud_status: 'unreachable' }
  } finally {
    cloudChecking.value = false
  }
}

const pingCloud = async () => {
  // Validar que haya una URL antes de llamar al backend
  const rawURL = cloud.value?.cloud_url?.trim()
  if (!rawURL) {
    cloudPing.value = { reachable: false, latency_ms: 0, status_code: 0, error: 'Ingresa una URL de servidor antes de validar.', tested_at: new Date().toISOString(), tested_url: '' }
    return
  }
  try { new URL(rawURL) } catch {
    cloudPing.value = { reachable: false, latency_ms: 0, status_code: 0, error: 'URL inválida. Debe comenzar con http:// o https://', tested_at: new Date().toISOString(), tested_url: rawURL }
    return
  }

  cloudPinging.value = true
  cloudPing.value = null
  try {
    const res = await axios.get('/api/gateway/edge/cloud/ping', {
      timeout: 12000,
      params: {
        url: rawURL,
        key: cloud.value?.cloud_api_key ?? ''
      }
    })
    cloudPing.value = res.data
  } catch (e) {
    cloudPing.value = { reachable: false, latency_ms: 0, status_code: 0, error: e.message, tested_at: new Date().toISOString(), tested_url: rawURL }
  } finally {
    cloudPinging.value = false
  }
}

const copyTabletURL = async () => {
  try { await navigator.clipboard.writeText(deepLinkURL.value); tabletCopied.value = true; setTimeout(() => { tabletCopied.value = false }, 3000) }
  catch { showMessage('No se pudo copiar', 'error') }
}

const copyEnv = async () => {
  try { await navigator.clipboard.writeText(generatedEnv.value); envCopied.value = true; setTimeout(() => { envCopied.value = false }, 3000) }
  catch { /* silently fail */ }
}

const saveEnvToJetson = async () => {
  envSaving.value = true
  envSaved.value = false
  try {
    await axios.post('/api/env-file', generatedEnv.value, { headers: { 'Content-Type': 'text/plain' } })
    envSaved.value = true
    showMessage('Archivo .env guardado en el Jetson.', 'success')
    setTimeout(() => { envSaved.value = false }, 5000)
  } catch {
    showMessage('No se pudo guardar el .env', 'error')
  } finally {
    envSaving.value = false
  }
}

const loadEnvFromJetson = async () => {
  try {
    const res = await axios.get('/api/env-file', { responseType: 'text' })
    const text = res.data || ''
    envRaw.value = text
    const get = (key, def = '') => { const m = text.match(new RegExp(`^${key}=(.*)$`, 'm')); return m ? m[1] : def }
    deploy.value.pg_password = get('POSTGRES_PASSWORD')
    deploy.value.tz = get('TZ', 'America/Lima')
  } catch { /* no .env yet */ }
}

const reloadEnvFromJetson = async () => {
  envReloading.value = true
  try {
    const res = await axios.get('/api/env-file', { responseType: 'text' })
    const text = res.data || ''
    envRaw.value = text
    const get = (key, def = '') => { const m = text.match(new RegExp(`^${key}=(.*)$`, 'm')); return m ? m[1] : def }
    deploy.value.pg_password = get('POSTGRES_PASSWORD')
    deploy.value.tz = get('TZ', deploy.value.tz || 'America/Lima')
    envPendingReload.value = false
    showMessage('Variables recargadas desde el .env del Jetson.', 'info')
  } catch {
    showMessage('No se pudo leer el .env del Jetson.', 'error')
  } finally {
    envReloading.value = false
  }
}

const drawRoiOverlayCanvas = () => {
  const oc = overlayCanvasRef.value
  if (!oc || !oc.width) return
  const [rx, ry, rw, rh, margin] = roi.value
  const ctx = oc.getContext('2d')
  ctx.clearRect(0, 0, oc.width, oc.height)
  ctx.fillStyle = 'rgba(0,0,0,0.45)'
  ctx.fillRect(0, 0, oc.width, oc.height)
  ctx.clearRect(rx, ry, rw, rh)
  ctx.strokeStyle = '#fbbf24'
  ctx.lineWidth   = 2
  ctx.strokeRect(rx, ry, rw, rh)
  if (margin > 0) {
    ctx.save()
    ctx.strokeStyle = 'rgba(251,191,36,0.55)'
    ctx.setLineDash([5, 4])
    ctx.lineWidth = 1.5
    ctx.beginPath()
    ctx.moveTo(rx, ry + rh - margin)
    ctx.lineTo(rx + rw, ry + rh - margin)
    ctx.stroke()
    ctx.restore()
  }
  ctx.fillStyle = '#fbbf24'
  ctx.font      = 'bold 11px monospace'
  ctx.fillText(`ROI ${rw}\u00d7${rh}`, rx + 4, ry + 14)
}

const renderFrame = (bitmap) => {
  const fc = frameCanvasRef.value
  if (!fc) return
  fc.width  = bitmap.width
  fc.height = bitmap.height
  fc.getContext('2d', { willReadFrequently: true }).drawImage(bitmap, 0, 0)
  const oc = overlayCanvasRef.value
  if (oc && (oc.width !== bitmap.width || oc.height !== bitmap.height)) {
    oc.width  = bitmap.width
    oc.height = bitmap.height
  }
  drawRoiOverlayCanvas()
}

const captureSnapshot = async () => {
  if (!camera.value.url) return
  if (liveMode.value) { liveMode.value = false; liveAbortCtrl?.abort(); liveAbortCtrl = null }
  snapshotLoading.value = true
  snapshotError.value   = null
  try {
    const resp = await fetch(`/api/gateway/edge/camera/snapshot?url=${encodeURIComponent(camera.value.url)}&t=${Date.now()}`)
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const bitmap = await createImageBitmap(await resp.blob())
    previewHasFrame.value = true
    await nextTick()
    renderFrame(bitmap)
    bitmap.close()
    snapshotTimestamp.value = new Date().toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    snapshotError.value = 'No se pudo capturar el frame. Verifica que la camara este activa y el gateway corriendo.'
  } finally {
    snapshotLoading.value = false
  }
}

const toggleLive = () => {
  if (liveMode.value) {
    liveMode.value = false
    liveAbortCtrl?.abort()
    liveAbortCtrl = null
  } else {
    if (!camera.value.url) return
    liveMode.value        = true
    snapshotError.value   = null
    previewHasFrame.value = false
    liveAbortCtrl = new AbortController()
    runLiveStream(liveAbortCtrl.signal)
  }
}

const runLiveStream = async (signal) => {
  const url = `/api/gateway/edge/camera/stream?url=${encodeURIComponent(camera.value.url)}`
  try {
    const resp = await fetch(url, { signal })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const reader = resp.body.getReader()
    let buf = new Uint8Array(512 * 1024)
    let len = 0
    let rendering = false

    const find = (b1, b2, from = 0) => {
      for (let i = from; i < len - 1; i++) if (buf[i] === b1 && buf[i + 1] === b2) return i
      return -1
    }
    const consume = (start) => {
      if (start > 0 && start < len) { buf.copyWithin(0, start, len); len -= start }
      else if (start >= len) { len = 0 }
    }

    let pendingBlob = null
    const scheduleRender = () => {
      if (rendering || !pendingBlob || signal.aborted) return
      rendering = true
      const blob = pendingBlob
      pendingBlob = null
      requestAnimationFrame(async () => {
        try {
          const bm = await createImageBitmap(blob)
          if (!signal.aborted) { previewHasFrame.value = true; renderFrame(bm) }
          bm.close()
        } catch {}
        rendering = false
        if (pendingBlob) scheduleRender()
      })
    }

    while (!signal.aborted) {
      const { done, value } = await reader.read()
      if (done) break
      if (len + value.length > buf.length) {
        const nb = new Uint8Array(Math.max(buf.length * 2, len + value.length))
        nb.set(buf.subarray(0, len))
        buf = nb
      }
      buf.set(value, len)
      len += value.length

      let latestFrame = null
      while (len > 4) {
        const si = find(0xFF, 0xD8)
        if (si < 0) { len = len > 1 ? 1 : len; if (len === 1) buf[0] = buf[len - 1]; break }
        if (si > 0) consume(si)
        const ei = find(0xFF, 0xD9, 2)
        if (ei < 0) break
        const end = ei + 2
        latestFrame = buf.slice(0, end)
        consume(end)
      }
      if (latestFrame) {
        pendingBlob = new Blob([latestFrame], { type: 'image/jpeg' })
        scheduleRender()
      }
      if (len > 1024 * 1024) { buf[0] = buf[len - 1]; len = 1 }
    }
  } catch (e) {
    if (!signal.aborted) {
      liveMode.value      = false
      snapshotError.value = 'Error en el stream. Verifica que la camara este activa.'
    }
  }
}

watch(roi, drawRoiOverlayCanvas, { deep: true })

watch(activeSection, (val) => {
  if (val === 'preview') nextTick(drawRoiOverlayCanvas)  // legacy: kept for safety
})

const checkCameraHealth = async () => {
  if (!camera.value.url) return
  cameraChecking.value = true
  cameraHealth.value = null
  try {
    cameraHealth.value = await cameraService.checkHealth(camera.value.url)
  } catch (err) {
    cameraHealth.value = err.response?.data || {
      status: 'tcp_unreachable',
      url_masked: camera.value.url.replace(/:\/\/[^@]+@/, '://****:****@'),
      tcp_reachable: false,
      rtsp_ok: false,
      rtsp_message: `Error: ${err.message}`,
      checked_at: new Date().toISOString(),
    }
  } finally {
    cameraChecking.value = false
  }
}

const loadConfig = async () => {
  try {
    const [data, sysDefaults] = await Promise.all([
      configService.getConfig(lineaId.value),
      configService.getSystemDefaults(),
    ])
    const legacyMode = data.mode !== 'textil'
    config.value = {
      thresholds: { edge: 0.4, color: 0.6, flow: 0.5, dy: 5.0, beige: 0.35, high: 0.7, low: 0.3, presencia_threshold: 0.05, presencia_scale: 0.005, presencia_hold: 750, presencia_window: 375, presencia_pixel_threshold: 8, ...data.thresholds },
      fsm: { n_frames: 3, cooldown: 8, exit_frames: 5, max_wait_exit_frames: 50, min_rearm_s: 6.0, ...data.fsm },
      mode: 'textil',
      config_version: data.config_version || 0,
      roi_presencia: (() => {
        const rp = data.roi_presencia
        if (!rp || typeof rp !== 'object') return null
        if (Array.isArray(rp)) return rp.length >= 4 ? rp.slice(0, 4) : null
        return rp.width > 0 && rp.height > 0 ? [rp.x, rp.y, rp.width, rp.height] : null
      })(),
    }
    roi.value = data.roi?.length >= 4 ? [...data.roi.slice(0, 4), data.roi[4] ?? 0] : [120, 60, 320, 200, 0]
    if (data.camera) camera.value = { url: data.camera.url || '', fps: data.camera.fps || 25 }
    if (data.oee && !legacyMode) {
      oee.value = {
        line_name: data.oee.line_name || '',
        micro_stop_max_s: data.oee.micro_stop_max_s ?? 120,
        stop_max_s: data.oee.stop_max_s ?? 86400,
        snapshot_interval_s: data.oee.snapshot_interval_s ?? 1800,
        vel_unit: data.oee.vel_unit || 'uh',
        vel_nominal_us: data.oee.vel_nominal_us ?? 0.008333333,
      }
    } else {
      oee.value = {
        line_name: data.oee?.line_name || '',
        micro_stop_max_s: 120,
        stop_max_s: 86400,
        snapshot_interval_s: 1800,
        vel_unit: 'uh',
        vel_nominal_us: 0.008333333,
      }
    }
    // Calcular el valor de display según la unidad
    velNominalDisplay.value = oee.value.vel_unit === 'uh'
      ? parseFloat((oee.value.vel_nominal_us * 3600).toFixed(4))
      : oee.value.vel_nominal_us
    // Usar URL del _system como fallback si el dispositivo no tiene una propia
    const defaultCloudUrl = sysDefaults?.cloud?.url || ''
    if (data.cloud) cloud.value = { sync_interval_s: data.cloud.sync_interval_s > 0 ? data.cloud.sync_interval_s : 300, cloud_url: data.cloud.cloud_url || defaultCloudUrl, cloud_api_key: data.cloud.cloud_api_key || '' }
    if (data.tablet) tablet.value = { host: data.tablet.host || '', port: data.tablet.port ?? 8090, gatewayPort: data.tablet.gatewayPort ?? 8005 }
    if (data.linea_id !== undefined) deploy.value.linea_id = String(data.linea_id || '')
    loadedLineaId.value = lineaId.value  // registrar qué línea está cargada
    checkCloudHealth()
    if (!tablet.value.host) autoDetectHost()
  } catch {
    showMessage('No se pudo cargar la configuracion', 'error')
  }
}

const saveConfig = async () => {
  if (!lineaId.value) {
    showMessage('Selecciona una línea para guardar.', 'error')
    return
  }
  saving.value = true
  try {
    await configService.updateConfig({
      roi: roi.value,
      thresholds: config.value.thresholds,
      fsm: config.value.fsm,
      mode: 'textil',
      camera: camera.value.url ? camera.value : undefined,
      oee: oee.value,
      cloud: cloud.value,
      tablet: tablet.value,
    }, lineaId.value)

    if (activeSection.value === 'deploy') {
      await saveEnvToJetson()
    }

    lastSaved.value = new Date().toLocaleTimeString('es-MX')
    showMessage('Configuración guardada.', 'success')
    // Si era una línea nueva, limpiar el flag ?new=1 de la URL
    if (route.query.new === '1') {
      router.replace('/config/' + lineaId.value)
    }
    await loadConfig()
    await loadLines()
  } catch {
    showMessage('Error al guardar', 'error')
  } finally {
    saving.value = false
  }
}

const startCalibration = async () => {
  calibrating.value = true
  try {
    await gatewayService.startCalibration()
    showMessage('Calibracion iniciada (30 muestras).', 'info')
  } catch {
    showMessage('Error al calibrar', 'error')
  } finally {
    setTimeout(() => { calibrating.value = false }, 3000)
  }
}

const showMessage = (text, type) => {
  message.value = text
  messageClass.value = { success: 'bg-green-900 text-green-200', error: 'bg-red-900 text-red-200', info: 'bg-blue-900 text-blue-200' }[type] || 'bg-slate-700 text-gray-200'
  setTimeout(() => { message.value = '' }, 5000)
}

onMounted(async () => {
  const paramId = route.params.lineaId
  if (!paramId) {
    // Sin linea_id → volver al listado principal
    router.replace('/config')
    return
  }
  lineaId.value = String(paramId)
  loadedLineaId.value = String(paramId)
  if (route.query.new === '1') {
    // Línea nueva: no existe en BD todavía. Pre-rellenar cloud URL desde defaults.
    try {
      const sysDefaults = await configService.getSystemDefaults()
      const defaultCloudUrl = sysDefaults?.cloud?.url || ''
      if (defaultCloudUrl) cloud.value = { ...cloud.value, cloud_url: defaultCloudUrl }
    } catch { /* ignorar */ }
  } else {
    await loadConfig()
  }
  await loadLines()
  await loadEnvFromJetson()

  // Polling del estado operacional cada 10s
  healthPollInterval = setInterval(() => {
    if (activeSection.value === 'cloud') checkCloudHealth()
  }, 10000)
})

let healthPollInterval = null
onUnmounted(() => {
  if (healthPollInterval) clearInterval(healthPollInterval)
  if (_labTimerHandle)    clearInterval(_labTimerHandle)
})

// ─── MODO LAB ────────────────────────────────────────────────────────────────

const labOpen      = ref(false)
const labGuideOpen = ref(false)
const labConfirmOpen = ref(false)
const labSaving    = ref(false)
const labSaved   = ref(false)

const labRoi   = ref([120, 60, 320, 200, 0])
const labRoiPresencia = ref([0, 0, 0, 0])
const labDragTarget = ref('count')  // 'count' | 'presence'
const labPresenceMotion = ref(false)  // estado local de presencia para etiquetas
const labMotionScore    = ref(0)      // micromovimiento promedio entre frames (Δpx)
const labPresenceScore  = ref(0)      // beige % en ROI de presencia

// ── Cronómetro de estado (sin BD) ────────────────────────────────────────────
const labStateStartMs  = ref(Date.now())   // ms en que empezó el estado actual
const labElapsedSec    = ref(0)            // segundos transcurridos en el estado
let   _labTimerHandle  = null
const _labStartTimer = () => {
  labStateStartMs.value = Date.now()
  labElapsedSec.value   = 0
  if (_labTimerHandle) clearInterval(_labTimerHandle)
  _labTimerHandle = setInterval(() => {
    labElapsedSec.value = Math.floor((Date.now() - labStateStartMs.value) / 1000)
  }, 1000)
}
const labFmtTime = (s) => {
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const ss = s % 60
  if (h > 0) return `${h}h ${String(m).padStart(2,'0')}m ${String(ss).padStart(2,'0')}s`
  if (m > 0) return `${m}m ${String(ss).padStart(2,'0')}s`
  return `${ss}s`
}
// Tipo: null = sin ROI, 'micro' = parada < 2 min, 'parada' = parada ≥ 2 min
const labStopType = computed(() => {
  if (labPresenceMotion.value) return null
  return labElapsedSec.value < 120 ? 'micro' : 'parada'
})
let   _labPrevPresencePx  = null       // compat (no usado en slow-window, se mantiene para reset)
let   _labPresenceBuffer  = []         // buffer circular para slow-window diff
let   _labPresenceN       = 0          // tamaño del ROI al buildear el buffer (reset si cambia)
const labThresh = ref({ edge: 0.4, color: 0.6, flow: 0.5, dy: 5.0, beige: 0.35, high: 0.7, low: 0.3, presencia_threshold: 0.05, presencia_scale: 0.005, presencia_hold: 750, presencia_window: 375, presencia_pixel_threshold: 8 })
const labFsm   = ref({ n_frames: 3, cooldown: 8, exit_frames: 5, max_wait_exit_frames: 50, min_rearm_s: 6.0 })

// Reinicia el cronómetro cada vez que cambia el estado produciendo/detenido
watch(labPresenceMotion, () => _labStartTimer())

// ── Contador de unidades (mini-FSM beige, espeja backend exacto) ────────
const labUnitCount   = ref(0)      // unidades detectadas en esta sesión
const labFsmState    = ref('IDLE')  // 'IDLE'|'DETECTING'|'WAIT_EXIT'|'COOLDOWN'
let   _labConfirmN   = 0           // frames consecutivos por encima del umbral alto
let   _labExitLowN   = 0           // frames consecutivos por DEBAJO del umbral bajo (WAIT_EXIT)
let   _labExitHighN  = 0           // frames consecutivos ENCIMA del umbral (reset histéresis)
let   _labWaitExitN  = 0           // total frames en WAIT_EXIT (timeout de seguridad)
let   _labCooldownN  = 0           // frames restantes de cooldown
let   _labLastCorteTime = 0        // ms timestamp of last emitted CORTE (for min_rearm_s gate)
const resetLabCounter = () => {
  labUnitCount.value = 0
  labFsmState.value  = 'IDLE'
  _labConfirmN  = 0
  _labExitLowN  = 0
  _labExitHighN = 0
  _labWaitExitN = 0
  _labCooldownN = 0
  _labLastCorteTime = 0
}

const labFrameRef    = ref(null)
const labOverlayRef  = ref(null)
const labHasFrame    = ref(false)
const labSnapshotLoading = ref(false)
const labSnapshotTs  = ref(null)
const labLiveMode    = ref(false)
const labFrameSize   = ref('')
let   labAbortCtrl   = null

// arrastre ROI
const labDragging      = ref(false)
let   labDragOrigin    = { x: 0, y: 0 }

const labDebugMode    = ref(false)
const labDebugChannel = ref('beige')

// Rango HSV para deteccion visual de beige (solo afecta el Lab, no el backend)
const labBeigeRange = ref({ hMin: 12, hMax: 50, sMin: 8, sMax: 160, vMin2: 100 })
const labScores       = ref({ beige: 0, edge: 0 })

const rgbToHsv = (r, g, b) => {
  const rn = r / 255, gn = g / 255, bn = b / 255
  const max = Math.max(rn, gn, bn), min = Math.min(rn, gn, bn), d = max - min
  let h = 0
  if (d > 0) {
    if (max === rn)      h = ((gn - bn) / d % 6) * 60
    else if (max === gn) h = ((bn - rn) / d + 2) * 60
    else                 h = ((rn - gn) / d + 4) * 60
    if (h < 0) h += 360
  }
  return [h, max === 0 ? 0 : (d / max) * 255, max * 255]
}

const labAnalyzeFrame = () => {
  const fc = labFrameRef.value
  const oc = labOverlayRef.value
  if (!fc || !oc || !labHasFrame.value) return
  if (oc.width !== fc.width || oc.height !== fc.height) {
    oc.width = fc.width; oc.height = fc.height
  }

  const [rx, ry, rw, rh, margin] = labRoi.value
  const crx  = Math.max(0, Math.min(rx, fc.width  - 1))
  const cry  = Math.max(0, Math.min(ry, fc.height - 1))
  const crw  = Math.max(1, Math.min(rw, fc.width  - crx))
  const activeH = Math.max(1, rh - (margin || 0))
  const crh  = Math.max(1, Math.min(activeH, fc.height - cry))

  const fctx = fc.getContext('2d', { willReadFrequently: true })
  const src  = fctx.getImageData(crx, cry, crw, crh)
  const { data } = src

  const ctx = oc.getContext('2d')
  ctx.clearRect(0, 0, oc.width, oc.height)

  // revelar área del ROI de presencia (sin overlay oscuro, igual que Produccion)
  const [px, py, pw, ph] = labRoiPresencia.value

  ctx.strokeStyle = '#fbbf24'
  ctx.lineWidth   = 2
  ctx.strokeRect(rx, ry, rw, rh)

  if ((margin || 0) > 0) {
    ctx.save()
    ctx.strokeStyle = 'rgba(251,191,36,0.5)'
    ctx.setLineDash([5, 4])
    ctx.lineWidth = 1.5
    ctx.beginPath()
    ctx.moveTo(rx, ry + rh - margin)
    ctx.lineTo(rx + rw, ry + rh - margin)
    ctx.stroke()
    ctx.fillStyle = 'rgba(249,115,22,0.18)'
    ctx.fillRect(rx, ry + rh - margin, rw, margin)
    ctx.restore()
  }

  const beigeOverlay = new Uint8ClampedArray(crw * crh * 4)
  const edgeOverlay  = new Uint8ClampedArray(crw * crh * 4)
  let beigePx = 0

  const showBeige = labDebugChannel.value === 'beige' || labDebugChannel.value === 'combined'
  const showEdge  = labDebugChannel.value === 'edge'  || labDebugChannel.value === 'combined'

  // Rango HSV configurable desde labBeigeRange (solo visual Lab)
  const { hMin, hMax, sMin, sMax, vMin2 } = labBeigeRange.value

  if (showBeige) {
    for (let i = 0; i < data.length; i += 4) {
      const [h, s, v] = rgbToHsv(data[i], data[i + 1], data[i + 2])
      if (h >= hMin && h <= hMax && s >= sMin && s <= sMax && v >= vMin2) {
        beigePx++
        beigeOverlay[i]     = 251
        beigeOverlay[i + 1] = 146
        beigeOverlay[i + 2] = 36
        beigeOverlay[i + 3] = 185
      }
    }
    labScores.value.beige = crw * crh > 0 ? Math.round(beigePx / (crw * crh) * 100) : 0
    const tc = document.createElement('canvas')
    tc.width = crw; tc.height = crh
    tc.getContext('2d').putImageData(new ImageData(beigeOverlay, crw, crh), 0, 0)
    ctx.drawImage(tc, crx, cry)
  }

  if (showEdge) {
    const gray = new Float32Array(crw * crh)
    for (let i = 0; i < data.length; i += 4)
      gray[i >> 2] = 0.299 * data[i] + 0.587 * data[i + 1] + 0.114 * data[i + 2]

    // edge slider: 0.1 → muy sensible, 0.8 → solo bordes fuertes
    const edgeThresh = Math.max(20, labThresh.value.edge * 280)
    let edgePx = 0

    for (let y = 1; y < crh - 1; y++) {
      for (let x = 1; x < crw - 1; x++) {
        const gx = -gray[(y-1)*crw+(x-1)] + gray[(y-1)*crw+(x+1)]
                 - 2*gray[y*crw+(x-1)]    + 2*gray[y*crw+(x+1)]
                 - gray[(y+1)*crw+(x-1)]  + gray[(y+1)*crw+(x+1)]
        const gy = -gray[(y-1)*crw+(x-1)] - 2*gray[(y-1)*crw+x] - gray[(y-1)*crw+(x+1)]
                 +  gray[(y+1)*crw+(x-1)] + 2*gray[(y+1)*crw+x] + gray[(y+1)*crw+(x+1)]
        if (Math.sqrt(gx * gx + gy * gy) > edgeThresh) {
          edgePx++
          const idx = (y * crw + x) * 4
          edgeOverlay[idx]     = 34
          edgeOverlay[idx + 1] = 211
          edgeOverlay[idx + 2] = 238
          edgeOverlay[idx + 3] = 220
        }
      }
    }
    labScores.value.edge = crw * crh > 0 ? Math.round(edgePx / (crw * crh) * 100) : 0
    const tc = document.createElement('canvas')
    tc.width = crw; tc.height = crh
    tc.getContext('2d').putImageData(new ImageData(edgeOverlay, crw, crh), 0, 0)
    ctx.drawImage(tc, crx, cry)
  }

  // borde ROI reactivo: verde si supera el umbral, rojo si no
  const beigePct     = labScores.value.beige / 100
  const threshTarget = labThresh.value.beige
  const aboveThresh  = beigePct >= threshTarget

  ctx.strokeStyle = aboveThresh ? '#22c55e' : '#ef4444'
  ctx.lineWidth   = aboveThresh ? 2.5 : 2
  ctx.strokeRect(rx, ry, rw, rh)

  // barra de score en la parte superior del ROI
  const barH   = 6
  const beigeW = Math.round((labScores.value.beige / 100) * rw)
  const edgeW  = Math.round(Math.min(labScores.value.edge * 4, 100) / 100 * rw)
  const threshX = rx + Math.round(threshTarget * rw)

  ctx.fillStyle = 'rgba(0,0,0,0.65)'
  ctx.fillRect(rx, ry - barH - 2, rw, barH + 2)

  if (showBeige) {
    ctx.fillStyle = aboveThresh ? 'rgba(34,197,94,0.9)' : 'rgba(251,146,36,0.85)'
    ctx.fillRect(rx, ry - barH - 1, beigeW, barH)
  }
  if (showEdge) {
    ctx.fillStyle = 'rgba(34,211,238,0.8)'
    ctx.fillRect(rx, showBeige ? ry - 4 : ry - barH - 1, edgeW, showBeige ? 3 : barH)
  }

  // linea vertical de umbral sobre la barra
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth   = 1.5
  ctx.setLineDash([3, 2])
  ctx.beginPath()
  ctx.moveTo(threshX, ry - barH - 3)
  ctx.lineTo(threshX, ry + 1)
  ctx.stroke()
  ctx.setLineDash([])

  // score text
  const scoreLabel = showBeige
    ? `${labScores.value.beige}% / min ${Math.round(threshTarget * 100)}%`
    : `bordes ${labScores.value.edge}%`
  ctx.fillStyle = aboveThresh ? '#86efac' : '#fca5a5'
  ctx.font      = 'bold 11px monospace'
  ctx.fillText(scoreLabel, rx + 4, ry + 14)

  // Etiqueta de estado en ROI de conteo cuando análisis está activo
  const isDetecting = labScores.value.beige >= Math.round(labThresh.value.beige * 100)
  if (isDetecting) {
    const countLabel = 'Detectando'
    ctx.font = 'bold 10px monospace'
    const cw = ctx.measureText(countLabel).width + 10
    ctx.fillStyle = 'rgba(30,58,138,0.85)'
    ctx.fillRect(rx + rw - cw - 2, ry + 2, cw, 16)
    ctx.fillStyle = '#93c5fd'
    ctx.fillText(countLabel, rx + rw - cw + 3, ry + 13)
  }

  // ── Mini-FSM de conteo (espeja exactamente el backend event_fsm.py) ──────
  // IDLE → DETECTING (beige >= high) → +1 unidad → WAIT_EXIT (espera que la
  // prenda SALGA: beige < low por exit_frames frames) → COOLDOWN → IDLE
  // Esto evita doble conteo cuando la prenda pasa lentamente.
  const nFrames     = labFsm.value.n_frames           ?? 3
  const cooldown    = labFsm.value.cooldown            ?? 8
  const exitFrames  = labFsm.value.exit_frames         ?? 5
  const maxWait     = labFsm.value.max_wait_exit_frames ?? 50
  const minRearmMs  = (labFsm.value.min_rearm_s ?? 6.0) * 1000
  // Umbral alto = beige slider (activación), bajo = la mitad (~salida)
  const highThr = labThresh.value.beige
  const lowThr  = labThresh.value.low ?? (highThr * 0.5)
  const aboveHigh = beigePct >= highThr
  const belowLow  = beigePct <  lowThr

  if (labFsmState.value === 'IDLE') {
    if (aboveHigh) {
      _labConfirmN = 1
      labFsmState.value = 'DETECTING'
    }
  } else if (labFsmState.value === 'DETECTING') {
    if (aboveHigh) {
      _labConfirmN++
      if (_labConfirmN >= nFrames) {
        const now = Date.now()
        if (minRearmMs <= 0 || (now - (_labLastCorteTime ?? 0)) >= minRearmMs) {
          labUnitCount.value++
          _labLastCorteTime = now
        }
        labFsmState.value = 'WAIT_EXIT'
        _labConfirmN  = 0
        _labExitLowN  = 0
        _labWaitExitN = 0
      }
    } else if (belowLow) {
      _labConfirmN = 0
      labFsmState.value = 'IDLE'
    }
  } else if (labFsmState.value === 'WAIT_EXIT') {
    _labWaitExitN++
    if (belowLow) {
      _labExitLowN++
      _labExitHighN = 0
    } else {
      _labExitHighN++
      if (_labExitHighN >= exitFrames) {
        _labExitLowN  = 0
        _labExitHighN = 0
      }
    }
    const exitConfirmed = _labExitLowN >= exitFrames
    // Timeout solo dispara si la prenda YA no está visible (beige < low).
    // Si sigue arriba del umbral, esperamos aunque se exceda max_wait.
    const timedOut = _labWaitExitN >= maxWait && belowLow
    if (exitConfirmed || timedOut) {
      labFsmState.value = 'COOLDOWN'
      _labCooldownN = cooldown
      _labExitLowN  = 0
      _labExitHighN = 0
      _labWaitExitN = 0
    }
  } else if (labFsmState.value === 'COOLDOWN') {
    _labCooldownN--
    if (_labCooldownN <= 0) {
      labFsmState.value = 'IDLE'
    }
  }

  // ROI de presencia: solo movimiento (slow-window diff), sin beige
  if (pw > 4 && ph > 4) {
    const motion = labPresenceMotion.value
    const motScore = labMotionScore.value  // Δpx promedio del slow-window diff
    ctx.save()
    ctx.strokeStyle = motion ? '#34d399' : '#f87171'
    ctx.lineWidth   = 2
    ctx.setLineDash([6, 3])
    ctx.strokeRect(px, py, pw, ph)
    ctx.setLineDash([])
    ctx.fillStyle = motion ? 'rgba(52,211,153,0.08)' : 'rgba(248,113,113,0.08)'
    ctx.fillRect(px, py, pw, ph)
    // Badge estado
    const label = motion ? 'En movimiento' : 'Detenida'
    const badgeColor = motion ? '#34d399' : '#f87171'
    ctx.font = 'bold 10px monospace'
    const textW = ctx.measureText(label).width + 10
    ctx.fillStyle = motion ? 'rgba(6,78,59,0.85)' : 'rgba(69,10,10,0.85)'
    ctx.fillRect(px + 2, py + 2, textW, 16)
    ctx.fillStyle = badgeColor
    ctx.fillText(label, px + 6, py + 13)
    // Barra de movimiento (slow-window Δpx — indiferente al color)
    const barW = Math.min(pw, Math.round(Math.min(motScore / 20, 1) * pw))
    ctx.fillStyle = 'rgba(0,0,0,0.6)'
    ctx.fillRect(px, py + ph - 8, pw, 8)
    ctx.fillStyle = motion ? 'rgba(52,211,153,0.9)' : 'rgba(248,113,113,0.7)'
    ctx.fillRect(px, py + ph - 7, barW, 6)
    ctx.fillStyle = motion ? '#86efac' : '#fca5a5'
    ctx.font = 'bold 9px monospace'
    ctx.fillText(`mv ${motScore.toFixed(1)}`, px + 4, py + ph - 1)
    ctx.restore()
  }
}

const labToggleDebug = () => {
  labDebugMode.value = !labDebugMode.value
  if (labDebugMode.value && labHasFrame.value) labAnalyzeFrame()
  else drawLabOverlay()
}

const labCanvasCoords = (e) => {
  const canvas = labOverlayRef.value
  if (!canvas) return { x: 0, y: 0 }
  const rect  = canvas.getBoundingClientRect()
  const scaleX = (labFrameRef.value?.width  || rect.width)  / rect.width
  const scaleY = (labFrameRef.value?.height || rect.height) / rect.height
  return {
    x: Math.round((e.clientX - rect.left) * scaleX),
    y: Math.round((e.clientY - rect.top)  * scaleY),
  }
}

const labDragStart = (e) => {
  labDragging.value = true
  labDragOrigin = labCanvasCoords(e)
}

const labDragMove = (e) => {
  if (!labDragging.value) return
  const { x, y } = labCanvasCoords(e)
  const x0 = Math.min(labDragOrigin.x, x)
  const y0 = Math.min(labDragOrigin.y, y)
  const w  = Math.abs(x - labDragOrigin.x)
  const h  = Math.abs(y - labDragOrigin.y)
  if (w > 4 && h > 4) {
    if (labDragTarget.value === 'presence') {
      labRoiPresencia.value[0] = x0
      labRoiPresencia.value[1] = y0
      labRoiPresencia.value[2] = w
      labRoiPresencia.value[3] = h
    } else {
      labRoi.value[0] = x0
      labRoi.value[1] = y0
      labRoi.value[2] = w
      labRoi.value[3] = h
    }
  }
  drawLabOverlay()
}

const labDragEnd = (e) => {
  if (!labDragging.value) return
  labDragging.value = false
  labDragMove(e)
}

const drawLabOverlay = () => {
  const oc = labOverlayRef.value
  const fc = labFrameRef.value
  if (!oc || !fc || !fc.width) return
  if (oc.width !== fc.width || oc.height !== fc.height) {
    oc.width  = fc.width
    oc.height = fc.height
  }
  const [rx, ry, rw, rh, margin] = labRoi.value
  const [px, py, pw, ph] = labRoiPresencia.value
  const ctx = oc.getContext('2d')
  ctx.clearRect(0, 0, oc.width, oc.height)

  // ROI de conteo — borde amarillo
  ctx.strokeStyle = '#fbbf24'
  ctx.lineWidth   = 2
  ctx.strokeRect(rx, ry, rw, rh)

  if (margin > 0) {
    ctx.save()
    ctx.strokeStyle = 'rgba(251,191,36,0.5)'
    ctx.setLineDash([5, 4])
    ctx.lineWidth = 1.5
    ctx.beginPath()
    ctx.moveTo(rx, ry + rh - margin)
    ctx.lineTo(rx + rw, ry + rh - margin)
    ctx.stroke()
    ctx.fillStyle   = 'rgba(249,115,22,0.18)'
    ctx.fillRect(rx, ry + rh - margin, rw, margin)
    ctx.restore()
  }

  ctx.fillStyle = '#fbbf24'
  ctx.font      = 'bold 11px monospace'
  ctx.fillText(`${rw}\u00d7${rh}`, rx + 4, ry + 14)

  // ROI de presencia — borde verde esmeralda (si tiene dimensiones válidas)
  if (pw > 4 && ph > 4) {
    const motion = labPresenceMotion.value
    ctx.save()
    ctx.strokeStyle = motion ? '#34d399' : '#f87171'
    ctx.lineWidth   = 2
    ctx.setLineDash([6, 3])
    ctx.strokeRect(px, py, pw, ph)
    ctx.setLineDash([])
    ctx.fillStyle = motion ? 'rgba(52,211,153,0.08)' : 'rgba(248,113,113,0.08)'
    ctx.fillRect(px, py, pw, ph)
    // Badge de estado
    const label = motion ? 'En movimiento' : 'Detenida'
    const badgeColor = motion ? '#34d399' : '#f87171'
    const textW = ctx.measureText(label).width + 10
    ctx.font = 'bold 10px monospace'
    ctx.fillStyle = motion ? 'rgba(6,78,59,0.85)' : 'rgba(69,10,10,0.85)'
    ctx.fillRect(px + 2, py + 2, textW, 16)
    ctx.fillStyle = badgeColor
    ctx.fillText(label, px + 6, py + 13)
    ctx.restore()
  }
}

const _labUpdatePresence = (fc) => {
  const [px, py, pw, ph] = labRoiPresencia.value
  if (!fc || pw < 8 || ph < 8) { labPresenceMotion.value = false; return }

  // Clampear ROI a los límites del frame.
  const x0 = Math.max(0, px), y0 = Math.max(0, py)
  const x1 = Math.min(fc.width,  px + pw)
  const y1 = Math.min(fc.height, py + ph)
  const cw = x1 - x0, ch = y1 - y0
  if (cw < 8 || ch < 8) { labPresenceMotion.value = false; return }

  try {
    const ctx  = fc.getContext('2d', { willReadFrequently: true })
    const data = ctx.getImageData(x0, y0, cw, ch).data
    const n    = cw * ch

    // Grayscale del ROI (cualquier color, sin depéndencia de beige)
    const gray = new Uint8Array(n)
    for (let i = 0; i < n; i++)
      gray[i] = (data[i * 4] + data[i * 4 + 1] + data[i * 4 + 2]) / 3

    // ── Slow-window diff (30 frames ≈ 2.5 s en modo Lab) ─────────────────
    // Espeja la lógica del backend PresenceDetector:
    // compara frame actual vs frame de N frames atrás en vez del inmediatamente anterior.
    // Detecta movimiento lento: banda de tela que se desplaza <1 px/frame.
    // Lee parámetros en vivo desde los sliders (en Lab usamos ventana reducida para respuesta inmediata)
    const PIXEL_THR = labThresh.value?.presencia_pixel_threshold ?? 8
    const MOTION_AREA = labThresh.value?.presencia_scale ?? 0.005
    // Escala la ventana del backend (375 frames) al Lab (~30 fps lab ≈ 8% de la ventana real)
    const backendWindow = labThresh.value?.presencia_window ?? 375
    const _LAB_WINDOW = Math.max(5, Math.round(backendWindow * 0.08))

    if (_labPresenceN !== n) { _labPresenceBuffer = []; _labPresenceN = n }  // reset si cambia el ROI

    let motionPx = 0, diffSum = 0
    if (_labPresenceBuffer.length >= _LAB_WINDOW) {
      const ref = _labPresenceBuffer[0]
      for (let i = 0; i < n; i++) {
        const d = Math.abs(gray[i] - ref[i])
        diffSum += d
        if (d > PIXEL_THR) motionPx++
      }
      labMotionScore.value = diffSum / n
    }
    _labPresenceBuffer.push(gray)
    if (_labPresenceBuffer.length > _LAB_WINDOW) _labPresenceBuffer.shift()

    // Presencia: fracción de píxeles que superan el umbral
    const motionFraction = motionPx / n
    labPresenceMotion.value = motionFraction >= MOTION_AREA
    labPresenceScore.value  = Math.round(motionFraction * 100)
  } catch (_) {}
}

const labRenderFrame = (bitmap) => {
  const fc = labFrameRef.value
  if (!fc) return
  fc.width  = bitmap.width
  fc.height = bitmap.height
  fc.getContext('2d', { willReadFrequently: true }).drawImage(bitmap, 0, 0)
  labFrameSize.value = `${bitmap.width} \u00d7 ${bitmap.height}`
  _labUpdatePresence(fc)
  if (labDebugMode.value) labAnalyzeFrame()
  else drawLabOverlay()
}

const labCaptureSnapshot = async () => {
  if (!camera.value.url) return
  if (labLiveMode.value) { labLiveMode.value = false; labAbortCtrl?.abort(); labAbortCtrl = null }
  labSnapshotLoading.value = true
  try {
    const resp = await fetch(`/api/gateway/edge/camera/snapshot?url=${encodeURIComponent(camera.value.url)}&t=${Date.now()}`)
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const bitmap = await createImageBitmap(await resp.blob())
    labHasFrame.value = true
    await nextTick()
    labRenderFrame(bitmap)
    bitmap.close()
    labSnapshotTs.value = new Date().toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    labHasFrame.value = false
  } finally {
    labSnapshotLoading.value = false
  }
}

const labToggleLive = () => {
  if (labLiveMode.value) {
    labLiveMode.value = false
    labAbortCtrl?.abort()
    labAbortCtrl = null
  } else {
    if (!camera.value.url) return
    labLiveMode.value = true
    labHasFrame.value = false
    labAbortCtrl = new AbortController()
    labRunStream(labAbortCtrl.signal)
  }
}

const labRunStream = async (signal) => {
  const url = `/api/gateway/edge/camera/stream?url=${encodeURIComponent(camera.value.url)}`
  try {
    const resp = await fetch(url, { signal })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const reader  = resp.body.getReader()
    let buf       = new Uint8Array(512 * 1024)
    let len       = 0
    let rendering = false

    const find = (b1, b2, from = 0) => {
      for (let i = from; i < len - 1; i++) if (buf[i] === b1 && buf[i + 1] === b2) return i
      return -1
    }
    const consume = (start) => {
      if (start > 0 && start < len) { buf.copyWithin(0, start, len); len -= start }
      else if (start >= len) { len = 0 }
    }

    let pendingBlob = null
    const scheduleRender = () => {
      if (rendering || !pendingBlob || signal.aborted) return
      rendering = true
      const blob = pendingBlob; pendingBlob = null
      requestAnimationFrame(async () => {
        try {
          const bm = await createImageBitmap(blob)
          if (!signal.aborted) { labHasFrame.value = true; labRenderFrame(bm) }
          bm.close()
        } catch {}
        rendering = false
        if (pendingBlob) scheduleRender()
      })
    }

    while (!signal.aborted) {
      const { done, value } = await reader.read()
      if (done) break
      if (len + value.length > buf.length) {
        const nb = new Uint8Array(Math.max(buf.length * 2, len + value.length))
        nb.set(buf.subarray(0, len)); buf = nb
      }
      buf.set(value, len); len += value.length

      let latestFrame = null
      while (len > 4) {
        const si = find(0xFF, 0xD8)
        if (si < 0) { len = len > 1 ? 1 : len; if (len === 1) buf[0] = buf[len - 1]; break }
        if (si > 0) consume(si)
        const ei = find(0xFF, 0xD9, 2)
        if (ei < 0) break
        latestFrame = buf.slice(0, ei + 2)
        consume(ei + 2)
      }
      if (latestFrame) { pendingBlob = new Blob([latestFrame], { type: 'image/jpeg' }); scheduleRender() }
      if (len > 1024 * 1024) { buf[0] = buf[len - 1]; len = 1 }
    }
  } catch (e) {
    if (!signal.aborted) { labLiveMode.value = false }
  }
}

const openLab = () => {
  labRoi.value          = [...roi.value]
  const rp = config.value.roi_presencia
  labRoiPresencia.value = rp && rp.length >= 4
    ? [rp[0], rp[1], rp[2], rp[3]]
    : [0, 0, 0, 0]
  labDragTarget.value   = 'count'
  labPresenceMotion.value = false
  labMotionScore.value    = 0
  labPresenceScore.value  = 0
  _labPrevPresencePx    = null
  _labPresenceBuffer    = []
  _labPresenceN         = 0
  labThresh.value       = { ...config.value.thresholds }
  labFsm.value          = { ...config.value.fsm }
  const br = config.value.thresholds?.beige_range
  if (br) {
    labBeigeRange.value = {
      hMin:  br.h_min ?? labBeigeRange.value.hMin,
      hMax:  br.h_max ?? labBeigeRange.value.hMax,
      sMin:  br.s_min ?? labBeigeRange.value.sMin,
      sMax:  br.s_max ?? labBeigeRange.value.sMax,
      vMin2: br.v_min ?? labBeigeRange.value.vMin2,
    }
  }
  labHasFrame.value     = false
  labFrameSize.value    = ''
  labDebugMode.value    = true
  labDebugChannel.value = 'beige'
  labOpen.value         = true
  nextTick(() => {
    labLiveMode.value = true
    labAbortCtrl = new AbortController()
    labRunStream(labAbortCtrl.signal)
  })
}

const closeLab = () => {
  labLiveMode.value = false
  labAbortCtrl?.abort()
  labAbortCtrl = null
  labOpen.value = false
}

watch([labThresh, labRoi, labRoiPresencia], () => {
  if (labDebugMode.value && labHasFrame.value) labAnalyzeFrame()
  else if (labHasFrame.value) drawLabOverlay()
}, { deep: true })

watch(labBeigeRange, () => {
  if (labDebugMode.value && labHasFrame.value) labAnalyzeFrame()
}, { deep: true })

watch(labDebugChannel, () => {
  if (labDebugMode.value && labHasFrame.value) labAnalyzeFrame()
})

const labRecalibrar = async () => {
  // Reinicia el buffer de slow-window tanto en el backend como en el Lab local
  _labPresenceBuffer = []
  _labPresenceN = 0
  labMotionScore.value = 0
  try {
    await configService.resetPresenceBackground()
    showMessage('Buffer de movimiento reiniciado', 'success')
  } catch {
    showMessage('Error al reiniciar el detector', 'error')
  }
}

const saveLabConfig = async () => {
  roi.value    = [...labRoi.value]
  const rp = labRoiPresencia.value
  config.value.roi_presencia = (rp[2] > 4 && rp[3] > 4) ? [...rp] : null
  config.value.thresholds = {
    ...labThresh.value,
    beige_range: {
      h_min: labBeigeRange.value.hMin,
      h_max: labBeigeRange.value.hMax,
      s_min: labBeigeRange.value.sMin,
      s_max: labBeigeRange.value.sMax,
      v_min: labBeigeRange.value.vMin2,
    },
  }
  config.value.fsm        = { ...labFsm.value }
  labSaving.value = true
  try {
    await configService.updateConfig({
      roi: roi.value,
      roi_presencia: config.value.roi_presencia,
      thresholds: config.value.thresholds,
      fsm: config.value.fsm,
      mode: config.value.mode,
      camera: camera.value.url ? camera.value : undefined,
      oee: oee.value,
      cloud: cloud.value,
      tablet: tablet.value,
    }, lineaId.value)
    lastSaved.value = new Date().toLocaleTimeString('es-MX')
    labSaved.value = true
    setTimeout(() => { labSaved.value = false }, 3000)
  } catch {
    showMessage('Error al guardar desde Modo Lab', 'error')
  } finally {
    labSaving.value = false
  }
}
</script>

<style scoped>
.card { @apply bg-slate-800 rounded-lg shadow p-6; }
.section-title { @apply text-base font-medium text-white mb-4; }
.input-field { @apply block w-full rounded-md border-slate-600 bg-slate-700 text-white shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm px-3 py-2; }
.label-field { @apply block text-sm font-medium text-gray-300; }
.btn-primary { @apply px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed; }
.btn-secondary { @apply px-4 py-2 bg-slate-600 text-white rounded-md hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-slate-500 disabled:opacity-50 disabled:cursor-not-allowed; }
.fade-enter-active, .fade-leave-active { transition: opacity .3s ease, transform .3s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translateY(-4px); }
.lab-fade-enter-active, .lab-fade-leave-active { transition: opacity .2s ease, transform .2s ease; }
.lab-fade-enter-from, .lab-fade-leave-to { opacity: 0; transform: scale(0.98); }
.guide-slide-enter-active, .guide-slide-leave-active { transition: transform .25s ease, opacity .2s ease; }
.guide-slide-enter-from, .guide-slide-leave-to { transform: translateX(100%); opacity: 0; }
</style>

