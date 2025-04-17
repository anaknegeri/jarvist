<script setup lang="ts">
import logo from "@/assets/logo-white.svg";
import { Application } from "@wailsio/runtime";
import { X } from "lucide-vue-next";

const titleBarStore = useTitleBarStore();

const appVersion = ref<string>("1.0.0");
const copyright = ref<string>("© 2025, Pitjarus Teknologi");

const windowClose = async () => {
  Application.Quit();
};

const loadConfig = async () => {
  appVersion.value = await GetProductVersion();
  copyright.value = await GetCopyright();
};

onMounted(async () => {
  loadConfig();
});
</script>

<template>
  <div
    class="select-none h-screen w-full flex flex-col items-center justify-center bg-gray-100 dark:bg-gray-800"
  >
    <div
      class="bg-[#0078d7] text-white flex justify-between items-center h-10 w-full"
    >
      <div
        class="w-full bg-gradient-to-r from-indigo-600 to-indigo-800 text-white px-4 py-2 flex items-center justify-between cursor-move"
        style="--wails-draggable: drag"
      >
        <div class="flex items-center gap-2">
          <img :src="logo" class="h-4" />
          <span class="font-medium">{{ titleBarStore.title }}</span>
        </div>
        <div class="flex items-center gap-1">
          <!-- <button @click="toggleMinimize" class="p-1 hover:bg-white/10 rounded">
            <Minimize2 v-if="!isMinimized" class="w-4 h-4" />
            <Maximize2 v-else class="w-4 h-4" />
          </button>
          <button
            @click="toggleFullscreen"
            class="p-1 hover:bg-white/10 rounded"
          >
            <Maximize2 class="w-4 h-4" />
          </button> -->
          <button class="p-1 hover:bg-red-500 rounded" @click="windowClose">
            <X class="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
    <div class="flex-1 flex flex-col overflow-hidden w-full">
      <div class="flex-1 overflow-auto p-4">
        <router-view />
      </div>
      <div
        class="mt-5 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 px-4 py-3"
      >
        <span>Version v{{ appVersion }}</span>
        <span>{{ copyright }}</span>
      </div>
    </div>
  </div>
</template>
