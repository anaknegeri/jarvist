<script setup lang="ts">
import {
  listCameras,
  setupConnectionStatusListener,
} from "@/services/cameraService";
import {
  checkRunningProcesses,
  processStatus,
  runBatFile,
  stopProcess,
} from "@/services/processManagerService";
import { Shield } from "lucide-vue-next";
import { computed, onMounted, ref } from "vue";

import { getCurrentTime } from "@/lib/common";

const router = useRouter();
const isLoading = ref(false);
const lastSyncTime = ref(getCurrentTime());

// Define service interface
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

// Service state
const services = ref<ServiceStatus[]>([
  {
    id: "people-counting",
    name: "People Counting",
    description: "Track and count people passing through designated areas",
    icon: "fa6-solid:person-walking-dashed-line-arrow-right",
    status: "inactive",
    lastStarted: "2025-02-25T14:30:00",
    version: "1.3.2",
    type: "counting",
    available: true,
    batFileName: "people_counter.bat",
  },
  {
    id: "motion-detection",
    name: "Motion Detection",
    description: "Detect motion in monitored areas and trigger alerts",
    icon: "mdi:motion-sensor",
    status: "coming-soon",
    lastStarted: "2025-02-26T08:15:00",
    version: "2.1.0",
    type: "detection",
    available: false,
    batFileName: "test.bat",
  },
  {
    id: "object-recognition",
    name: "Object Recognition",
    description: "Identify and classify objects in the camera view",
    icon: "mdi:cube-scan",
    status: "coming-soon",
    lastStarted: "2025-02-25T09:45:00",
    version: "1.8.5",
    type: "detection",
    available: false,
    batFileName: "test.bat",
  },
  {
    id: "traffic-analysis",
    name: "Traffic Analysis",
    description: "Analyze traffic patterns and generate flow reports",
    icon: "mdi:traffic-light",
    status: "coming-soon",
    lastStarted: "2025-02-26T10:20:00",
    version: "1.2.0",
    type: "analytics",
    available: false,
    batFileName: "test.bat",
  },
  {
    id: "occupancy-tracking",
    name: "Occupancy Tracking",
    description: "Monitor and report on zone occupancy in real-time",
    icon: "mdi:account-group",
    status: "coming-soon",
    lastStarted: "2025-02-24T16:10:00",
    version: "1.1.7",
    type: "counting",
    available: false,
    batFileName: "test.bat",
  },
  {
    id: "scheduler-sync",
    name: "Scheduler Sync",
    description: "Synchronize and schedule AI processing tasks",
    icon: "mdi:file-sync",
    status: "inactive",
    lastStarted: "2025-02-26T09:00:00",
    version: "1.4.1",
    type: "analytics",
    available: true,
    batFileName: "sync_manager.bat",
  },
]);

const countingServices = computed(() =>
  services.value.filter((s) => s.type === "counting")
);

const detectionServices = computed(() =>
  services.value.filter((s) => s.type === "detection")
);

const analyticsServices = computed(() =>
  services.value.filter((s) => s.type === "analytics")
);

const handleServiceAction = (service: ServiceStatus) => {
  if (service.available) {
    const fileName = service.batFileName;
    if (processStatus.value && processStatus.value[fileName]?.running) {
      stopProcess(fileName)
        .then(() => {
          console.log(`Successfully stopped ${fileName}`);
          lastSyncTime.value = getCurrentTime();
        })
        .catch((err) => {
          console.error(`Failed to stop ${fileName}:`, err);
        });
    } else {
      runBatFile(fileName)
        .then(() => {
          console.log(`Successfully started ${fileName}`);
          lastSyncTime.value = getCurrentTime();
        })
        .catch((err) => {
          console.error(`Failed to start ${fileName}:`, err);
        });
    }
  }
};

const openServiceSettings = (id: string) => {
  router.push(`jarvist/view/${id}`);
};

function isProcessRunning(fileName: string): boolean {
  return (
    !!processStatus.value &&
    !!processStatus.value[fileName] &&
    !!processStatus.value[fileName].running
  );
}

const activeServicesCount = computed(() => {
  let count = 0;
  services.value.forEach((service: ServiceStatus) => {
    if (service.available) {
      if (isProcessRunning(service.batFileName)) count++;
    }
  });
  return count;
});

onMounted(async () => {
  isLoading.value = true;
  await Promise.all([checkRunningProcesses(), listCameras()]);
  isLoading.value = false;

  setupConnectionStatusListener();
});
</script>

<template>
  <div class="flex flex-col h-full space-y-4">
    <div
      class="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-md shadow-sm border border-gray-200"
    >
      <div class="flex items-center gap-3">
        <div
          class="w-10 h-10 bg-indigo-50 rounded-full flex items-center justify-center"
        >
          <Shield class="w-5 h-5 text-indigo-500" />
        </div>
        <div>
          <h2 class="text-xl font-semibold text-gray-800 dark:text-gray-100">
            AI Services
          </h2>
          <p class="text-xs text-muted-foreground mt-1">
            Manage and monitor AI processing services
          </p>
        </div>
      </div>
      <div class="flex items-center space-x-2">
        <div
          class="text-xs text-muted-foreground flex items-center bg-muted/40 px-2 py-1 rounded"
        >
          <div class="w-2 h-2 bg-green-500 rounded-full mr-1.5"></div>
          <span>{{ activeServicesCount }} services running</span>
        </div>
      </div>
    </div>

    <!-- Main content area -->
    <div class="flex-1 overflow-auto">
      <!-- Services list -->
      <div class="space-y-4">
        <!-- Service categories -->
        <ServiceCategory
          title="Counting Services"
          :services="countingServices"
          @action="handleServiceAction"
          @settings="openServiceSettings"
        />

        <ServiceCategory
          title="Detection Services"
          :services="detectionServices"
          @action="handleServiceAction"
          @settings="openServiceSettings"
        />

        <ServiceCategory
          title="Analytics Services"
          :services="analyticsServices"
          @action="handleServiceAction"
          @settings="openServiceSettings"
        />
      </div>
    </div>
  </div>
</template>
