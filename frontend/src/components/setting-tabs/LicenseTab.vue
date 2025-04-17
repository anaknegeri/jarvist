<script setup lang="ts">
import { FileText, Key, X } from "lucide-vue-next";
import { onMounted, ref } from "vue";

const viewLicenseOpen = ref(false);
const isDeactivating = ref(false);
const deactivationSuccess = ref(false);
const deactivationError = ref("");
const deactivateDialogOpen = ref(false);

const licenseData = ref<any>({
  company: "",
  contactName: "",
  daysLeft: 0,
  deviceInfo: {
    Name: "",
    OS: "",
    Architecture: "",
  },
  email: "",
  expiryDate: "",
  issueDate: "",
  licenseKey: "",
  licensed: true,
  message: "",
  status: 1,
  daysRemaining: 0,
  percentRemaining: 0,
});

const calculateLicenseMetrics = (
  from: string | number | Date,
  until: string | number | Date
) => {
  const startDate = new Date(from);
  const endDate = new Date(until);
  const currentDate = new Date();

  const totalDays = Math.ceil(
    (endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24)
  );
  const daysRemaining = Math.ceil(
    (endDate.getTime() - currentDate.getTime()) / (1000 * 60 * 60 * 24)
  );
  const percentRemaining = Math.round((daysRemaining / totalDays) * 100);

  return { daysRemaining, percentRemaining };
};

const formatDate = (dateString: string | number | Date) => {
  const date = new Date(dateString);
  return date.toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
};

const handleDeactivation = async () => {
  isDeactivating.value = true;
  deactivationError.value = "";

  try {
    const { success } = await DeactivateLicense();
    if (success) {
      await CloseApp();
    } else {
      deactivationError.value =
        "Server error: Could not deactivate license. Please try again later.";
    }
  } catch (error) {
    deactivationError.value = "An unexpected error occurred";
  } finally {
    isDeactivating.value = false;
  }
};

