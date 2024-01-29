<script setup lang="ts">
import { onMounted, computed, ref } from 'vue';
import type { PropType, Ref } from 'vue';
import { useForm } from 'vee-validate';
import { useField } from 'vee-validate';
import * as yup from 'yup';
import { TrashIcon } from '@heroicons/vue/24/outline'
import { useEquipmentStore } from '../stores/equipment'
import { useLDCStore } from '../stores/ldc'
import { storeToRefs } from 'pinia'
import AlertSuccess from "./AlertSuccess.vue";
import AlertError from './AlertError.vue';
import type { Equipment } from '@/gen/api_pb';
import { duckApiClient } from '@/api/client';


const emit = defineEmits(['created'])
const store = useEquipmentStore()
const ldcStore = useLDCStore()
const { ldcs } = storeToRefs(ldcStore)

const props = defineProps({
  equipment: { required: false, type: Object as PropType<any> },
})
const id: Ref<number> = ref(0)
const errorMsg: Ref<string> = ref("")
const success: Ref<boolean> = ref(false);

const { values, errors, meta, handleSubmit, setValues, setFieldValue, defineInputBinds, submitCount } = useForm({
  validationSchema: yup.object({
    //type: yup.number().required("Device type is mandatory"),
    deviceIp: yup.string().required("Device IP address is mandatory"),
    hostIp: yup.string().required("Host IP address is mandatory"),
    hostPort: yup.number().required("Host port number is mandatory"),
    ldcId: yup.number().required("LDC ID is mandatory"),
  })
});

//const { value: type } = useField('type')
const { value: deviceIp } = useField('deviceIp')
const { value: hostIp } = useField('hostIp')
const { value: hostPort } = useField('hostPort')
const { value: ldcId } = useField('ldcId')
const { value: enabled } = useField('enabled')

const showErrors = computed(() => {
  return (!meta.value.valid) && (submitCount.value > 0)
})

export type EquipmentForm = {
  type: number,
  deviceIp: string,
  hostIp: string,
  hostPort: number,
  ldcId: number,
  enabled: boolean,
}

async function createNewEquipment(values: EquipmentForm) {
  success.value = false;
  try {
    const response = await duckApiClient.createEquipment({
      type: values.type,
      deviceIp: values.deviceIp,
      hostIp: values.hostIp,
      hostPort: values.hostPort,
      ldcId: values.ldcId,
      enabled: values.enabled,
    });
    errorMsg.value = ""
    console.log(response)
    store.getEquipments()
    emit("created")
    success.value = true;
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
}

async function updateEquipment(values: EquipmentForm) {
  success.value = false;
  if (props.equipment == null) return
  try {
    const response = await duckApiClient.updateEquipment({
      id: props.equipment.id,
      type: values.type,
      deviceIp: values.deviceIp,
      hostIp: values.hostIp,
      hostPort: values.hostPort,
      ldcId: values.ldcId,
      enabled: values.enabled,
    });
    errorMsg.value = ""
    console.log(response)
    store.getEquipments()
    success.value = true;
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
}

async function deleteEquipment() {
  success.value = false;
  if (props.equipment == null) return
  try {
    const response = await duckApiClient.deleteEquipment({
      id: props.equipment.id,
    });
    errorMsg.value = ""
    console.log(response)
    store.getEquipments()
    success.value = true;
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
}

const onSubmit = handleSubmit(values => {
  let valuesCasted: EquipmentForm = {
    type: 22,
    deviceIp: values.deviceIp,
    hostIp: values.hostIp,
    hostPort: Number(values.hostPort),
    ldcId: Number(values.ldcId),
    enabled: values.enabled,
  }
  if (props.equipment != null) {
    updateEquipment(valuesCasted)
  } else {
    createNewEquipment(valuesCasted)
  }
})

onMounted(() => {
  if (props.equipment != null) {
    //type.value = props.equipment.type
    deviceIp.value = props.equipment.deviceIp
    hostIp.value = props.equipment.hostIp
    hostPort.value = props.equipment.hostPort
    ldcId.value = props.equipment.ldcId
    enabled.value = props.equipment.enabled

    id.value = props.equipment.id ?? 0
  }
})

const modal: Ref<{ showModal: () => null } | null> = ref(null)
</script>

<template>
  <AlertSuccess v-if="success" text="Updated successfully!" />
  <AlertError v-if="errorMsg" :text="`Error: ${errorMsg}`" />
  <div class="border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
    <h2 class="px-3 text-xl font-bold">Equipment</h2>
    <fieldset>
      <div class="flex flex-wrap items-end">
        <div class="form-control mx-2 w-40">
          <label class="label" :for="`eq-device-ip-${id}`">
            <span class="label-text">Device IP</span>
          </label>
          <input type="text" v-model="deviceIp" :id="`eq-device-ip-${id}`" :data-testid="`equipment-device-ip-input-${id}`" placeholder="Device IP"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors['deviceIp'] }}</div>
        </div>

        <div class="form-control mx-2 w-40">
          <label class="label" :for="`eq-host-ip-${id}`">
            <span class="label-text">Host IP</span>
          </label>
          <input type="text" v-model="hostIp" :id="`eq-host-ip-${id}`" :data-testid="`equipment-host-ip-input-${id}`" placeholder="Host IP"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors['hostIp'] }}</div>
        </div>

        <div class="form-control mx-2 w-28">
          <label class="label" :for="`eq-host-port-${id}`">
            <span class="label-text">Host Port</span>
          </label>
          <input type="number" v-model="hostPort" :id="`eq-host-port-${id}`" :data-testid="`equipment-host-port-input-${id}`" placeholder="Host Port"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors['hostPort'] }}</div>
        </div>

        <div class="form-control mx-2 w-28">
          <label class="label" :for="`eq-ldcid-${id}`">
            <span class="label-text">LDC ID</span>
          </label>
          <select v-model="ldcId" :id="`eq-ldcid-${id}`" :data-testid="`equipment-ldc-select-${id}`" class="select select-bordered">
            <option v-for="ldc in ldcs" :value="ldc.id">{{ ldc.hostname }}</option>
          </select>
          <div v-if="showErrors" class="text-red-600">{{ errors['ldcId'] }}</div>
        </div>

        <div class="form-control mx-2">
          <label class="label" :for="`eq-enabled-${id}`">
            <span class="label-text">Enabled</span>
          </label>
          <input type="checkbox" v-model="enabled" :id="`eq-enabled-${id}`" :data-testid="`equipment-enabled-checkbox-${id}`" class="checkbox checkbox-md" />
          <div v-if="showErrors" class="text-red-600">{{ errors['enabled'] }}</div>
        </div>

        <button @click="onSubmit" :data-testid="`equipment-submit-button-${id}`" class="btn btn-primary w-auto text-xl mr-2">
          <span>{{ props.equipment != null ? "Update" : "Create Equipment" }}</span>
        </button>

        <div v-if="props.equipment != null" class="tooltip" data-tip="Delete">
          <div>
            <button @click="modal?.showModal()" :data-testid="`equipment-delete-button-${id}`" class="btn btn-error w-full mx-auto object-center text-xl">
              <TrashIcon class="text-red-900 w-6" />
            </button>
          </div>
          <dialog ref="modal" class="modal">
            <form method="dialog" class="modal-box">
              <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
              <p class="py-4">Confirm you want to delete equipment {{ props.equipment.deviceIp }}</p>
              <button class="btn btn-sm btn-ghost">Cancel</button>
              <button class="btn btn-sm btn-error" @click="deleteEquipment()">Delete</button>
            </form>
          </dialog>
        </div>
      </div>
    </fieldset>
  </div>
</template>