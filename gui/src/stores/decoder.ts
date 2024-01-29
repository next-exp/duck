import { shallowRef, computed } from 'vue'
import type { Ref } from 'vue'
import { defineStore } from 'pinia'
import duckApiClient from '@/api/client'
import { DecoderConfiguration } from '@/gen/api_pb'

export const useDecoderStore = defineStore('decoder', () => {
  const decoderConfig: Ref<DecoderConfiguration> = shallowRef(new DecoderConfiguration())
  const errorDecoder = shallowRef("")

  async function getDecoderConfig() {
    try {
      const response = await duckApiClient.getDecoderConfiguration({})
      if (response.configuration) {
        decoderConfig.value = response.configuration
      }
      errorDecoder.value = ""
      console.log(response)
    } catch (error: any) {
      console.log(error)
      errorDecoder.value = error.message
    }
  }

  getDecoderConfig()

  return { decoderConfig, getDecoderConfig, errorDecoder }
})