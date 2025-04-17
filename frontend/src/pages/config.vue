<script setup lang="ts">
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
} from "@/components/ui/form";
import { useToast } from "@/components/ui/toast";

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

import { toTypedSchema } from "@vee-validate/zod";
import { Link, Settings } from "lucide-vue-next";
import { useForm } from "vee-validate";
import { onMounted, ref } from "vue";
import { z } from "zod";

interface PlaceCategory {
  id: number;
  name: string;
}

interface TimeZone {
  ID: number;
  Zone: string;
  UTCOffset: string;
  Name: string;
}

const isLoading = ref(false);
const error = ref<string | null>(null);
const success = ref<string | null>(null);
const showConfirmDialog = ref(false);
const dialogMessage = ref("");
const existingPlace = ref<any>(null);
const usingExistingPlace = ref(false);
const placeCategories = ref<PlaceCategory[]>([]);
const timezones = ref<TimeZone[]>([]);
const { toast } = useToast();
const titleBarStore = useTitleBarStore();

const formSchema = z.object({
  site_code: z
    .string({ required_error: "Please enter a location code" })
    .min(1, "Please enter a location code"),
  site_name: z
    .string({ required_error: "Please enter a location name" })
    .min(1, "Please enter a location name"),
  site_category: z.number({ required_error: "Please select a category" }),
  default_timezone: z
    .string({ required_error: "Please select a timezone" })
    .min(1, "Please select a timezone"),
});

export type FormValues = z.infer<typeof formSchema>;

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {
    default_timezone: "Asia/Jakarta",
  },
});

const saveSettings = async (values: FormValues) => {
  isLoading.value = true;
  error.value = null;
  success.value = null;

  try {
    let saveResponse;

    if (usingExistingPlace.value && existingPlace.value?.id) {
      saveResponse = await UpdatePlaceConfig(existingPlace.value.id, values);
    } else {
      saveResponse = await SavePlaceConfig(values);
    }

    console.log(saveResponse);
    const { success } = saveResponse;
    if (success) {
      toast({
        title: "Settings saved",
        description: "Settings saved successfully",
        variant: "default",
      });

      const isConfigured = await IsConfigured();
      if (isConfigured) {
        Restart();
      }
    } else {
      toast({
        title: "Save failed",
        description: "Failed to save settings",
        variant: "destructive",
      });

      error.value = "Failed to save configuration";
    }
  } catch (err) {
    console.error("Error saving settings:", err);
    toast({
      title: "Error occurred",
      description: "An error occurred while saving settings",
      variant: "destructive",
    });

    error.value = "An error occurred while saving configuration";
  } finally {
    isLoading.value = false;
  }
};

const onSubmit = form.handleSubmit(async (values) => {
  isLoading.value = true;
  error.value = null;
  success.value = null;

  try {
    if (usingExistingPlace.value) {
      await saveSettings(values);
      return;
    }

    const { success, message, data } = await ValidateSiteCode(values.site_code);

    if (!success) {
      showConfirmDialog.value = true;
      dialogMessage.value = message;
      existingPlace.value = data;
      isLoading.value = false;
      return;
    }

    await saveSettings(values);
  } catch (err) {
    console.error("Error validating place code:", err);
    toast({
      title: "Error occurred",
      description: "An error occurred while validating place code",
      variant: "destructive",
    });
    error.value = "An error occurred while validating place code";
    isLoading.value = false;
  }
});

const useExistingPlace = async () => {
  if (existingPlace.value) {
    usingExistingPlace.value = true;

    form.setValues({
      site_code: existingPlace.value.place_code,
      site_name: existingPlace.value.name,
      site_category: existingPlace.value.category.id || null,
    });
  }
  showConfirmDialog.value = false;
};

const changeLocationCode = () => {
  showConfirmDialog.value = false;
  usingExistingPlace.value = false;
  setTimeout(() => {
    const locationCodeInput = document.querySelector('input[name="site_code"]');
    if (locationCodeInput) {
      (locationCodeInput as HTMLInputElement).focus();
    }
  }, 100);
};

onMounted(async () => {
  titleBarStore.setTitle("Jarvist AI - System Configuration");

  const licence = await GetLicenseDetails();
  console.log(licence);
  try {
    await LoadLicense();
    timezones.value = await GetTimeZones();
    placeCategories.value = await GetSiteCetegories();
  } catch (err) {
    console.error("Error loading initial data:", err);
    error.value = "Failed to load initial data";
  }
});
</script>