onMounted(async () => {
  try {
    const data: any = await GetLicenseDetails();

    const { daysRemaining, percentRemaining } = calculateLicenseMetrics(
      data.issueDate,
      data.expiryDate
    );

    licenseData.value = {
      ...data,
      daysRemaining,
      percentRemaining,
    };

    console.log(licenseData.value);
  } catch (error) {
    console.error("Failed to load license data:", error);
  }
});
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle class="flex items-center">
        <Key class="mr-2 h-4 w-4" />
        License Information
      </CardTitle>
    </CardHeader>
    <CardContent class="space-y-6">
      <Separator />
      <Card>
        <CardContent class="px-4 pt-2 pb-3">
          <div class="flex flex-col gap-2">
            <div class="text-sm font-medium">
              Active until {{ formatDate(licenseData.expiryDate) }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              We will send you a notification upon subscription expiration
            </div>
            <div class="mt-2">
              <div class="flex items-center gap-2">
                <div
                  class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2"
                >
                  <div
                    class="bg-blue-500 dark:bg-blue-600 h-2 rounded-full"
                    :style="`width: ${licenseData.percentRemaining}%`"
                  ></div>
                </div>
                <span class="text-xs font-medium"
                  >{{ licenseData.percentRemaining }}%</span
                >
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                {{ licenseData.daysRemaining }} days remaining on your license
              </div>
            </div>
          </div>
        </CardContent>
        <CardFooter
          class="px-4 py-3 bg-gray-50 dark:bg-gray-800/30 border-t border-gray-200 dark:border-gray-700 flex justify-end rounded-b-lg"
        >
          <Dialog v-model:open="viewLicenseOpen">
            <DialogTrigger asChild>
              <Button variant="outline" size="sm" class="h-8 text-xs">
                <FileText class="mr-1.5 h-3.5 w-3.5" />
                View License Details
              </Button>
            </DialogTrigger>
            <DialogContent class="sm:max-w-md p-0 overflow-hidden">
              <div
                class="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-4 py-3 flex items-center justify-between"
              >
                <div>
                  <h3 class="text-sm font-semibold">License Details</h3>
                  <p class="text-xs text-gray-500 dark:text-gray-400">
                    Your JARVIST License information
                  </p>
                </div>
                <button
                  @click="viewLicenseOpen = false"
                  class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-400"
                >
                  <X :size="16" />
                </button>
              </div>
              <div class="px-4 py-4 overflow-y-auto max-h-[70vh]">
                <div class="grid gap-3">
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">Client:</div>
                    <div class="col-span-2 text-xs">
                      {{ licenseData.company }}
                    </div>
                  </div>
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">Email:</div>
                    <div class="col-span-2 text-xs">
                      {{ licenseData.email }}
                    </div>
                  </div>
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">License Key:</div>
                    <div
                      class="col-span-2 text-xs font-mono bg-gray-100 dark:bg-gray-800 p-1 rounded"
                    >
                      {{ licenseData.licenseKey }}
                    </div>
                  </div>
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">Status:</div>
                    <div class="col-span-2 text-xs">
                      <span
                        :class="
                          licenseData.licensed
                            ? 'text-green-600'
                            : 'text-red-600'
                        "
                      >
                        {{ licenseData.licensed ? "Active" : "Inactive" }}
                      </span>
                    </div>
                  </div>
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">Valid From:</div>
                    <div class="col-span-2 text-xs">
                      {{ formatDate(licenseData.issueDate) }}
                    </div>
                  </div>
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">Valid Until:</div>
                    <div class="col-span-2 text-xs">
                      {{ formatDate(licenseData.expiryDate) }}
                    </div>
                  </div>
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">Device ID:</div>
                    <div
                      class="col-span-2 font-mono bg-gray-100 dark:bg-gray-800 p-1 rounded text-xs"
                    >
                      {{ licenseData?.device_id }}
                    </div>
                  </div>
                  <div class="grid grid-cols-3 items-center gap-3">
                    <div class="text-xs font-medium">Registered At:</div>
                    <div class="col-span-2 text-xs">
                      {{ formatDate(licenseData.issueDate) }}
                    </div>
                  </div>
                </div>
              </div>
              <div
                class="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/30 px-4 py-3 flex justify-end space-x-2"
              >
                <Button
                  variant="outline"
                  size="sm"
                  class="h-8 text-xs"
                  @click="viewLicenseOpen = false"
                  >Close</Button
                >
                <Button variant="default" size="sm" class="h-8 text-xs"
                  >Contact Support</Button
                >
              </div>
            </DialogContent>
          </Dialog>
        </CardFooter>
      </Card>
      <!-- Deactivation -->
      <Alert variant="destructive">
        <AlertTitle>License Deactivation</AlertTitle>
        <AlertDescription class="space-y-2">
          <p>Deactivating your license will remove it from this device.</p>

          <Dialog v-model:open="deactivateDialogOpen">
            <DialogTrigger asChild>
              <Button variant="destructive" size="sm" class="h-8 text-xs">
                <X class="w-4 h-4 mr-1" />
                Deactivate License
              </Button>
            </DialogTrigger>
            <DialogContent class="sm:max-w-md p-0 overflow-hidden">
              <div
                class="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-4 py-3 flex items-center justify-between"
              >
                <div>
                  <h3 class="text-sm font-semibold">Deactivate License</h3>
                  <p class="text-xs text-gray-500 dark:text-gray-400">
                    This action cannot be undone
                  </p>
                </div>
                <button
                  @click="deactivateDialogOpen = false"
                  class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-400"
                  :disabled="isDeactivating"
                >
                  <X :size="16" />
                </button>
              </div>

              <div class="px-4 py-4">
                <div v-if="!deactivationSuccess">
                  <Alert variant="destructive" class="mb-4">
                    <AlertTitle>Warning</AlertTitle>
                    <AlertDescription>
                      Are you sure you want to deactivate your license on this
                      device? This will immediately end your license activation
                      and you'll need to reactivate it later.
                    </AlertDescription>
                  </Alert>

                  <div class="mb-4">
                    <p class="text-sm mb-2">License Key to deactivate:</p>
                    <div
                      class="text-sm font-mono bg-gray-100 dark:bg-gray-800 p-2 rounded"
                    >
                      {{ licenseData.licenseKey }}
                    </div>
                  </div>

                  <div class="mb-4">
                    <p class="text-sm mb-2">Device ID:</p>
                    <div
                      class="text-sm font-mono bg-gray-100 dark:bg-gray-800 p-2 rounded"
                    >
                      {{ licenseData.deviceInfo.device_id }}
                    </div>
                  </div>

                  <div
                    v-if="deactivationError"
                    class="mb-4 text-sm text-red-600 dark:text-red-500"
                  >
                    {{ deactivationError }}
                  </div>
                </div>

                <div v-if="deactivationSuccess" class="text-center py-4">
                  <div
                    class="mb-2 text-lg font-medium text-green-600 dark:text-green-500"
                  >
                    License Successfully Deactivated
                  </div>
                  <p class="text-sm text-gray-600 dark:text-gray-400">
                    Your license has been removed from this device
                  </p>
                </div>
              </div>

              <div
                class="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/30 px-4 py-3 flex justify-end space-x-2"
              >
                <Button
                  v-if="!deactivationSuccess"
                  variant="outline"
                  size="sm"
                  class="h-8 text-xs"
                  @click="deactivateDialogOpen = false"
                  :disabled="isDeactivating"
                  >Cancel</Button
                >
                <Button
                  v-if="!deactivationSuccess"
                  variant="destructive"
                  size="sm"
                  class="h-8 text-xs"
                  :disabled="isDeactivating"
                  @click="handleDeactivation"
                >
                  <span v-if="isDeactivating">Deactivating...</span>
                  <span v-else>Confirm Deactivation</span>
                </Button>

                <Button
                  v-if="deactivationSuccess"
                  variant="default"
                  size="sm"
                  class="h-8 text-xs"
                  @click="deactivateDialogOpen = false"
                  >Close</Button
                >
              </div>
            </DialogContent>
          </Dialog>
        </AlertDescription>
      </Alert>
    </CardContent>
  </Card>
</template>
