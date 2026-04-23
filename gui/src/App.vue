<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'

const enableTestDevices = import.meta.env.VITE_ENABLE_TEST_DEVICES === 'true'
</script>

<template>
  <header>
    <div class="wrapper">
      <div class="navbar bg-base-300">
        <div class="flex-1">
          <RouterLink to="/" class="btn btn-ghost text-xl">NEXT Duck</RouterLink>
        </div>
        <div class="flex-none">
          <ul class="menu menu-horizontal px-1">
            <li>
              <RouterLink to="/">Data taking</RouterLink>
            </li>
            <li>
              <RouterLink to="/detector">Configuration</RouterLink>
            </li>
            <li v-if="enableTestDevices">
              <RouterLink to="/test-devices">Test Devices</RouterLink>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </header>
  <RouterView />
</template>