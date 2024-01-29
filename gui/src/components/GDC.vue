<script setup lang="ts">
import { onMounted, computed, ref } from "vue";
import type { PropType, Ref } from "vue";
import { useForm } from "vee-validate";
import { useField } from "vee-validate";
import { watch } from "vue";
import * as yup from "yup";
import { TrashIcon } from "@heroicons/vue/24/outline";
import { useGDCStore } from "../stores/gdc";
import AlertError from "./AlertError.vue";
import AlertSuccess from "./AlertSuccess.vue";
import type { GDC } from "@/gen/api_pb";
import { duckApiClient } from "@/api/client";

const emit = defineEmits(["created"]);
const store = useGDCStore();

const props = defineProps({
  gdc: { required: false, type: Object as PropType<any> },
});
const id: Ref<number> = ref(0);
const errorMsg: Ref<string> = ref("");
const success: Ref<boolean> = ref(false);

const {
  values,
  errors,
  meta,
  handleSubmit,
  setValues,
  setFieldValue,
  defineInputBinds,
  submitCount,
} = useForm({
  validationSchema: yup.object({
    name: yup.string().required("Name is mandatory"),
    hostname: yup.string().required("Hostname is mandatory"),
    ip: yup.string().required("IP address is mandatory"),
    port: yup.number().required("Port number is mandatory"),
    datapath: yup.string().required("Data path is mandatory"),
    grpcPort: yup.number().required("Port number is mandatory"),
    prometheusPort: yup.number().required("Port number is mandatory"),
    enabled: yup.boolean(),
    writeOutput: yup.boolean(),
    decode: yup.boolean(),
  }),
});

const { value: name } = useField("name");
const { value: hostname } = useField("hostname");
const { value: ip } = useField("ip");
const { value: port } = useField("port");
const { value: grpcPort } = useField("grpcPort");
const { value: prometheusPort } = useField("prometheusPort");
const { value: datapath } = useField("datapath");
const { value: enabled } = useField("enabled");
const { value: writeOutput } = useField("writeOutput");
const { value: decode } = useField("decode");

const showErrors = computed(() => {
  return !meta.value.valid && submitCount.value > 0;
});

export type GDCForm = {
  name: string;
  hostname: string;
  ip: string;
  port: number;
  grpcPort: number;
  prometheusPort: number;
  datapath: string;
  enabled: boolean;
  writeOutput: boolean;
  decode: boolean;
};

async function createNewGDC(values: GDCForm) {
  success.value = false;
  try {
    const response = await duckApiClient.createGDC({
      name: values.name,
      hostname: values.hostname,
      ip: values.ip,
      port: values.port,
      grpcPort: values.grpcPort,
      prometheusPort: values.prometheusPort,
      datapath: values.datapath,
      enabled: values.enabled,
      writeOutput: values.writeOutput,
      decode: values.decode,
    });
    errorMsg.value = "";
    console.log(response);
    store.getGDCs();
    emit("created");
    success.value = true;
  } catch (error: any) {
    console.log(error);
    errorMsg.value = error.message;
  }
}

async function updateGDC(values: GDCForm) {
  success.value = false;
  if (props.gdc == null) return;
  try {
    const response = await duckApiClient.updateGDC({
      id: props.gdc.id,
      name: values.name,
      hostname: values.hostname,
      ip: values.ip,
      port: values.port,
      grpcPort: values.grpcPort,
      prometheusPort: values.prometheusPort,
      datapath: values.datapath,
      enabled: values.enabled,
      writeOutput: values.writeOutput,
      decode: values.decode,
    });
    errorMsg.value = "";
    console.log(response);
    store.getGDCs();
    success.value = true;
  } catch (error: any) {
    console.log(error);
    errorMsg.value = error.message;
  }
}

async function deleteGDC() {
  success.value = false;
  if (props.gdc == null) return;
  try {
    const response = await duckApiClient.deleteGDC({
      id: props.gdc.id,
    });
    errorMsg.value = "";
    console.log(response);
    store.getGDCs();
    success.value = true;
  } catch (error: any) {
    console.log(error);
    errorMsg.value = error.message;
  }
}

const onSubmit = handleSubmit((values) => {
  let valuesCasted: GDCForm = {
    name: values.name,
    hostname: values.hostname,
    ip: values.ip,
    datapath: values.datapath,
    port: Number(values.port),
    grpcPort: Number(values.grpcPort),
    prometheusPort: Number(values.prometheusPort),
    enabled: values.enabled,
    writeOutput: values.writeOutput,
    decode: values.decode,
  };
  if (props.gdc != null) {
    updateGDC(valuesCasted);
  } else {
    createNewGDC(valuesCasted);
  }
});

onMounted(() => {
  if (props.gdc != null) {
    name.value = props.gdc.name;
    hostname.value = props.gdc.hostname;
    ip.value = props.gdc.ip;
    port.value = props.gdc.port;
    grpcPort.value = props.gdc.grpcPort;
    prometheusPort.value = props.gdc.prometheusPort;
    datapath.value = props.gdc.datapath;
    enabled.value = props.gdc.enabled;
    writeOutput.value = props.gdc.writeOutput;
    decode.value = props.gdc.decode;

    id.value = props.gdc.id ?? 0;
  }
});

