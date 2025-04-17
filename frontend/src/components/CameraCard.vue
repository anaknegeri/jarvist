<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Card, CardFooter, CardHeader } from "@/components/ui/card";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Camera, Edit, RefreshCw, Trash2 } from "lucide-vue-next";
import { ref } from "vue";
import CameraStatus from "./CameraStatus.vue";

const props = defineProps<{
  camera: any;
  refreshingCamera: number | null;
}>();

const emit = defineEmits<{
  (e: "refresh", id: number, event: Event): void;
  (e: "edit", id: number): void;
  (e: "delete", id: number): void;
  (e: "view", id: number): void;
}>();

const hoveredCamera = ref<number | null>(null);

// Format the last checked time
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

const handleRefresh = (event: Event) => {
  emit("refresh", props.camera.ID, event);
};

const handleEdit = (event: Event) => {
  event.stopPropagation();
  emit("edit", props.camera.ID);
};

const handleDelete = (event: Event) => {
  event.stopPropagation();
  emit("delete", props.camera.ID);
};

const handleView = () => {
  emit("view", props.camera.ID);
};
</script>

<template>
  <Card
    class="overflow-hidden bg-white rounded-lg p-4 gap-3 shadow-sm border border-gray-200 hover:border-blue-300 hover:shadow-md transition-all duration-200 group"
    @mouseenter="hoveredCamera = camera.ID"
    @mouseleave="hoveredCamera = null"
    @click="handleView"
  >
    <CardHeader
      class="p-0 pb-3 flex flex-row items-center justify-between space-y-0 space-x-2"
    >
      <div class="flex items-center">
        <div
          class="w-10 h-10 bg-blue-50 rounded-full flex items-center justify-center shrink-0 group-hover:bg-blue-100 transition-colors"
        >
          <Camera class="w-5 h-5 text-blue-500" />
        </div>
        <div class="ml-3">
          <h3 class="font-medium text-sm truncate">{{ camera.Name }}</h3>
          <p class="text-xs text-muted-foreground truncate">
            {{ camera.Location?.name || "No location" }}
          </p>
        </div>
      </div>

      <!-- Status badge -->
      <CameraStatus :status="camera.is_connected" />
    </CardHeader>

    <CardFooter
      class="p-0 pt-2 border-t text-xs text-muted-foreground flex justify-between items-center"
    >
      <div>Last checked: {{ formatLastChecked(camera.last_checked) }}</div>
      <div class="flex space-x-1">
        <div class="flex space-x-1" v-if="hoveredCamera === camera.ID">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7 p-1.5"
                  @click="handleEdit"
                >
                  <Edit :size="12" class="text-muted-foreground" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Edit camera</TooltipContent>
            </Tooltip>
          </TooltipProvider>

          <Button
            variant="ghost"
            size="icon"
            class="h-7 w-7 hover:bg-destructive/10 hover:text-destructive"
            @click="handleDelete"
          >
            <Trash2 :size="14" />
          </Button>
        </div>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                class="h-7 w-7 p-1.5"
                @click="handleRefresh"
              >
                <RefreshCw
                  :size="12"
                  :class="{
                    'animate-spin': refreshingCamera === camera.ID,
                  }"
                  class="text-muted-foreground"
                />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Check connection</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </CardFooter>
  </Card>
</template>
