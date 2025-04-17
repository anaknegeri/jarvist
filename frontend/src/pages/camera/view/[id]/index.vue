<script setup lang="ts">
import { Line } from "@/components/ImageCanvas.vue";
import {
  camerasState,
  checkCameraConnection,
  cleanupConnectionStatusListener,
  getCamera,
  setupConnectionStatusListener,
} from "@/services/cameraService";
import { listLocations, locationsState } from "@/services/locationService";
import { Icon } from "@iconify/vue";
import {
  ArrowLeft,
  Computer,
  Edit,
  Link,
  MapPin,
  RefreshCw,
  Tag,
} from "lucide-vue-next";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";

import CameraStatus from "@/components/CameraStatus.vue";
import ImageCanvas from "@/components/ImageCanvas.vue";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";

const router = useRouter();
const props = defineProps(["id"]);

const isLoading = ref(true);
const cameraData = ref<any>(null);
const cameraId = Number(props.id);
const base64Image = ref("");
const lines = ref<Line[]>([]);
const selectedLocation = ref<any>(null);

const formatLastChecked = (lastChecked?: string) => {
  if (!lastChecked) return "Never checked";

  try {
    const date = new Date(lastChecked);
    return date.toLocaleString();
  } catch (e) {
    console.error(e);
    return lastChecked;
  }
};

const directionMap: Record<string, string> = {
  ttb: "Top to Bottom",
  btt: "Bottom to Top",
  ltr: "Left to Right",
  rtl: "Right to Left",
};

const icons: any = {
  ltr: "prime:arrow-right",
  rtl: "prime:arrow-left",
  ttb: "prime:arrow-down",
  btt: "prime:arrow-up",
};

const goToListPage = () => {
  router.push("/camera");
};

const goToEditPage = () => {
  router.push(`/camera/edit/${cameraId}`);
};

function constructRtspUrl(data: any): string {
  if (!data || !data.Host) return "";

  const { Schema, Host, Port, Path, Username } = data;

  const schema = Schema || "rtsp";
  const credentials = Username ? `${Username}:***@` : "";
  const portPart = Port ? `:${Port}` : "";
  const pathPart = Path ? `/${Path}` : "";

  return `${schema}://${credentials}${Host}${portPart}${pathPart}`;
}

const loadCameraData = async () => {
  isLoading.value = true;
  try {
    await listLocations();

    const { success, data, lines: cameraLines } = await getCamera(cameraId);

    if (success && data) {
      cameraData.value = data;

      selectedLocation.value = locationsState.value.find(
        (p) => p.id === data.Location?.id
      );

      if (cameraLines && cameraLines.length) {
        lines.value = cameraLines.map((line: any) => ({
          start: line.start,
          end: line.end,
          direction: line.direction || data.Direction,
          color: line.color,
        }));
      }

      if (data.ImageData) {
        base64Image.value = data.ImageData;
      }
    }
  } catch (error) {
    console.error("Error loading camera data:", error);
  } finally {
    isLoading.value = false;
  }
};

const isRefreshing = ref(false);

const currentCamera = computed(() => {
  return camerasState.value.find((c) => c.ID === cameraId);
});

watch(
  currentCamera,
  (newCamera) => {
    if (newCamera && cameraData.value) {
      cameraData.value.is_connected = newCamera.is_connected;
      cameraData.value.last_checked = newCamera.last_checked;
      cameraData.value.status_message = newCamera.status_message;
    }
  },
  { deep: true }
);

const refreshCameraConnection = async () => {
  isRefreshing.value = true;
  await checkCameraConnection(cameraId);
  isRefreshing.value = false;
};

onMounted(async () => {
  await loadCameraData();
  setupConnectionStatusListener();
});

onUnmounted(() => {
  cleanupConnectionStatusListener();
});

const goToSetupCountingZone = () => {
  router.push(`/camera/edit/${cameraId}?setupConfig=true`);
};
</script>

