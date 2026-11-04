<script setup lang="ts">
import CheckConnection from '@/components/CheckConnection.vue';
import CheckDisabled from '@/components/CheckDisabled.vue';
import LogMessages from '@/components/LogMessages.vue'
import RunControl from '@/components/RunControl.vue';
import RunStatistics from '@/components/RunStatistics.vue';
import Status from '@/components/Status.vue';
import RateHistory from '@/components/RateHistory.vue';
import { computed } from 'vue';
import { useGDCStore } from '@/stores/gdc';
import { useLDCStore } from '@/stores/ldc';
const gdcStore = useGDCStore();
const ldcStore = useLDCStore();
const compactOverview = computed(() => {
  const count = gdcStore.gdcs.length + ldcStore.ldcs.length;
  return count > 0 && count <= 4;
});
</script>

<template>
  <main>
    <CheckConnection />
    <CheckDisabled />
    <div class="flex gap-2 justify-center">
      <RunControl />
      <Status />
    </div>
    <div class="run-overview" :class="{ 'run-overview--compact': compactOverview }" data-testid="run-overview">
      <div class="statistics-pane"><RunStatistics :compact="compactOverview" /></div>
      <RateHistory />
    </div>
    <LogMessages />
  </main>
</template>

<style scoped>
.run-overview { display: grid; gap: 16px; max-width: 1536px; margin: 16px auto; padding: 0 16px; }
.statistics-pane { min-width: 0; overflow-x: auto; }
.run-overview :deep(.rate-history) { min-width: 0; width: 100%; max-width: none; margin: 0; }
@media (min-width: 1200px) {
  .run-overview--compact { grid-template-columns: max-content minmax(0, 1fr); align-items: start; }
  .run-overview--compact :deep(.rate-chart) { height: 300px; }
}
</style>
