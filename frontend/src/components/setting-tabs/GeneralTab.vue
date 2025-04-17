<script setup lang="ts">
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
} from "@/components/ui/form";
import { toast, ToastAction } from "@/components/ui/toast";
import { siteCategories, siteCategoryState } from "@/services/apiService";
import { Link, Settings, Zap } from "lucide-vue-next";

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
import { listTimezones, timezoneState } from "@/services/timezoneService";
import { Events } from "@wailsio/runtime";

// Define the UpdateEvent type based on Go struct
interface UpdateEvent {
  event: string;
  message: string;
  progress?: number;
  timestamp: string;
  success: boolean;
  data?: any;
}

const props = defineProps({
  form: {
    type: Object,
    required: true,
  },
  validateOnChange: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits([
  "loadExistingSite",
  "settingsLoaded",
  "loadExistingPlace",
]);

const isLoading = ref(false);
const error = ref<string | null>(null);
const showConfirmDialog = ref(false);
const dialogMessage = ref("");
const existingSite = ref<any>(null);
const usingExistingSite = ref(false);
const setting = ref<any>({});
const originalLocationCode = ref("");
const currentVersion = ref("1.0.0");
const updateListeners = ref<(() => void) | null>(null);
const isPendingUpdate = ref(false);

const validateLocationCode = async (locationCode: string) => {
  if (locationCode === originalLocationCode.value) {
    return true;
  }

  try {
    const { success, message, data } = await ValidateSiteCode(locationCode);

    if (!success) {
      showConfirmDialog.value = true;
      dialogMessage.value = message;
      existingSite.value = data;
      return false;
    }
    return true;
  } catch (err) {
    console.error("Error validating site code:", err);
    toast({
      title: "Error occurred",
      description: "An error occurred while validating site code",
      variant: "destructive",
    });
    error.value = "An error occurred while validating site code";
    return false;
  }
};

const useExistingSite = async () => {
  if (existingSite.value) {
    usingExistingSite.value = true;

    props.form.setValues({
      site_code: existingSite.value.siteCode,
      site_name: existingSite.value.siteName,
      site_category: parseInt(existingSite.value.siteCategory) || 0,
      timezone: existingSite.value.default_timezone || "Asia/Jakarta",
    });

    originalLocationCode.value = existingSite.value.siteCode;

    emit("loadExistingSite", existingSite.value);
  }
  showConfirmDialog.value = false;
};

const changeLocationCode = () => {
  showConfirmDialog.value = false;
  setTimeout(() => {
    const locationCodeInput = document.querySelector('input[name="site_code"]');
    if (locationCodeInput) {
      (locationCodeInput as HTMLInputElement).focus();
    }
  }, 100);
};

const checkForUpdates = async () => {
  try {
    const updateEventListener = Events.On(
      "update_event",
      async (event: Events.WailsEvent) => {
        try {
          const eventData: UpdateEvent = JSON.parse(event.data);

          switch (eventData.event) {
            case "update_checking":
              toast({
                title: "Checking for updates",
                description: eventData.message,
                variant: "default",
              });
              isLoading.value = true;
              break;

            case "update_available":
              toast({
                title: "Update available!",
                description: `Version ${eventData.data.version} is available. Click to install.`,
                variant: "default",
                action: h(
                  ToastAction,
                  {
                    altText: "Update now",
                    onClick: () => downloadUpdate(eventData.data),
                  },
                  {
                    default: () => "Update",
                  }
                ),
              });
              isLoading.value = false;
              break;

            case "update_check_complete":
              isLoading.value = false;
              toast({
                title: "You're up to date!",
                description: eventData.message,
                variant: "default",
              });
              break;

            case "update_check_error":
              toast({
                title: "Update check failed",
                description: eventData.message,
                variant: "destructive",
              });
              isLoading.value = false;
              break;

            case "update_download_start":
              toast({
                title: "Downloading update",
                description: eventData.message,
                variant: "default",
              });
              break;

            case "update_download_progress":
              // You could update a progress bar here if needed
              console.log("Download progress:", eventData.data);
              break;

            case "update_download_complete":
              await checkPendingUpdate();
              toast({
                title: "Update downloaded",
                description:
                  "Update has been downloaded and will be installed when you restart the application.",
                variant: "default",
                action: h(
                  ToastAction,
                  {
                    altText: "Install now",
                    onClick: () => installUpdate(eventData.data),
                  },
                  {
                    default: () => "Install",
                  }
                ),
              });
              break;

            case "update_download_error":
              toast({
                title: "Download failed",
                description: eventData.message,
                variant: "destructive",
              });
              break;

            case "update_install_start":
              toast({
                title: "Installing update",
                description: "Installing update, please wait...",
                variant: "default",
              });
              break;

            case "update_install_complete":
              toast({
                title: "Update installed",
                description: eventData.message,
                variant: "default",
              });
              break;

            case "update_install_error":
              toast({
                title: "Installation failed",
                description: eventData.message,
                variant: "destructive",
              });
              isLoading.value = false;
              break;

            case "update_restart_ready":
              toast({
                title: "Update ready",
                description: "Update is ready to be installed on restart",
                variant: "default",
              });
              break;
          }
        } catch (error) {
          console.error("Error processing update event:", error);
        }
      }
    );

    updateListeners.value = updateEventListener;

    // Call the Go backend function to check for updates
    isLoading.value = true;
    await CheckForUpdates();

    // Remove listener after 2 minutes if no response
    setTimeout(() => {
      if (updateListeners.value) {
        updateListeners.value();
        updateListeners.value = null;
      }
    }, 120000);
  } catch (error) {
    toast({
      title: "Update check failed",
      description: "Could not connect to update server",
      variant: "destructive",
    });
    isLoading.value = false;
  }
};

const downloadUpdate = async (updateInfo: UpdateInfo) => {
  try {
    toast({
      title: "Downloading update",
      description: "Please wait while we download the update...",
      variant: "default",
    });
    await DownloadUpdate(updateInfo);
  } catch (error) {
    toast({
      title: "Download failed",
      description: "Could not start the download process",
      variant: "destructive",
    });
  }
};

const installUpdate = async (downloadPath?: string) => {
  isLoading.value = true;
  try {
    if (downloadPath) {
      await InstallUpdate(downloadPath);
    } else {
      await InstallPendingUpdates();
    }
  } catch (error) {
    toast({
      title: "Installation failed",
      description: "Could not install the update",
      variant: "destructive",
    });
    isLoading.value = false;
  }
};

const loadSettings = async () => {
  try {
    const settingsData = await GetAllSettings();
    setting.value = settingsData;

    props.form.setValues({
      site_code: settingsData.site_code || "",
      site_name: settingsData.site_name || "",
      site_category: parseInt(settingsData.site_category) || 0,
      timezone: settingsData.default_timezone || "Asia/Jakarta",
      log_retention: settingsData.log_retention || "30",
    });

    originalLocationCode.value = settingsData.site_code || "";
    emit("settingsLoaded", settingsData);

    await listTimezones();
    await siteCategories();
  } catch (err) {
    console.error("Error loading settings:", err);
    error.value = "Failed to load settings";
  }
};

const loadVersion = async () => {
  currentVersion.value = await GetProductVersion();
};

const checkPendingUpdate = async () => {
  isPendingUpdate.value = await CheckPendingUpdates();
};

onMounted(async () => {
  await loadVersion();
  await loadSettings();
  await checkPendingUpdate();
});

watch(
  () => props.form.values.site_code,
  async (newValue, oldValue) => {
    if (
      props.validateOnChange &&
      newValue &&
      newValue !== oldValue &&
      newValue !== originalLocationCode.value &&
      !usingExistingSite.value
    ) {
      await validateLocationCode(newValue);
    }
  },
  { flush: "post" }
);

onUnmounted(() => {
  if (updateListeners.value) {
    updateListeners.value();
    updateListeners.value = null;
  }
});
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle class="flex items-center">
        <Settings class="mr-2 h-4 w-4" />
        System Configuration
      </CardTitle>
      <div
        v-if="usingExistingSite"
        class="text-sm text-amber-600 dark:text-amber-400 mt-1 flex items-center justify-center"
      >
        <Link class="w-4 h-4 mr-1" />
        Using existing location data
      </div>
    </CardHeader>
    <CardContent class="space-y-6">
      <Separator />

      <div class="grid grid-cols-2 gap-x-8 gap-y-2">
        <FormField
          v-slot="{ componentField, errorMessage, errors }"
          name="site_code"
          :form="form"
        >
          <FormItem :class="{ 'has-error': errors.length > 0 }">
            <FormLabel>Location Code</FormLabel>
            <FormControl>
              <Input
                type="text"
                v-bind="componentField"
                readonly
                :class="{
                  'border-red-500 focus:ring-red-500': errors.length > 0,
                  'bg-gray-100 dark:bg-gray-800 cursor-not-allowed':
                    usingExistingSite,
                }"
                autocomplete="off"
                @blur="validateLocationCode($event.target.value)"
              />
            </FormControl>
            <div v-if="errors.length > 0" class="text-red-500 text-xs mt-1">
              {{ errorMessage }}
            </div>
          </FormItem>
        </FormField>

        <FormField
          v-slot="{ componentField, errorMessage, errors }"
          name="site_name"
          :form="form"
        >
          <FormItem :class="{ 'has-error': errors.length > 0 }">
            <FormLabel>Location Name</FormLabel>
            <FormControl>
              <Input
                type="text"
                v-bind="componentField"
                :readonly="usingExistingSite"
                :class="{
                  'border-red-500 focus:ring-red-500': errors.length > 0,
                  'bg-gray-100 dark:bg-gray-800 cursor-not-allowed':
                    usingExistingSite,
                }"
                autocomplete="off"
              />
            </FormControl>
            <div v-if="errors.length > 0" class="text-red-500 text-xs mt-1">
              {{ errorMessage }}
            </div>
          </FormItem>
        </FormField>

        <FormField
          v-slot="{ componentField, errorMessage, errors }"
          name="site_category"
          :form="form"
        >
          <FormItem :class="{ 'has-error': errors.length > 0 }">
            <FormLabel>Category</FormLabel>
            <FormControl>
              <Select v-bind="componentField">
                <SelectTrigger
                  :class="{
                    'border-red-500 focus:ring-red-500': errors.length > 0,
                  }"
                >
                  <SelectValue siteholder="Select category" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem
                      v-for="category in siteCategoryState"
                      :value="category.id"
                      :key="category.id"
                    >
                      {{ category.name }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </FormControl>
            <div v-if="errors.length > 0" class="text-red-500 text-xs mt-1">
              {{ errorMessage }}
            </div>
          </FormItem>
        </FormField>

        <FormField
          v-slot="{ componentField, errorMessage, errors }"
          name="timezone"
          :form="form"
        >
          <FormItem :class="{ 'has-error': errors.length > 0 }">
            <FormLabel>Time Zone</FormLabel>
            <FormControl>
              <Select v-bind="componentField">
                <SelectTrigger
                  :class="{
                    'border-red-500 focus:ring-red-500': errors.length > 0,
                  }"
                >
                  <SelectValue siteholder="Select a TimeZone" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectLabel>Timezones</SelectLabel>
                    <SelectItem
                      v-for="tz in timezoneState"
                      :value="tz.Zone"
                      :key="tz.ID"
                    >
                      {{ tz.Zone }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </FormControl>
            <div v-if="errors.length > 0" class="text-red-500 text-xs mt-1">
              {{ errorMessage }}
            </div>
          </FormItem>
        </FormField>
      </div>

      <div class="form-group">
        <h3 class="font-semibold leading-none tracking-tight mb-3">Updates</h3>
        <div
          class="bg-gray-50 dark:bg-slate-800 p-4 rounded-lg border border-gray-200 dark:border-slate-700"
        >
          <div class="flex justify-between items-center">
            <div>
              <div class="font-medium text-gray-800 dark:text-gray-200">
                Current Version
              </div>
              <div class="text-sm text-gray-500 dark:text-gray-400">
                Jarvist v{{ currentVersion }}
              </div>
            </div>
            <Button
              v-if="!isPendingUpdate"
              @click="checkForUpdates"
              variant="outline"
              type="button"
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

            <Button
              v-else
              @click="installUpdate"
              variant="outline"
              type="button"
            >
              Install Update
            </Button>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>

  <!-- Alert Dialog for confirmation when site already exists -->
  <AlertDialog
    :open="showConfirmDialog"
    @update:open="showConfirmDialog = $event"
  >
    <AlertDialogContent class="max-w-md mx-auto rounded-lg">
      <div class="p-4 max-w-md mx-auto">
        <AlertDialogHeader class="pb-2">
          <AlertDialogTitle class="text-lg font-semibold text-center"
            >Location Code Already Exists</AlertDialogTitle
          >
          <AlertDialogDescription class="text-center">
            {{ dialogMessage }}
            <div class="mt-2 text-sm">
              Would you like to use the existing location data or change the
              location code?
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter
          class="flex flex-col sm:flex-row gap-2 pt-4 sm:justify-center border-t border-gray-200 dark:border-gray-700 mt-4"
        >
          <AlertDialogAction
            @click="useExistingSite"
            class="bg-amber-600 hover:bg-amber-700 text-white order-1 sm:order-2 w-full sm:w-auto"
          >
            Use Existing Location
          </AlertDialogAction>
          <AlertDialogCancel
            @click="changeLocationCode"
            class="order-2 sm:order-1 w-full sm:w-auto"
          >
            Change Location Code
          </AlertDialogCancel>
        </AlertDialogFooter>
      </div>
    </AlertDialogContent>
  </AlertDialog>
</template>