<template>
  <div v-if="isLoading" class="flex justify-center items-center h-full">
    <div class="flex flex-col items-center space-y-4">
      <Icon
        icon="line-md:loading-twotone-loop"
        class="w-12 h-12 text-primary"
      />
      <p class="text-sm text-gray-600">Loading camera data...</p>
    </div>
  </div>

  <div v-else class="flex flex-col h-full">
    <!-- Page header -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <div
          class="w-10 h-10 bg-blue-50 rounded-full flex items-center justify-center"
        >
          <Computer class="w-5 h-5 text-blue-500" />
        </div>
        <div>
          <h2 class="text-lg font-medium text-gray-800">
            {{ cameraData?.Name }}
          </h2>
          <div class="flex items-center gap-2">
            <MapPin class="w-3.5 h-3.5 text-gray-500" />
            <p class="text-sm text-gray-500">
              {{ selectedLocation?.name || "No location" }}
            </p>
          </div>
        </div>
      </div>
      <div class="flex items-center space-x-2">
        <div class="flex items-center gap-2">
          <CameraStatus :status="cameraData?.is_connected" />
          <Button
            variant="ghost"
            size="sm"
            class="h-8 w-8 p-0"
            @click="refreshCameraConnection"
            :disabled="isRefreshing"
          >
            <RefreshCw :size="16" :class="{ 'animate-spin': isRefreshing }" />
          </Button>
        </div>
        <Button variant="outline" type="button" @click="goToListPage">
          <ArrowLeft class="w-4 h-4 mr-1.5" /> Back
        </Button>
        <Button type="button" @click="goToEditPage">
          <Edit class="w-4 h-4 mr-1.5" /> Edit
        </Button>
      </div>
    </div>

    <!-- Main content area -->
    <Card class="shadow-sm border-gray-200">
      <div class="grid grid-cols-1 lg:grid-cols-5 h-full">
        <!-- Left column with camera image -->
        <div class="col-span-1 lg:col-span-3 p-0 border-r border-gray-200">
          <div class="h-full relative overflow-hidden p-6">
            <ImageCanvas
              v-if="base64Image"
              :defaultLines="lines"
              :image-src="base64Image"
              :direction="cameraData?.Direction || 'ttb'"
              :disable="true"
              class="h-full w-full"
            />
            <div
              v-else
              class="h-full flex items-center justify-center bg-gray-100"
            >
              <p class="text-sm text-gray-500">No camera image available</p>
            </div>
          </div>
        </div>

        <!-- Right column with details -->
        <div class="col-span-1 lg:col-span-2 p-0">
          <!-- Camera Information -->
          <div class="p-4">
            <h3 class="text-base font-medium mb-3">Camera Information</h3>

            <div class="space-y-4">
              <div class="flex items-start gap-3">
                <Link class="w-4 h-4 text-gray-500 mt-0.5 flex-shrink-0" />
                <div class="flex-1">
                  <p class="text-xs text-gray-500 mb-1">URL</p>
                  <p class="text-sm break-all">
                    {{ constructRtspUrl(cameraData) }}
                  </p>
                </div>
              </div>

              <div class="flex items-start gap-3">
                <Icon
                  icon="ph:text-align-left"
                  class="w-4 h-4 text-gray-500 mt-0.5 flex-shrink-0"
                />
                <div class="flex-1">
                  <p class="text-xs text-gray-500 mb-1">Description</p>
                  <p class="text-sm">
                    {{ cameraData?.Description || "No description" }}
                  </p>
                </div>
              </div>

              <div class="flex items-start gap-3">
                <Tag class="w-4 h-4 text-gray-500 mt-0.5 flex-shrink-0" />
                <div class="flex-1">
                  <p class="text-xs text-gray-500 mb-1">Tags</p>
                  <div class="flex flex-wrap gap-2">
                    <Badge
                      v-if="cameraData?.Tags"
                      variant="outline"
                      class="text-xs"
                      >CCTV</Badge
                    >
                    <Badge
                      v-if="cameraData?.Schema === 'rtsp'"
                      variant="outline"
                      class="text-xs"
                      >RTSP</Badge
                    >
                    <Badge
                      v-if="cameraData?.Tags?.includes('PTZ')"
                      variant="outline"
                      class="text-xs"
                      >PTZ</Badge
                    >
                    <p v-if="!cameraData?.Tags" class="text-sm">No tags</p>
                  </div>
                </div>
              </div>

              <div class="flex items-start gap-3">
                <Icon
                  icon="ph:clock"
                  class="w-4 h-4 text-gray-500 mt-0.5 flex-shrink-0"
                />
                <div class="flex-1">
                  <p class="text-xs text-gray-500 mb-1">Last Checked</p>
                  <p class="text-sm">
                    {{ formatLastChecked(cameraData?.last_checked) }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <Separator />

          <!-- People Flow Counting -->
          <div class="p-4">
            <div class="flex justify-between items-center mb-3">
              <h3 class="text-base font-medium">People Flow Counting</h3>
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="goToSetupCountingZone"
              >
                Setup Config
              </Button>
            </div>

            <div class="space-y-4">
              <div>
                <p class="text-xs text-gray-500 mb-1">Virtual Lines</p>
                <p class="text-sm">{{ lines.length }} Lines</p>
              </div>

              <div>
                <p class="text-xs text-gray-500 mb-1">Enter Direction</p>
                <div class="flex items-center gap-2">
                  <Icon
                    :icon="icons[cameraData?.Direction || 'ttb']"
                    class="w-4 h-4 text-gray-700"
                  />
                  <p class="text-sm">
                    {{ directionMap[cameraData?.Direction || "ttb"] }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>
