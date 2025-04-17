<script setup>
import { useToast } from "@/components/ui/toast/use-toast";
import {
  Bell,
  Check,
  ChevronRight,
  Database,
  Globe,
  Key,
  Laptop,
  Moon,
  Palette,
  Save,
  Server,
  Share2,
  Shield,
  Sun,
  Trash2,
  Users,
  Wifi,
  X,
  Zap,
} from "lucide-vue-next";
import { onMounted, ref } from "vue";

// Import shadcn components
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

// Form states
const settings = ref({
  general: {
    theme: "system",
    language: "en",
    startOnBoot: true,
    notifications: true,
    checkUpdates: true,
  },
  camera: {
    defaultTimeout: 5000,
    reconnectAttempts: 3,
    streamQuality: "medium",
  },
  storage: {
    localPath: "C:\\Jarvist\\Data",
    retention: 30,
    autoCleanup: true,
  },
  network: {
    apiEndpoint: "https://api.jarvist.ai",
    proxyEnabled: false,
    proxyAddress: "",
    proxyPort: "",
  },
  license: {
    key: "DEMO-1234-5678-ABCD",
    status: "active",
    expiryDate: "2025-03-25",
    maxCameras: 2,
  },
});

const activeTab = ref("general");
const isLoading = ref(false);
const { toast } = useToast();
const unsavedChanges = ref(false);

// Tabs configuration
const tabs = [
  { id: "general", label: "General", icon: Palette },
  { id: "camera", label: "Camera", icon: Wifi },
  { id: "storage", label: "Storage", icon: Database },
  { id: "network", label: "Network", icon: Globe },
  { id: "license", label: "License", icon: Shield },
];

// Quality options
const qualityOptions = [
  { value: "low", label: "Low (480p)", description: "Use less bandwidth" },
  { value: "medium", label: "Medium (720p)", description: "Balanced option" },
  { value: "high", label: "High (1080p)", description: "Best quality" },
];

// Retention period options
const retentionOptions = [
  { value: 7, label: "7 days" },
  { value: 14, label: "14 days" },
  { value: 30, label: "30 days" },
  { value: 60, label: "60 days" },
  { value: 90, label: "90 days" },
];

// Save settings
const saveSettings = async () => {
  isLoading.value = true;

  try {
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 800));

    // Save to localStorage for demo purposes
    localStorage.setItem("jarvist_settings", JSON.stringify(settings.value));

    toast({
      title: "Settings saved",
      description: "Your preferences have been updated successfully",
    });

    unsavedChanges.value = false;
  } catch (error) {
    console.log(error);
    toast({
      title: "Error saving settings",
      description: "There was a problem saving your settings",
      variant: "destructive",
    });
  } finally {
    isLoading.value = false;
  }
};

// Load settings
const loadSettings = () => {
  try {
    const savedSettings = localStorage.getItem("jarvist_settings");
    if (savedSettings) {
      settings.value = JSON.parse(savedSettings);
    }
  } catch (error) {
    console.log(error);
    console.error("Error loading settings:", error);
  }
};

// Open file browser to select directory (mock function)
const selectDirectory = () => {
  // In a real app, this would open a file browser dialog
  settings.value.storage.localPath =
    "C:\\Users\\Admin\\Documents\\Jarvist\\Data";
  unsavedChanges.value = true;
};

// Check for updates
const checkForUpdates = async () => {
  isLoading.value = true;

  try {
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1500));

    toast({
      title: "You're up to date!",
      description: "Jarvist v1.5.2 is the latest version",
    });
  } catch (error) {
    console.log(error);
    toast({
      title: "Update check failed",
      description: "Could not connect to update server",
      variant: "destructive",
    });
  } finally {
    isLoading.value = false;
  }
};

// Clear storage
const clearStorage = async () => {
  if (
    !confirm(
      "Are you sure you want to clear all stored data? This cannot be undone."
    )
  ) {
    return;
  }

  isLoading.value = true;

  try {
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1200));

    toast({
      title: "Storage cleared",
      description: "All data has been deleted",
    });
  } catch (error) {
    console.log(error);
    toast({
      title: "Error clearing storage",
      description: "There was a problem clearing the data",
      variant: "destructive",
    });
  } finally {
    isLoading.value = false;
  }
};

// Handle form changes
const handleChange = () => {
  unsavedChanges.value = true;
};

