<script setup lang="ts">
import {
  camerasState,
  checkCameraConnection,
  listCameras,
  setupConnectionStatusListener,
} from "@/services/cameraService";
import {
  checkRunningProcesses,
  getStatusClass,
  processStatus,
  registerEventHandlers,
  runBatFile,
  stopProcess,
} from "@/services/processManagerService";
import { Clock, Cpu, MonitorPlay, Plus } from "lucide-vue-next";
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRouter } from "vue-router";

// Import utility functions
import { getCurrentDate, getCurrentTime } from "@/lib/common";

// Import custom components
import CameraCard from "@/components/CameraCard.vue";
import EmptyState from "@/components/EmptyState.vue";
import ServiceCard from "@/components/ServiceCard.vue";
import ServiceControlCard from "@/components/ServiceControlCard.vue";

// Import shadcn components
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Events } from "@wailsio/runtime";

const router = useRouter();
const isLoading = ref(false);
const lastSyncTime = ref(getCurrentTime());
const refreshingCamera = ref<number | null>(null);

// Add new camera
const addNewCamera = () => {
  router.push("/camera/create");
};

const viewCameraDetails = (id: number) => {
  router.push(`/camera/view/${id}`);
};

const refreshCameraConnection = async (id: number, event: Event) => {
  // Prevent click from bubbling to parent elements
  event.stopPropagation();

  refreshingCamera.value = id;
  await checkCameraConnection(id);
  refreshingCamera.value = null;
};

