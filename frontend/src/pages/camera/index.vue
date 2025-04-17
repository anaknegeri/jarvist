<script setup lang="ts">
import { getCurrentTime } from "@/lib/common";
import {
  camerasState,
  checkCameraConnection,
  cleanupConnectionStatusListener,
  deleteCamera,
  listCameras,
  setupConnectionStatusListener,
} from "@/services/cameraService";
import {
  Grid,
  List,
  MonitorPlay,
  Plus,
  RefreshCw,
  Search,
} from "lucide-vue-next";
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRouter } from "vue-router";

// Import custom components
import CameraCard from "@/components/CameraCard.vue";
import CameraList from "@/components/CameraList.vue";
import EmptyState from "@/components/EmptyState.vue";

// Import UI components
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

const uiStore = useUIStore();
const router = useRouter();
const searchQuery = ref("");
const isLoading = ref(false);
const selectedView = computed({
  get: () => uiStore.cameraViewPreference,
  set: (value) => uiStore.setCameraView(value),
});
const refreshingCamera = ref<number | null>(null);
const cameraToDelete = ref<number | null>(null);
const confirmDialogOpen = ref<boolean>(false);

// Referensi untuk unsubscribe function
let unsubscribeFromEvents: (() => void) | undefined;

const newCamera = () => {
  router.push("/camera/create");
};

const editCamera = (id: number) => {
  router.push(`/camera/edit/${id}`);
};

const confirmDeleteCamera = (id: number) => {
  cameraToDelete.value = id;
  confirmDialogOpen.value = true;
};

const handleDeleteCamera = async () => {
  console.log(cameraToDelete.value);
  if (!cameraToDelete.value) return;

  isLoading.value = true;
  const response = await deleteCamera(cameraToDelete.value);
  console.log(response);
  isLoading.value = false;
  cameraToDelete.value = null;

  if (!response.success) {
    // Show error notification
    alert(`Failed to delete camera: ${response.error}`);
  }
  confirmDialogOpen.value = false;
};

const refreshCameras = async () => {
  isLoading.value = true;
  await listCameras();
  isLoading.value = false;
};

const refreshCameraConnection = async (id: number, event: Event) => {
  // Prevent click from bubbling to parent elements
  event.stopPropagation();

  refreshingCamera.value = id;
  await checkCameraConnection(id);
  refreshingCamera.value = null;
};

const viewCameraDetails = (id: number) => {
  router.push(`/camera/view/${id}`);
};

onMounted(async () => {
  const savedView = localStorage.getItem("cameraViewPreference");
  if (savedView) {
    selectedView.value = savedView;
  }

  isLoading.value = true;
  await listCameras();
  isLoading.value = false;

  // Set up listener for real-time status updates
  unsubscribeFromEvents = setupConnectionStatusListener();
});

onUnmounted(() => {
  // Clean up listeners using unsubscribe function if available
  if (unsubscribeFromEvents) {
    unsubscribeFromEvents();
  } else {
    // Fallback to the standard cleanup method
    cleanupConnectionStatusListener();
  }
});

// Filter cameras based on search query
const filteredCameras = computed(() => {
  if (!searchQuery.value) return camerasState.value;

  const query = searchQuery.value.toLowerCase();
  return camerasState.value.filter(
    (camera) =>
      (camera.Name?.toLowerCase() || "").includes(query) ||
      (camera.Location?.name?.toLowerCase() || "").includes(query)
  );
});

// Get online and offline counts
const onlineCount = computed(() => {
  return filteredCameras.value.filter((c) => c.is_connected === true).length;
});

const offlineCount = computed(() => {
  return filteredCameras.value.filter((c) => c.is_connected === false).length;
});
</script>

