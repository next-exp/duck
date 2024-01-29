import { shallowRef, computed } from 'vue'
import type { Ref } from 'vue'
import { defineStore } from 'pinia'
import duckApiClient from '@/api/client'
import type { GDC } from '@/gen/api_pb'

export const useGDCStore = defineStore('gdc', () => {
  const gdcs: Ref<Array<GDC>> = shallowRef([])
  const nActiveGDCs = computed(() => {
    return gdcs.value?.filter(gdc => gdc.enabled).length || 0
  })
  const errorGDCs = shallowRef("")

  async function getGDCs() {
    try {
      const response = await duckApiClient.getGDCs({})
      gdcs.value = response.gdcs
      errorGDCs.value = ""
      console.log(response)
    } catch (error: any) {
      console.log(error)
      errorGDCs.value = error.message
    }
  }

  getGDCs()

  return { gdcs, getGDCs, errorGDCs, nActiveGDCs }
})