const modal: Ref<{ showModal: () => null } | null> = ref(null);
</script>

<template>
  <AlertError v-if="errorMsg" :text="`Error: ${errorMsg}`" />
  <AlertSuccess v-if="success" text="Updated successfully!" />
  <div class="border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
    <h2 class="px-3 text-xl font-bold">GDC</h2>
    <fieldset>
      <div class="flex flex-wrap items-end">
        <div class="form-control mx-2 w-32">
          <label class="label" :for="`gdc-name-${id}`">
            <span class="label-text">Name</span>
          </label>
          <input type="text" v-model="name" :id="`gdc-name-${id}`" :data-testid="`gdc-name-input-${id}`" placeholder="Name" class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["name"] }}</div>
        </div>

        <div class="form-control mx-2 w-32">
          <label class="label" :for="`gdc-hostname-${id}`">
            <span class="label-text">Hostname</span>
          </label>
          <input type="text" v-model="hostname" :id="`gdc-hostname-${id}`" :data-testid="`gdc-hostname-input-${id}`" placeholder="Hostname"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["hostname"] }}</div>
        </div>

        <div class="form-control mx-2 w-40">
          <label class="label" :for="`gdc-ip-${id}`">
            <span class="label-text">IP</span>
          </label>
          <input type="text" v-model="ip" :id="`gdc-ip-${id}`" :data-testid="`gdc-ip-input-${id}`" placeholder="IP" class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["ip"] }}</div>
        </div>

        <div class="form-control mx-2 w-28">
          <label class="label" :for="`gdc-port-${id}`">
            <span class="label-text">Port</span>
          </label>
          <input type="number" v-model="port" :id="`gdc-port-${id}`" :data-testid="`gdc-port-input-${id}`" placeholder="Port" class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["port"] }}</div>
        </div>

        <div class="form-control mx-2 w-96">
          <label class="label" :for="`gdc-datapath-${id}`">
            <span class="label-text">Datapath</span>
          </label>
          <input type="text" v-model="datapath" :id="`gdc-datapath-${id}`" :data-testid="`gdc-datapath-input-${id}`" placeholder="Data path"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["datapath"] }}</div>
        </div>

        <div class="form-control mx-2 w-28">
          <label class="label" :for="`gdc-grpc-port-${id}`">
            <span class="label-text">gRPC port</span>
          </label>
          <input type="number" v-model="grpcPort" :id="`gdc-port-${id}`" :data-testid="`gdc-grpc-port-input-${id}`" placeholder="gRPC port"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["grpcPort"] }}</div>
        </div>

        <div class="form-control mx-2 w-28">
          <label class="label" :for="`gdc-prometheus-port-${id}`">
            <span class="label-text">Prom. port</span>
          </label>
          <input type="number" v-model="prometheusPort" :id="`gdc-prometheus-port-${id}`" :data-testid="`gdc-prometheus-port-input-${id}`" placeholder="Prometheus port"
            class="input input-bordered" />
          <div v-if="showErrors" class="text-red-600">{{ errors["prometheusPort"] }}</div>
        </div>

        <div class="form-control mx-2">
          <label class="label" :for="`gdc-enabled-${id}`">
            <span class="label-text">Enabled</span>
          </label>
          <input type="checkbox" v-model="enabled" :id="`gdc-enabled-${id}`" class="checkbox checkbox-md" />
          <div v-if="showErrors" class="text-red-600">{{ errors["enabled"] }}</div>
        </div>

        <div class="form-control mx-2">
          <label class="label" :for="`gdc-write-enabled-${id}`">
            <span class="label-text">Write binary file</span>
          </label>
          <input type="checkbox" v-model="writeOutput" :id="`gdc-write-enabled-${id}`" class="checkbox checkbox-md" />
          <div v-if="showErrors" class="text-red-600">{{ errors["writeOutput"] }}</div>
        </div>

        <div class="form-control mx-2">
          <label class="label" :for="`gdc-decode-${id}`">
            <span class="label-text">Decode</span>
          </label>
          <input type="checkbox" v-model="decode" :id="`gdc-decode-${id}`" class="checkbox checkbox-md" />
          <div v-if="showErrors" class="text-red-600">{{ errors["decode"] }}</div>
        </div>

        <button @click="onSubmit" data-cy="submit" :data-testid="`gdc-submit-button-${id}`" class="btn btn-primary w-auto text-xl mr-2">
          <span>{{ props.gdc != null ? "Update" : "Create GDC" }}</span>
        </button>

        <div v-if="props.gdc != null" class="tooltip" data-tip="Delete">
          <div>
            <button @click="modal?.showModal()" :data-testid="`gdc-delete-button-${id}`" class="btn btn-error w-auto text-xl">
              <TrashIcon class="text-red-900 w-6" />
            </button>
          </div>
          <dialog ref="modal" class="modal">
            <form method="dialog" class="modal-box">
              <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">
                ✕
              </button>
              <p class="py-4">Confirm you want to delete GDC {{ props.gdc.hostname }}</p>
              <button class="btn btn-sm btn-ghost">Cancel</button>
              <button class="btn btn-sm btn-error" @click="deleteGDC()">Delete</button>
            </form>
          </dialog>
        </div>
      </div>
    </fieldset>
  </div>
</template>
