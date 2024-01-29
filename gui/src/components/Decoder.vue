<script setup lang="ts">
import { ref } from 'vue';
import { onMounted, computed, watch } from 'vue';
import type { Ref } from 'vue';
import { useForm } from 'vee-validate';
import { useField } from 'vee-validate';
import * as yup from 'yup';
import { useDecoderStore } from '../stores/decoder'
import { storeToRefs } from 'pinia'
import AlertSuccess from "./AlertSuccess.vue";
import AlertError from './AlertError.vue';
import { duckApiClient } from '@/api/client';

const store = useDecoderStore()
const { decoderConfig, errorDecoder } = storeToRefs(store)

const compression_algorithms: Ref<Array<string>> = ref([
  "blosclz",
  "lz4",
  "lz4hc",
  "snappy",
  "zlib",
  "zstd",
]);

const compression_shuffles: Ref<Array<string>> = ref([
  "no-shuffle",
  "byte-shuffle",
  "bit-shuffle",
]);

const errorMsg: Ref<string> = ref("")
const success: Ref<boolean> = ref(false);

const { values, errors, meta, handleSubmit, setValues, setFieldValue, defineInputBinds, submitCount } = useForm({
  validationSchema: yup.object({
    ext_trigger: yup.number()
      .min(0, "The channel number must be positive")
      .required("External trigger channel is mandatory"),
    trg_code_1: yup.number()
      .min(0, "The trigger code must be positive")
      .required("Trigger 1 code is mandatory"),
    trg_code_2: yup.number()
      .min(0, "The trigger code must be positive")
      .required("Trigger 2 code is mandatory"),
    read_pmts: yup.boolean(),
    read_sipms: yup.boolean(),
    read_trigger: yup.boolean(),
    split_trigger: yup.boolean(),
    no_db: yup.boolean(),
    discard: yup.boolean(),
    host: yup.string().required("Database host is mandatory"),
    user: yup.string().required("Database user is mandatory"),
    password: yup.string().required("Database password is mandatory"),
    db_name: yup.string().required("Database name is mandatory"),
    write_data: yup.boolean(),
    use_blosc: yup.boolean(),
    blosc_algorithm: yup.string().required("Compression algorithm is mandatory"),
    compression_level: yup.string().required("Compression level is mandatory"),
    bit_shuffle: yup.string().required("Bit shuffle is mandatory"),
  })
});

const { value: ext_trigger } = useField('ext_trigger');
const { value: trg_code_1 } = useField('trg_code_1');
const { value: trg_code_2 } = useField('trg_code_2');
const { value: read_pmts } = useField('read_pmts');
const { value: read_sipms } = useField('read_sipms');
const { value: read_trigger } = useField('read_trigger');
const { value: split_trigger } = useField('split_trigger');
const { value: no_db } = useField('no_db');
const { value: discard } = useField('discard');
const { value: host } = useField('host');
const { value: user } = useField('user');
const { value: password } = useField('password');
const { value: db_name } = useField('db_name');
const { value: write_data } = useField('write_data');
const { value: use_blosc } = useField('use_blosc');
const { value: blosc_algorithm } = useField('blosc_algorithm');
const { value: compression_level } = useField('compression_level');
const { value: bit_shuffle } = useField('bit_shuffle');

const showErrors = computed(() => {
  return (!meta.value.valid) && (submitCount.value > 0)
})

export type DecoderForm = {
  ext_trigger: number,
  trg_code_1: number,
  trg_code_2: number,
  read_pmts: boolean,
  read_sipms: boolean,
  read_trigger: boolean,
  split_trigger: boolean,
  no_db: boolean,
  discard: boolean,
  host: string,
  user: string,
  password: string,
  db_name: string,
  write_data: boolean,
  use_blosc: boolean,
  blosc_algorithm: string,
  compression_level: string,
  bit_shuffle: string,
}

