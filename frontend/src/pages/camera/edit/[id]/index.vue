<script setup lang="ts">
import { Line } from "@/components/ImageCanvas.vue";
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { getCamera, updateCamera } from "@/services/cameraService";
import {
  createLocation,
  listLocations,
  locationsState,
} from "@/services/locationService";
import { checkRTSPWithConfig, type RTSPResponse } from "@/services/rtspService";
import { toTypedSchema } from "@vee-validate/zod";
import { ArrowLeft, Computer, Plus, Trash2 } from "lucide-vue-next";
import { useForm } from "vee-validate";
import { onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import * as z from "zod";

import AddLocationDialog from "@/components/AddLocationDialog.vue";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { useToast } from "@/components/ui/toast";
import { Icon } from "@iconify/vue";

const router = useRouter();
const props = defineProps(["id"]);
const tagsModel = ref<string[]>([]);
const toast = useToast();

const rtspUrl = ref("");
const isValidUrl = ref(false);
const connectionStatus = ref<RTSPResponse | null>(null);
const isTestingConnection = ref(false);
const showSetupConfig = ref(false);
const imageCanvasRef = ref<any>(null);
const lines = ref<Line[]>([]);
const selectedDirection = ref("ttb");
const isDialogOpen = ref(false);
const isLoading = ref(true);
const isSaving = ref(false);
const cameraData = ref<any>(null);
const cameraId = Number(props.id);
const route = useRoute();

const base64Image = ref("");

const directions = ref([
  { name: "Top To Bottom", direction: "ttb" },
  { name: "Bottom To Top", direction: "btt" },
  { name: "Left To Right", direction: "ltr" },
  { name: "Right To Left", direction: "rtl" },
]);

const schemas = ref([{ name: "RTSP", value: "rtsp" }]);

const icons: any = {
  ltr: "prime:arrow-right",
  rtl: "prime:arrow-left",
  ttb: "prime:arrow-down",
  btt: "prime:arrow-up",
};

const formSchema = z.object({
  name: z.string().min(1, "Name is required"),
  location: z.string().min(1, "Location is required"),
  description: z.string().optional().nullable(),
  tags: z.string().optional().nullable(),
  schema: z.string().min(1, "Schema is required"),
  host: z
    .string()
    .regex(
      /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
      "Invalid IP address format"
    ),
  port: z
    .string()
    .transform((val) => parseInt(val, 10))
    .refine(
      (val) => !isNaN(val) && val > 0 && val <= 65535,
      "Port must be a valid number between 1-65535"
    ),
  path: z.string().optional().nullable(),
  username: z.string().optional().nullable(),
  password: z.string().optional().nullable(),
});

export type FormValues = z.infer<typeof formSchema>;

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {
    schema: "rtsp",
    port: "554",
  },
});

function formatPassword(password: string, isShowing: boolean): string {
  return isShowing ? password : "*".repeat(password.length);
}

function convertToRtspUrl(config: any): string {
  const { schema, host, port, path, username, password } = config;

  if (!host?.length) {
    throw new Error("Schema and host are required");
  }

  const credentials = username
    ? password
      ? `${username}:${formatPassword(password, true)}@`
      : `${username}@`
    : "";

  const portPart = port ? `:${port}` : "";
  const pathPart = path ? `/${path}` : "";
  return `${schema}://${credentials}${host}${portPart}${pathPart}`;
}

async function testConnection() {
  if (!isValidUrl.value) return;

  isTestingConnection.value = true;
  connectionStatus.value = null;

  try {
    const config: any = {
      schema: form.values.schema,
      host: form.values.host,
      port: Number(form.values.port),
      path: form.values.path,
      username: form.values.username,
      password: form.values.password,
    };

    const response = await checkRTSPWithConfig(config, true);

    if (response.screenshotPath)
      base64Image.value = await GetImageAsBase64(response.screenshotPath);

    connectionStatus.value = response;
  } catch (error) {
    connectionStatus.value = {
      success: false,
      message: "Failed to test connection",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  } finally {
    isTestingConnection.value = false;
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    isSaving.value = true; // Set loading state to true when saving starts

    const value: any = {
      ...values,
      image_data: base64Image.value,
      direction: selectedDirection.value,
      lines: lines.value.map((line) => ({
        start: line.start,
        end: line.end,
        direction: line.direction,
        color: line.color,
      })),
    };

    const { success } = await updateCamera(cameraId, value);

    if (success) {
      toast.toast({
        title: "Success",
        description: "Camera settings updated successfully",
      });
      router.push("/camera");
    } else {
      toast.toast({
        title: "Error",
        description: "Failed to update camera settings",
        variant: "destructive",
      });
    }
  } catch (error) {
    console.error("Error saving camera settings:", error);
    toast.toast({
      title: "Error",
      description: "An unexpected error occurred while saving",
      variant: "destructive",
    });
  } finally {
    isSaving.value = false;
  }
});

