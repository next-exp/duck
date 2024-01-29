<script setup lang="ts">
import CheckConnection from '@/components/CheckConnection.vue';
import CheckDisabled from '@/components/CheckDisabled.vue';
import LogMessages from '@/components/LogMessages.vue'
import RunControl from '@/components/RunControl.vue';
import RunStatistics from '@/components/RunStatistics.vue';
import Status from '@/components/Status.vue';
</script>

<template>
  <main>
    <CheckConnection />
    <CheckDisabled />
    <div class="flex gap-2 justify-center">
      <RunControl />
      <Status />
    </div>
    <RunStatistics />
    <LogMessages />
  </main>
</template>
