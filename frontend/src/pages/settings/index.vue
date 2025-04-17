<script setup lang="ts">
import { Activity, Palette, Shield } from "lucide-vue-next";
import { ref } from "vue";

import { useToast } from "@/components/ui/toast";
import { toTypedSchema } from "@vee-validate/zod";
import { useForm } from "vee-validate";
import { z } from "zod";

const { toast } = useToast();
const activeTab = ref("general");
const existingPlaceId = ref<any>(null);
const originalPlaceId = ref<any>(null);
const originalLocationCode = ref("");
const isLoading = ref(false);

const formSchema = z.object({
  site_name: z.string().min(2, "Please enter a location name").max(50),
  site_code: z.string().min(2, "Please enter a location code").max(50),
  site_category: z.number({ required_error: "Please select a category" }),
  timezone: z.string().min(2, "Please select a timezone").max(50),
});

export type FormValues = z.infer<typeof formSchema>;

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
});

const onSubmit = form.handleSubmit(async (values: FormValues) => {
  console.log(values);
  if (isLoading.value) return;
  isLoading.value = true;

  try {
    const saveResponse = await UpdatePlaceConfig(originalPlaceId.value, {
      site_code: values.site_code,
      site_name: values.site_name,
      site_category: values.site_category,
      default_timezone: values.timezone,
    });
    console.log(saveResponse);
    if (saveResponse.success) {
      toast({
        title: "Settings saved",
        description: "Your settings have been updated successfully",
        variant: "default",
      });
      originalLocationCode.value = values.site_code;
      return true;
    } else {
      toast({
        title: "Save failed",
        description: "Failed to save settings",
        variant: "destructive",
      });
      return false;
    }
  } catch (error) {
    console.error("Error saving settings:", error);
    toast({
      title: "Error occurred",
      description: "An error occurred while saving settings",
      variant: "destructive",
    });
    return false;
  } finally {
    isLoading.value = false;
  }
});

const tabs = [
  { id: "general", label: "General", icon: Palette },
  { id: "license", label: "License", icon: Shield },
  { id: "monitoring", label: "Monitoring", icon: Activity },
];

const handleSettingsLoaded = (settings: any) => {
  originalLocationCode.value = settings.site_code || "";
  originalPlaceId.value = Number(settings.site_id);
};

const handleLoadExistingPlace = (placeData: any) => {
  if (placeData && placeData.id) {
    existingPlaceId.value = placeData.id;
    originalLocationCode.value = placeData.placeCode || "";
  }
};

// Reset settings to defaults
const resetSettings = async () => {
  // if (isLoading.value) return;
  // isLoading.value = true;
  // try {
  //   const response = await ResetSettings();
  //   if (response.success) {
  //     const settings = await GetAllSettings();
  //     form.setValues({
  //       site_code: settings.site_code || "",
  //       site_name: settings.site_name || "",
  //       site_category: parseInt(settings.site_category) || 0,
  //       timezone: settings.default_timezone || "Asia/Jakarta",
  //     });
  //     originalLocationCode.value = settings.site_code || "";
  //     existingPlaceId.value = null;
  //     toast({
  //       title: "Settings reset",
  //       description: "All settings have been reset to default values",
  //       variant: "default",
  //     });
  //   } else {
  //     throw new Error("Failed to reset settings");
  //   }
  // } catch (error) {
  //   console.error("Error resetting settings:", error);
  //   toast({
  //     title: "Reset failed",
  //     description: "Could not reset settings to defaults",
  //     variant: "destructive",
  //   });
  // } finally {
  //   isLoading.value = false;
  // }
};

const saveSettings = () => {
  onSubmit();
};
</script>

<template>
  <form @submit.prevent="onSubmit">
    <div class="h-full w-full flex flex-col">
      <!-- Page header -->
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">Settings</h1>
          <p class="text-sm text-muted-foreground">
            Configure system settings and preferences.
          </p>
        </div>
        <div class="flex space-x-2">
          <Button
            variant="outline"
            :disabled="isLoading"
            @click.prevent="resetSettings"
          >
            <template v-if="isLoading">
              <svg
                class="animate-spin -ml-1 mr-2 h-4 w-4"
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
              <span>Resetting...</span>
            </template>
            <template v-else> Reset to Defaults </template>
          </Button>
          <Button
            type="submit"
            :disabled="isLoading"
            @click.prevent="saveSettings"
          >
            <template v-if="isLoading">
              <svg
                class="animate-spin -ml-1 mr-2 h-4 w-4"
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
              <span>Saving...</span>
            </template>
            <template v-else> Save Changes </template>
          </Button>
        </div>
      </div>

      <!-- Settings tabs -->
      <Tabs v-model="activeTab" class="flex-1">
        <div class="border-b">
          <TabsList
            class="w-full justify-start rounded-none h-auto p-0 bg-transparent"
          >
            <TabsTrigger
              v-for="tab in tabs"
              :key="tab.id"
              :value="tab.id"
              class="px-4 py-2.5 rounded-none border-b-2 border-transparent data-[state=active]:border-primary font-medium capitalize"
            >
              <div class="flex items-center space-x-2">
                <component :is="tab.icon" class="w-5 h-5" />
                <span>{{ tab.label }}</span>
              </div>
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="general" class="py-6 space-y-6">
          <GeneralTab
            :form="form"
            @loadExistingPlace="handleLoadExistingPlace"
            @settingsLoaded="handleSettingsLoaded"
          />
        </TabsContent>

        <!-- License Settings -->
        <TabsContent value="license" class="py-6 space-y-6">
          <LicenseTab />
        </TabsContent>

        <!-- Monitoring and Logging -->
        <TabsContent value="monitoring" class="py-6 space-y-6">
          <MonitoringTab :form="form" />
        </TabsContent>
      </Tabs>
    </div>
  </form>
</template>
