<script setup lang="ts">
import { useMessagesStore } from "@/stores/messages";
import { storeToRefs } from 'pinia'
import { onMounted, onUnmounted, computed, ref, type Ref } from 'vue';
import duckApiClient from '@/api/client';
import AlertError from './AlertError.vue';

const store = useMessagesStore()
const { controlEnabled, startRunEnabled, stopRunEnabled, waitForSummaries, nSummariesReceived} = storeToRefs(store)
const errorMsg: Ref<string> = ref("")
const enabledControlButtons: Ref<boolean> = ref(false)

async function startRun() {
  waitForSummaries.value = false
  nSummariesReceived.value = 0

  startRunEnabled.value = false
  controlEnabled.value = false

  try {
    const response = await duckApiClient.startRun({})
    console.log(response)
    errorMsg.value = ""
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
  store.getRunNumber()
}

async function stopRun() {
  waitForSummaries.value = true
  nSummariesReceived.value = 0

  stopRunEnabled.value = false
  controlEnabled.value = false

  try {
    const response = await duckApiClient.stopRun({})
    console.log(response)
    store.restartRun()
    errorMsg.value = ""
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
  store.getRunNumber()
}

async function forceStopRun() {
  waitForSummaries.value = false
  nSummariesReceived.value = 0

  try {
    const response = await duckApiClient.forceStopRun({})
    console.log(response)
    store.restartRun()
    errorMsg.value = ""
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
  store.getRunNumber()
}

const restarting = ref(false)
const restartId = ref("")
const restartProgress = ref<string[]>([])
let timer: ReturnType<typeof setTimeout>
let mounted = true

async function pollRestart() {
  try {
    const status = await duckApiClient.getRunTransitionStatus({})
    if (!mounted) return
    if (restartId.value && !status.restartOperationId) {
      errorMsg.value = "The API lost the restart status. Check the service states."
      await duckApiClient.getProcessStates({})
    }
    restartId.value = status.restartOperationId
    restarting.value = status.state === "restarting"
    restartProgress.value = status.restartTargets.map(t =>
      [t.name, t.host, t.state, t.detail].filter(Boolean).join(" · "))
    if (status.restartError) errorMsg.value = status.restartError
  } catch (error: any) {
    if (restartId.value) errorMsg.value = error.message
  } finally {
    if (mounted) timer = setTimeout(pollRestart, 1000)
  }
}
onMounted(pollRestart)
onUnmounted(() => { mounted = false; clearTimeout(timer) })

async function restartServices() {
  restarting.value = true
  try {
    const response = await duckApiClient.restartServices({})
    restartId.value = response.operationId
    errorMsg.value = ""
  } catch (error: any) {
    restarting.value = false
    errorMsg.value = error.message
  }
}
</script>

<template>
  <div class="bg-slate-300 flex flex-col gap-2 w-fit m-2 p-2 rounded-md">
    <AlertError v-if="errorMsg" :text="`Error: ${errorMsg}`" />

    <div class="flex">
      <h2 class="text-xl font-bold">Run control</h2>
      <div class="form-control mx-2">
        <label class="label flex gap-2" for="lock-controls">
          <input type="checkbox" v-model="enabledControlButtons" id="lock-controls" class="toggle toggle-md" />
          <span v-if="enabledControlButtons" class="label-text">Disable control</span>
          <span v-if="!enabledControlButtons" class="label-text">Enable control</span>
        </label>
      </div>
    </div>

    <ul v-if="restartProgress.length" aria-live="polite"><li v-for="line in restartProgress" :key="line">{{ line }}</li></ul>
    <div class="flex flex-col md:flex-row gap-2">
      <button data-testid="start-run-button" :disabled="restarting || !enabledControlButtons || !controlEnabled || !startRunEnabled || waitForSummaries" @click="startRun" class="btn btn-success">Start run</button>
      <button data-testid="stop-run-button" :disabled="restarting || !enabledControlButtons || !controlEnabled || !stopRunEnabled || waitForSummaries" @click="stopRun" class="btn btn-error">Stop run</button>
      <button data-testid="force-stop-button" :disabled="restarting || !enabledControlButtons" @click="forceStopRun" class="btn btn-error">Force stop</button>
      <button data-testid="restart-services-button" :disabled="restarting || !enabledControlButtons" @click="restartServices" class="btn bg-amber-500 border-amber-600">Restart services</button>
    </div>
  </div>
</template>
