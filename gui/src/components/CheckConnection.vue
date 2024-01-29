<script setup lang="ts">
import AlertWarning from './AlertWarning.vue';
import AlertError from './AlertError.vue';
import { useMessagesStore } from "@/stores/messages";
import { storeToRefs } from 'pinia'

const store = useMessagesStore()
const { errorCentrifuge, warningCentrifuge } = storeToRefs(store)

</script>

<template>
  <div class="flex gap-1 w-fit m-2 p-2 px-4 rounded-md mx-auto">
    <AlertError v-if="errorCentrifuge" :text="`${errorCentrifuge}`" />
    <AlertWarning v-if="warningCentrifuge" :text="`${warningCentrifuge}`" />
  </div>
</template>
