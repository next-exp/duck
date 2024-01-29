<script setup lang="ts">
import { onMounted, ref } from "vue";
import type { Ref } from "vue";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import { useMessagesStore } from "@/stores/messages";
import { storeToRefs } from 'pinia'

dayjs.extend(utc)
dayjs.extend(timezone)

function parseDates(date: string, lscTime: boolean): string {
  let result = "—";
  if (date) {
    if (lscTime) {
      result = dayjs(date).tz("Europe/Madrid").format("DD/MM/YY - HH:mm:ss.SSS");
    } else {
      result = dayjs(date).format("DD/MM/YY - HH:mm:ss.SSS");
    }
  }
  return result;
}
const store = useMessagesStore()
const { messages, messagesDebug } = storeToRefs(store)

const showDebug: Ref<boolean> = ref(false)
</script>

<template>
  <div class="bg-slate-300 rounded-lg md:w-2/3 mx-auto p-2">
    <h2 class="text-xl font-bold m-4">Messages</h2>
    <div class="bg-gray-200 rounded-lg m-1 md:m-4 overflow-scroll">
      <p v-for="message in [...messages].reverse()" class="mx-2 my-1 whitespace-pre-wrap" :class="{ 'text-red-600': message.type == 'error' }">{{
        parseDates(message.timestamp,
          true) }} - {{ message.type }} {{ message.host }}: {{ message.value }}
      </p>
    </div>
  </div>

  <div class="bg-slate-300 rounded-lg md:w-2/3 mx-auto p-2 m-4">
    <div class="flex">
      <h2 class="text-xl font-bold m-4">Debugging</h2>
      <div class="form-control mx-2">
        <label class="label" for="show-debugging">
          <span class="label-text">Show messages</span>
        </label>
        <input type="checkbox" v-model="showDebug" id="show-debugging" class="toggle toggle-md" />
      </div>
    </div>
    <div v-if="showDebug" class="bg-gray-200 rounded-lg m-1 md:m-4 overflow-scroll">
      <p v-for="message in [...messagesDebug].reverse()" class="mx-2 my-1 whitespace-pre-wrap">{{
        parseDates(message.timestamp, true) }} - {{ message.host }}: {{ message.value }}
      </p>
    </div>
  </div>
</template>
