<script setup lang="ts">
const props = defineProps({
  text: { required: true },
})
</script>

<template>
  <div data-testid="alert-warning" class="alert alert-warning my-2 mx-2 text-lg font-bold">
    <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
        d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
    <div><span>{{ text }}.</span></div>
  </div>
</template>