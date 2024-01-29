<script setup lang="ts">
const props = defineProps({
  text: { required: true },
})
</script>

<template>
  <div data-testid="alert-success" class="alert alert-success my-2 mx-2 text-lg font-bold">
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor"
      class="size-6">
      <path stroke-linecap="round" stroke-linejoin="round"
        d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
    </svg>
    <div><span>{{ text }}.</span></div>
  </div>
</template>