const toggleSetupConfig = async () => {
  const isValid = await form.validate();

  if (isValid.valid) {
    showSetupConfig.value = !showSetupConfig.value;
  }
};

const handleChangeLines = (lineData: any) => {
  lines.value = lineData;
  localStorage.setItem("lineData", JSON.stringify(lineData));
};

const handleLineSelect = (line: any) => {
  console.log(line);
};

const addNewLine = () => {
  imageCanvasRef.value?.addLine(selectedDirection.value);
};

const deleteLine = (index: number) => {
  imageCanvasRef.value?.removeLine(index);
};

async function handleAddLocation(value: { name: string; description: string }) {
  const { success, data } = await createLocation(value);
  if (success) {
    await listLocations();

    form.setFieldValue("location", data?.id);
    toast.toast({
      title: "Success",
      description: "New location created successfully",
    });
  } else {
    toast.toast({
      title: "Error",
      description: "Failed to create location",
      variant: "destructive",
    });
  }
  isDialogOpen.value = false;
}

// Open dialog
function openAddLocationDialog() {
  isDialogOpen.value = true;
}

const goToListPage = () => {
  router.push("/camera");
};

const loadCameraData = async () => {
  isLoading.value = true;
  try {
    // Get camera data
    const { success, data, lines: cameraLines } = await getCamera(cameraId);

    if (success && data) {
      cameraData.value = data;

      // Set form values from camera data
      form.setFieldValue("name", data.Name || "");
      form.setFieldValue("location", data.Location.id || "");
      form.setFieldValue("description", data.Description || "");

      // Parse connection details
      form.setFieldValue("schema", data.Schema || "rtsp");
      form.setFieldValue("host", data.Host);
      form.setFieldValue("port", `${data.Port}`);
      form.setFieldValue("path", data.Path);
      form.setFieldValue("username", data.Username);
      form.setFieldValue("password", data.Password);

      // Set direction and lines
      if (data.Direction) {
        selectedDirection.value = data.Direction;
      }

      if (cameraLines && cameraLines.length) {
        lines.value = cameraLines.map((line: any) => ({
          start: line.start,
          end: line.end,
          direction: line.direction || selectedDirection.value,
          color: line.color,
        }));
      }

      if (data.Tags) {
        tagsModel.value = data.Tags.split(",");
      }

      if (data.ImageData) {
        base64Image.value = data.ImageData;
      }
    } else {
      toast.toast({
        title: "Error",
        description: "Failed to load camera data",
        variant: "destructive",
      });
    }
  } catch (error) {
    console.error("Error loading camera data:", error);
    toast.toast({
      title: "Error",
      description:
        "Error loading camera data: " +
        (error instanceof Error ? error.message : String(error)),
      variant: "destructive",
    });
  } finally {
    isLoading.value = false;
  }
};

watch(
  () => form.values,
  (newState) => {
    try {
      rtspUrl.value = convertToRtspUrl(newState);
      isValidUrl.value = true;
      connectionStatus.value = null;
    } catch {
      rtspUrl.value = "";
      isValidUrl.value = false;
      connectionStatus.value = null;
    }
  },
  { deep: true }
);

watch(
  () => tagsModel.value,
  (tags) => {
    form.setFieldValue("tags", tags.join(","));
  }
);

const checkUrlParams = () => {
  if (route.query.setupConfig === "true") {
    showSetupConfig.value = true;
  }
};