// On component mount
onMounted(() => {
  loadSettings();
});
</script>
<template>
  <div class="container mx-auto">
    <!-- Settings header with save button -->
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold">Settings</h1>

      <Button
        @click="saveSettings"
        :disabled="isLoading || !unsavedChanges"
        :class="{ 'opacity-60': !unsavedChanges }"
      >
        <Save v-if="!isLoading" class="w-4 h-4 mr-2" />
        <span v-if="isLoading" class="mr-2">
          <svg
            class="animate-spin h-4 w-4"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
        </span>
        {{ isLoading ? "Saving..." : "Save Changes" }}
      </Button>
    </div>

    <!-- Settings container -->
    <div class="flex gap-6">
      <!-- Tabs -->
      <Tabs value="general" class="w-full" @update:value="activeTab = $event">
        <div class="flex">
          <!-- Settings tabs sidebar -->
          <div class="w-64 mr-6">
            <TabsList
              class="flex flex-col h-auto p-1 space-y-1 bg-muted/40 border-0"
            >
              <TabsTrigger
                v-for="tab in tabs"
                :key="tab.id"
                :value="tab.id"
                class="justify-start w-full px-4 py-2 h-10 data-[state=active]:bg-background data-[state=active]:shadow-sm"
              >
                <component :is="tab.icon" class="w-5 h-5 mr-3" />
                <span>{{ tab.label }}</span>
                <ChevronRight
                  v-if="activeTab === tab.id"
                  class="ml-auto w-4 h-4 opacity-70"
                />
              </TabsTrigger>
            </TabsList>

            <div v-if="unsavedChanges" class="mt-6">
              <Card
                class="bg-yellow-50 border-yellow-200 dark:bg-yellow-900/20 dark:border-yellow-800/30"
              >
                <CardContent class="p-3">
                  <p
                    class="flex items-start gap-2 text-sm text-yellow-700 dark:text-yellow-400"
                  >
                    <Bell class="w-4 h-4 mt-0.5" />
                    <span>You have unsaved changes</span>
                  </p>
                </CardContent>
              </Card>
            </div>
          </div>

          <!-- Settings content panels -->
          <div class="flex-1">
            <!-- General Settings -->
            <TabsContent value="general" class="space-y-6 mt-0">
              <Card>
                <CardHeader>
                  <CardTitle>General Settings</CardTitle>
                </CardHeader>
                <CardContent class="space-y-6">
                  <!-- Theme selection -->
                  <div>
                    <Label class="text-base">Theme</Label>
                    <div class="flex gap-4 mt-2">
                      <div
                        class="relative flex flex-col items-center"
                        v-for="option in [
                          { value: 'light', icon: Sun, label: 'Light' },
                          { value: 'dark', icon: Moon, label: 'Dark' },
                          { value: 'system', icon: Laptop, label: 'System' },
                        ]"
                        :key="option.value"
                      >
                        <RadioGroup
                          v-model="settings.general.theme"
                          @update:model-value="handleChange"
                        >
                          <div class="sr-only">
                            <RadioGroupItem
                              :value="option.value"
                              :id="`theme-${option.value}`"
                            />
                          </div>
                        </RadioGroup>
                        <Label
                          :for="`theme-${option.value}`"
                          :class="[
                            'w-20 h-16 rounded-lg flex items-center justify-center border-2 transition-colors cursor-pointer',
                            settings.general.theme === option.value
                              ? 'border-primary bg-primary/10'
                              : 'border-border bg-card',
                          ]"
                          @click="
                            settings.general.theme = option.value;
                            handleChange();
                          "
                        >
                          <component
                            :is="option.icon"
                            class="w-6 h-6"
                            :class="
                              settings.general.theme === option.value
                                ? 'text-primary'
                                : 'text-muted-foreground'
                            "
                          />
                        </Label>
                        <span class="mt-1 text-sm text-muted-foreground">{{
                          option.label
                        }}</span>
                      </div>
                    </div>
                  </div>

                  <!-- Language selection -->
                  <div>
                    <Label for="language" class="text-base">Language</Label>
                    <Select
                      v-model="settings.general.language"
                      @update:model-value="handleChange"
                    >
                      <SelectTrigger id="language" class="w-full">
                        <SelectValue placeholder="Select language" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="en">English</SelectItem>
                        <SelectItem value="es">Español</SelectItem>
                        <SelectItem value="fr">Français</SelectItem>
                        <SelectItem value="de">Deutsch</SelectItem>
                        <SelectItem value="id">Indonesia</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <!-- Toggles section -->
                  <div class="space-y-3">
                    <div class="flex items-start space-x-3 pt-2">
                      <Checkbox
                        id="start-on-boot"
                        v-model:checked="settings.general.startOnBoot"
                        @update:checked="handleChange"
                      />
                      <div>
                        <Label
                          for="start-on-boot"
                          class="text-base font-medium cursor-pointer"
                        >
                          Start on system boot
                        </Label>
                        <p class="text-sm text-muted-foreground">
                          Application will start automatically when system
                          starts
                        </p>
                      </div>
                    </div>

                    <div class="flex items-start space-x-3 pt-2">
                      <Checkbox
                        id="notifications"
                        v-model:checked="settings.general.notifications"
                        @update:checked="handleChange"
                      />
                      <div>
                        <Label
                          for="notifications"
                          class="text-base font-medium cursor-pointer"
                        >
                          Enable notifications
                        </Label>
                        <p class="text-sm text-muted-foreground">
                          Show system notifications for important events
                        </p>
                      </div>
                    </div>

                    <div class="flex items-start space-x-3 pt-2">
                      <Checkbox
                        id="check-updates"
                        v-model:checked="settings.general.checkUpdates"
                        @update:checked="handleChange"
                      />
                      <div>
                        <Label
                          for="check-updates"
                          class="text-base font-medium cursor-pointer"
                        >
                          Check for updates
                        </Label>
                        <p class="text-sm text-muted-foreground">
                          Automatically check for new versions
                        </p>
                      </div>
                    </div>
                  </div>

                  <!-- Updates section -->
                  <div>
                    <h3 class="text-sm font-medium mb-2">Updates</h3>
                    <Card>
                      <CardContent class="p-4">
                        <div class="flex justify-between items-center">
                          <div>
                            <div class="font-medium">Current Version</div>
                            <div class="text-sm text-muted-foreground">
                              Jarvist v1.5.2
                            </div>
                          </div>
                          <Button
                            variant="outline"
                            @click="checkForUpdates"
                            :disabled="isLoading"
                          >
                            <Zap v-if="!isLoading" class="w-4 h-4 mr-2" />
                            <svg
                              v-else
                              class="animate-spin -ml-1 mr-2 h-4 w-4 text-primary"
                              xmlns="http://www.w3.org/2000/svg"
                              fill="none"
                              viewBox="0 0 24 24"
                            >
                              <circle
                                class="opacity-25"
                                cx="12"
                                cy="12"
                                r="10"
                                stroke="currentColor"
                                stroke-width="4"
                              ></circle>
                              <path
                                class="opacity-75"
                                fill="currentColor"
                                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                              ></path>
                            </svg>
                            Check for Updates
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <!-- Camera Settings -->
            <TabsContent value="camera" class="space-y-6 mt-0">
              <Card>
                <CardHeader>
                  <CardTitle>Camera Settings</CardTitle>
                </CardHeader>
                <CardContent class="space-y-6">
                  <!-- Timeout setting -->
                  <div>
                    <Label for="timeout" class="text-base"
                      >Connection Timeout (ms)</Label
                    >
                    <Input
                      id="timeout"
                      type="number"
                      min="1000"
                      max="10000"
                      step="500"
                      v-model="settings.camera.defaultTimeout"
                      @input="handleChange"
                    />
                    <p class="text-sm text-muted-foreground mt-1">
                      Time to wait before marking a camera as offline
                    </p>
                  </div>

                  <!-- Reconnect attempts -->
                  <div>
                    <Label for="reconnect" class="text-base"
                      >Reconnection Attempts</Label
                    >
                    <Input
                      id="reconnect"
                      type="number"
                      min="1"
                      max="10"
                      v-model="settings.camera.reconnectAttempts"
                      @input="handleChange"
                    />
                    <p class="text-sm text-muted-foreground mt-1">
                      Number of times to attempt reconnection
                    </p>
                  </div>

                  <!-- Stream quality options -->
                  <div>
                    <Label class="text-base mb-2 block">Stream Quality</Label>
                    <RadioGroup
                      v-model="settings.camera.streamQuality"
                      @update:model-value="handleChange"
                      class="space-y-2"
                    >
                      <Label
                        v-for="option in qualityOptions"
                        :key="option.value"
                        :for="`quality-${option.value}`"
                        class="flex items-center p-3 rounded-lg border cursor-pointer transition-colors"
                        :class="
                          settings.camera.streamQuality === option.value
                            ? 'bg-primary/10 border-primary/50'
                            : 'bg-card border-border'
                        "
                      >
                        <RadioGroupItem
                          :value="option.value"
                          :id="`quality-${option.value}`"
                          class="sr-only"
                        />
                        <div class="flex-1">
                          <div class="font-medium">{{ option.label }}</div>
                          <div class="text-sm text-muted-foreground">
                            {{ option.description }}
                          </div>
                        </div>
                        <div
                          class="w-5 h-5 rounded-full border flex items-center justify-center ml-3"
                          :class="
                            settings.camera.streamQuality === option.value
                              ? 'border-primary bg-primary'
                              : 'border-muted'
                          "
                        >
                          <Check
                            v-if="
                              settings.camera.streamQuality === option.value
                            "
                            class="w-3 h-3 text-white"
                          />
                        </div>
                      </Label>
                    </RadioGroup>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <!-- Storage Settings -->
            <TabsContent value="storage" class="space-y-6 mt-0">
              <Card>
                <CardHeader>
                  <CardTitle>Storage Settings</CardTitle>
                </CardHeader>
                <CardContent class="space-y-6">
                  <!-- Storage location -->
                  <div>
                    <Label for="storage-path" class="text-base"
                      >Data Storage Location</Label
                    >
                    <div class="flex">
                      <Input
                        id="storage-path"
                        type="text"
                        v-model="settings.storage.localPath"
                        readonly
                        class="rounded-r-none"
                      />
                      <Button
                        @click="selectDirectory"
                        variant="outline"
                        class="rounded-l-none border-l-0"
                      >
                        Browse
                      </Button>
                    </div>
                    <p class="text-sm text-muted-foreground mt-1">
                      Location where application data will be stored
                    </p>
                  </div>

                  <!-- Data retention -->
                  <div>
                    <Label for="retention" class="text-base"
                      >Data Retention Period</Label
                    >
                    <Select
                      v-model="settings.storage.retention"
                      @update:model-value="handleChange"
                    >
                      <SelectTrigger id="retention" class="w-full">
                        <SelectValue placeholder="Select retention period" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem
                          v-for="option in retentionOptions"
                          :key="option.value"
                          :value="option.value"
                        >
                          {{ option.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <p class="text-sm text-muted-foreground mt-1">
                      How long to keep recorded data
                    </p>
                  </div>

                  <!-- Auto cleanup toggle -->
                  <div class="flex items-center justify-between space-x-2">
                    <div class="space-y-0.5">
                      <Label for="auto-cleanup" class="text-base"
                        >Automatic cleanup</Label
                      >
                      <p class="text-sm text-muted-foreground">
                        Automatically delete data older than retention period
                      </p>
                    </div>
                    <Switch
                      id="auto-cleanup"
                      v-model:checked="settings.storage.autoCleanup"
                      @update:checked="handleChange"
                    />
                  </div>

                  <!-- Danger zone -->
                  <div class="pt-4">
                    <Card class="border-destructive/50 bg-destructive/5">
                      <CardHeader class="pb-2">
                        <CardTitle class="text-sm text-destructive">
                          Danger Zone
                        </CardTitle>
                        <CardDescription class="text-destructive/80">
                          The following actions are destructive and cannot be
                          undone.
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <Button
                          @click="clearStorage"
                          variant="destructive"
                          :disabled="isLoading"
                        >
                          <Trash2 class="w-4 h-4 mr-2" />
                          Clear All Data
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <!-- Network Settings -->
            <TabsContent value="network" class="space-y-6 mt-0">
              <Card>
                <CardHeader>
                  <CardTitle>Network Settings</CardTitle>
                </CardHeader>
                <CardContent class="space-y-6">
                  <!-- API Endpoint -->
                  <div>
                    <Label for="api-endpoint" class="text-base"
                      >API Endpoint</Label
                    >
                    <Input
                      id="api-endpoint"
                      type="text"
                      v-model="settings.network.apiEndpoint"
                      @input="handleChange"
                    />
                    <p class="text-sm text-muted-foreground mt-1">
                      URL for Jarvist API services
                    </p>
                  </div>

                  <!-- Proxy Settings -->
                  <div>
                    <div class="flex items-center justify-between mb-2">
                      <Label class="text-base">Proxy Settings</Label>
                      <Switch
                        id="proxy-enabled"
                        v-model:checked="settings.network.proxyEnabled"
                        @update:checked="handleChange"
                      />
                    </div>

                    <Card
                      :class="{ 'opacity-50': !settings.network.proxyEnabled }"
                    >
                      <CardContent class="p-4">
                        <div class="grid grid-cols-2 gap-4">
                          <div>
                            <Label for="proxy-address" class="text-sm"
                              >Proxy Address</Label
                            >
                            <Input
                              id="proxy-address"
                              type="text"
                              v-model="settings.network.proxyAddress"
                              :disabled="!settings.network.proxyEnabled"
                              @input="handleChange"
                            />
                          </div>
                          <div>
                            <Label for="proxy-port" class="text-sm"
                              >Proxy Port</Label
                            >
                            <Input
                              id="proxy-port"
                              type="text"
                              v-model="settings.network.proxyPort"
                              :disabled="!settings.network.proxyEnabled"
                              @input="handleChange"
                            />
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  <!-- Connection test -->
                  <div>
                    <h3 class="text-sm font-medium mb-2">Connection Test</h3>
                    <Card>
                      <CardContent class="p-4">
                        <div class="flex justify-between items-center">
                          <div>
                            <div class="font-medium">Test API Connection</div>
                            <div class="text-sm text-muted-foreground">
                              Verify connection to API server
                            </div>
                          </div>
                          <Button variant="outline">
                            <Server class="w-4 h-4 mr-2" />
                            Test Connection
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <!-- License Settings -->
            <TabsContent value="license" class="space-y-6 mt-0">
              <Card>
                <CardHeader>
                  <CardTitle>License Information</CardTitle>
                </CardHeader>
                <CardContent class="space-y-6">
                  <!-- License info card -->
                  <Card class="overflow-hidden">
                    <div
                      class="p-5 bg-gradient-to-r from-blue-600 to-indigo-600 text-white"
                    >
                      <div class="flex items-center gap-3">
                        <div class="p-2 bg-white/20 rounded-lg">
                          <Key class="w-6 h-6" />
                        </div>
                        <div>
                          <h3 class="text-lg font-semibold">
                            {{
                              settings.license.key.startsWith("DEMO")
                                ? "Demo License"
                                : "Professional License"
                            }}
                          </h3>
                          <p class="text-blue-100">
                            {{ settings.license.key }}
                          </p>
                        </div>
                      </div>
                    </div>

                    <CardContent class="p-5">
                      <div class="grid grid-cols-2 gap-x-4 gap-y-3">
                        <div>
                          <div class="text-sm text-muted-foreground">
                            Status
                          </div>
                          <div
                            class="font-medium flex items-center gap-1.5"
                            :class="
                              settings.license.status === 'active'
                                ? 'text-green-600 dark:text-green-400'
                                : 'text-destructive'
                            "
                          >
                            <span
                              class="w-2 h-2 rounded-full"
                              :class="
                                settings.license.status === 'active'
                                  ? 'bg-green-500'
                                  : 'bg-destructive'
                              "
                            ></span>
                            {{
                              settings.license.status === "active"
                                ? "Active"
                                : "Inactive"
                            }}
                          </div>
                        </div>

                        <div>
                          <div class="text-sm text-muted-foreground">
                            Expiry Date
                          </div>
                          <div class="font-medium">
                            {{
                              new Date(
                                settings.license.expiryDate
                              ).toLocaleDateString()
                            }}
                          </div>
                        </div>

                        <div>
                          <div class="text-sm text-muted-foreground">
                            Max Cameras
                          </div>
                          <div class="font-medium">
                            {{ settings.license.maxCameras }}
                          </div>
                        </div>

                        <div>
                          <div class="text-sm text-muted-foreground">Type</div>
                          <div class="font-medium">
                            {{
                              settings.license.key.startsWith("DEMO")
                                ? "Demo"
                                : "Professional"
                            }}
                          </div>
                        </div>
                      </div>

                      <Separator class="my-4" />

                      <div class="flex justify-between items-center">
                        <Button variant="outline">
                          <Share2 class="w-4 h-4 mr-2" />
                          Transfer License
                        </Button>
                        <Button>
                          <Users class="w-4 h-4 mr-2" />
                          Upgrade Plan
                        </Button>
                      </div>
                    </CardContent>
                  </Card>

                  <!-- Manual activation -->
                  <div>
                    <h3 class="text-sm font-medium mb-2">Manual Activation</h3>
                    <Card>
                      <CardContent class="p-4">
                        <p class="text-sm text-muted-foreground mb-3">
                          If you have a new license key, you can activate it
                          manually here.
                        </p>
                        <div class="flex gap-2">
                          <Input
                            type="text"
                            placeholder="XXXX-XXXX-XXXX-XXXX"
                            class="flex-1"
                          />
                          <Button>
                            <Key class="w-4 h-4 mr-2" />
                            Activate
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  <!-- Deactivation -->
                  <div>
                    <Card class="border-destructive/50 bg-destructive/5">
                      <CardHeader class="pb-2">
                        <CardTitle class="text-sm text-destructive">
                          License Deactivation
                        </CardTitle>
                        <CardDescription class="text-destructive/80">
                          Deactivating your license will remove it from this
                          device.
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <Button variant="destructive">
                          <X class="w-4 h-4 mr-2" />
                          Deactivate License
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </div>
        </div>
      </Tabs>
    </div>
  </div>
</template>
