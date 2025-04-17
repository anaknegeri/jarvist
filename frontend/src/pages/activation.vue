<script setup lang="ts">
import { useToast } from "@/components/ui/toast";
import { AlertCircle, Check, Copy, Cpu, Key, Shield } from "lucide-vue-next";
import { onMounted, ref } from "vue";

interface RegistrationState {
  productKey: string;
  hardwareId: string;
  activated: boolean;
  registered: boolean;
  expiryDate: string | null;
  maxCameras: number;
}

const state = ref<RegistrationState>({
  productKey: "",
  hardwareId: "",
  activated: false,
  registered: false,
  expiryDate: null,
  maxCameras: 0,
});

const isLoading = ref(false);
const error = ref<string | null>(null);
const success = ref<string | null>(null);
const { toast } = useToast();
const copied = ref(false);
const titleBarStore = useTitleBarStore();
const router = useRouter();

const loadRegistrationData = async () => {
  state.value.hardwareId = await GetHardwareFingerprint();
};

const copyToClipboard = async () => {
  try {
    await navigator.clipboard.writeText(state.value.hardwareId);
    copied.value = true;
    setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch (err) {
    console.error("Failed to copy: ", err);
  }
};

const registerApp = async () => {
  isLoading.value = true;
  error.value = null;
  success.value = null;

  try {
    if (!validateProductKey(state.value.productKey)) {
      error.value = "Invalid product key format. Please enter a valid key.";
      isLoading.value = false;
      return;
    }

    const validate = await RegisterLicense(state.value.productKey);
    isLoading.value = false;

    if (validate.success) {
      toast({
        title: "Registration Successful",
        description: validate.message,
        variant: "default",
      });

      router.push("/config");
      return;
    }

    error.value = "Invalid product key. Please contact support.";
    return;
  } catch (err) {
    console.error("Error registering app:", err);
    error.value = "An error occurred during registration. Please try again.";
  } finally {
    isLoading.value = false;
  }
};

const formatDate = (dateString: string | null): string => {
  if (!dateString) return "N/A";
  return new Date(dateString).toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
};

const getDaysRemaining = (): number => {
  if (!state.value.expiryDate) return 0;

  const expiryDate = new Date(state.value.expiryDate);
  const today = new Date();
  const diffTime = expiryDate.getTime() - today.getTime();
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  return diffDays > 0 ? diffDays : 0;
};

onMounted(async () => {
  titleBarStore.setTitle("Jarvist AI - License Activation");
  await loadRegistrationData();

  const licenseDetail = await GetLicenseDetails();
  const isConfigured = await IsConfigured();

  if (licenseDetail.licensed && !isConfigured) {
    router.push("/config");
  }
});
</script>

<template>
  <div>
    <div class="flex justify-center mb-5">
      <div
        class="w-24 h-24 bg-gradient-to-r from-indigo-500 to-blue-600 rounded-full flex items-center justify-center shadow-lg"
      >
        <Shield class="w-12 h-12 text-white" />
      </div>
    </div>

    <h1
      class="text-center text-2xl font-bold text-gray-900 dark:text-gray-100 mb-1"
    >
      Jarvist AI
    </h1>
    <p class="text-center text-gray-500 dark:text-gray-400 mb-4">
      {{ state.registered ? "License Information" : "Software Registration" }}
    </p>

    <Card
      class="shadow-md border-0 bg-white/90 dark:bg-gray-700/50 backdrop-blur-sm"
    >
      <CardHeader
        class="text-center bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700"
      >
        <CardTitle>{{
          state.registered ? "License Status" : "Activate Your License"
        }}</CardTitle>
      </CardHeader>

      <CardContent class="p-6">
        <!-- Show registration form if not registered -->
        <div v-if="!state.registered" class="space-y-5">
          <div class="space-y-2">
            <Label for="hardwareId" class="text-sm font-medium"
              >Hardware ID</Label
            >
            <div class="flex items-center gap-2">
              <div class="relative flex-1">
                <Input
                  id="hardwareId"
                  v-model="state.hardwareId"
                  readonly
                  class="font-mono text-sm pr-10 bg-gray-50 dark:bg-gray-800 border-gray-300 dark:border-gray-600"
                />
                <Button
                  variant="ghost"
                  size="sm"
                  class="absolute right-1 top-1/2 transform -translate-y-1/2 h-7 w-8"
                  @click="copyToClipboard"
                  title="Copy to clipboard"
                >
                  <Check v-if="copied" class="w-4 h-4 text-green-500" />
                  <Copy v-else class="w-4 h-4" />
                </Button>
              </div>
            </div>
            <p class="text-xs text-muted-foreground">
              This is your unique hardware ID. You'll need to provide this when
              purchasing a license.
            </p>
          </div>

          <div class="space-y-2">
            <Label for="productKey" class="text-sm font-medium"
              >Product Key</Label
            >
            <Input
              id="productKey"
              v-model="state.productKey"
              placeholder="XXXX-XXXX-XXXX-XXXX"
              class="font-mono text-sm bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600"
            />
            <p class="text-xs text-muted-foreground">
              Enter the product key you received after purchase.
            </p>
          </div>

          <div
            v-if="error"
            class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md text-red-600 dark:text-red-400 text-sm flex items-start animate-in fade-in"
          >
            <AlertCircle class="w-4 h-4 mr-2 mt-0.5" />
            <span>{{ error }}</span>
          </div>

          <div
            v-if="success"
            class="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-md text-green-600 dark:text-green-400 text-sm flex items-start animate-in fade-in"
          >
            <Check class="w-4 h-4 mr-2 mt-0.5" />
            <span>{{ success }}</span>
          </div>
        </div>

        <!-- Show registration info if registered -->
        <div v-else class="space-y-5">
          <div class="flex items-center justify-center mb-2">
            <div
              class="w-16 h-16 bg-gradient-to-r from-green-400 to-emerald-500 rounded-full flex items-center justify-center"
            >
              <Check class="w-8 h-8 text-white" />
            </div>
          </div>

          <div
            class="border border-gray-200 dark:border-gray-600 rounded-lg divide-y divide-gray-200 dark:divide-gray-700 overflow-hidden"
          >
            <div
              class="flex justify-between p-3 bg-gray-50 dark:bg-gray-800/50"
            >
              <span class="text-gray-500 dark:text-gray-400">Status</span>
              <span
                class="font-medium text-gray-900 dark:text-gray-100 flex items-center"
              >
                <span class="w-2 h-2 bg-green-500 rounded-full mr-1.5"></span>
                {{ state.activated ? "Active" : "Inactive" }}
              </span>
            </div>

            <div class="flex justify-between p-3">
              <span class="text-gray-500 dark:text-gray-400">License Type</span>
              <span class="font-medium text-gray-900 dark:text-gray-100">
                {{
                  state.productKey.startsWith("DEMO") ? "Demo" : "Professional"
                }}
              </span>
            </div>

            <div
              class="flex justify-between p-3 bg-gray-50 dark:bg-gray-800/50"
            >
              <span class="text-gray-500 dark:text-gray-400">Expires On</span>
              <div class="text-right">
                <span
                  class="font-medium text-gray-900 dark:text-gray-100 block"
                >
                  {{ formatDate(state.expiryDate) }}
                </span>
                <span class="text-xs text-green-600 dark:text-green-400">
                  {{ getDaysRemaining() }} days remaining
                </span>
              </div>
            </div>

            <div class="flex justify-between p-3">
              <span class="text-gray-500 dark:text-gray-400">Camera Limit</span>
              <span class="font-medium text-gray-900 dark:text-gray-100">
                {{ state.maxCameras }}
              </span>
            </div>

            <div
              class="flex justify-between p-3 bg-gray-50 dark:bg-gray-800/50"
            >
              <span class="text-gray-500 dark:text-gray-400">Hardware ID</span>
              <div class="flex items-center gap-1">
                <span
                  class="font-mono text-xs text-gray-900 dark:text-gray-100"
                >
                  {{ state.hardwareId }}
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-6 w-6 p-0"
                  @click="copyToClipboard"
                >
                  <Check v-if="copied" class="w-3 h-3 text-green-500" />
                  <Copy v-else class="w-3 h-3" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </CardContent>

      <CardFooter
        class="px-6 py-4 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700"
      >
        <div v-if="!state.registered" class="w-full space-y-3">
          <Button
            class="w-full bg-gradient-to-r from-indigo-600 to-indigo-700 hover:from-indigo-700 hover:to-indigo-800 shadow-lg text-white"
            :disabled="isLoading || !state.productKey"
            @click="registerApp"
          >
            <Key v-if="!isLoading" class="w-4 h-4 mr-2" />
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
            {{ isLoading ? "Activating..." : "Activate License" }}
          </Button>
        </div>
        <div v-else class="w-full">
          <Button
            class="w-full bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 shadow-lg text-white"
            @click="$router ? $router.push('/') : null"
          >
            <Cpu class="w-4 h-4 mr-2" />
            Launch Application
          </Button>
        </div>
      </CardFooter>
    </Card>
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