<template>
  <div class="flex flex-col h-full space-y-4">
    <!-- Header with controls using shadcn components -->
    <div
      class="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-md shadow-sm border border-gray-200"
    >
      <div class="flex items-center">
        <MonitorPlay class="w-5 h-5 mr-2 text-indigo-500" />
        <h2 class="text-base font-medium text-gray-800">Cameras</h2>
      </div>

      <div class="flex items-center space-x-2">
        <!-- Search input -->
        <div class="relative">
          <Search
            class="absolute left-2 top-1/2 transform -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground"
          />
          <Input
            type="text"
            placeholder="Search cameras..."
            v-model="searchQuery"
            class="w-48 h-8 pl-7 text-xs"
          />
        </div>

        <!-- View toggle -->
        <div class="bg-accent rounded-md flex h-8 overflow-hidden">
          <Button
            variant="ghost"
            size="sm"
            class="px-3 h-full rounded-none"
            :class="selectedView === 'grid' ? 'bg-primary/10 text-primary' : ''"
            @click="selectedView = 'grid'"
          >
            <Grid :size="14" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="px-3 h-full rounded-none"
            :class="selectedView === 'list' ? 'bg-primary/10 text-primary' : ''"
            @click="selectedView = 'list'"
          >
            <List :size="14" />
          </Button>
        </div>

        <!-- Refresh button -->
        <Button variant="outline" size="sm" @click="refreshCameras">
          <RefreshCw
            :size="14"
            :class="{ 'animate-spin': isLoading }"
            class="mr-1"
          />
          <span class="text-xs">Refresh</span>
        </Button>

        <!-- Add button -->
        <Button size="sm" @click="newCamera">
          <Plus :size="14" class="mr-1" />
          <span class="text-xs">Add Camera</span>
        </Button>
      </div>
    </div>

    <!-- Camera list content -->
    <div class="flex-1 overflow-auto bg-muted/30">
      <!-- Grid view -->
      <div
        v-if="selectedView === 'grid' && filteredCameras.length > 0"
        class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3"
      >
        <CameraCard
          v-for="camera in filteredCameras"
          :key="camera.ID"
          :camera="camera"
          :refreshingCamera="refreshingCamera"
          @refresh="refreshCameraConnection"
          @edit="editCamera"
          @delete="confirmDeleteCamera"
          @view="viewCameraDetails"
        />
      </div>

      <!-- List view -->
      <CameraList
        v-else-if="selectedView === 'list' && filteredCameras.length > 0"
        :cameras="filteredCameras"
        :refreshingCamera="refreshingCamera"
        @refresh="refreshCameraConnection"
        @edit="editCamera"
        @delete="confirmDeleteCamera"
        @view="viewCameraDetails"
      />

      <!-- Empty state -->
      <EmptyState
        v-else
        :icon="MonitorPlay"
        :title="searchQuery ? 'No cameras found' : 'No cameras configured'"
        :description="
          searchQuery
            ? 'Try adjusting your search query or check if your cameras are properly configured.'
            : 'Get started by adding your first camera to monitor your premises.'
        "
        :showButton="!searchQuery"
        buttonText="Add Camera"
        @action="newCamera"
      />
    </div>

    <!-- Status bar -->
    <div
      class="mt-4 flex items-center justify-between bg-gray-50 py-2 px-4 rounded text-xs text-gray-500 border border-gray-200"
    >
      <div>
        {{ filteredCameras.length }} camera{{
          filteredCameras.length !== 1 ? "s" : ""
        }}
        |
        <span class="text-green-600 dark:text-green-400">
          {{ onlineCount }} online
        </span>
        |
        <span class="text-red-600 dark:text-red-400">
          {{ offlineCount }} offline
        </span>
      </div>
      <div>Last updated: {{ getCurrentTime() }}</div>
    </div>

    <!-- Alert Dialog for delete confirmation -->
    <AlertDialog
      :open="confirmDialogOpen"
      @update:open="
        (isOpen) => {
          if (!isOpen) confirmDialogOpen = false;
        }
      "
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you sure?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. This will permanently delete the
            camera and remove its data from our servers.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction @click="handleDeleteCamera"
            >Delete</AlertDialogAction
          >
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
