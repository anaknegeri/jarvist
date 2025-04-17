<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Dialog, DialogContent, DialogTrigger } from "@/components/ui/dialog";
import {
  ChevronDown,
  ChevronRight,
  Circle,
  Download,
  FileText,
  RotateCw,
  X,
} from "lucide-vue-next";
import { ref } from "vue";

// Refs for controlling collapsible sections
const basicInfoOpen = ref(true);
const licenseOpen = ref(true);
const aboutOpen = ref(true);

// Refs for dialogs
const viewLicenseOpen = ref(false);
const renewLicenseOpen = ref(false);
const checkUpdateOpen = ref(false);

// Mock license data
const licenseData = {
  licenseType: "Enterprise",
  licenseKey: "JRVST-ENTP-2024-XXXX-XXXX",
  purchaseDate: "February 12, 2023",
  expiryDate: "February 12, 2026",
  seats: 150,
  usedSeats: 98,
  modules: [
    "Core",
    "Analytics",
    "Team Management",
    "Integration",
    "Advanced Security",
  ],
  supportLevel: "Premium",
};

// Mock update data
const updateData = {
  currentVersion: "1.0.1",
  latestVersion: "1.0.2",
  releaseDate: "February 15, 2025",
  updateSize: "24.5 MB",
  updateNotes: [
    "Fixed performance issues when handling large datasets",
    "Improved UI response time in dashboard view",
    "Added new visualization options in analytics module",
    "Enhanced security for API integrations",
    "Updated third-party dependencies",
  ],
  isUpdateAvailable: true,
};

function checkForUpdates() {
  // Mock checking for updates
  checkUpdateOpen.value = true;
}

function downloadUpdate() {
  // Mock downloading update
  alert(
    "Update download started. The application will restart after installation."
  );
  checkUpdateOpen.value = false;
}

function renewLicense() {
  // Mock renew license logic
  alert(
    "License renewed successfully. Your subscription is now active until February 12, 2027."
  );
  renewLicenseOpen.value = false;
}
</script>

