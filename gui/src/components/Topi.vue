<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { duckApiClient } from '@/api/client'
import AlertError from './AlertError.vue'
import AlertSuccess from './AlertSuccess.vue'

type Available = { name: string; parseError: string; isDefault: boolean }
const form = reactive({
  enabled: false, daemonUrl: '', rabbitmqAddress: '', rabbitmqPort: 5672,
  rabbitmqUser: '', rabbitmqVhost: '/', exchangeName: 'production',
  controlQueue: 'topi_daemon_control', selectedConfiguration: '',
  apiToken: '', rabbitmqPassword: '', hasApiToken: false, hasRabbitmqPassword: false,
})
const configurations = ref<Available[]>([])
const error = ref('')
const success = ref(false)
const loading = ref(false)

async function load() {
  try {
    const response = await duckApiClient.getTopiConfiguration({})
    if (response.configuration) Object.assign(form, response.configuration)
  } catch (e: any) { error.value = e.message }
}

async function fetchConfigurations() {
  loading.value = true; error.value = ''
  try {
    const response = await duckApiClient.listTopiConfigurations({ daemonUrl: form.daemonUrl, apiToken: form.apiToken })
    configurations.value = response.configurations
    if (!form.selectedConfiguration) {
      form.selectedConfiguration = configurations.value.find(c => c.isDefault && !c.parseError)?.name || ''
    }
  } catch (e: any) { error.value = e.message }
  finally { loading.value = false }
}

async function save() {
  error.value = ''; success.value = false
  try {
    await duckApiClient.updateTopiConfiguration({
      configuration: {
        enabled: form.enabled, daemonUrl: form.daemonUrl,
        rabbitmqAddress: form.rabbitmqAddress, rabbitmqPort: Number(form.rabbitmqPort),
        rabbitmqUser: form.rabbitmqUser, rabbitmqVhost: form.rabbitmqVhost,
        exchangeName: form.exchangeName, controlQueue: form.controlQueue,
        selectedConfiguration: form.selectedConfiguration,
      },
      apiToken: form.apiToken, rabbitmqPassword: form.rabbitmqPassword,
    })
    form.apiToken = ''; form.rabbitmqPassword = ''; success.value = true
    await load()
  } catch (e: any) { error.value = e.message }
}

onMounted(load)
</script>

<template>
  <div class="border min-w-screen-lg rounded-lg bg-violet-50 p-4 my-2">
    <h2 class="px-3 text-xl font-bold">TOPI integration</h2>
    <p class="px-3 py-2 text-sm">Duck fetches processing configurations from TOPI and sends non-blocking start/stop notifications. TOPI failures never stop data taking.</p>
    <AlertError v-if="error" :text="error" />
    <AlertSuccess v-if="success" text="TOPI configuration saved" />
    <form class="p-3 space-y-4" @submit.prevent="save">
      <label class="label cursor-pointer justify-start gap-3"><input v-model="form.enabled" type="checkbox" class="toggle" /><span class="label-text">Enable TOPI notifications</span></label>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <label class="form-control"><span class="label-text">TOPI daemon URL</span><input v-model.trim="form.daemonUrl" class="input input-bordered" placeholder="http://topi-host:8080" /></label>
        <label class="form-control"><span class="label-text">API token</span><input v-model="form.apiToken" type="password" class="input input-bordered" :placeholder="form.hasApiToken ? 'Stored — leave blank to keep' : 'Token'" autocomplete="new-password" /></label>
        <label class="form-control"><span class="label-text">RabbitMQ host</span><input v-model.trim="form.rabbitmqAddress" class="input input-bordered" /></label>
        <label class="form-control"><span class="label-text">RabbitMQ port</span><input v-model.number="form.rabbitmqPort" type="number" min="1" max="65535" class="input input-bordered" /></label>
        <label class="form-control"><span class="label-text">RabbitMQ user</span><input v-model.trim="form.rabbitmqUser" class="input input-bordered" /></label>
        <label class="form-control"><span class="label-text">RabbitMQ password</span><input v-model="form.rabbitmqPassword" type="password" class="input input-bordered" :placeholder="form.hasRabbitmqPassword ? 'Stored — leave blank to keep' : 'Password'" autocomplete="new-password" /></label>
        <label class="form-control"><span class="label-text">Virtual host</span><input v-model.trim="form.rabbitmqVhost" class="input input-bordered" /></label>
        <label class="form-control"><span class="label-text">Exchange</span><input v-model.trim="form.exchangeName" class="input input-bordered" /></label>
        <label class="form-control"><span class="label-text">Control queue / routing key</span><input v-model.trim="form.controlQueue" class="input input-bordered" /></label>
        <label class="form-control"><span class="label-text">Processing configuration</span>
          <div class="join"><select v-model="form.selectedConfiguration" class="select select-bordered join-item grow"><option value="">Select a configuration</option><option v-for="item in configurations" :key="item.name" :value="item.name" :disabled="!!item.parseError">{{ item.name }}{{ item.isDefault ? ' (default)' : '' }}{{ item.parseError ? ` — invalid: ${item.parseError}` : '' }}</option></select><button type="button" class="btn join-item" :disabled="loading" @click="fetchConfigurations">{{ loading ? 'Loading…' : 'Fetch' }}</button></div>
        </label>
      </div>
      <button type="submit" class="btn btn-primary">Save TOPI settings</button>
    </form>
  </div>
</template>
