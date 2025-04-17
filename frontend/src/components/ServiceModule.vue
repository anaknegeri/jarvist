<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Icon } from "@iconify/vue";
import { MonitorPlay, Play } from "lucide-vue-next";

// Define ServiceStatus interface
interface ServiceStatus {
  id: string;
  name: string;
  description: string;
  icon: string | any;
  status: "active" | "inactive" | "error" | "coming-soon";
  version?: string;
  type: "analytics" | "detection" | "counting";
  configuration?: Record<string, any>;
  available: boolean;
}

const props = defineProps<{
  service: ServiceStatus;
  isRunning: boolean;
  status: string;
  statusClass: string;
}>();

const emit = defineEmits<{
  (e: "action", id: string): void;
  (e: "settings", id: string): void;
}>();

const handleAction = () => {
  emit("action", props.service.id);
};

const openSettings = () => {
  emit("settings", props.service.id);
};

const formatMemory = (memoryInMB: number) => {
  return `${memoryInMB} MB`;
};
</script>

<template>
  <Card class="relative">
    <div
      v-if="!service.available"
      class="absolute inset-0 bg-gray-50/80 rounded backdrop-blur-[0.1px] z-10 flex flex-col items-center justify-center"
    >
      <Badge
        class="bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400 mb-1"
      >
        Coming Soon
      </Badge>
    </div>
    <CardHeader class="px-4 py-3 flex flex-row justify-between items-center">
      <div class="flex items-center gap-2">
        <Icon :icon="service.icon" class="w-5 h-5 text-gray-600" />
        <CardTitle class="text-sm font-medium">{{ service.name }}</CardTitle>
      </div>
      <Badge :class="statusClass" v-if="service.available">
        {{ status }}
      </Badge>
      <Badge
        class="bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400"
        v-else
      >
        Coming Soon
      </Badge>
    </CardHeader>
    <CardContent class="p-4">
      <p class="text-xs text-gray-500 mb-3">
        {{ service.description }}
      </p>
      <div class="flex justify-between items-center">
        <div></div>
        <div class="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            @click="openSettings"
            :disabled="!service.available"
          >
            <MonitorPlay class="w-3.5 h-3.5" />
          </Button>

          <Button
            size="sm"
            @click="handleAction"
            :variant="isRunning ? 'destructive' : 'default'"
            :disabled="!service.available"
            class="min-w-[80px]"
          >
            <Icon icon="ion:stop" v-if="isRunning" class="mr-1.5" />
            <Play v-else class="w-3.5 h-3.5 mr-1.5" />
            {{ isRunning ? "Stop" : "Start" }}
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
