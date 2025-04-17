<script setup>
import { AppWindow, Computer, Power, RefreshCcw, Trash2 } from 'lucide-vue-next'
import { ref } from 'vue'

const memoryUsage = ref(75)
const tempFiles = ref(4.2)
const diskSpace = ref({
  used: 146.4,
  total: 353.3,
})
const startupTime = ref(6)
const runningApps = ref(10)
</script>

<template>
  <div
    class="bg-white rounded-lg p-4 mb-6 flex items-center gap-4 shadow-sm border border-gray-200"
  >
    <div class="w-10 h-10 bg-blue-50 rounded-full flex items-center justify-center">
      <Computer class="w-6 h-6 text-blue-500" />
    </div>
    <div>
      <h3 class="text-gray-800 font-medium">Annoyed by pop-up ads in apps?</h3>
      <p class="text-gray-500 text-sm">
        Turn on Pop-up Management to reduce app pop-up harassment.
      </p>
    </div>
  </div>

  <!-- PC Boost -->
  <div class="bg-white rounded-lg p-4 mb-6 shadow-sm border border-gray-200">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <RefreshCcw class="w-5 h-5 text-gray-500" />
        <span class="text-gray-800 font-medium">PC boost</span>
      </div>
      <button class="text-blue-500 text-sm hover:text-blue-600">Set Smart boost</button>
    </div>
    <p class="text-gray-500 text-sm mb-4">Free up resources to help your PC run faster</p>

    <div class="grid grid-cols-2 gap-4 mb-4">
      <div>
        <div class="flex justify-between text-sm mb-1">
          <span class="text-gray-500">Memory usage</span>
          <span class="text-gray-800">{{ memoryUsage }}%</span>
        </div>
        <div class="h-1 bg-gray-100 rounded-full">
          <div class="h-full bg-blue-500 rounded-full" :style="{ width: `${memoryUsage}%` }"></div>
        </div>
      </div>
      <div>
        <div class="flex justify-between text-sm mb-1">
          <span class="text-gray-500">Temporary files</span>
          <span class="text-gray-800">{{ tempFiles }}GB</span>
        </div>
        <div class="h-1 bg-gray-100 rounded-full">
          <div class="h-full bg-blue-500 rounded-full" style="width: 40%"></div>
        </div>
      </div>
    </div>

    <button
      class="w-full bg-blue-500 hover:bg-blue-600 text-white py-2 rounded-lg transition-colors"
    >
      Boost
    </button>
  </div>

  <!-- Status Grid -->
  <div class="grid grid-cols-2 gap-4">
    <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-200">
      <div class="flex items-center gap-2 mb-4">
        <Computer class="w-5 h-5 text-gray-500" />
        <span class="text-gray-800 font-medium">Health check</span>
      </div>
      <div class="text-sm">
        <div class="text-gray-500">Last check</div>
        <div class="text-gray-800">--</div>
      </div>
    </div>

    <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-200">
      <div class="flex items-center gap-2 mb-4">
        <AppWindow class="w-5 h-5 text-gray-500" />
        <span class="text-gray-800 font-medium">Process</span>
      </div>
      <div class="text-sm">
        <div class="text-gray-500">In progress</div>
        <div class="text-gray-800">{{ runningApps }} Apps</div>
      </div>
    </div>

    <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-200">
      <div class="flex items-center gap-2 mb-4">
        <Trash2 class="w-5 h-5 text-gray-500" />
        <span class="text-gray-800 font-medium">Deep cleanup</span>
      </div>
      <div class="text-sm">
        <div class="text-gray-500">Local Disk (C:)</div>
        <div class="text-gray-800">{{ diskSpace.used }}GB/{{ diskSpace.total }}GB</div>
      </div>
    </div>

    <div class="bg-white rounded-lg p-4 shadow-sm border border-gray-200">
      <div class="flex items-center gap-2 mb-4">
        <Power class="w-5 h-5 text-gray-500" />
        <span class="text-gray-800 font-medium">Startup</span>
      </div>
      <div class="text-sm">
        <div class="text-gray-500">Start time</div>
        <div class="text-gray-800">{{ startupTime }} Sec</div>
      </div>
    </div>
  </div>
</template>
