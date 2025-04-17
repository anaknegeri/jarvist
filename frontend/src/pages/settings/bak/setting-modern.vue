<script setup>
import { useToast } from "@/components/ui/toast";
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
  Users,
  Wifi,
  X,
  Zap,
} from "lucide-vue-next";
import { onMounted, ref } from "vue";

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
      variant: "default",
    });

    unsavedChanges.value = false;
  } catch (error) {
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
      variant: "default",
    });
  } catch (error) {
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
      variant: "default",
    });
  } catch (error) {
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
  <div class="settings-page">
    <!-- Settings header with save button -->
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-800 dark:text-white">Settings</h1>

      <Button
        @click="saveSettings"
        class="bg-blue-600 hover:bg-blue-700 text-white flex items-center gap-2"
        :class="{
          'opacity-60 cursor-not-allowed': isLoading || !unsavedChanges,
        }"
        :disabled="isLoading || !unsavedChanges"
      >
        <Save v-if="!isLoading" class="w-4 h-4" />
        <svg
          v-else
          class="animate-spin h-4 w-4 text-white"
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
        <span>{{ isLoading ? "Saving..." : "Save Changes" }}</span>
      </Button>
    </div>

    <!-- Settings container -->
    <div
      class="flex gap-6 bg-white dark:bg-slate-900 rounded-xl shadow-sm border border-gray-200 dark:border-slate-700 overflow-hidden"
    >
      <!-- Settings tabs sidebar -->
      <div class="w-64 border-r border-gray-200 dark:border-slate-700 p-4">
        <ul class="space-y-1">
          <li v-for="tab in tabs" :key="tab.id">
            <button
              @click="activeTab = tab.id"
              class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
              :class="{
                'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400':
                  activeTab === tab.id,
                'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-800':
                  activeTab !== tab.id,
              }"
            >
              <component :is="tab.icon" class="w-5 h-5" />
              <span>{{ tab.label }}</span>
              <ChevronRight
                v-if="activeTab === tab.id"
                class="ml-auto w-4 h-4"
              />
            </button>
          </li>
        </ul>

        <div
          v-if="unsavedChanges"
          class="mt-6 p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800/30 rounded-lg text-sm text-amber-700 dark:text-amber-300"
        >
          <p class="flex items-start gap-2">
            <Bell class="w-4 h-4 mt-0.5" />
            <span>You have unsaved changes</span>
          </p>
        </div>
      </div>

      <!-- Settings content panels -->
      <div class="flex-1 p-6">
        <!-- General Settings -->
        <div v-if="activeTab === 'general'" class="space-y-6">
          <h2 class="text-lg font-medium text-gray-800 dark:text-white mb-4">
            General Settings
          </h2>

          <!-- Theme selection -->
          <div class="form-group">
            <label class="label">Theme</label>
            <div class="flex gap-4 mt-2">
              <label class="relative flex flex-col items-center">
                <input
                  type="radio"
                  v-model="settings.general.theme"
                  value="light"
                  class="sr-only"
                  @change="handleChange"
                />
                <div
                  class="w-20 h-16 rounded-lg flex items-center justify-center border-2 transition-colors cursor-pointer"
                  :class="
                    settings.general.theme === 'light'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800'
                  "
                >
                  <Sun
                    class="w-6 h-6"
                    :class="
                      settings.general.theme === 'light'
                        ? 'text-blue-500'
                        : 'text-gray-400 dark:text-gray-500'
                    "
                  />
                </div>
                <span class="mt-1 text-sm text-gray-700 dark:text-gray-300"
                  >Light</span
                >
              </label>

              <label class="relative flex flex-col items-center">
                <input
                  type="radio"
                  v-model="settings.general.theme"
                  value="dark"
                  class="sr-only"
                  @change="handleChange"
                />
                <div
                  class="w-20 h-16 rounded-lg flex items-center justify-center border-2 transition-colors cursor-pointer"
                  :class="
                    settings.general.theme === 'dark'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800'
                  "
                >
                  <Moon
                    class="w-6 h-6"
                    :class="
                      settings.general.theme === 'dark'
                        ? 'text-blue-500'
                        : 'text-gray-400 dark:text-gray-500'
                    "
                  />
                </div>
                <span class="mt-1 text-sm text-gray-700 dark:text-gray-300"
                  >Dark</span
                >
              </label>

              <label class="relative flex flex-col items-center">
                <input
                  type="radio"
                  v-model="settings.general.theme"
                  value="system"
                  class="sr-only"
                  @change="handleChange"
                />
                <div
                  class="w-20 h-16 rounded-lg flex items-center justify-center border-2 transition-colors cursor-pointer"
                  :class="
                    settings.general.theme === 'system'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800'
                  "
                >
                  <Laptop
                    class="w-6 h-6"
                    :class="
                      settings.general.theme === 'system'
                        ? 'text-blue-500'
                        : 'text-gray-400 dark:text-gray-500'
                    "
                  />
                </div>
                <span class="mt-1 text-sm text-gray-700 dark:text-gray-300"
                  >System</span
                >
              </label>
            </div>
          </div>

          <!-- Language selection -->
          <div class="form-group">
            <label class="label">Language</label>
            <select
              v-model="settings.general.language"
              class="input"
              @change="handleChange"
            >
              <option value="en">English</option>
              <option value="es">Español</option>
              <option value="fr">Français</option>
              <option value="de">Deutsch</option>
              <option value="id">Indonesia</option>
            </select>
          </div>

          <!-- Toggles section -->
          <div class="form-group">
            <div class="space-y-3">
              <!-- Start on boot -->
              <label class="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  v-model="settings.general.startOnBoot"
                  class="checkbox"
                  @change="handleChange"
                />
                <div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    Start on system boot
                  </div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Application will start automatically when system starts
                  </div>
                </div>
              </label>

              <!-- Notifications -->
              <label class="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  v-model="settings.general.notifications"
                  class="checkbox"
                  @change="handleChange"
                />
                <div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    Enable notifications
                  </div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Show system notifications for important events
                  </div>
                </div>
              </label>

              <!-- Check for updates -->
              <label class="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  v-model="settings.general.checkUpdates"
                  class="checkbox"
                  @change="handleChange"
                />
                <div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    Check for updates
                  </div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Automatically check for new versions
                  </div>
                </div>
              </label>
            </div>
          </div>

          <!-- Updates section -->
          <div class="form-group">
            <h3
              class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              Updates
            </h3>
            <div
              class="bg-gray-50 dark:bg-slate-800 p-4 rounded-lg border border-gray-200 dark:border-slate-700"
            >
              <div class="flex justify-between items-center">
                <div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    Current Version
                  </div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Jarvist v1.5.2
                  </div>
                </div>
                <Button
                  @click="checkForUpdates"
                  variant="outline"
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
            </div>
          </div>
        </div>

        <!-- Camera Settings -->
        <div v-if="activeTab === 'camera'" class="space-y-6">
          <h2 class="text-lg font-medium text-gray-800 dark:text-white mb-4">
            Camera Settings
          </h2>

          <!-- Timeout setting -->
          <div class="form-group">
            <label class="label">Connection Timeout (ms)</label>
            <input
              type="number"
              min="1000"
              max="10000"
              step="500"
              v-model="settings.camera.defaultTimeout"
              class="input"
              @input="handleChange"
            />
            <div class="hint">
              Time to wait before marking a camera as offline
            </div>
          </div>

          <!-- Reconnect attempts -->
          <div class="form-group">
            <label class="label">Reconnection Attempts</label>
            <input
              type="number"
              min="1"
              max="10"
              v-model="settings.camera.reconnectAttempts"
              class="input"
              @input="handleChange"
            />
            <div class="hint">Number of times to attempt reconnection</div>
          </div>

          <!-- Stream quality options -->
          <div class="form-group">
            <label class="label">Stream Quality</label>
            <div class="mt-2 space-y-2">
              <label
                v-for="option in qualityOptions"
                :key="option.value"
                class="flex items-center p-3 rounded-lg border border-gray-200 dark:border-slate-700 cursor-pointer transition-colors"
                :class="
                  settings.camera.streamQuality === option.value
                    ? 'bg-blue-50 border-blue-200 dark:bg-blue-900/20 dark:border-blue-800/50'
                    : 'bg-white dark:bg-slate-800'
                "
              >
                <input
                  type="radio"
                  v-model="settings.camera.streamQuality"
                  :value="option.value"
                  class="sr-only"
                  @change="handleChange"
                />
                <div class="flex-1">
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    {{ option.label }}
                  </div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    {{ option.description }}
                  </div>
                </div>
                <div
                  class="w-5 h-5 rounded-full border flex items-center justify-center"
                  :class="
                    settings.camera.streamQuality === option.value
                      ? 'border-blue-500 bg-blue-500'
                      : 'border-gray-300 dark:border-slate-600'
                  "
                >
                  <Check
                    v-if="settings.camera.streamQuality === option.value"
                    class="w-3 h-3 text-white"
                  />
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- Storage Settings -->
        <div v-if="activeTab === 'storage'" class="space-y-6">
          <h2 class="text-lg font-medium text-gray-800 dark:text-white mb-4">
            Storage Settings
          </h2>

          <!-- Storage location -->
          <div class="form-group">
            <label class="label">Data Storage Location</label>
            <div class="flex">
              <input
                type="text"
                v-model="settings.storage.localPath"
                class="input rounded-r-none flex-1"
                readonly
              />
              <Button
                @click="selectDirectory"
                variant="outline"
                class="rounded-l-none border-l-0"
              >
                Browse
              </Button>
            </div>
            <div class="hint">
              Location where application data will be stored
            </div>
          </div>

          <!-- Data retention -->
          <div class="form-group">
            <label class="label">Data Retention Period</label>
            <select
              v-model="settings.storage.retention"
              class="input"
              @change="handleChange"
            >
              <option
                v-for="option in retentionOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
            <div class="hint">How long to keep recorded data</div>
          </div>

          <!-- Auto cleanup toggle -->
          <div class="form-group">
            <label class="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                v-model="settings.storage.autoCleanup"
                class="checkbox"
                @change="handleChange"
              />
              <div>
                <div class="font-medium text-gray-800 dark:text-gray-200">
                  Automatic cleanup
                </div>
                <div class="text-sm text-gray-500 dark:text-gray-400">
                  Automatically delete data older than retention period
                </div>
              </div>
            </label>
          </div>

          <!-- Danger zone -->
          <div class="form-group mt-8">
            <div
              class="p-4 border border-red-200 dark:border-red-900/50 rounded-lg bg-red-50 dark:bg-red-900/20"
            >
              <h3
                class="text-sm font-semibold text-red-600 dark:text-red-400 mb-2"
              >
                Danger Zone
              </h3>
              <p class="text-sm text-red-600/80 dark:text-red-400/80 mb-3">
                The following actions are destructive and cannot be undone.
              </p>
              <Button
                @click="clearStorage"
                variant="destructive"
                class="bg-red-600 hover:bg-red-700"
                :disabled="isLoading"
              >
                <Trash2 class="w-4 h-4 mr-2" />
                Clear All Data
              </Button>
            </div>
          </div>
        </div>

        <!-- Network Settings -->
        <div v-if="activeTab === 'network'" class="space-y-6">
          <h2 class="text-lg font-medium text-gray-800 dark:text-white mb-4">
            Network Settings
          </h2>

          <!-- API Endpoint -->
          <div class="form-group">
            <label class="label">API Endpoint</label>
            <input
              type="text"
              v-model="settings.network.apiEndpoint"
              class="input"
              @input="handleChange"
            />
            <div class="hint">URL for Jarvist API services</div>
          </div>

          <!-- Proxy Settings -->
          <div class="form-group">
            <div class="flex items-center justify-between mb-2">
              <label class="label mb-0">Proxy Settings</label>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  v-model="settings.network.proxyEnabled"
                  class="sr-only peer"
                  @change="handleChange"
                />
                <div
                  class="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"
                ></div>
              </label>
            </div>

            <div
              class="bg-gray-50 dark:bg-slate-800 rounded-lg p-4 border border-gray-200 dark:border-slate-700"
              :class="{ 'opacity-50': !settings.network.proxyEnabled }"
            >
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1 block"
                    >Proxy Address</label
                  >
                  <input
                    type="text"
                    v-model="settings.network.proxyAddress"
                    class="input"
                    :disabled="!settings.network.proxyEnabled"
                    @input="handleChange"
                  />
                </div>
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1 block"
                    >Proxy Port</label
                  >
                  <input
                    type="text"
                    v-model="settings.network.proxyPort"
                    class="input"
                    :disabled="!settings.network.proxyEnabled"
                    @input="handleChange"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- Connection test -->
          <div class="form-group">
            <h3
              class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              Connection Test
            </h3>
            <div
              class="bg-gray-50 dark:bg-slate-800 p-4 rounded-lg border border-gray-200 dark:border-slate-700"
            >
              <div class="flex justify-between items-center">
                <div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    Test API Connection
                  </div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Verify connection to API server
                  </div>
                </div>
                <Button variant="outline">
                  <Server class="w-4 h-4 mr-2" />
                  Test Connection
                </Button>
              </div>
            </div>
          </div>
        </div>

        <!-- License Settings -->
        <div v-if="activeTab === 'license'" class="space-y-6">
          <h2 class="text-lg font-medium text-gray-800 dark:text-white mb-4">
            License Information
          </h2>

          <!-- License info card -->
          <div
            class="bg-white dark:bg-slate-800 rounded-lg overflow-hidden border border-gray-200 dark:border-slate-700 shadow-sm"
          >
            <div
              class="p-5 bg-gradient-to-r from-blue-500 to-indigo-600 text-white"
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
                  <p class="text-blue-100">{{ settings.license.key }}</p>
                </div>
              </div>
            </div>

            <div class="p-5 space-y-4">
              <div class="grid grid-cols-2 gap-x-4 gap-y-3">
                <div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Status
                  </div>
                  <div
                    class="font-medium flex items-center gap-1.5"
                    :class="
                      settings.license.status === 'active'
                        ? 'text-green-600 dark:text-green-400'
                        : 'text-red-600 dark:text-red-400'
                    "
                  >
                    <span
                      class="w-2 h-2 rounded-full"
                      :class="
                        settings.license.status === 'active'
                          ? 'bg-green-500'
                          : 'bg-red-500'
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
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Expiry Date
                  </div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    {{
                      new Date(settings.license.expiryDate).toLocaleDateString()
                    }}
                  </div>
                </div>

                <div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Max Cameras
                  </div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    {{ settings.license.maxCameras }}
                  </div>
                </div>

                <div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">
                    Type
                  </div>
                  <div class="font-medium text-gray-800 dark:text-gray-200">
                    {{
                      settings.license.key.startsWith("DEMO")
                        ? "Demo"
                        : "Professional"
                    }}
                  </div>
                </div>
              </div>

              <div
                class="pt-3 border-t border-gray-200 dark:border-slate-700 flex justify-between items-center"
              >
                <div>
                  <Button variant="outline">
                    <Share2 class="w-4 h-4 mr-2" />
                    Transfer License
                  </Button>
                </div>
                <div>
                  <Button>
                    <Users class="w-4 h-4 mr-2" />
                    Upgrade Plan
                  </Button>
                </div>
              </div>
            </div>
          </div>

          <!-- Manual activation -->
          <div class="form-group">
            <h3
              class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              Manual Activation
            </h3>
            <div
              class="bg-gray-50 dark:bg-slate-800 p-4 rounded-lg border border-gray-200 dark:border-slate-700"
            >
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-3">
                If you have a new license key, you can activate it manually
                here.
              </p>
              <div class="flex gap-2">
                <input
                  type="text"
                  placeholder="XXXX-XXXX-XXXX-XXXX"
                  class="input flex-1"
                />
                <Button>
                  <Key class="w-4 h-4 mr-2" />
                  Activate
                </Button>
              </div>
            </div>
          </div>

          <!-- Deactivation -->
          <div class="form-group mt-8">
            <div
              class="p-4 border border-red-200 dark:border-red-900/50 rounded-lg bg-red-50 dark:bg-red-900/20"
            >
              <h3
                class="text-sm font-semibold text-red-600 dark:text-red-400 mb-2"
              >
                License Deactivation
              </h3>
              <p class="text-sm text-red-600/80 dark:text-red-400/80 mb-3">
                Deactivating your license will remove it from this device.
              </p>
              <Button variant="destructive" class="bg-red-600 hover:bg-red-700">
                <X class="w-4 h-4 mr-2" />
                Deactivate License
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Form styling */
.form-group {
  @apply mb-6;
}

.label {
  @apply block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1;
}

.input {
  @apply w-full px-3 py-2 bg-white dark:bg-slate-800 border border-gray-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500;
}

.hint {
  @apply mt-1 text-sm text-gray-500 dark:text-gray-400;
}

.checkbox {
  @apply h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-800;
}

/* Button component (dummy styling if you don't have it) */
.Button {
  @apply inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed;
}
</style>
