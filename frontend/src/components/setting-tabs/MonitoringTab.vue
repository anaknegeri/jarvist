<template>
  <Card>
    <CardHeader>
      <div class="flex justify-between items-center">
        <CardTitle class="flex items-center">
          <Activity class="mr-2 h-4 w-4" />
          System Logs
        </CardTitle>

        <div class="flex space-x-2">
          <Button
            size="sm"
            variant="outline"
            @click="fetchLogs"
            :disabled="isLoading"
          >
            <RefreshCw class="w-3.5 h-3.5 mr-1.5" />
            {{ isLoading ? "Refreshing..." : "Refresh" }}
          </Button>
          <Button
            size="sm"
            variant="outline"
            @click="handleDownloadLogs"
            :disabled="logs.length === 0"
          >
            <Download class="w-3.5 h-3.5 mr-1.5" />
            Export
          </Button>
        </div>
      </div>
    </CardHeader>
    <CardContent class="space-y-4">
      <!-- Filters -->
      <div
        class="flex gap-2 mb-3 pb-3 border-b border-gray-200 dark:border-gray-700 justify-between"
      >
        <div class="relative min-w-[200px]">
          <Search
            class="absolute left-2 top-2 h-4 w-4 text-gray-500 dark:text-gray-400"
          />
          <Input
            v-model="logSearch"
            placeholder="Search logs..."
            class="pl-8 h-8 text-xs"
          />
        </div>
        <div class="flex gap-2">
          <Select v-model="selectedLogLevel">
            <SelectTrigger class="h-8 text-xs">
              <Filter class="w-3.5 h-3.5 mr-1.5" />
              <span v-if="selectedLogLevel === 'all'">All Levels</span>
              <span v-else>{{ selectedLogLevel }}</span>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Levels</SelectItem>
              <SelectItem
                v-for="level in LOG_LEVELS"
                :key="level.value"
                :value="level.value"
              >
                {{ level.label }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="selectedComponent">
            <SelectTrigger class="h-8 text-xs">
              <span v-if="selectedComponent === 'all'">All Components</span>
              <span v-else>{{ selectedComponent }}</span>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Components</SelectItem>
              <SelectItem
                v-for="component in logComponents"
                :key="component"
                :value="component"
              >
                {{ component }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Button
            size="sm"
            variant="ghost"
            class="h-8 text-xs"
            @click="handleClearFilters"
          >
            <X class="w-3.5 h-3.5 mr-1.5" />
            Clear Filters
          </Button>
        </div>
      </div>

      <!-- Conditional Rendering Based on Log State -->
      <template v-if="isLoading">
        <div class="flex justify-center items-center h-64">
          <div class="flex flex-col items-center">
            <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
            <p class="mt-2 text-sm text-muted-foreground">Loading logs...</p>
          </div>
        </div>
      </template>

      <template v-else-if="error">
        <div class="flex justify-center items-center h-64">
          <div class="flex flex-col items-center text-center">
            <AlertTriangle class="h-8 w-8 text-destructive mb-2" />
            <p class="text-sm text-destructive mb-2">{{ error }}</p>
            <Button @click="fetchLogs" variant="outline">
              <RefreshCw class="mr-2 h-4 w-4" />
              Try Again
            </Button>
          </div>
        </div>
      </template>

      <template v-else-if="logs.length === 0">
        <div class="flex justify-center items-center h-64">
          <div class="flex flex-col items-center text-center">
            <FileText class="h-8 w-8 text-muted-foreground mb-2" />
            <p class="text-sm text-muted-foreground mb-2">No logs available</p>
            <Button @click="fetchLogs" variant="outline">
              <RefreshCw class="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </div>
        </div>
      </template>

      <template v-else>
        <!-- Log Table with Scrollable Body -->
        <div
          class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden"
        >
          <Table class="w-full table-fixed">
            <TableHeader>
              <TableRow>
                <TableHead class="w-2/12">Timestamp</TableHead>
                <TableHead class="w-1/12">Level</TableHead>
                <TableHead class="w-2/12">Component</TableHead>
                <TableHead class="w-7/12">Message</TableHead>
              </TableRow>
            </TableHeader>
          </Table>
          <div class="max-h-[350px] overflow-y-auto">
            <Table class="w-full table-fixed">
              <TableBody>
                <TableRow
                  v-for="log in filteredLogs"
                  :key="log.timestamp + log.message"
                  class="text-xs"
                >
                  <TableCell class="w-2/12 py-2">
                    <div class="flex items-center">
                      <Clock class="w-3.5 h-3.5 mr-1 text-gray-500" />
                      {{ formatTimestamp(log.timestamp) }}
                    </div>
                  </TableCell>
                  <TableCell class="w-1/12 py-2">
                    <Badge
                      :class="getLevelBadgeClass(log.level)"
                      class="font-normal px-1.5 py-0.5"
                    >
                      <div class="flex items-center space-x-1">
                        <Info v-if="log.level === 'info'" class="w-3 h-3" />
                        <AlertTriangle
                          v-else-if="log.level === 'warn'"
                          class="w-3 h-3"
                        />
                        <XCircle
                          v-else-if="
                            log.level === 'error' || log.level === 'critical'
                          "
                          class="w-3 h-3"
                        />
                        <span>{{ log.level }}</span>
                      </div>
                    </Badge>
                  </TableCell>
                  <TableCell class="w-2/12 py-2">{{ log.component }}</TableCell>
                  <TableCell class="w-7/12 py-2">
                    <p class="font-medium">{{ log.message }}</p>
                    <p
                      class="text-gray-500 dark:text-gray-400 text-[11px] mt-0.5"
                    >
                      {{ log.details }}
                    </p>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </div>

        <!-- Filtered Results Empty State -->
        <div
          v-if="filteredLogs.length === 0"
          class="text-center py-4 text-muted-foreground"
        >
          No logs match the current filter criteria.
        </div>
      </template>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import {
  Activity,
  AlertTriangle,
  Clock,
  Download,
  FileText,
  Filter,
  Info,
  Loader2,
  RefreshCw,
  Search,
  X,
  XCircle,
} from "lucide-vue-next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { computed, onMounted, ref } from "vue";

// Log entry interface
interface LogEntry {
  timestamp: string;
  level: string;
  component: string;
  message: string;
  details: string;
}

// Log levels for filtering
const LOG_LEVELS = [
  { value: "debug", label: "Debug" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warning" },
  { value: "error", label: "Error" },
  { value: "critical", label: "Critical" },
];

// State variables
const logs = ref<LogEntry[]>([]);
const isLoading = ref(false);
const error = ref<string | null>(null);
const logSearch = ref("");
const selectedLogLevel = ref("all");
const selectedComponent = ref("all");

// Compute unique components
const logComponents = computed(() => {
  return [...new Set(logs.value.map((log) => log.component))];
});

// Filtered logs
const filteredLogs = computed(() => {
  return logs.value.filter((log) => {
    // Search filter
    const matchesSearch =
      logSearch.value === "" ||
      log.message.toLowerCase().includes(logSearch.value.toLowerCase()) ||
      log.details.toLowerCase().includes(logSearch.value.toLowerCase()) ||
      log.component.toLowerCase().includes(logSearch.value.toLowerCase());

    // Level filter
    const matchesLevel =
      selectedLogLevel.value === "all" || log.level === selectedLogLevel.value;

    // Component filter
    const matchesComponent =
      selectedComponent.value === "all" ||
      log.component === selectedComponent.value;

    return matchesSearch && matchesLevel && matchesComponent;
  });
});

// Utility functions
const getLevelBadgeClass = (level: string) => {
  switch (level) {
    case "debug":
      return "bg-gray-200 text-gray-800 dark:bg-gray-700 dark:text-gray-300";
    case "info":
      return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300";
    case "warn":
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300";
    case "error":
      return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300";
    case "critical":
      return "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300";
    default:
      return "bg-gray-200 text-gray-800 dark:bg-gray-700 dark:text-gray-300";
  }
};

const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp);
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
};

// Utility function to parse log lines
const processLogLines = (rawLogs: string[]): LogEntry[] => {
  const logRegex = /^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) - (\w+) -\s*(.+)$/;
  return rawLogs
    .map((line) => {
      const match = line.match(logRegex);
      if (!match) return null;

      return {
        timestamp: match[1],
        level: match[2].toLowerCase(),
        component: "Scheduler", // Fixed component name based on log source
        message: "Database Update",
        details: match[3],
      };
    })
    .filter((entry): entry is LogEntry => entry !== null);
};