// Function to handle process actions (Start/Stop)
function handleProcessAction(fileName: string): void {
  // Make sure processStatus.value[fileName] exists before accessing running property
  if (processStatus.value && processStatus.value[fileName]?.running) {
    // If already running, stop the process
    stopProcess(fileName)
      .then(() => {
        console.log(`Successfully stopped ${fileName}`);
        lastSyncTime.value = getCurrentTime();
      })
      .catch((err) => {
        console.error(`Failed to stop ${fileName}:`, err);
      });
  } else {
    // If not running, start the process
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

// Helper function to safely access process status
function isProcessRunning(fileName: string): boolean {
  return (
    !!processStatus.value &&
    !!processStatus.value[fileName] &&
    !!processStatus.value[fileName].running
  );
}

// Helper function to safely get process status
function getProcessStatus(fileName: string): string {
  if (!processStatus.value || !processStatus.value[fileName]) {
    return "Idle";
  }
  return processStatus.value[fileName].status || "Idle";
}

// Helper to get summary status - at least one service is running
const hasActiveServices = computed(() => {
  return (
    isProcessRunning("people_counter.bat") ||
    isProcessRunning("sync_manager.bat")
  );
});

// Get camera count
const cameraCount = computed(() => {
  return camerasState.value?.length || 0;
});

const currentDate = computed(() => {
  return getCurrentDate();
});

// Update last sync time when status changes
function updateLastSyncTime() {
  lastSyncTime.value = getCurrentTime();
}

let statusCheckInterval: number | null = null;

function checkProcessStatus() {
  const processes = ["people_counter.bat", "sync_manager.bat"];

  processes.forEach((process) => {
    // Jika status sedang terkunci, lewati pemeriksaan
    if (processStatus.value[process]?.statusLocked) {
      return;
    }

    // Periksa status proses
    Promise.all([IsProcessRunning(process), GetDetailedProcessStatus(process)])
      .then(([isRunning, status]) => {
        // Jika UI status berbeda dengan status sebenarnya
        const uiRunning = isProcessRunning(process);

        if (isRunning !== uiRunning) {
          console.log(
            `Status mismatch for ${process}: UI=${uiRunning}, actual=${isRunning}`
          );

          // Jika proses sebenarnya berjalan tapi UI menunjukkan tidak
          if (isRunning && !uiRunning && processStatus.value[process]) {
            processStatus.value[process].running = true;
            processStatus.value[process].status = status || "Running";
          }
          // Jika proses sebenarnya tidak berjalan tapi UI menunjukkan berjalan
          else if (!isRunning && uiRunning && processStatus.value[process]) {
            processStatus.value[process].running = false;
            processStatus.value[process].status = "Stopped";
          }
        }
      })
      .catch((err) => {
        console.error(`Error checking status for ${process}:`, err);
      });
  });
}

// Mulai interval check
function startStatusCheckInterval() {
  const intervalId = setInterval(checkProcessStatus, 5000);

  // Juga periksa status segera
  checkProcessStatus();

  return intervalId;
}

// Perbaiki onMounted dan onUnmounted

onMounted(async () => {
  isLoading.value = true;

  // Register event handlers first
  registerEventHandlers();

  // Load initial data
  await Promise.all([checkRunningProcesses(), listCameras()]);

  // Set up additional listeners
  setupConnectionStatusListener();

  // Mulai interval pemeriksaan status
  startStatusCheckInterval();

  // Listen for events to update last sync time
  Events.On("process_status", () => updateLastSyncTime());
  Events.On("process_status_updated", () => updateLastSyncTime());
  Events.On("process_running", () => updateLastSyncTime());
  Events.On("process_stopped", () => updateLastSyncTime());

  isLoading.value = false;

  // Force verify all process status immediately and after 3 seconds
  VerifyAllProcessStatusConsistency();
  setTimeout(() => {
    VerifyAllProcessStatusConsistency();
  }, 3000);
});

onUnmounted(() => {
  // Bersihkan interval status check
  if (statusCheckInterval !== null) {
    clearInterval(statusCheckInterval);
    statusCheckInterval = null;
  }

  // Matikan event listeners
  Events.Off("process_status");
  Events.Off("process_status_updated");
  Events.Off("process_running");
  Events.Off("process_stopped");
});
</script>

<template>
  <div class="flex flex-col h-full space-y-4">
    <!-- Dashboard header with summary stats -->
    <div
      class="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-md shadow-sm border border-gray-200"
    >
      <div>
        <h2 class="text-xl font-semibold text-gray-800 dark:text-gray-100">
          Dashboard
        </h2>
        <p class="text-xs text-muted-foreground mt-1">{{ currentDate }}</p>
      </div>
      <div class="flex items-center space-x-2">
        <div
          class="text-xs text-muted-foreground flex items-center bg-muted/40 px-2 py-1 rounded"
        >
          <Clock class="w-3 h-3 mr-1" />
          Last sync: {{ lastSyncTime }}
        </div>
      </div>
    </div>

    <div class="flex-1 overflow-auto space-y-4">
      <!-- Service status card -->
      <Card>
        <CardHeader class="flex flex-row justify-between items-center border-b">
          <div class="flex items-center gap-2">
            <Cpu class="w-5 h-5 text-gray-700 dark:text-gray-300" />
            <CardTitle class="text-base font-medium">AI Services</CardTitle>
          </div>
          <!-- <div class="flex items-center">
            <Badge
              :class="
                hasActiveServices
                  ? 'bg-green-100 hover:bg-green-100 text-green-800 hover:text-green-800'
                  : 'bg-gray-100 hover:bg-gray-100 text-gray-800 hover:text-gray-800'
              "
            >
              {{
                hasActiveServices ? "Services Active" : "All Services Inactive"
              }}
            </Badge>
          </div> -->
        </CardHeader>

        <CardContent class="p-4">
          <!-- Services grid -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- People Counting service -->
            <ServiceCard
              serviceName="People Counting"
              serviceFileName="people_counter.bat"
              description="AI-powered people detection and counting"
              iconName="fa6-solid:person-walking-dashed-line-arrow-right"
              iconColorClass="text-blue-500 dark:text-blue-400"
              bgColorClass="bg-blue-50 dark:bg-blue-900/20"
              :isRunning="isProcessRunning('people_counter.bat')"
              :status="getProcessStatus('people_counter.bat')"
              :statusClass="getStatusClass('people_counter.bat')"
              @toggle="handleProcessAction"
            />

            <!-- Scheduler Sync service -->
            <!-- <ServiceCard
              serviceName="Scheduler Sync"
              serviceFileName="sync_manager.bat"
              description="Automated data synchronization service"
              iconName="mdi:file-sync"
              iconColorClass="text-purple-500 dark:text-purple-400"
              bgColorClass="bg-purple-50 dark:bg-purple-900/20"
              :isRunning="isProcessRunning('sync_manager.bat')"
              :status="getProcessStatus('sync_manager.bat')"
              :statusClass="getStatusClass('sync_manager.bat')"
              @toggle="handleProcessAction"
            /> -->

            <ServiceControlCard
              serviceName="Sync Service"
              serviceDescription="Background data synchronization service"
              iconName="mdi:file-sync"
              iconColorClass="text-purple-500 dark:text-purple-400"
              bgColorClass="bg-purple-50 dark:bg-purple-900/20"
            />
          </div>
        </CardContent>
      </Card>

      <!-- Cameras section -->
      <Card class="flex-1 flex flex-col">
        <CardHeader class="flex flex-row justify-between items-center border-b">
          <div class="flex items-center gap-2">
            <MonitorPlay class="w-5 h-5 text-gray-800 dark:text-gray-200" />
            <CardTitle class="text-base font-medium">Cameras</CardTitle>
          </div>
          <Button
            size="sm"
            variant="outline"
            class="h-8 text-xs"
            @click="addNewCamera"
          >
            <Plus :size="14" class="mr-1.5" /> Add Camera
          </Button>
        </CardHeader>

        <CardContent class="p-4 flex-1 overflow-auto">
          <!-- Camera list with enhanced styling -->
          <div
            v-if="cameraCount > 0"
            class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3"
          >
            <CameraCard
              v-for="camera in camerasState"
              :key="camera.ID"
              :camera="camera"
              :refreshingCamera="refreshingCamera"
              @refresh="refreshCameraConnection"
              @edit="(id) => router.push(`/camera/edit/${id}`)"
              @view="viewCameraDetails"
            />
          </div>

          <!-- Empty camera state with enhanced styling -->
          <EmptyState
            v-else
            :icon="MonitorPlay"
            title="No cameras configured"
            description="Add cameras to start tracking people and collecting data for analytics"
            :showButton="true"
            buttonText="Add Camera"
            @action="addNewCamera"
          />
        </CardContent>
      </Card>
    </div>
  </div>
</template>
