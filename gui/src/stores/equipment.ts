import { shallowRef, computed } from 'vue'
import type { Ref } from 'vue'
import { defineStore } from 'pinia'
import duckApiClient from '@/api/client'
import type { Equipment } from '@/gen/api_pb'

export const useEquipmentStore = defineStore('equipment', () => {
  const equipments: Ref<Array<Equipment>> = shallowRef([])
  const errorEquipments = shallowRef("")

  async function getEquipments() {
    try {
      const response = await duckApiClient.getEquipments({})
      equipments.value = response.equipments
      errorEquipments.value = ""
      console.log(response)
    } catch (error: any) {
      console.log(error)
      errorEquipments.value = error.message
    }
  }

  getEquipments()

  return { equipments, getEquipments, errorEquipments }
})