async function updateDecoderConfig(values: DecoderForm) {
  success.value = false;
  try {
    const response = await duckApiClient.updateDecoderConfiguration({
      configuration: {
        extTrigger: values.ext_trigger,
        trgCode1: values.trg_code_1,
        trgCode2: values.trg_code_2,
        readPmts: values.read_pmts,
        readSipms: values.read_sipms,
        readTrigger: values.read_trigger,
        splitTrigger: values.split_trigger,
        noDb: values.no_db,
        discard: values.discard,
        host: values.host,
        user: values.user,
        password: values.password,
        dbName: values.db_name,
        writeData: values.write_data,
        useBlosc: values.use_blosc,
        bloscAlgorithm: values.blosc_algorithm,
        compressionLevel: Number(values.compression_level),
        bitShuffle: values.bit_shuffle,
      },
    });
    errorMsg.value = ""
    console.log(response)
    store.getDecoderConfig()
    success.value = true;
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
}

const onSubmit = handleSubmit(values => {
  let valuesCasted: DecoderForm = {
    ext_trigger: Number(values.ext_trigger),
    trg_code_1: Number(values.trg_code_1),
    trg_code_2: Number(values.trg_code_2),
    read_pmts: values.read_pmts,
    read_sipms: values.read_sipms,
    read_trigger: values.read_trigger,
    split_trigger: values.split_trigger,
    no_db: values.no_db,
    discard: values.discard,
    host: values.host,
    user: values.user,
    password: values.password,
    db_name: values.db_name,
    write_data: values.write_data,
    use_blosc: values.use_blosc,
    blosc_algorithm: values.blosc_algorithm,
    compression_level: values.compression_level,
    bit_shuffle: values.bit_shuffle,
  }
  updateDecoderConfig(valuesCasted)
})

function updateDecoderConfigGUI(newConfig: any) {
  ext_trigger.value = newConfig.extTrigger;
  trg_code_1.value = newConfig.trgCode1;
  trg_code_2.value = newConfig.trgCode2;
  read_pmts.value = newConfig.readPmts;
  read_sipms.value = newConfig.readSipms;
  read_trigger.value = newConfig.readTrigger;
  split_trigger.value = newConfig.splitTrigger;
  no_db.value = newConfig.noDb;
  discard.value = newConfig.discard;
  host.value = newConfig.host;
  user.value = newConfig.user;
  password.value = newConfig.password;
  db_name.value = newConfig.dbName;
  write_data.value = newConfig.writeData;
  use_blosc.value = newConfig.useBlosc;
  blosc_algorithm.value = newConfig.bloscAlgorithm;
  compression_level.value = newConfig.compressionLevel;
  bit_shuffle.value = newConfig.bitShuffle;
}

watch(decoderConfig, (newConfig) => {
  updateDecoderConfigGUI(newConfig)
});

onMounted(() => {
  updateDecoderConfigGUI(decoderConfig.value)
})
</script>

<template>
  <AlertError v-if="errorDecoder" :text="`Error: ${errorDecoder}`" />
  <div class="border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
    <h2 class="px-3 text-xl font-bold">Decoder configuration</h2>
    <fieldset>
      <div class="flex flex-col flex-wrap gap-2">
        <div class="flex flex-wrap border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
          <div class="flex flex-col gap-2">
            <h3 class="px-3 text-xl font-bold">Trigger configuration</h3>
            <div class="flex flex-wrap gap-2">
              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-external-trigger`">
                  <span class="label-text">External trigger</span>
                </label>
                <input type="text" v-model="ext_trigger" :id="`decoder-external-trigger`" placeholder="Ext trg ch"
                  class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['ext_trigger'] }}</div>
              </div>

              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-trigger-code-1`">
                  <span class="label-text">Trigger code 1</span>
                </label>
                <input type="number" v-model="trg_code_1" :id="`decoder-trigger-code-1`" placeholder="Trg1 code"
                  class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['trg_code_1'] }}</div>
              </div>

              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-trigger-code-2`">
                  <span class="label-text">Trigger code 2</span>
                </label>
                <input type="number" v-model="trg_code_2" :id="`decoder-trigger-code-2`" placeholder="Trg2 code"
                  class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['trg_code_2'] }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
          <div class="flex flex-col gap-2">
            <h3 class="px-3 text-xl font-bold">Database configuration</h3>
            <div class="flex flex-wrap gap-2">
              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-db-host`">
                  <span class="label-text">Database host</span>
                </label>
                <input type="text" v-model="host" :id="`decoder-db-host`" placeholder="DB host"
                  class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['host'] }}</div>
              </div>

              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-db-user`">
                  <span class="label-text">Database user</span>
                </label>
                <input type="text" v-model="user" :id="`decoder-db-user`" placeholder="DB user"
                  class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['user'] }}</div>
              </div>

              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-db-password`">
                  <span class="label-text">Database password</span>
                </label>
                <input type="password" v-model="password" :id="`decoder-db-password`" placeholder="DB password"
                  class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['password'] }}</div>
              </div>

              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-db-name`">
                  <span class="label-text">Database name</span>
                </label>
                <input type="text" v-model="db_name" :id="`decoder-db-name`" placeholder="DB name"
                  class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['db_name'] }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
          <div class="flex flex-col gap-2">
            <h3 class="px-3 text-xl font-bold">Decode process</h3>
            <div class="flex flex-wrap gap-2">
              <div class="form-control mx-2">
                <label class="label" :for="`decoder-read-pmts`">
                  <span class="label-text">Read PMTs</span>
                </label>
                <input type="checkbox" v-model="read_pmts" :id="`decoder-read-pmts`" class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['read_pmts'] }}</div>
              </div>

              <div class="form-control mx-2">
                <label class="label" :for="`decoder-read-sipms`">
                  <span class="label-text">Read SiPMs</span>
                </label>
                <input type="checkbox" v-model="read_sipms" :id="`decoder-read-sipms`" class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['read_sipms'] }}</div>
              </div>

              <div class="form-control mx-2">
                <label class="label" :for="`decoder-read-trigger`">
                  <span class="label-text">Read Trigger</span>
                </label>
                <input type="checkbox" v-model="read_trigger" :id="`decoder-read-trigger`"
                  class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['read_trigger'] }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
          <div class="flex flex-col gap-2">
            <h3 class="px-3 text-xl font-bold">Flags</h3>
            <div class="flex flex-wrap gap-2">
              <div class="form-control mx-2">
                <label class="label" :for="`decoder-split-trigger`">
                  <span class="label-text">Split trigger</span>
                </label>
                <input type="checkbox" v-model="split_trigger" :id="`decoder-split-trigger`"
                  class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['split_trigger'] }}</div>
              </div>

              <div class="form-control mx-2">
                <label class="label" :for="`decoder-no-db`">
                  <span class="label-text">No DB</span>
                </label>
                <input type="checkbox" v-model="no_db" :id="`decoder-no-db`" class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['no_db'] }}</div>
              </div>

              <div class="form-control mx-2">
                <label class="label" :for="`decoder-discard`">
                  <span class="label-text">Discard errors</span>
                </label>
                <input type="checkbox" v-model="discard" :id="`decoder-discard`" class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['discard'] }}</div>
              </div>

              <div class="form-control mx-2">
                <label class="label" :for="`decoder-write-data`">
                  <span class="label-text">Write data</span>
                </label>
                <input type="checkbox" v-model="write_data" :id="`decoder-write-data`" class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['write_data'] }}</div>
              </div>

              <div class="form-control mx-2">
                <label class="label" :for="`decoder-use-blosc`">
                  <span class="label-text">Use blosc</span>
                </label>
                <input type="checkbox" v-model="use_blosc" :id="`decoder-use-blosc`" class="checkbox checkbox-md" />
                <div v-if="showErrors" class="text-red-600">{{ errors['use_blosc'] }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
          <div class="flex flex-col gap-2">
            <h3 class="px-3 text-xl font-bold">Compression</h3>
            <div class="flex flex-wrap gap-2">
              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-blosc-algorithm`">
                  <span class="label-text">Blosc algorithm</span>
                </label>
                <select id="decoder-blosc-algorithm" v-model="blosc_algorithm" class="select select-bordered">
                  <option v-for="algorithm in compression_algorithms" :value="algorithm">{{ algorithm }}</option>
                </select>
                <div v-if="showErrors" class="text-red-600">{{ errors['blosc_algorithm'] }}</div>
              </div>

              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-compression-level`">
                  <span class="label-text">Compression level</span>
                </label>
                <input type="number" v-model="compression_level" :id="`decoder-compression-level`" min="0" max="9"
                  placeholder="Compression level" class="input input-bordered" />
                <div v-if="showErrors" class="text-red-600">{{ errors['compression_level'] }}</div>
              </div>

              <div class="form-control mx-2 w-40">
                <label class="label" :for="`decoder-bit-shuffle`">
                  <span class="label-text">Blosc bit shuffle</span>
                </label>
                <select id="decoder-bit-shuffle" v-model="bit_shuffle" class="select select-bordered">
                  <option v-for="shuffle in compression_shuffles" :value="shuffle">{{ shuffle }}</option>
                </select>
                <div v-if="showErrors" class="text-red-600">{{ errors['bit_shuffle'] }}</div>
              </div>
            </div>
          </div>
        </div>

        <AlertSuccess v-if="success" text="Updated successfully!" />
        <AlertError v-if="errorMsg" :text="`Error: ${errorMsg}`" />
        <button @click="onSubmit" data-testid="decoder-submit-button" class="btn btn-primary w-auto text-xl mr-2">
          <span>Update</span>
        </button>
      </div>
    </fieldset>
  </div>
</template>