onMounted(async () => {
  await Promise.all([listLocations(), loadCameraData()]);
  checkUrlParams();
});
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
        <h2 class="text-lg font-medium text-gray-800">Edit Camera</h2>
      </div>
      <div class="flex items-center space-x-2">
        <Button
          variant="outline"
          type="button"
          v-if="showSetupConfig"
          @click="toggleSetupConfig"
        >
          <ArrowLeft class="w-4 h-4 mr-1.5" /> Back
        </Button>
        <Button variant="outline" type="button" @click="goToListPage"
          >Cancel</Button
        >
        <Button type="button" @click="onSubmit" :disabled="isSaving">
          <Icon
            v-if="isSaving"
            icon="line-md:loading-twotone-loop"
            class="w-4 h-4 mr-1.5"
          />
          {{ isSaving ? "Saving..." : "Save Changes" }}
        </Button>
      </div>
    </div>

    <!-- Main content area -->
    <div class="flex-1">
      <form @submit.prevent="onSubmit">
        <!-- Camera config section -->
        <div
          v-show="!showSetupConfig"
          class="grid grid-cols-1 md:grid-cols-8 gap-6 h-full"
        >
          <!-- Left column - Combined Camera information & People Flow Counting -->
          <Card
            class="shadow-sm border-gray-200 h-full flex flex-col col-span-5"
          >
            <CardHeader class="px-4 py-3 bg-gray-50">
              <CardTitle class="text-sm font-medium"
                >Camera Information</CardTitle
              >
            </CardHeader>
            <CardContent class="p-4 flex-1">
              <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-2">
                <!-- Name field -->
                <FormField
                  v-slot="{ componentField, errors }"
                  name="name"
                  class="md:col-span-2"
                >
                  <FormItem :class="{ 'has-error': errors.length > 0 }">
                    <FormLabel class="text-xs font-medium text-gray-700"
                      >Name</FormLabel
                    >
                    <FormControl>
                      <Input
                        type="text"
                        v-bind="componentField"
                        :class="{
                          'border-red-500 focus:ring-red-500':
                            errors.length > 0,
                        }"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <!-- Location field -->
                <FormField v-slot="{ componentField, errors }" name="location">
                  <FormItem :class="{ 'has-error': errors.length > 0 }">
                    <FormLabel class="text-xs font-medium text-gray-700"
                      >Location</FormLabel
                    >
                    <FormControl>
                      <div class="flex items-center space-x-2">
                        <Select
                          v-bind="componentField"
                          :class="{ 'border-red-500': errors.length > 0 }"
                        >
                          <FormControl>
                            <SelectTrigger
                              :class="{
                                'border-red-500 focus:ring-red-500':
                                  errors.length > 0,
                              }"
                            >
                              <SelectValue placeholder="Select a Location" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectGroup>
                              <SelectItem
                                v-for="option in locationsState"
                                :value="option.id"
                                :key="option.id"
                              >
                                {{ option.name }}
                              </SelectItem>
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                        <Button
                          class="flex-none h-9"
                          size="sm"
                          type="button"
                          @click="openAddLocationDialog"
                        >
                          <Plus class="w-4 h-4" />
                        </Button>
                      </div>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <!-- Description field -->
                <FormField
                  v-slot="{ componentField, errors }"
                  name="description"
                  class="md:col-span-2"
                >
                  <FormItem :class="{ 'has-error': errors.length > 0 }">
                    <FormLabel class="text-xs font-medium text-gray-700"
                      >Description</FormLabel
                    >
                    <FormControl>
                      <Input
                        type="text"
                        v-bind="componentField"
                        :class="{
                          'border-red-500 focus:ring-red-500':
                            errors.length > 0,
                        }"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <!-- Tags field -->
                <FormField
                  v-slot="{ errorMessage, errors }"
                  name="tags"
                  class="md:col-span-2"
                >
                  <FormItem :class="{ 'has-error': errors.length > 0 }">
                    <FormLabel class="text-xs font-medium text-gray-700"
                      >Tags</FormLabel
                    >
                    <FormControl>
                      <TagsInput v-model="tagsModel">
                        <TagsInputItem
                          v-for="item in tagsModel"
                          :key="item"
                          :value="item"
                        >
                          <TagsInputItemText />
                          <TagsInputItemDelete />
                        </TagsInputItem>

                        <TagsInputInput />
                      </TagsInput>
                    </FormControl>
                    <div
                      v-if="errors.length > 0"
                      class="text-red-500 text-xs mt-1"
                    >
                      {{ errorMessage }}
                    </div>
                  </FormItem>
                </FormField>
              </div>
            </CardContent>

            <!-- People Flow Counting section -->
            <Separator class="my-2" />

            <div class="px-4 py-3 bg-gray-50 flex justify-between items-center">
              <h3 class="text-sm font-medium">People Flow Counting</h3>
              <Button type="button" size="sm" @click="toggleSetupConfig"
                >Setup Config</Button
              >
            </div>
          </Card>

          <!-- Right column - Connection settings -->
          <Card
            class="shadow-sm border-gray-200 h-full flex flex-col col-span-3"
          >
            <CardHeader class="px-4 py-3 bg-gray-50">
              <CardTitle class="text-sm font-medium"
                >Connection Setting</CardTitle
              >
            </CardHeader>
            <CardContent
              class="p-4 grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-2 flex-1"
            >
              <!-- Schema field -->
              <FormField v-slot="{ componentField, errors }" name="schema">
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700"
                    >Schema</FormLabel
                  >
                  <Select v-bind="componentField">
                    <FormControl>
                      <SelectTrigger
                        :class="{
                          'border-red-500 focus:ring-red-500':
                            errors.length > 0,
                        }"
                      >
                        <SelectValue placeholder="Select a Schema" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem
                          v-for="option in schemas"
                          :value="option.value"
                          :key="option.value"
                        >
                          {{ option.name }}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Host field -->
              <FormField v-slot="{ componentField, errors }" name="host">
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700"
                    >Host/IP</FormLabel
                  >
                  <FormControl>
                    <Input
                      type="text"
                      v-bind="componentField"
                      :class="{
                        'border-red-500 focus:ring-red-500': errors.length > 0,
                      }"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Port field -->
              <FormField v-slot="{ componentField, errors }" name="port">
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700"
                    >Port</FormLabel
                  >
                  <FormControl>
                    <Input
                      type="text"
                      v-bind="componentField"
                      :class="{
                        'border-red-500 focus:ring-red-500': errors.length > 0,
                      }"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Path field -->
              <FormField v-slot="{ componentField, errors }" name="path">
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700"
                    >Path</FormLabel
                  >
                  <FormControl>
                    <Input
                      type="text"
                      v-bind="componentField"
                      :class="{
                        'border-red-500 focus:ring-red-500': errors.length > 0,
                      }"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Username field -->
              <FormField
                v-slot="{ componentField, errors }"
                name="username"
                class="md:col-span-2"
              >
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700"
                    >Username</FormLabel
                  >
                  <FormControl>
                    <Input
                      type="text"
                      v-bind="componentField"
                      :class="{
                        'border-red-500 focus:ring-red-500': errors.length > 0,
                      }"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Password field -->
              <FormField
                v-slot="{ componentField, errors }"
                name="password"
                class="md:col-span-2"
              >
                <FormItem :class="{ 'has-error': errors.length > 0 }">
                  <FormLabel class="text-xs font-medium text-gray-700"
                    >Password</FormLabel
                  >
                  <FormControl>
                    <Input
                      type="password"
                      v-bind="componentField"
                      :class="{
                        'border-red-500 focus:ring-red-500': errors.length > 0,
                      }"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Connection URL alert -->
              <Alert class="md:col-span-2">
                <AlertTitle class="text-xs font-medium"
                  >Camera will connect at:</AlertTitle
                >
                <AlertDescription class="break-words">
                  <code class="text-xs">{{ rtspUrl }}</code>
                </AlertDescription>
              </Alert>

              <!-- Connection test result alert -->
              <Alert
                v-if="connectionStatus"
                class="md:col-span-2"
                :variant="connectionStatus.success ? 'default' : 'destructive'"
              >
                <AlertTitle class="text-xs font-medium">
                  {{
                    connectionStatus.success
                      ? "Connection successful"
                      : "Connection failed"
                  }}
                </AlertTitle>
                <AlertDescription class="text-xs">
                  {{ connectionStatus.message }}
                  <div v-if="connectionStatus.error" class="mt-2">
                    Error: {{ connectionStatus.error }}
                  </div>
                </AlertDescription>
              </Alert>
            </CardContent>

            <CardFooter class="p-4 bg-gray-50 border-t border-gray-200 mt-auto">
              <Button
                @click.prevent="testConnection"
                :disabled="!isValidUrl || isTestingConnection"
                type="button"
                size="sm"
                class="w-full"
              >
                {{ isTestingConnection ? "Testing..." : "Test Connection" }}
              </Button>
            </CardFooter>
          </Card>
        </div>

        <!-- Configuration setup section -->
        <div
          v-if="showSetupConfig"
          class="grid grid-cols-1 md:grid-cols-5 gap-4 h-full"
        >
          <!-- Left column - Image canvas -->
          <div class="md:col-span-3">
            <Card class="shadow-sm border-gray-200 h-full">
              <CardHeader class="px-4 py-3 bg-gray-50">
                <CardTitle class="text-sm font-medium"
                  >Configuration Setup</CardTitle
                >
              </CardHeader>
              <CardContent class="p-4 h-[calc(100%-56px)]">
                <ImageCanvas
                  ref="imageCanvasRef"
                  :defaultLines="lines"
                  :image-src="base64Image"
                  @onChange="handleChangeLines"
                  @onSelectedLine="handleLineSelect"
                  :direction="selectedDirection"
                  class="h-full"
                />
              </CardContent>
            </Card>
          </div>

          <!-- Right column - Line configuration -->
          <div class="md:col-span-2">
            <Card class="shadow-sm border-gray-200 h-full flex flex-col">
              <CardHeader class="px-4 py-3 bg-gray-50">
                <CardTitle class="text-sm font-medium"
                  >Line Configuration</CardTitle
                >
              </CardHeader>
              <CardContent class="p-4 space-y-4 flex-1">
                <!-- Direction selection -->
                <div class="space-y-2">
                  <Label class="text-xs font-medium text-gray-700"
                    >Direction</Label
                  >
                  <Select v-model="selectedDirection">
                    <SelectTrigger>
                      <SelectValue placeholder="Select Direction">
                        <div class="flex items-center gap-2">
                          <Icon
                            :icon="icons[selectedDirection]"
                            class="w-4 h-4"
                          />
                          <span class="text-xs">
                            {{
                              directions.find(
                                (d) => d.direction === selectedDirection
                              )?.name
                            }}
                          </span>
                        </div>
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectLabel class="text-xs">Direction</SelectLabel>
                        <SelectItem
                          :value="option.direction"
                          v-for="option in directions"
                          :key="option.direction"
                        >
                          <div class="flex items-center gap-2">
                            <Icon
                              :icon="icons[option.direction]"
                              class="w-4 h-4"
                            />
                            <span class="text-xs">{{ option.name }}</span>
                          </div>
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </div>

                <!-- Virtual lines list -->
                <div class="space-y-2 flex-1 flex flex-col">
                  <Label class="text-xs font-medium text-gray-700"
                    >Virtual Lines</Label
                  >
                  <div
                    class="rounded border border-gray-200 p-3 flex flex-col gap-3 flex-1"
                  >
                    <ScrollArea class="flex-1" v-if="lines.length">
                      <ul class="list-none flex flex-col gap-2">
                        <li v-for="(line, index) in lines" :key="index">
                          <div class="flex items-center gap-2 justify-between">
                            <div class="flex items-center gap-2">
                              <div
                                class="w-2 h-2 rounded-full"
                                :style="{
                                  backgroundColor: line.color || '#4B39EF',
                                }"
                              ></div>
                              <div class="text-xs">
                                Virtual Line {{ index + 1 }}
                              </div>
                            </div>
                            <Button
                              type="button"
                              @click="deleteLine(index)"
                              variant="destructive"
                              size="sm"
                              class="h-7 w-7 p-0"
                            >
                              <Trash2 class="w-3.5 h-3.5" />
                            </Button>
                          </div>
                          <Separator class="my-2" />
                        </li>
                      </ul>
                    </ScrollArea>

                    <div
                      v-else
                      class="flex-1 flex items-center justify-center text-xs text-gray-500"
                    >
                      No virtual lines defined
                    </div>

                    <Button
                      type="button"
                      @click="addNewLine"
                      variant="outline"
                      class="w-full h-8 text-xs gap-1.5"
                    >
                      <Plus class="w-3.5 h-3.5" />
                      Add Line
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </form>
    </div>
  </div>

  <!-- Location dialog -->
  <AddLocationDialog v-model:open="isDialogOpen" @submit="handleAddLocation" />
</template>
