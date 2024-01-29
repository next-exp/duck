<script setup lang="ts">
import { ref, type Ref } from 'vue';
import Equipment from './Equipment.vue';
import { useEquipmentStore } from '../stores/equipment'
import { storeToRefs } from 'pinia'
import { PlusIcon } from '@heroicons/vue/24/outline'
import AlertError from './AlertError.vue';

const store = useEquipmentStore()
const { equipments, errorEquipments } = storeToRefs(store)

const showNew: Ref<boolean> = ref(false)
</script>

<template>
  <div class="bg-slate-300 rounded-md p-2">
    <AlertError v-if="errorEquipments" :text="`Error loading Equipments: ${errorEquipments}`" />
    <Equipment v-for="equipment in equipments" :equipment="equipment" :key="equipment.id" />
    <button v-if="!showNew" @click="showNew = !showNew" class="btn btn-info w-auto text-xl">
      <PlusIcon class="text-red-900 w-6" />
      <p>Add Equipment</p>
    </button>
    <Equipment v-if="showNew" @created="showNew = !showNew"/>
  </div>
</template>
