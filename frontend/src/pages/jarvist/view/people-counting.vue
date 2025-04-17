<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { ArrowLeft } from "lucide-vue-next";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";

import {
  checkRunningProcesses,
  processStatus,
} from "@/services/processManagerService";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

const router = useRouter();
const isLoading = ref(false);

const videoFeed = ref({
  url: "",
  status: "inactive",
  isLoading: false,
});

const serviceData = ref({
  id: "people-counting",
  name: "People Counting",
  description: "AI-powered people detection and counting service",
  icon: "fa6-solid:person-walking-dashed-line-arrow-right",
  batFile: "people_counter.bat",
});

// Status stream dari Wails backend
const isStreamRunning = ref(false);

// Komputer status gabungan
const isServiceRunning = computed(() => {
  // Periksa baik status proses bat maupun status stream
  const isBatRunning =
    !!processStatus.value &&
    !!processStatus.value[serviceData.value.batFile] &&
    !!processStatus.value[serviceData.value.batFile].running;

  return isBatRunning || isStreamRunning.value;
});

const goBack = () => {
  router.push("/jarvist");
};

const statusInterval = ref<number | null>(null);

onMounted(async () => {
  isLoading.value = true;

  // Initial check of process status
  await checkRunningProcesses();

  // Periksa status stream
  try {
    isStreamRunning.value = await IsStreamRunning();

    // Jika stream tidak berjalan, start stream
    if (!isStreamRunning.value) {
      console.log("Starting stream...");
      await StartStream();
      isStreamRunning.value = true;
    }

    // Dapatkan URL stream
    videoFeed.value.url = await GetStreamURL();
    videoFeed.value.status = "active";

    // Set interval untuk memeriksa status stream
    statusInterval.value = window.setInterval(async () => {
      const wasRunning = isStreamRunning.value;
      isStreamRunning.value = await IsStreamRunning();

      // Jika stream berhenti berjalan, coba restart
      if (!isStreamRunning.value && wasRunning) {
        console.log("Stream stopped unexpectedly, attempting to restart...");
        videoFeed.value.isLoading = true;

        try {
          await StartStream();
          console.log("Stream restarted successfully");
          videoFeed.value.isLoading = false;
        } catch (error) {
          console.error("Failed to restart stream:", error);
          videoFeed.value.isLoading = false;
        }
      }
    }, 5000);
  } catch (error) {
    console.error("Error checking stream status:", error);
  }

  isLoading.value = false;
});

onBeforeUnmount(() => {
  // Clean up interval
  if (statusInterval.value) {
    clearInterval(statusInterval.value);
  }
});

// Generate a unique query string to avoid browser cache
// const getUniqueStreamURL = () => {
//   return `${videoFeed.value.url}?t=${Date.now()}`;
// };
</script>

<template>
  <div class="flex flex-col h-full space-y-4">
    <!-- Simple header -->
    <div
      class="flex items-center justify-between bg-white dark:bg-gray-800 rounded-md shadow-sm border border-gray-200 p-3"
    >
      <div class="flex items-center gap-3">
        <Button variant="ghost" size="sm" @click="goBack">
          <ArrowLeft class="h-4 w-4" />
        </Button>

        <div class="flex items-center gap-3">
          <div
            class="w-10 h-10 bg-blue-50 dark:bg-blue-900/20 rounded-full flex items-center justify-center"
          >
            <Icon
              :icon="serviceData.icon"
              class="h-5 w-5 text-blue-500 dark:text-blue-400"
            />
          </div>
          <div>
            <h2 class="text-lg font-medium text-gray-800 dark:text-gray-100">
              {{ serviceData.name }}
            </h2>
          </div>
        </div>
      </div>

      <div class="flex items-center">
        <Badge
          :class="
            isServiceRunning
              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
              : 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400'
          "
          class="mr-3"
        >
          {{ isServiceRunning ? "Running" : "Stopped" }}
        </Badge>
      </div>
    </div>

    <!-- Video feed card -->
    <Card class="flex-1 flex flex-col">
      <CardContent class="p-1 flex-1 flex flex-col">
        <div
          v-if="videoFeed.isLoading"
          class="flex-1 bg-gray-100 dark:bg-gray-800 rounded flex items-center justify-center"
        >
          <div class="text-center">
            <div
              class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent align-[-0.125em] motion-reduce:animate-[spin_1.5s_linear_infinite]"
            ></div>
            <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
              Connecting to camera...
            </p>
          </div>
        </div>

        <div
          class="flex-1 bg-gray-900 rounded overflow-hidden relative flex justify-center"
        >
          <img
            :src="videoFeed.url"
            class="h-full max-h-full w-auto mx-auto object-contain"
            alt="Live Camera Feed"
          />

          <div
            class="absolute top-2 right-2 bg-red-600 px-2 py-1 rounded text-white text-xs flex items-center"
          >
            <span
              class="animate-pulse w-2 h-2 bg-white rounded-full mr-1"
            ></span>
            LIVE
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
