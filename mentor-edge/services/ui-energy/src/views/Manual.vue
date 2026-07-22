<template>
  <div class="space-y-8 max-w-3xl">

    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold text-white">Manual de uso</h1>
      <p class="mt-1 text-sm text-gray-400">Guia de referencia para el operador — Mentor Energy v1</p>
    </div>

    <!-- TOC -->
    <nav class="rounded-xl border border-slate-700/60 bg-slate-800/40 px-5 py-4">
      <p class="text-[10px] uppercase tracking-widest text-gray-500 mb-3">Contenido</p>
      <ol class="space-y-1 text-sm">
        <li v-for="s in sections" :key="s.id">
          <a :href="'#' + s.id"
            class="flex items-center gap-2 text-gray-400 hover:text-white transition-colors">
            <span class="text-[10px] font-mono text-gray-600 w-5 shrink-0">{{ s.num }}</span>
            {{ s.title }}
          </a>
        </li>
      </ol>
    </nav>

    <!-- 1. Sistema -->
    <section id="sistema" class="space-y-3">
      <h2 class="section-heading">1. Descripcion del sistema</h2>
      <p class="body-text">
        Mentor Energy es un sistema de monitoreo electrico embebido que corre en una
        Raspberry Pi. Se conecta a medidores MC-60 a traves de un bus RS-485 y almacena
        snapshots en una base de datos PostgreSQL local. La interfaz web se accede desde
        cualquier navegador en la misma red local.
      </p>
      <div class="info-box">
        <span class="label">Direccion por defecto</span>
        <code class="font-mono text-blue-300">http://192.168.0.134:8087</code>
      </div>
      <p class="body-text">
        Los datos se actualizan segun el intervalo configurado en cada medidor (tipicamente
        300 segundos). La pagina Monitor se refresca automaticamente cada 10 segundos para
        mostrar el estado mas reciente.
      </p>
    </section>

    <!-- 2. Dashboard -->
    <section id="dashboard" class="space-y-3">
      <h2 class="section-heading">2. Dashboard</h2>
      <p class="body-text">
        Vista de resumen con los indicadores clave de todos los medidores registrados.
        Muestra el consumo acumulado, la potencia activa total y el estado de conexion
        de cada dispositivo.
      </p>
      <p class="body-text">
        Los indicadores se colorean segun el estado: <span class="text-green-400">verde</span>
        indica medidor en linea con datos recientes, <span class="text-slate-400">gris</span>
        indica sin datos en el ultimo ciclo.
      </p>
    </section>

    <!-- 3. Monitor -->
    <section id="monitor" class="space-y-3">
      <h2 class="section-heading">3. Monitor</h2>
      <p class="body-text">
        Pagina de supervision en tiempo real. Permite ver el estado actual de cada medidor
        y su historial de las ultimas horas.
      </p>

      <h3 class="subsection-heading">3.1 Selector de medidor</h3>
      <p class="body-text">
        La fila de botones en la parte superior muestra todos los medidores configurados.
        El punto verde indica que el medidor esta en linea (recibio datos en el ultimo
        intervalo). El numero en el badge es el UID Modbus del dispositivo.
        Haz clic en un boton para ver el detalle de ese medidor.
      </p>

      <h3 class="subsection-heading">3.2 Panel de detalle</h3>
      <p class="body-text">
        Al seleccionar un medidor se despliega el panel con 6 metricas principales:
      </p>
      <table class="metric-table">
        <thead>
          <tr>
            <th>Metrica</th>
            <th>Descripcion</th>
            <th>Unidad</th>
          </tr>
        </thead>
        <tbody>
          <tr><td>Tension</td><td>Tension promedio fase-neutro (Vavg)</td><td>V</td></tr>
          <tr><td>Corriente</td><td>Corriente promedio de fase (Iavg)</td><td>A</td></tr>
          <tr><td>Potencia</td><td>Potencia activa total (P)</td><td>W / kW</td></tr>
          <tr><td>Energia</td><td>Energia activa importada acumulada (EPImp)</td><td>Wh / kWh</td></tr>
          <tr><td>Frecuencia</td><td>Frecuencia de red (Freq)</td><td>Hz</td></tr>
          <tr><td>Factor P.</td><td>Factor de potencia total (PF)</td><td>cos φ</td></tr>
        </tbody>
      </table>

      <h3 class="subsection-heading">3.3 Graficas de historial</h3>
      <p class="body-text">
        Debajo de las metricas se muestran 3 graficas de serie temporal con las
        ultimas 48 lecturas almacenadas (~4 horas con intervalo de 5 min):
        Tension (amarillo), Corriente (cian) y Potencia activa (azul).
        Las graficas se cargan automaticamente al seleccionar un medidor.
      </p>

      <h3 class="subsection-heading">3.4 Resumen compacto</h3>
      <p class="body-text">
        Al pie de la pagina hay tarjetas compactas con la potencia y tension/corriente
        actuales de cada medidor. Hacer clic en una tarjeta selecciona ese medidor.
      </p>
    </section>

    <!-- 4. Lecturas -->
    <section id="lecturas" class="space-y-3">
      <h2 class="section-heading">4. Lecturas</h2>
      <p class="body-text">
        Muestra el ultimo snapshot completo de un medidor, con todos los registros
        agrupados por categoria. Cada fila incluye una mini grafica de tendencia
        (sparkline) basada en el historial reciente.
      </p>

      <h3 class="subsection-heading">4.1 Seleccion de medidor</h3>
      <p class="body-text">
        Usa los botones de la parte superior para cambiar de medidor. El sistema
        carga automaticamente el snapshot mas reciente y el historial al mismo tiempo.
        Los datos se refrescan cada 30 segundos.
      </p>

      <h3 class="subsection-heading">4.2 Grupos de registros</h3>
      <div class="overflow-x-auto">
        <table class="metric-table">
          <thead>
            <tr>
              <th>Grupo</th>
              <th>Registros</th>
              <th>Unidad</th>
            </tr>
          </thead>
          <tbody>
            <tr><td>Corrientes</td><td>Ia, Ib, Ic, In, Iavg</td><td>A</td></tr>
            <tr><td>Tension fase-neutro</td><td>Va, Vb, Vc, Vavg, V0</td><td>V</td></tr>
            <tr><td>Tension fase-fase</td><td>Vab, Vbc, Vca, VLavg</td><td>V</td></tr>
            <tr><td>Potencia activa</td><td>Pa, Pb, Pc, P</td><td>kW</td></tr>
            <tr><td>Potencia reactiva</td><td>Qa, Qb, Qc, Q</td><td>kVAR</td></tr>
            <tr><td>Potencia aparente</td><td>Sa, Sb, Sc, S</td><td>kVA</td></tr>
            <tr><td>Factor de potencia</td><td>PFa, PFb, PFc, PF</td><td>cos φ</td></tr>
            <tr><td>DPF (fundamental)</td><td>DPFa, DPFb, DPFc, DPF</td><td>—</td></tr>
            <tr><td>Frecuencia</td><td>FreqA, FreqB, FreqC, Freq</td><td>Hz</td></tr>
            <tr><td>Energia activa importada</td><td>EPAImp, EPBImp, EPCImp, EPImp</td><td>kWh</td></tr>
            <tr><td>Energia activa exportada</td><td>EPAExp, EPBExp, EPCExp, EPExp</td><td>kWh</td></tr>
            <tr><td>Energia reactiva import.</td><td>EQAImp, EQBImp, EQCImp, EQImp</td><td>kVARh</td></tr>
            <tr><td>Energia reactiva export.</td><td>EQAExp, EQBExp, EQCExp, EQExp</td><td>kVARh</td></tr>
            <tr><td>Energia aparente</td><td>ESA, ESB, ESC, ES</td><td>kVAh</td></tr>
            <tr><td>THD corriente</td><td>THDia, THDib, THDic</td><td>%</td></tr>
            <tr><td>THD tension</td><td>THDua, THDub, THDuc</td><td>%</td></tr>
          </tbody>
        </table>
      </div>

      <div class="info-box">
        <span class="label">Nota sobre sparklines</span>
        Las mini graficas por fila muestran la tendencia del campo de historial disponible
        mas cercano. Los registros por fase (Ia, Ib, Ic) utilizan el promedio de fase (Iavg)
        ya que el historial solo almacena valores consolidados.
      </div>
    </section>

    <!-- 5. Configuracion -->
    <section id="configuracion" class="space-y-3">
      <h2 class="section-heading">5. Configuracion</h2>
      <p class="body-text">
        Permite agregar, editar y eliminar medidores del bus RS-485.
        Cada medidor requiere:
      </p>
      <ul class="list-disc list-inside space-y-1 text-sm text-gray-300 pl-2">
        <li><span class="font-mono text-gray-400">meter_id</span> — Identificador unico (ej. <code class="font-mono text-blue-300">medidor-01</code>)</li>
        <li><span class="font-mono text-gray-400">unit_id</span> — Direccion Modbus del dispositivo (1–247)</li>
        <li><span class="font-mono text-gray-400">interval_s</span> — Intervalo de lectura en segundos (minimo 60)</li>
      </ul>
      <div class="warning-box">
        Cambiar el <code class="font-mono">unit_id</code> o eliminar un medidor no borra
        el historial existente en la base de datos. Los snapshots anteriores se conservan.
      </div>
    </section>

    <!-- 6. Programar -->
    <section id="programar" class="space-y-3">
      <h2 class="section-heading">6. Programar medidor</h2>
      <p class="body-text">
        Permite escribir parametros de configuracion directamente al medidor via Modbus.
        Usar con precaucion: una escritura incorrecta puede alterar el comportamiento
        del dispositivo.
      </p>
      <div class="warning-box">
        Esta funcion es para tecnicos. No modificar registros de calibracion o de
        proteccion sin conocimiento del protocolo del medidor MC-60.
      </div>
    </section>

    <!-- 7. Auditoria -->
    <section id="auditoria" class="space-y-3">
      <h2 class="section-heading">7. Auditoria</h2>
      <p class="body-text">
        Registro de eventos del sistema: conexiones, desconexiones, errores de lectura
        y cambios de configuracion. Util para diagnosticar problemas de comunicacion
        con el bus RS-485.
      </p>
    </section>

    <!-- 8. Solucion de problemas -->
    <section id="problemas" class="space-y-3">
      <h2 class="section-heading">8. Solucion de problemas</h2>
      <div class="space-y-4">

        <div class="problem-item">
          <p class="problem-title">El medidor aparece como Offline</p>
          <p class="body-text">
            Verificar que el cable RS-485 este conectado correctamente (A+ / B−).
            Confirmar que el <code class="font-mono text-blue-300">unit_id</code> configurado
            coincide con la direccion programada fisicamente en el medidor.
            Revisar la seccion Auditoria para ver el error exacto.
          </p>
        </div>

        <div class="problem-item">
          <p class="problem-title">Las lecturas muestran guiones (—)</p>
          <p class="body-text">
            El medidor no tiene registrado ese campo en el ultimo snapshot, o el
            campo no existe para ese modelo. Esto es normal para registros opcionales
            como V0 o THD si el medidor no los soporta.
          </p>
        </div>

        <div class="problem-item">
          <p class="problem-title">Las sparklines no aparecen</p>
          <p class="body-text">
            El historial requiere al menos 2 lecturas almacenadas. Si el medidor
            se acaba de agregar o se reinicio la base de datos, esperar al menos
            2 ciclos de lectura para que aparezcan las graficas.
          </p>
        </div>

        <div class="problem-item">
          <p class="problem-title">La pagina no carga o muestra error de red</p>
          <p class="body-text">
            Verificar que los contenedores Docker esten corriendo en la Raspberry Pi:
          </p>
          <pre class="code-block">cd /home/py/mentor-energy
