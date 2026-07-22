<template>
  <svg viewBox="0 0 78 120" :width="width" :height="height" xmlns="http://www.w3.org/2000/svg">
    <rect x="6" y="6" width="66" height="108" rx="5" fill="#0f172a" stroke="#334155" stroke-width="1.5"/>
    <rect x="10" y="6" width="58" height="13" rx="3" fill="#1e293b" stroke="#475569" stroke-width="1"/>
    <rect x="14" y="7" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="18.5" cy="12" r="2.5" fill="#475569"/>
    <rect x="26" y="7" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="30.5" cy="12" r="2.5" fill="#475569"/>
    <rect x="44" y="7" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="48.5" cy="12" r="2.5" fill="#475569"/>
    <rect x="55" y="7" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="59.5" cy="12" r="2.5" fill="#475569"/>
    <text x="18.5" y="6.5" text-anchor="middle" fill="#64748b" font-size="3.5" font-family="sans-serif">L</text>
    <text x="30.5" y="6.5" text-anchor="middle" fill="#64748b" font-size="3.5" font-family="sans-serif">N</text>
    <text x="48.5" y="6.5" text-anchor="middle" fill="#64748b" font-size="3.5" font-family="sans-serif">L'</text>
    <text x="59.5" y="6.5" text-anchor="middle" fill="#64748b" font-size="3.5" font-family="sans-serif">N'</text>
    <rect x="10" y="23" width="58" height="26" rx="3" fill="#020d1a" stroke="#1e3a5f" stroke-width="1"/>
    <rect x="12" y="25" width="54" height="22" rx="2" fill="#0a1f35"/>
    <text x="39" y="37" text-anchor="middle" fill="#4ade80" font-size="9.5" font-family="'Courier New',monospace" letter-spacing="1.5">00.00</text>
    <text x="57" y="44" text-anchor="middle" fill="#22d3ee" font-size="5" font-family="sans-serif">kWh</text>
    <circle cx="17" cy="37" r="3.5" fill="#14532d"/>
    <circle cx="17" cy="37" r="2" fill="#4ade80"/>
    <rect x="10" y="53" width="58" height="1.5" rx="0.75" fill="#1e3a5f"/>
    <rect x="10" y="58" width="58" height="16" rx="2" fill="#0c1a2e" stroke="#1e3a5f" stroke-width="0.75"/>
    <text x="39" y="67" text-anchor="middle" fill="#3b82f6" font-size="7" font-family="sans-serif" font-weight="700">MC-60</text>
    <text x="39" y="72.5" text-anchor="middle" fill="#475569" font-size="4.5" font-family="sans-serif">ENERGY METER</text>
    <rect x="10" y="78" width="58" height="20" rx="2" fill="#0c1a2e" stroke="#1e3a5f" stroke-width="0.75"/>
    <rect x="14" y="81" width="14" height="13" rx="2" fill="#172554" stroke="#3b82f6" stroke-width="0.5"/>
    <text x="21" y="89.5" text-anchor="middle" fill="#93c5fd" font-size="5.5" font-family="sans-serif" font-weight="600">A</text>
    <rect x="32" y="81" width="14" height="13" rx="2" fill="#172554" stroke="#3b82f6" stroke-width="0.5"/>
    <text x="39" y="89.5" text-anchor="middle" fill="#93c5fd" font-size="5.5" font-family="sans-serif" font-weight="600">V</text>
    <rect x="50" y="81" width="14" height="13" rx="2" fill="#172554" stroke="#3b82f6" stroke-width="0.5"/>
    <text x="57" y="89.5" text-anchor="middle" fill="#93c5fd" font-size="5.5" font-family="sans-serif" font-weight="600">W</text>
    <rect x="10" y="101" width="58" height="13" rx="3" fill="#1e293b" stroke="#475569" stroke-width="1"/>
    <rect x="14" y="102" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="18.5" cy="107" r="2.5" fill="#475569"/>
    <rect x="26" y="102" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="30.5" cy="107" r="2.5" fill="#475569"/>
    <rect x="44" y="102" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="48.5" cy="107" r="2.5" fill="#475569"/>
    <rect x="55" y="102" width="9" height="10" rx="1.5" fill="#334155" stroke="#64748b" stroke-width="0.75"/>
    <circle cx="59.5" cy="107" r="2.5" fill="#475569"/>
    <rect x="6" y="111" width="66" height="3" rx="1.5" fill="#1e3a5f"/>
  </svg>
</template>

<script setup>
defineProps({
  width:  { type: [Number, String], default: 78 },
  height: { type: [Number, String], default: 120 },
})
</script>
