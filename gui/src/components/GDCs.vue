<script setup lang="ts">
import { ref, type Ref } from 'vue';
import GDC from './GDC.vue';
import { useGDCStore } from '../stores/gdc'
import { storeToRefs } from 'pinia'
import { PlusIcon } from '@heroicons/vue/24/outline'
import AlertError from './AlertError.vue';

const store = useGDCStore()
const { gdcs, errorGDCs } = storeToRefs(store)

const showNew: Ref<boolean> = ref(false)
</script>

<template>
  <div class="bg-slate-300 rounded-md p-2">
    <AlertError v-if="errorGDCs" :text="`Error loading GDCS: ${errorGDCs}`" />
    <GDC v-for="gdc in gdcs" :gdc="gdc" :key="gdc.id" />

    <button v-if="!showNew" @click="showNew = !showNew" class="btn btn-info w-auto text-xl">
      <PlusIcon class="text-red-900 w-6" />
      <p>Add GDC</p>
    </button>
    <GDC v-if="showNew" @created="showNew = !showNew"/>
  </div>
</template>
