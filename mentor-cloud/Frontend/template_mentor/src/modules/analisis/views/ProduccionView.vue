<script setup>
import Card from '@/shared/components/ui/Card.vue'
</script>

<template>
  <div class="produccion-view">
    <div class="view-header">
      <div>
        <h1 class="view-title">Análisis Producción</h1>
        <p class="view-subtitle">Monitoreo de producción y eficiencia</p>
      </div>
    </div>

    <Card>
      <div class="empty-state">
        <p>Vista de Análisis Producción</p>
        <p class="text-sm text-gray-500 mt-2">Implementación pendiente</p>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.produccion-view {
  @apply space-y-6;
}

.view-header {
  @apply flex items-start justify-between;
}

.view-title {
  @apply text-2xl font-bold text-gray-900;
}

.view-subtitle {
  @apply text-gray-600 mt-1;
}

.empty-state {
  @apply text-center py-12;
}
</style>
