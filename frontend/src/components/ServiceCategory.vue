<script setup lang="ts">
import {
  getStatusClass,
  processStatus,
} from "@/services/processManagerService";
import ServiceModule from "./ServiceModule.vue";

// Define ServiceStatus interface
interface ServiceStatus {
  id: string;
  name: string;
  description: string;
  icon: string | any;
  status: "active" | "inactive" | "error" | "coming-soon";
  lastStarted?: string;
  version?: string;
  type: "analytics" | "detection" | "counting";
  available: boolean;
  batFileName: string;
}

defineProps<{
  title: string;
  services: ServiceStatus[];
}>();

const emit = defineEmits<{
  (e: "action", serive: ServiceStatus): void;
  (e: "settings", id: string): void;
}>();

function isProcessRunning(fileName: string): boolean {
  return (
    !!processStatus.value &&
    !!processStatus.value[fileName] &&
    !!processStatus.value[fileName].running
  );
}

function getProcessStatus(fileName: string): string {
  if (!processStatus.value || !processStatus.value[fileName]) {
    return "Idle";
  }
  return processStatus.value[fileName].status || "Idle";
}

const handleAction = (serive: ServiceStatus) => {
  emit("action", serive);
};

const handleSettings = (id: string) => {
  emit("settings", id);
};
</script>

<template>
  <div>
    <h3 class="text-sm font-medium text-gray-700 mb-2">
      {{ title }}
    </h3>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <ServiceModule
        v-for="service in services"
        :key="service.id"
        :service="service"
        :isRunning="isProcessRunning(service.batFileName)"
        :status="getProcessStatus(service.batFileName)"
        :statusClass="getStatusClass(service.batFileName)"
        @action="handleAction(service)"
        @settings="handleSettings"
      />
    </div>
  </div>
</template>
