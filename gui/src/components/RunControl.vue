<script setup lang="ts">
import { useMessagesStore } from "@/stores/messages";
import { storeToRefs } from 'pinia'
import { onMounted, computed, ref, type Ref } from 'vue';
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

async function restartServices() {
  waitForSummaries.value = false
  nSummariesReceived.value = 0

  try {
    const response = await duckApiClient.restartServices({})
    console.log(response)
    errorMsg.value = ""
  } catch (error: any) {
    console.log(error)
    errorMsg.value = error.message
  }
  store.getRunNumber()
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

    <div class="flex flex-col md:flex-row gap-2">
      <button data-testid="start-run-button" :disabled="!enabledControlButtons || !controlEnabled || !startRunEnabled || waitForSummaries" @click="startRun" class="btn btn-success">Start run</button>
      <button data-testid="stop-run-button" :disabled="!enabledControlButtons || !controlEnabled || !stopRunEnabled || waitForSummaries" @click="stopRun" class="btn btn-error">Stop run</button>
      <button data-testid="force-stop-button" :disabled="!enabledControlButtons" @click="forceStopRun" class="btn btn-error">Force stop</button>
      <button data-testid="restart-services-button" :disabled="!enabledControlButtons" @click="restartServices" class="btn bg-amber-500 border-amber-600">Restart services</button>
    </div>
  </div>
</template>
