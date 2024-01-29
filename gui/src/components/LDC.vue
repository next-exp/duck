<script setup lang="ts">
import { onMounted, computed, ref } from 'vue';
import type { PropType, Ref } from 'vue';
import { useForm } from 'vee-validate';
import { useField } from 'vee-validate';
import * as yup from 'yup';
import { TrashIcon } from '@heroicons/vue/24/outline'
import { useLDCStore } from '../stores/ldc'
import Equipment from './Equipment.vue';
import AlertSuccess from "./AlertSuccess.vue";
import AlertError from './AlertError.vue';
import type { LDC } from '@/gen/api_pb';
import { duckApiClient } from '@/api/client';

const emit = defineEmits(['created'])
const store = useLDCStore()

const props = defineProps({
  ldc: { required: false, type: Object as PropType<any> },
})
const id: Ref<number> = ref(0)
const errorMsg: Ref<string> = ref("")
const success: Ref<boolean> = ref(false);

const { values, errors, meta, handleSubmit, setValues, setFieldValue, defineInputBinds, submitCount } = useForm({
  validationSchema: yup.object({
    name: yup.string().required("Name is mandatory"),
    hostname: yup.string().required("Hostname is mandatory"),
    ip: yup.string().required("IP address is mandatory"),
    grpcPort: yup.number().required("Port number is mandatory"),
    prometheusPort: yup.number().required("Port number is mandatory"),
  })
});

const { value: name } = useField('name')
const { value: hostname } = useField('hostname')
const { value: ip } = useField('ip')
const { value: enabled } = useField('enabled')
const { value: grpcPort } = useField("grpcPort");
const { value: prometheusPort } = useField("prometheusPort");

const showErrors = computed(() => {
  return (!meta.value.valid) && (submitCount.value > 0)
})

export type LDCForm = {
  name: string,
  hostname: string,
  ip: string,
  enabled: boolean,
  grpcPort: number;
  prometheusPort: number;
}