<template>
  <div class="h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
    <SettingLayout>
      <div class="flex-1 overflow-y-auto p-6">
        <div class="max-w-3xl">
          <!-- Basic Information Section -->
          <Collapsible v-model:open="basicInfoOpen" class="mb-4">
            <Card class="shadow-sm border-gray-200 dark:border-gray-700">
              <CardHeader class="px-4 py-3 bg-gray-50 dark:bg-gray-800/50">
                <div class="flex items-center justify-between">
                  <h2 class="text-sm font-semibold">Basic Information</h2>
                  <CollapsibleTrigger asChild>
                    <button
                      class="focus:outline-none text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                    >
                      <ChevronRight v-if="!basicInfoOpen" :size="18" />
                      <ChevronDown v-else :size="18" />
                    </button>
                  </CollapsibleTrigger>
                </div>
              </CardHeader>

              <CollapsibleContent>
                <CardContent class="px-4 pt-2 pb-3">
                  <div class="grid gap-3">
                    <div class="grid grid-cols-2 gap-4">
                      <div>
                        <label
                          class="text-xs font-medium text-gray-500 dark:text-gray-400"
                          >Company Name</label
                        >
                        <div class="text-sm mt-1">Acme Corporation</div>
                      </div>
                      <div>
                        <label
                          class="text-xs font-medium text-gray-500 dark:text-gray-400"
                          >Contact Email</label
                        >
                        <div class="text-sm mt-1">contact@acmecorp.com</div>
                      </div>
                    </div>

                    <div>
                      <label
                        class="text-xs font-medium text-gray-500 dark:text-gray-400"
                        >Address</label
                      >
                      <div class="text-sm mt-1">
                        123 Main Street, Anytown, Anystate, 12345
                      </div>
                    </div>
                  </div>
                </CardContent>
              </CollapsibleContent>
            </Card>
          </Collapsible>

          <!-- License Section -->
          <Collapsible v-model:open="licenseOpen" class="mb-4">
            <Card class="shadow-sm border-gray-200 dark:border-gray-700">
              <CardHeader class="px-4 py-3 bg-gray-50 dark:bg-gray-800/50">
                <div class="flex items-center justify-between">
                  <h2 class="text-sm font-semibold">License</h2>
                  <CollapsibleTrigger asChild>
                    <button
                      class="focus:outline-none text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                    >
                      <ChevronRight v-if="!licenseOpen" :size="18" />
                      <ChevronDown v-else :size="18" />
                    </button>
                  </CollapsibleTrigger>
                </div>
              </CardHeader>

              <CollapsibleContent>
                <CardContent class="px-4 pt-2 pb-3">
                  <div class="flex flex-col gap-2">
                    <div class="text-sm font-medium">
                      Active until Feb 12, 2026
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      We will send you a notification upon subscription
                      expiration
                    </div>
                    <div class="mt-2">
                      <div class="flex items-center gap-2">
                        <div
                          class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2"
                        >
                          <div
                            class="bg-blue-500 dark:bg-blue-600 h-2 rounded-full w-3/4"
                          ></div>
                        </div>
                        <span class="text-xs font-medium">75%</span>
                      </div>
                      <div
                        class="text-xs text-gray-500 dark:text-gray-400 mt-1"
                      >
                        456 days remaining on your license
                      </div>
                    </div>
                  </div>
                </CardContent>
                <CardFooter
                  class="px-4 py-3 bg-gray-50 dark:bg-gray-800/30 border-t border-gray-200 dark:border-gray-700 flex justify-between"
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
                            Your JARVIST Enterprise License information
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
                            <div class="text-xs font-medium">License Type:</div>
                            <div class="col-span-2 text-xs">
                              {{ licenseData.licenseType }}
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
                            <div class="text-xs font-medium">
                              Purchase Date:
                            </div>
                            <div class="col-span-2 text-xs">
                              {{ licenseData.purchaseDate }}
                            </div>
                          </div>
                          <div class="grid grid-cols-3 items-center gap-3">
                            <div class="text-xs font-medium">Expiry Date:</div>
                            <div class="col-span-2 text-xs">
                              {{ licenseData.expiryDate }}
                            </div>
                          </div>
                          <div class="grid grid-cols-3 items-center gap-3">
                            <div class="text-xs font-medium">Seats:</div>
                            <div class="col-span-2 text-xs">
                              {{ licenseData.usedSeats }} /
                              {{ licenseData.seats }} ({{
                                Math.round(
                                  (licenseData.usedSeats / licenseData.seats) *
                                    100
                                )
                              }}% used)
                            </div>
                          </div>
                          <div class="grid grid-cols-3 items-start gap-3">
                            <div class="text-xs font-medium">Modules:</div>
                            <div class="col-span-2">
                              <ul
                                class="list-disc list-inside text-xs space-y-1"
                              >
                                <li
                                  v-for="module in licenseData.modules"
                                  :key="module"
                                >
                                  {{ module }}
                                </li>
                              </ul>
                            </div>
                          </div>
                          <div class="grid grid-cols-3 items-center gap-3">
                            <div class="text-xs font-medium">
                              Support Level:
                            </div>
                            <div class="col-span-2 text-xs">
                              {{ licenseData.supportLevel }}
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

                  <Dialog v-model:open="renewLicenseOpen">
                    <DialogTrigger asChild>
                      <Button size="sm" class="h-8 text-xs">
                        <RotateCw class="mr-1.5 h-3.5 w-3.5" />
                        Renew License
                      </Button>
                    </DialogTrigger>
                    <DialogContent class="p-0 overflow-hidden">
                      <div
                        class="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-4 py-3 flex items-center justify-between"
                      >
                        <div>
                          <h3 class="text-sm font-semibold">
                            Renew Your License
                          </h3>
                          <p class="text-xs text-gray-500 dark:text-gray-400">
                            Extend your JARVIST license for another year
                          </p>
                        </div>
                        <button
                          @click="renewLicenseOpen = false"
                          class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-400"
                        >
                          <X :size="16" />
                        </button>
                      </div>
                      <div class="px-4 py-4 overflow-y-auto max-h-[70vh]">
                        <p class="text-xs mb-3">
                          Your current license is valid until
                          <span class="font-medium">February 12, 2026</span>.
                          Renewing now will extend your license until
                          <span class="font-medium">February 12, 2027</span>.
                        </p>
                        <div
                          class="bg-gray-50 dark:bg-gray-800 p-3 rounded-md mb-3"
                        >
                          <div class="flex justify-between items-center">
                            <div>
                              <div class="text-xs font-medium">
                                Enterprise Plan - 1 Year Extension
                              </div>
                              <div
                                class="text-xs text-gray-500 dark:text-gray-400"
                              >
                                {{ licenseData.seats }} user seats with all
                                modules included
                              </div>
                            </div>
                            <div class="text-sm font-bold">$4,999</div>
                          </div>
                        </div>
                        <div class="flex items-center space-x-2">
                          <input
                            id="agree"
                            type="checkbox"
                            class="w-3 h-3 rounded border-gray-300"
                          />
                          <label for="agree" class="text-xs">
                            I agree to the
                            <a href="#" class="text-blue-600 hover:underline"
                              >terms and conditions</a
                            >
                          </label>
                        </div>
                      </div>
                      <div
                        class="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/30 px-4 py-3 flex justify-end space-x-2"
                      >
                        <Button
                          variant="outline"
                          size="sm"
                          class="h-8 text-xs"
                          @click="renewLicenseOpen = false"
                          >Cancel</Button
                        >
                        <Button
                          variant="default"
                          size="sm"
                          class="h-8 text-xs"
                          @click="renewLicense"
                          >Proceed to Payment</Button
                        >
                      </div>
                    </DialogContent>
                  </Dialog>
                </CardFooter>
              </CollapsibleContent>
            </Card>
          </Collapsible>

          <!-- About Section -->
          <Collapsible v-model:open="aboutOpen" class="mb-4">
            <Card class="shadow-sm border-gray-200 dark:border-gray-700">
              <CardHeader class="px-4 py-3 bg-gray-50 dark:bg-gray-800/50">
                <div class="flex items-center justify-between">
                  <h2 class="text-sm font-semibold">About JARVIST</h2>
                  <CollapsibleTrigger asChild>
                    <button
                      class="focus:outline-none text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                    >
                      <ChevronRight v-if="!aboutOpen" :size="18" />
                      <ChevronDown v-else :size="18" />
                    </button>
                  </CollapsibleTrigger>
                </div>
              </CardHeader>

              <CollapsibleContent>
                <CardContent class="px-4 pt-2 pb-3">
                  <div class="flex justify-between items-center">
                    <div class="space-y-1">
                      <div class="text-xs text-gray-600 dark:text-gray-400">
                        Current version: 1.0.1
                      </div>
                      <div class="text-xs text-gray-600 dark:text-gray-400">
                        Last Updated: November 08, 2024 at 13:00 WIB
                      </div>
                    </div>

                    <Dialog v-model:open="checkUpdateOpen">
                      <DialogTrigger asChild>
                        <Button
                          size="sm"
                          class="h-8 text-xs"
                          @click="checkForUpdates"
                        >
                          <RotateCw class="mr-1.5 h-3.5 w-3.5" />
                          Check for updates
                        </Button>
                      </DialogTrigger>
                      <DialogContent class="p-0 overflow-hidden">
                        <div
                          class="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-4 py-3 flex items-center justify-between"
                        >
                          <div>
                            <h3 class="text-sm font-semibold">
                              Software Update
                            </h3>
                            <p class="text-xs text-gray-500 dark:text-gray-400">
                              Check for the latest version of JARVIST
                            </p>
                          </div>
                          <button
                            @click="checkUpdateOpen = false"
                            class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-400"
                          >
                            <X :size="16" />
                          </button>
                        </div>
                        <div class="px-4 py-4 overflow-y-auto max-h-[70vh]">
                          <div
                            v-if="updateData.isUpdateAvailable"
                            class="bg-green-50 dark:bg-green-900/20 p-3 rounded-md border border-green-200 dark:border-green-900/30 mb-3"
                          >
                            <div
                              class="text-green-800 dark:text-green-400 text-xs font-medium flex items-center gap-2"
                            >
                              <Circle
                                class="w-2 h-2 fill-green-600 dark:fill-green-400 stroke-0"
                              />
                              Update available: Version
                              {{ updateData.latestVersion }}
                            </div>
                          </div>
                          <div
                            v-else
                            class="bg-gray-50 dark:bg-gray-800 p-3 rounded-md mb-3"
                          >
                            <div
                              class="text-gray-800 dark:text-gray-300 text-xs font-medium"
                            >
                              You're already on the latest version ({{
                                updateData.currentVersion
                              }})
                            </div>
                          </div>

                          <div class="grid gap-2.5">
                            <div class="grid grid-cols-3 items-center gap-3">
                              <div class="text-xs font-medium">
                                Current Version:
                              </div>
                              <div class="col-span-2 text-xs">
                                {{ updateData.currentVersion }}
                              </div>
                            </div>
                            <div class="grid grid-cols-3 items-center gap-3">
                              <div class="text-xs font-medium">
                                Latest Version:
                              </div>
                              <div class="col-span-2 text-xs">
                                {{ updateData.latestVersion }}
                              </div>
                            </div>
                            <div class="grid grid-cols-3 items-center gap-3">
                              <div class="text-xs font-medium">
                                Release Date:
                              </div>
                              <div class="col-span-2 text-xs">
                                {{ updateData.releaseDate }}
                              </div>
                            </div>
                            <div class="grid grid-cols-3 items-center gap-3">
                              <div class="text-xs font-medium">
                                Update Size:
                              </div>
                              <div class="col-span-2 text-xs">
                                {{ updateData.updateSize }}
                              </div>
                            </div>
                          </div>

                          <div class="mt-3">
                            <div class="text-xs font-medium mb-1.5">
                              What's New:
                            </div>
                            <ul class="list-disc list-inside text-xs space-y-1">
                              <li
                                v-for="note in updateData.updateNotes"
                                :key="note"
                              >
                                {{ note }}
                              </li>
                            </ul>
                          </div>
                        </div>
                        <div
                          class="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/30 px-4 py-3 flex justify-end space-x-2"
                        >
                          <Button
                            variant="outline"
                            size="sm"
                            class="h-8 text-xs"
                            @click="checkUpdateOpen = false"
                            >Cancel</Button
                          >
                          <Button
                            v-if="updateData.isUpdateAvailable"
                            variant="default"
                            size="sm"
                            class="h-8 text-xs"
                            @click="downloadUpdate"
                          >
                            <Download class="mr-1.5 h-3.5 w-3.5" />
                            Download and Install
                          </Button>
                        </div>
                      </DialogContent>
                    </Dialog>
                  </div>
                </CardContent>
                <CardFooter
                  class="px-4 py-3 bg-gray-50 dark:bg-gray-800/30 border-t border-gray-200 dark:border-gray-700"
                >
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    © 2024 JARVIST Inc. All rights reserved.
                  </div>
                </CardFooter>
              </CollapsibleContent>
            </Card>
          </Collapsible>

          <!-- Status bar within main content area -->
          <div
            class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-4"
          >
            <div>
              Server status:
              <span class="text-green-500 dark:text-green-400">Online</span>
            </div>
            <div>Last sync: Today, 15:42</div>
          </div>
        </div>
      </div>
    </SettingLayout>

    <!-- Status bar - desktop app footer -->
    <div
      class="h-6 bg-gray-100 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-4 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400"
    >
      <div class="flex items-center space-x-4">
        <div class="flex items-center">
          <Circle class="w-2 h-2 fill-green-500 stroke-0 mr-1.5" />
          Connected
        </div>
        <div>v1.0.1</div>
      </div>
      <div>© 2024 JARVIST Inc.</div>
    </div>
  </div>
</template>
