<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Camera,
  Edit,
  ExternalLink,
  Eye,
  RefreshCw,
  Trash2,
} from "lucide-vue-next";
import CameraStatus from "./CameraStatus.vue";

defineProps<{
  cameras: any[];
  refreshingCamera: number | null;
}>();

const emit = defineEmits<{
  (e: "refresh", id: number, event: Event): void;
  (e: "edit", id: number): void;
  (e: "delete", id: number): void;
  (e: "view", id: number): void;
}>();

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

// Truncate long strings with ellipsis
const truncate = (str: string, maxLength = 30) => {
  if (!str) return "";
  return str.length > maxLength ? str.substring(0, maxLength) + "..." : str;
};

const handleRefresh = (id: number, event: Event) => {
  event.stopPropagation();
  emit("refresh", id, event);
};

const handleEdit = (id: number, event: Event) => {
  event.stopPropagation();
  emit("edit", id);
};

const handleDelete = (id: number, event: Event) => {
  event.stopPropagation();
  emit("delete", id);
};

const handleView = (id: number) => {
  emit("view", id);
};
</script>

<template>
  <div class="rounded-lg border overflow-hidden bg-white">
    <Table>
      <TableHeader>
        <TableRow class="bg-gray-50">
          <TableHead class="w-[200px]">Name</TableHead>
          <TableHead class="w-[180px]">Location</TableHead>
          <TableHead class="w-[250px]">Connection</TableHead>
          <TableHead class="w-[100px]">Status</TableHead>
          <TableHead class="w-[210px]">Last Checked</TableHead>
          <TableHead class="text-center w-[120px]">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow
          v-for="camera in cameras"
          :key="camera.ID"
          class="cursor-pointer hover:bg-gray-50 transition-colors"
          @click="handleView(camera.ID)"
        >
          <TableCell>
            <div class="flex items-center">
              <div
                class="w-9 h-9 bg-blue-50 rounded-full flex items-center justify-center mr-3 group-hover:bg-blue-100 transition-colors"
              >
                <Camera class="w-4 h-4 text-blue-500" />
              </div>
              <div class="flex flex-col">
                <span class="font-medium text-sm">{{ camera.Name }}</span>
                <span class="text-xs text-muted-foreground" v-if="camera.Tags">
                  <Badge variant="outline" class="text-xs mr-1">CCTV</Badge>
                  <Badge
                    v-if="camera.Schema === 'rtsp'"
                    variant="outline"
                    class="text-xs"
                    >RTSP</Badge
                  >
                </span>
              </div>
            </div>
          </TableCell>
          <TableCell class="text-sm">
            <div class="flex items-center gap-1">
              <div v-if="camera.Location?.name" class="text-sm">
                {{ camera.Location.name }}
              </div>
              <div v-else class="text-sm text-muted-foreground">
                No location
              </div>
            </div>
          </TableCell>
          <TableCell>
            <div class="flex items-center gap-1 text-sm">
              <code class="bg-muted px-1.5 py-0.5 rounded text-xs">
                {{ camera.Schema || "rtsp" }}://{{ truncate(camera.Host, 15)
                }}{{ camera.Port ? ":" + camera.Port : ""
                }}{{ camera.Path ? "/" + truncate(camera.Path, 10) : "" }}
              </code>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="ghost" size="icon" class="h-6 w-6">
                      <ExternalLink :size="12" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <code>
                      {{ camera.Schema || "rtsp" }}://{{ camera.Host
                      }}{{ camera.Port ? ":" + camera.Port : ""
                      }}{{ camera.Path ? "/" + camera.Path : "" }}
                    </code>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </TableCell>
          <TableCell>
            <CameraStatus :status="camera.is_connected" />
          </TableCell>
          <TableCell class="text-sm text-muted-foreground">
            {{ formatLastChecked(camera.last_checked) }}
          </TableCell>
          <TableCell class="text-right">
            <div class="flex items-center justify-end">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8 text-muted-foreground hover:text-primary"
                      @click.stop="(event) => handleView(camera.ID)"
                    >
                      <Eye :size="16" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>View details</TooltipContent>
                </Tooltip>
              </TooltipProvider>

              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8 text-muted-foreground hover:text-primary"
                      @click="(event) => handleRefresh(camera.ID, event)"
                    >
                      <RefreshCw
                        :size="16"
                        :class="{
                          'animate-spin': refreshingCamera === camera.ID,
                        }"
                      />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Check connection</TooltipContent>
                </Tooltip>
              </TooltipProvider>

              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8 text-muted-foreground hover:text-primary"
                      @click="(event) => handleEdit(camera.ID, event)"
                    >
                      <Edit :size="16" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Edit camera</TooltipContent>
                </Tooltip>
              </TooltipProvider>

              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8 text-muted-foreground hover:text-destructive"
                      @click="(event) => handleDelete(camera.ID, event)"
                    >
                      <Trash2 :size="16" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Delete camera</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