const fetchLogs = async () => {
  // Reset previous state
  error.value = null;
  isLoading.value = true;
  logs.value = [];

  try {
    // Simulate log fetching
    const rawLogs = await ReadLogs(); // Assuming this is defined elsewhere

    // Check if no logs were returned
    // if (!rawLogs || rawLogs.length === 0) {
    //   throw new Error("No logs available");
    // }

    logs.value = processLogLines(rawLogs);
    console.log(logs.value);
  } catch (err) {
    // Set error message
    error.value =
      err instanceof Error
        ? err.message
        : "Failed to fetch logs. Please try again.";

    // Log the full error for debugging
    console.error("Log fetching error:", err);
  } finally {
    isLoading.value = false;
  }
};

const handleDownloadLogs = () => {
  // Convert logs to CSV
  const csvContent = [
    "Timestamp,Level,Component,Message,Details",
    ...logs.value.map(
      (log) =>
        `"${log.timestamp}","${log.level}","${log.component}","${log.message}","${log.details}"`
    ),
  ].join("\n");

  // Create and download CSV file
  const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  const url = URL.createObjectURL(blob);
  link.setAttribute("href", url);
  link.setAttribute(
    "download",
    `logs_${new Date().toISOString().split("T")[0]}.csv`
  );
  link.style.visibility = "hidden";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
};

const handleClearFilters = () => {
  logSearch.value = "";
  selectedLogLevel.value = "all";
  selectedComponent.value = "all";
};

onMounted(() => {
  fetchLogs();
});
</script>