<template>
  <div class="flex-1">
    <div class="flex justify-center mb-3">
      <div
        class="w-24 h-24 bg-gradient-to-r from-blue-500 to-indigo-600 rounded-full flex items-center justify-center shadow-lg"
      >
        <Settings class="w-12 h-12 text-white" />
      </div>
    </div>

    <h1
      class="text-center text-2xl font-bold text-gray-900 dark:text-gray-100 mb-1"
    >
      System Configuration
    </h1>
    <p class="text-center text-gray-500 dark:text-gray-400 mb-4">
      Configure your application settings
    </p>
    <form @submit.prevent="onSubmit">
      <Card
        class="shadow-md border-0 bg-white/90 dark:bg-gray-700/50 backdrop-blur-sm"
      >
        <CardHeader
          class="text-center bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700"
        >
          <CardTitle>Location Settings</CardTitle>
          <div
            v-if="usingExistingPlace"
            class="text-sm text-amber-600 dark:text-amber-400 mt-1 flex items-center justify-center"
          >
            <Link class="w-4 h-4 mr-1" />
            Using existing location data
          </div>
        </CardHeader>

        <CardContent class="p-6">
          <div class="space-y-2">
            <FormField
              v-slot="{ componentField, errorMessage, errors }"
              name="site_code"
              class="md:col-span-2"
            >
              <FormItem :class="{ 'has-error': errors.length > 0 }">
                <FormLabel class="text-xs font-medium text-gray-700">
                  Location Code
                </FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    v-bind="componentField"
                    :readonly="usingExistingPlace"
                    class="h-8 text-sm"
                    :class="{
                      'border-red-500 focus:ring-red-500': errors.length > 0,
                      'bg-gray-100 dark:bg-gray-800 cursor-not-allowed':
                        usingExistingPlace,
                    }"
                  />
                </FormControl>
                <div v-if="errors.length > 0" class="text-red-500 text-xs mt-1">
                  {{ errorMessage }}
                </div>
                <p
                  v-else-if="!usingExistingPlace"
                  class="text-xs text-muted-foreground"
                >
                  Enter a unique code for this location/installation.
                </p>
                <p v-else class="text-xs text-amber-600 dark:text-amber-400">
                  Using existing location code.
                </p>
              </FormItem>
            </FormField>

            <FormField
              v-slot="{ componentField, errorMessage, errors }"
              name="site_name"
              class="md:col-span-2"
            >
              <FormItem :class="{ 'has-error': errors.length > 0 }">
                <FormLabel class="text-xs font-medium text-gray-700">
                  Location Name
                </FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    v-bind="componentField"
                    :readonly="usingExistingPlace"
                    class="h-8 text-sm"
                    :class="{
                      'border-red-500 focus:ring-red-500': errors.length > 0,
                      'bg-gray-100 dark:bg-gray-800 cursor-not-allowed':
                        usingExistingPlace,
                    }"
                  />
                </FormControl>
                <div v-if="errors.length > 0" class="text-red-500 text-xs">
                  {{ errorMessage }}
                </div>
                <p
                  v-else-if="!usingExistingPlace"
                  class="text-xs text-muted-foreground"
                >
                  Enter a descriptive name for this location.
                </p>
                <p v-else class="text-xs text-amber-600 dark:text-amber-400">
                  Using existing location name.
                </p>
              </FormItem>
            </FormField>
            <div class="grid grid-cols-2 space-x-2">
              <FormField
                v-slot="{ componentField, errorMessage, errors }"
                name="site_category"
              >
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700">
                    Category
                  </FormLabel>
                  <Select v-bind="componentField">
                    <FormControl>
                      <SelectTrigger
                        class="h-8 text-sm"
                        :class="{
                          'border-red-500 focus:ring-red-500':
                            errors.length > 0,
                        }"
                      >
                        <SelectValue placeholder="Select category" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem
                          v-for="category in placeCategories"
                          :value="category.id"
                          :key="category.id"
                        >
                          {{ category.name }}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <div v-if="errors.length > 0" class="text-red-500 text-xs">
                    {{ errorMessage }}
                  </div>
                  <p v-else class="text-xs text-muted-foreground">
                    Select the type of location
                  </p>
                </FormItem>
              </FormField>
              <FormField
                v-slot="{ componentField, errorMessage, errors }"
                name="default_timezone"
              >
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700">
                    Timezone
                  </FormLabel>
                  <Select v-bind="componentField">
                    <FormControl>
                      <SelectTrigger
                        class="h-8 text-sm"
                        :class="{
                          'border-red-500 focus:ring-red-500':
                            errors.length > 0,
                        }"
                      >
                        <SelectValue placeholder="Select a TimeZone" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem
                          v-for="tz in timezones"
                          :value="tz.Zone"
                          :key="tz.ID"
                        >
                          {{ tz.Zone }}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <div
                    v-if="errors.length > 0"
                    class="text-red-500 text-xs mt-1"
                  >
                    {{ errorMessage }}
                  </div>
                  <p v-else class="text-xs text-muted-foreground">
                    For accurate time-based operations
                  </p>
                </FormItem>
              </FormField>
            </div>
          </div>
        </CardContent>

        <CardFooter
          class="px-6 py-4 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700"
        >
          <div class="w-full space-y-3">
            <Button
              class="w-full shadow-lg text-white"
              :class="
                usingExistingPlace
                  ? 'bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-600 hover:to-amber-700'
                  : 'bg-gradient-to-r from-blue-600 to-indigo-700 hover:from-blue-700 hover:to-indigo-800'
              "
              :disabled="isLoading"
              type="submit"
            >
              <template v-if="!isLoading">
                <Link v-if="usingExistingPlace" class="w-4 h-4 mr-2" />
                <Settings v-else class="w-4 h-4 mr-2" />
              </template>
              <svg
                v-else
                class="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
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
              {{
                isLoading
                  ? "Saving..."
                  : usingExistingPlace
                  ? "Continue with Existing Location"
                  : "Save Configuration"
              }}
            </Button>
          </div>
        </CardFooter>
      </Card>
    </form>

    <!-- Alert Dialog for confirmation when place already exists -->
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
              @click="useExistingPlace"
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
  </div>
</template>

<style scoped>
.animate-in {
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>

<route lang="yaml">
meta:
  layout: config
</route>
