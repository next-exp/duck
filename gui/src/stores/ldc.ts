import { shallowRef, computed } from 'vue'
import type { Ref } from 'vue'
import { defineStore } from 'pinia'
import duckApiClient from '@/api/client'
import type { LDC } from '@/gen/api_pb'

export const useLDCStore = defineStore('ldc', () => {
  const ldcs: Ref<Array<LDC>> = shallowRef([])
  const nActiveLDCs = computed(() => {
    return ldcs.value?.filter(ldc => ldc.enabled).length || 0
  })
  const errorLDCs = shallowRef("")

  async function getLDCs() {
    console.log("get ldcs")
    try {
      const response = await duckApiClient.getLDCs({})
      ldcs.value = response.ldcs
      errorLDCs.value = ""
      console.log(response)
    } catch (error: any) {
      console.log(error)
      errorLDCs.value = error.message
    }
  }

  getLDCs()

  return { ldcs, getLDCs, errorLDCs, nActiveLDCs }
})