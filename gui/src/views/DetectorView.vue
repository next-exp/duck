<script setup lang="ts">
import { TabGroup, TabList, Tab, TabPanels, TabPanel } from '@headlessui/vue'
import GDCs from '@/components/GDCs.vue';
import LDCs from '@/components/LDCs.vue';
import Equipments from '@/components/Equipments.vue';
import Decoder from '@/components/Decoder.vue';
import Topi from '@/components/Topi.vue';
</script>

<template>
  <main>
    <div class="flex flex-col gap-2">
      <TabGroup>
        <div class="w-fit">
          <TabList class="tabs tabs-bordered">
            <Tab v-slot="{ selected }" class="tab tab-lg ui-selected:tab-active">
              <span class="text-xl" :class="{ 'font-bold': selected }">GDCs</span>
            </Tab>
            <Tab v-slot="{ selected }" class="tab tab-lg ui-selected:tab-active">
              <span class="text-xl" :class="{ 'font-bold': selected }">LDCs</span>
            </Tab>
            <Tab v-slot="{ selected }" class="tab tab-lg ui-selected:tab-active">
              <span class="text-xl" :class="{ 'font-bold': selected }">Equipments</span>
            </Tab>
            <Tab v-slot="{ selected }" class="tab tab-lg ui-selected:tab-active">
              <span class="text-xl" :class="{ 'font-bold': selected }">Decoder</span>
            </Tab>
            <Tab v-slot="{ selected }" class="tab tab-lg ui-selected:tab-active">
              <span class="text-xl" :class="{ 'font-bold': selected }">TOPI</span>
            </Tab>
          </TabList>
        </div>
        <TabPanels>
          <TabPanel>
            <GDCs />
          </TabPanel>
          <TabPanel>
            <LDCs />
          </TabPanel>
          <TabPanel>
            <Equipments />
          </TabPanel>
          <TabPanel>
            <Decoder/>
          </TabPanel>
          <TabPanel>
            <Topi />
          </TabPanel>
        </TabPanels>
      </TabGroup>
    </div>
  </main>
</template>
