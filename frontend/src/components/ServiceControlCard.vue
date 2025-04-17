<!-- ServiceControlCard.vue -->
<template>
  <div class="relative bg-muted/30 rounded-lg p-4 border overflow-hidden group">
    <!-- Service status indicator -->
    <div
      class="absolute top-0 bottom-0 left-0 w-1"
      :class="statusBarClass"
    ></div>

    <div class="flex justify-between items-start">
      <div class="pl-2">
        <div class="flex items-center gap-2 mb-3">
          <div
            class="w-10 h-10 rounded-full flex items-center justify-center"
            :class="bgColorClass"
          >
            <Icon :icon="iconName" class="w-5 h-5" :class="iconColorClass" />
          </div>
          <div>
            <span class="text-gray-800 dark:text-gray-200 font-medium block">
              {{ serviceName }}
            </span>
            <span class="text-xs text-muted-foreground">{{
              serviceDescription
            }}</span>
          </div>
        </div>

        <div class="ml-1">
          <Badge :class="statusClass" variant="outline" class="mb-2">
            {{ displayStatus }}
          </Badge>
        </div>
      </div>

      <!-- Action button -->
      <Button
        size="sm"
        @click="toggleService"
        :variant="buttonVariant"
        class="min-w-[80px]"
        :disabled="isLoading"
      >
        <template v-if="isLoading">
          <Loader2 class="w-3.5 h-3.5 animate-spin" />
          Loading...
        </template>
        <template v-else>
          <Icon icon="ion:stop" v-if="isRunning" />
          <Play v-else class="w-3.5 h-3.5" />
          {{ actionText }}
        </template>
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@iconify/vue";
import { Loader2, Play } from "lucide-vue-next";
import { computed, onMounted, onUnmounted, ref } from "vue";

// Props
interface Props {
  serviceName?: string;
  serviceDescription?: string;
  autoRefresh?: boolean;
  refreshInterval?: number;
  iconName: string;
  iconColorClass: string;
  bgColorClass: string;
}

const props = withDefaults(defineProps<Props>(), {
  serviceName: "Sync Service",
  serviceDescription: "Windows background service",
  autoRefresh: true,
  refreshInterval: 15000,
});

// State
const serviceStatus = ref<string>("Unknown");
const isLoading = ref<boolean>(false);
const refreshTimer = ref<number | null>(null);

const isRunning = computed(() => serviceStatus.value === "Running");
const isInstalled = computed(
  () =>
    serviceStatus.value !== "Not installed" && serviceStatus.value !== "Unknown"
);

const displayStatus = computed(() => {
  switch (serviceStatus.value) {
    case "Running":
      return "Running";
    case "Stopped":
      return "Stopped";
    case "Not installed":
      return "Not Installed";
    default:
      return "Unknown";
  }
});

const statusClass = computed(() => {
  switch (serviceStatus.value) {
    case "Running":
      return "bg-green-100 hover:bg-green-100 text-green-800 hover:text-green-800";
    case "Stopped":
    case "Not installed":
    default:
      return "bg-gray-100 hover:bg-gray-100 text-gray-800 hover:text-gray-800";
  }
});

const statusBarClass = computed(() => {
  switch (serviceStatus.value) {
    case "Running":
      return "bg-green-500 dark:bg-green-400";
    case "Stopped":
    case "Not installed":
    default:
      return "bg-gray-300 dark:bg-gray-600";
  }
});

const actionText = computed(() => {
  return isRunning.value ? "Stop" : "Start";
});

const buttonVariant = computed(() => {
  return isRunning.value ? "destructive" : "default";
});

// Methods
const fetchServiceStatus = async (silent: boolean = false): Promise<void> => {
  if (!silent) {
    isLoading.value = true;
  }

  try {
    const status = await ServiceManager.GetServiceStatus();
    serviceStatus.value = status;
  } catch (error) {
    console.error("Error fetching service status:", error);
    serviceStatus.value = "Error";
  } finally {
    if (!silent) {
      isLoading.value = false;
    }
  }
};

const startService = async (): Promise<void> => {
  isLoading.value = true;

  try {
    await ServiceManager.EnsureServiceRunning();
    await fetchServiceStatus(true);
  } catch (error) {
    console.error("Error starting service:", error);
  } finally {
    isLoading.value = false;
  }
};

const stopService = async (): Promise<void> => {
  isLoading.value = true;

  try {
    await ServiceManager.StopService();
    await fetchServiceStatus(true);
  } catch (error) {
    console.error("Error stopping service:", error);
  } finally {
    isLoading.value = false;
  }
};

const toggleService = async (): Promise<void> => {
  if (isLoading.value) return;

  if (isRunning.value) {
    await stopService();
  } else {
    await startService();
  }
};

// Lifecycle hooks
onMounted(async () => {
  await fetchServiceStatus();

  if (props.autoRefresh) {
    refreshTimer.value = window.setInterval(() => {
      fetchServiceStatus(true);
    }, props.refreshInterval);
  }
});

onUnmounted(() => {
  if (refreshTimer.value !== null) {
    clearInterval(refreshTimer.value);
  }
});
</script>