/usr/local/bin/docker-compose ps</pre>
          <p class="body-text mt-2">
            Si algun contenedor aparece como <code class="font-mono text-red-400">Exit</code>,
            reiniciarlo con:
          </p>
          <pre class="code-block">/usr/local/bin/docker-compose up -d</pre>
        </div>

      </div>
    </section>

  </div>
</template>

<script setup>
const sections = [
  { id: 'sistema',       num: '01', title: 'Descripcion del sistema' },
  { id: 'dashboard',     num: '02', title: 'Dashboard' },
  { id: 'monitor',       num: '03', title: 'Monitor' },
  { id: 'lecturas',      num: '04', title: 'Lecturas' },
  { id: 'configuracion', num: '05', title: 'Configuracion' },
  { id: 'programar',     num: '06', title: 'Programar medidor' },
  { id: 'auditoria',     num: '07', title: 'Auditoria' },
  { id: 'problemas',     num: '08', title: 'Solucion de problemas' },
]
</script>

<style scoped>
.section-heading {
  @apply text-base font-semibold text-white border-b border-slate-700/60 pb-2;
}
.subsection-heading {
  @apply text-sm font-semibold text-gray-300 mt-4;
}
.body-text {
  @apply text-sm text-gray-400 leading-relaxed;
}
.info-box {
  @apply rounded-lg border border-blue-900/40 bg-blue-950/20 px-4 py-3 text-sm text-blue-300 flex flex-col gap-0.5;
}
.info-box .label {
  @apply text-[10px] uppercase tracking-wider text-blue-500 font-medium;
}
.warning-box {
  @apply rounded-lg border border-amber-900/40 bg-amber-950/20 px-4 py-3 text-sm text-amber-300;
}
.metric-table {
  @apply w-full text-sm border-collapse;
}
.metric-table th {
  @apply text-left text-[10px] uppercase tracking-wider text-gray-500 pb-2 border-b border-slate-700/40 font-medium;
}
.metric-table td {
  @apply py-1.5 text-gray-300 border-b border-slate-800/60 text-xs;
}
.metric-table td:first-child {
  @apply font-medium text-gray-200 w-44;
}
.metric-table td:last-child {
  @apply font-mono text-gray-500 text-[11px];
}
.problem-item {
  @apply rounded-lg border border-slate-700/40 bg-slate-800/30 px-4 py-3 space-y-2;
}
.problem-title {
  @apply text-sm font-semibold text-white;
}
.code-block {
  @apply font-mono text-xs text-green-300 bg-slate-900/60 rounded px-3 py-2 border border-slate-700/40 whitespace-pre-wrap;
}
</style>