async function createNewLDC(values: LDCForm) {
  success.value = false;
  try {
    const response = await duckApiClient.createLDC({
      name: values.name,
      hostname: values.hostname,
      ip: values.ip,
      enabled: values.enabled,
      grpcPort: values.grpcPort,
      prometheusPort: values.prometheusPort,
    });
    errorMsg.value = ""
    console.log(response)
    store.getLDCs()
    emit("created")
    success.value = true;
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
}

async function updateLDC(values: LDCForm) {
  success.value = false;
  if (props.ldc == null) return
  try {
    const response = await duckApiClient.updateLDC({
      id: props.ldc.id,
      name: values.name,
      hostname: values.hostname,
      ip: values.ip,
      enabled: values.enabled,
      grpcPort: values.grpcPort,
      prometheusPort: values.prometheusPort,
    });
    errorMsg.value = ""
    console.log(response)
    store.getLDCs()
    success.value = true;
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
}

async function deleteLDC() {
  success.value = false;
  if (props.ldc == null) return
  try {
    const response = await duckApiClient.deleteLDC({
      id: props.ldc.id,
    });
    errorMsg.value = ""
    console.log(response)
    store.getLDCs()
    success.value = true;
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
}

const onSubmit = handleSubmit(values => {
  let valuesForm = {
    name: values.name,
    hostname: values.hostname,
    ip: values.ip,
    enabled: values.enabled,
    grpcPort: Number(values.grpcPort),
    prometheusPort: Number(values.prometheusPort),
  }
  if (props.ldc != null) {
    updateLDC(valuesForm)
  } else {
    createNewLDC(valuesForm)
  }
})

onMounted(() => {
  if (props.ldc != null) {
    name.value = props.ldc.name
    hostname.value = props.ldc.hostname
    ip.value = props.ldc.ip
    id.value = props.ldc.id ?? 0
    enabled.value = props.ldc.enabled
    grpcPort.value = props.ldc.grpcPort;
    prometheusPort.value = props.ldc.prometheusPort;
  }
})

const modal: Ref<{ showModal: () => null } | null> = ref(null)
</script>

<template>
  <AlertSuccess v-if="success" text="Updated successfully!" />
  <AlertError v-if="errorMsg" :text="`Error: ${errorMsg}`" />
  <div class="border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
    <h2 class="px-3 text-xl font-bold">LDC</h2>
    <fieldset>
      <div class="flex flex-wrap items-end">
        <div class="form-control mx-2 w-32">
          <label class="label" :for="`ldc-name-${id}`">
            <span class="label-text">Name</span>
          </label>
          <input type="text" v-model="name" :id="`ldc-name-${id}`" :data-testid="`ldc-name-input-${id}`" placeholder="Name" class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors['name'] }}</div>
        </div>

        <div class="form-control mx-2 w-32">
          <label class="label" :for="`ldc-hostname-${id}`">
            <span class="label-text">Hostname</span>
          </label>
          <input type="text" v-model="hostname" :id="`ldc-hostname-${id}`" :data-testid="`ldc-hostname-input-${id}`" placeholder="Hostname"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors['hostname'] }}</div>
        </div>

        <div class="form-control mx-2 w-40">
          <label class="label" :for="`ldc-ip-${id}`">
            <span class="label-text">IP</span>
          </label>
          <input type="text" v-model="ip" :id="`ldc-ip-${id}`" :data-testid="`ldc-ip-input-${id}`" placeholder="IP" class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors['ip'] }}</div>
        </div>

        <div class="form-control mx-2 w-28">
          <label class="label" :for="`ldc-grpc-port-${id}`">
            <span class="label-text">gRPC port</span>
          </label>
          <input type="number" v-model="grpcPort" :id="`ldc-port-${id}`" :data-testid="`ldc-grpc-port-input-${id}`" placeholder="gRPC port"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["grpcPort"] }}</div>
        </div>

        <div class="form-control mx-2 w-28">
          <label class="label" :for="`ldc-prometheus-port-${id}`">
            <span class="label-text">Prom. port</span>
          </label>
          <input type="number" v-model="prometheusPort" :id="`ldc-prometheus-port-${id}`" :data-testid="`ldc-prometheus-port-input-${id}`" placeholder="Prometheus port"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">
            {{ errors["prometheusPort"] }}
          </div>
        </div>

        <div class="form-control mx-2">
          <label class="label" :for="`ldc-enabled-${id}`">
            <span class="label-text">Enabled</span>
          </label>
          <input type="checkbox" v-model="enabled" :id="`ldc-enabled-${id}`" class="checkbox checkbox-md" />
          <div v-if="showErrors" class="text-red-600">{{ errors['enabled'] }}</div>
        </div>

        <button @click="onSubmit" data-cy="submit" :data-testid="`ldc-submit-button-${id}`" class="btn btn-primary w-auto text-xl mr-2">
          <span>{{ props.ldc != null ? "Update LDC" : "Create LDC" }}</span>
        </button>

        <div v-if="props.ldc != null" class="tooltip" data-tip="Delete">
          <div>
            <button @click="modal?.showModal()" :data-testid="`ldc-delete-button-${id}`" class="btn btn-error w-auto text-xl">
              <TrashIcon class="text-red-900 w-6" />
            </button>
          </div>
          <dialog ref="modal" class="modal">
            <form method="dialog" class="modal-box">
              <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
              <p class="py-4">Confirm you want to delete LDC {{ props.ldc.hostname }}</p>
              <button class="btn btn-sm btn-ghost">Cancel</button>
              <button class="btn btn-sm btn-error" @click="deleteLDC()">Delete</button>
            </form>
          </dialog>
        </div>
      </div>

      <div v-if="props.ldc != null" class="collapse bg-base-200 m-2 w-1/2" data-testid="ldc-equipment-collapse">
        <input type="checkbox" />
        <div class="collapse-title text-xl font-medium">
          <p>Show LDC Equipments</p>
        </div>
        <div class="collapse-content">
          <Equipment v-for="equipment in props.ldc.equipments" :equipment="equipment" :key="equipment.id" />
        </div>
      </div>
    </fieldset>
  </div>
</template>