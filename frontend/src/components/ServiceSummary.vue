<script setup lang="ts">
import { Card, CardContent } from "@/components/ui/card";
import { Icon } from "@iconify/vue";
import { computed } from "vue";

// Define ServiceStatus interface
interface ServiceStatus {
  id: string;
  name: string;
  description: string;
  icon: string | any;
  status: "active" | "inactive" | "error" | "coming-soon";
  lastStarted?: string;
  cpuUsage?: number;
  memoryUsage?: number;
  version?: string;
  type: "analytics" | "detection" | "counting";
  configuration?: Record<string, any>;
  available: boolean;
}

const props = defineProps<{
  title: string;
  icon: any;
  iconColor: string;
  iconBgColor: string;
  services: ServiceStatus[];
}>();

// Calculate active services
const activeCount = computed(() => {
  return props.services.filter((s) => s.status === "active").length;
});

// Calculate available services
const availableCount = computed(() => {
  return props.services.filter((s) => s.available).length;
});

// Calculate coming soon services
const comingSoonCount = computed(() => {
  return props.services.filter((s) => !s.available).length;
});
</script>

<template>
  <Card class="shadow-sm border-gray-200">
    <CardContent class="p-4 flex items-center gap-4">
      <div
        class="w-10 h-10 rounded-full flex items-center justify-center"
        :class="iconBgColor"
      >
        <component
          :is="icon"
          v-if="typeof icon !== 'string'"
          class="w-5 h-5"
          :class="iconColor"
        />
        <Icon v-else :icon="icon" class="w-5 h-5" :class="iconColor" />
      </div>
      <div>
        <h3 class="text-gray-800 font-medium text-sm">{{ title }}</h3>
        <p class="text-gray-500 text-xs mt-0.5">
          {{ activeCount }} of {{ availableCount }} available
          <span v-if="comingSoonCount > 0" class="text-blue-600"
            >(+{{ comingSoonCount }} coming soon)</span
          >
        </p>
      </div>
    </CardContent>
  </Card>
</template>
