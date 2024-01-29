<script setup lang="ts">
import { useMessagesStore } from "@/stores/messages";
import AlertError from './AlertError.vue';
import { storeToRefs } from 'pinia'
import { onMounted, computed, ref } from 'vue';
import type { PropType, Ref } from 'vue';

const store = useMessagesStore()
const { status, runNumber, errorStates, errorRunNumber } = storeToRefs(store)
</script>

<template>
  <div data-testid="status-container" class="bg-slate-300 flex flex-col gap-2 w-fit m-2 p-2 px-4 rounded-md">
  <AlertError v-if="errorStates" :text="`Error: ${errorStates}`" />
  <AlertError v-if="errorRunNumber" :text="`Error: ${errorRunNumber}`" />
    <h2 class="text-xl font-bold">Status</h2>
    <div class="flex-col gap-2">
      <p><span class="font-bold">State:</span> {{ status }}</p>
      <p><span class="font-bold">Run number:</span> {{ runNumber }}</p>
    </div>
  </div>
</template>
