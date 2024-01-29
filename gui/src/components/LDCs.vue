<script setup lang="ts">
import { ref, type Ref } from 'vue';
import LDC from './LDC.vue';
import { useLDCStore } from '../stores/ldc'
import { storeToRefs } from 'pinia'
import { PlusIcon } from '@heroicons/vue/24/outline'
import AlertError from './AlertError.vue';

const store = useLDCStore()
const { ldcs, errorLDCs } = storeToRefs(store)

const showNew: Ref<boolean> = ref(false)
</script>

<template>
  <div class="bg-slate-300 rounded-md p-2">
    <AlertError v-if="errorLDCs" :text="`Error loading LDCS: ${errorLDCs}`" />
    <LDC v-for="ldc in ldcs" :ldc="ldc" :key="ldc.id" />
    <button v-if="!showNew" @click="showNew = !showNew" class="btn btn-info w-auto text-xl">
      <PlusIcon class="text-red-900 w-6" />
      <p>Add LDC</p>
    </button>
    <LDC v-if="showNew" @created="showNew = !showNew" />
  </div>
</template>
