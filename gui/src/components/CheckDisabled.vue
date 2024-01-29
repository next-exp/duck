<script setup lang="ts">
import AlertWarning from './AlertWarning.vue';
import AlertError from './AlertError.vue';
import { onMounted, ref, type Ref } from 'vue';
import duckApiClient from '@/api/client';

export type DisabledWarning = {
  gdcs: Array<string>,
  ldcs: Array<string>,
  equipments: Array<string>,
  writing: Array<string>,
}

const disabled: Ref<DisabledWarning> = ref({
  gdcs: [],
  ldcs: [],
  equipments: [],
  writing: [],
})
const errorCheckDisabled: Ref<string> = ref("")

function checkDisabled() {
  duckApiClient.checkDisabled({})
    .then((response) => {
      console.log(response)
      disabled.value = {
        gdcs: response.warnings?.gdcs || [],
        ldcs: response.warnings?.ldcs || [],
        equipments: response.warnings?.equipments || [],
        writing: response.warnings?.writing || []
      }
      errorCheckDisabled.value = ""
    })
    .catch((error: any) => {
      console.log(error)
      errorCheckDisabled.value = error.message
    })
}

onMounted(() => {
  checkDisabled()
})  
</script>

<template>
  <div class="flex gap-1 w-fit m-2 p-2 px-4 rounded-md mx-auto">
    <AlertError v-if="errorCheckDisabled" :text="`Error loading GDCS: ${errorCheckDisabled}`" />
    <AlertWarning v-if="disabled?.gdcs?.length > 0" :text="`Disabled GDCs: ${disabled.gdcs}`" />
    <AlertWarning v-if="disabled?.ldcs?.length > 0" :text="`Disabled LDCs: ${disabled.ldcs}`" />
    <AlertWarning v-if="disabled?.equipments?.length > 0" :text="`Disabled Equipments: ${disabled.equipments}`" />
    <AlertWarning v-if="disabled?.writing?.length > 0" :text="`File writing disabled on: ${disabled.writing}`" />
  </div>
</template>
