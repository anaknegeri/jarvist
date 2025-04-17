<script setup lang="ts">
import { AlertTriangle, CheckCircle2, Circle } from "lucide-vue-next";

const props = defineProps<{
  status: boolean | undefined;
}>();

const getStatusInfo = () => {
  if (props.status === true) {
    return {
      text: "Online",
      variant: "default" as const,
      color: "text-green-600 dark:text-green-400",
      fillColor: "fill-green-500 dark:fill-green-400",
      dotColor: "bg-green-500",
      icon: CheckCircle2,
      bgClass:
        "bg-green-100 dark:bg-green-900/20 text-green-700 dark:text-green-400",
    };
  } else if (props.status === false) {
    return {
      text: "Offline",
      variant: "destructive" as const,
      color: "text-red-600 dark:text-red-400",
      fillColor: "fill-red-500 dark:fill-red-400",
      dotColor: "bg-red-500",
      icon: AlertTriangle,
      bgClass: "bg-red-100 dark:bg-red-900/20 text-red-700 dark:text-red-400",
    };
  } else {
    return {
      text: "Unknown",
      variant: "outline" as const,
      color: "text-gray-600 dark:text-gray-400",
      fillColor: "fill-gray-500 dark:fill-gray-400",
      dotColor: "bg-gray-400",
      icon: Circle,
      bgClass: "bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400",
    };
  }
};
</script>

<template>
  <div
    class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
    :class="getStatusInfo().bgClass"
  >
    <div
      class="w-1.5 h-1.5 rounded-full mr-1"
      :class="getStatusInfo().dotColor"
    ></div>
    {{ getStatusInfo().text }}
  </div>
</template>
