<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@iconify/vue";
import { Loader2, Play } from "lucide-vue-next";
import { computed, ref, watch } from "vue";

const props = defineProps<{
  serviceName: string;
  serviceFileName: string;
  description: string;
  iconName: string;
  iconColorClass: string;
  bgColorClass: string;
  isRunning: boolean;
  status: string;
  statusClass: string;
}>();

const emit = defineEmits<{
  (e: "toggle", fileName: string): void;
}>();

const isActionInProgress = ref(false);

// Reset isActionInProgress when status changes to a final state
watch(
  () => props.status,
  (newStatus) => {
    if (
      newStatus === "Stopped" ||
      newStatus === "Error" ||
      newStatus === "Idle" ||
      newStatus === "Running" ||
      newStatus === "Completed"
    ) {
      isActionInProgress.value = false;
    }
  },
  { immediate: true }
);

const handleToggle = () => {
  isActionInProgress.value = true;
  emit("toggle", props.serviceFileName);

  // Safety timeout to reset button if event doesn't come back
  setTimeout(() => {
    isActionInProgress.value = false;
  }, 10000); // Longer timeout
};

// Check if status indicates it's running
const effectivelyRunning = computed(() => {
  return (
    props.isRunning ||
    props.status === "Running" ||
    props.status === "Initializing" ||
    props.status === "Loading" ||
    props.status === "Starting..."
  );
});

// Determine button text
const buttonText = computed(() => {
  if (isActionInProgress.value) {
    return effectivelyRunning.value ? "Stopping..." : "Starting...";
  }

  if (props.status === "Stopping...") return "Stopping...";
  if (props.status === "Starting...") return "Starting...";

  return effectivelyRunning.value ? "Stop" : "Start";
});

// Determine button variant
const buttonVariant = computed(() => {
  if (isActionInProgress.value) {
    return effectivelyRunning.value ? "destructive" : "default";
  }

  if (props.status === "Stopping...") return "destructive";
  if (props.status === "Starting...") return "default";

  return effectivelyRunning.value ? "destructive" : "default";
});

// Determine if button should be disabled
const isButtonDisabled = computed(() => {
  return (
    isActionInProgress.value ||
    props.status === "Stopping..." ||
    props.status === "Starting..."
  );
});

// Determine status bar color
const statusBarClass = computed(() => {
  switch (props.status) {
    case "Error":
      return "bg-red-500 dark:bg-red-400";
    case "Initializing":
    case "Starting...":
    case "Loading":
      return "bg-yellow-500 dark:bg-yellow-400";
    case "Running":
      return "bg-green-500 dark:bg-green-400";
    case "Stopping...":
      return "bg-orange-500 dark:bg-orange-400";
    default:
      return "bg-gray-300 dark:bg-gray-600";
  }
});

// Format status for display
const displayStatus = computed(() => {
  // Return status as-is
  return props.status;
});
</script>

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
            <span class="text-gray-800 dark:text-gray-200 font-medium block">{{
              serviceName
            }}</span>
            <span class="text-xs text-muted-foreground">{{ description }}</span>
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
        @click="handleToggle"
        :variant="buttonVariant"
        class="min-w-[80px]"
        :disabled="isButtonDisabled"
      >
        <template v-if="isButtonDisabled">
          <Loader2 class="w-3.5 h-3.5 animate-spin" />
          {{ buttonText }}
        </template>
        <template v-else>
          <Icon icon="ion:stop" v-if="effectivelyRunning" />
          <Play v-else class="w-3.5 h-3.5" />
          {{ buttonText }}
        </template>
      </Button>
    </div>
  </div>
</template>
