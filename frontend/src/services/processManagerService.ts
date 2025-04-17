import { Events } from "@wailsio/runtime";
import { ref, Ref } from "vue";

// Define interfaces
export interface ProcessState {
  running: boolean;
  status: string;
  outputs: string[];
  errors: string[];
  lastUpdated?: Date;
  pid?: number;
  statusLocked?: boolean; // Flag to lock status during transitions
}

export interface ProcessStatusMap {
  [key: string]: ProcessState;
}

export interface EventData {
  processId: string;
  message: string;
  pid?: number;
  timestamp: string;
  success: boolean;
  data?: any;
}

// Create reactive state
const processStatus: Ref<ProcessStatusMap> = ref({
  "people_counter.bat": {
    running: false,
    status: "Idle",
    outputs: [],
    errors: [],
    statusLocked: false,
  },
  "sync_manager.bat": {
    running: false,
    status: "Idle",
    outputs: [],
    errors: [],
    statusLocked: false,
  },
});

// Keep track of status change times
const statusChangeTimes: any = {
  "people_counter.bat": 0,
  "sync_manager.bat": 0,
};

// Variable to track if the event handlers are registered
let handlersRegistered = false;
const statusCheckInterval: number | null = null;
const unsubscribeFunctions: (() => void)[] = [];

// Log status changes for debugging
function logStatusChange(
  processId: string,
  oldStatus: string,
  newStatus: string
) {
  console.log(
    `[${new Date().toISOString()}] ${processId}: ${oldStatus} -> ${newStatus}`
  );
}

// Function to update process status with locking mechanism
function updateProcessStatus(
  processId: string,
  status: string,
  running: boolean,
  force: boolean = false
) {
  if (!processStatus.value[processId]) return;

  // Don't update if status is locked and force is false
  if (processStatus.value[processId].statusLocked && !force) {
    console.log(`Status update ignored for ${processId}: ${status} (locked)`);
    return;
  }

  const oldStatus = processStatus.value[processId].status;

  // Don't update if status is the same
  if (
    oldStatus === status &&
    processStatus.value[processId].running === running
  ) {
    return;
  }

  // Log change
  logStatusChange(processId, oldStatus, status);

  // Update status
  processStatus.value[processId].status = status;
  processStatus.value[processId].running = running;
  processStatus.value[processId].lastUpdated = new Date();

  // Record change time
  statusChangeTimes[processId] = Date.now();
}

// Lock status during critical operations
function lockProcessStatus(processId: string, lock: boolean) {
  if (processStatus.value[processId]) {
    processStatus.value[processId].statusLocked = lock;
    console.log(`${processId} status ${lock ? "locked" : "unlocked"}`);
  }
}

// Function to run a batch file
export function runBatFile(fileName: string): Promise<void> {
  // Ensure the process status exists
  if (!processStatus.value[fileName]) {
    processStatus.value[fileName] = {
      running: false,
      status: "Idle",
      outputs: [],
      errors: [],
      statusLocked: false,
    };
  }

  // Kunci status dan perbarui ke Starting
  lockProcessStatus(fileName, true);
  updateProcessStatus(fileName, "Starting...", true, true);

  // Clear outputs and errors for new run
  processStatus.value[fileName].outputs = [];
  processStatus.value[fileName].errors = [];

  // Panggil backend function
  return ProcessManagerService.RunBatFile(fileName)
    .then(() => {
      console.log(`Successfully called RunBatFile for ${fileName}`);

      // Jadwalkan transisi status
      // Pertama, ubah ke 'Loading' setelah 2 detik
      setTimeout(() => {
        // Pastikan status masih terkunci dan aplikasi dianggap berjalan
        if (processStatus.value[fileName].statusLocked) {
          updateProcessStatus(fileName, "Loading", true, true);
        }

        // Kemudian, cek status aktual setiap 2 detik hingga aplikasi benar-benar berjalan
        const statusCheckInterval = setInterval(() => {
          // Cek apakah proses masih berjalan
          ProcessManagerService.IsProcessRunning(fileName)
            .then((isRunning) => {
              if (!isRunning) {
                // Jika proses tidak berjalan lagi, update status dan bersihkan
                updateProcessStatus(fileName, "Stopped", false, true);
                lockProcessStatus(fileName, false);
                clearInterval(statusCheckInterval);
                return;
              }

              // Proses masih berjalan, cek status aktual
              ProcessManagerService.GetDetailedProcessStatus(fileName)
                .then((actualStatus) => {
                  if (actualStatus === "Running") {
                    // Jika status sudah "Running", update status dan buka kunci
                    updateProcessStatus(fileName, "Running", true, true);
                    lockProcessStatus(fileName, false);
                    clearInterval(statusCheckInterval);
                  } else if (
                    processStatus.value[fileName].status !== "Loading"
                  ) {
                    // Update status ke Loading jika saat ini bukan Loading
                    updateProcessStatus(fileName, "Loading", true, true);
                  }
                })
                .catch(() => {
                  // Fallback jika GetDetailedProcessStatus gagal
                  if (processStatus.value[fileName].statusLocked) {
                    updateProcessStatus(fileName, "Loading", true, true);
                  }
                });
            })
            .catch(() => {
              // Error fallback
              clearInterval(statusCheckInterval);
              lockProcessStatus(fileName, false);
            });
        }, 2000);

        // Batas waktu maksimum untuk kunci status (30 detik)
        setTimeout(() => {
          if (processStatus.value[fileName].statusLocked) {
            // Jika masih terkunci setelah 30 detik, buka kunci
            lockProcessStatus(fileName, false);
            clearInterval(statusCheckInterval);
          }
        }, 30000);
      }, 2000);
    })
    .catch((err: Error) => {
      console.error("Failed to start process:", err);

      // Update status to Error and unlock
      updateProcessStatus(fileName, "Error", false, true);
      lockProcessStatus(fileName, false);

      throw err;
    });
}

// Function to stop a process
export function stopProcess(fileName: string): Promise<boolean> {
  // Lock status and update to Stopping
  lockProcessStatus(fileName, true);
  updateProcessStatus(fileName, "Stopping...", true, true);

  // Call backend function
  return ProcessManagerService.StopProcess(fileName)
    .then((result) => {
      console.log(`Successfully called StopProcess for ${fileName}:`, result);

      // Keep status locked for 3 seconds
      setTimeout(() => {
        updateProcessStatus(fileName, "Stopped", false, true);
        lockProcessStatus(fileName, false);
      }, 3000);

      return result;
    })
    .catch((err: Error) => {
      console.error("Failed to stop process:", err);

      // Unlock after error
      setTimeout(() => {
        lockProcessStatus(fileName, false);
      }, 3000);

      throw err;
    });
}

// Cleanup function to unsubscribe all event handlers
export function cleanupEventHandlers(): void {
  // Unregister all event handlers
  unsubscribeFunctions.forEach((unsubscribe) => unsubscribe());
  unsubscribeFunctions.length = 0;
  handlersRegistered = false;
}

// Function to register all event handlers
export function registerEventHandlers(): void {
  if (handlersRegistered) {
    return;
  }

  // Unsubscribe from any existing handlers first
  cleanupEventHandlers();

  // Register event handlers using Wails v3 API
  const processStartedUnsubscribe = Events.On(
    "process_started",
    (event: Events.WailsEvent) => {
      try {
        const data: EventData = event.data;
        console.log("Event: process_started", data);

        // Only update if not locked
        if (
          processStatus.value[data.processId] &&
          !processStatus.value[data.processId].statusLocked
        ) {
          updateProcessStatus(data.processId, "Initializing", true, false);
        }
      } catch (error) {
        console.error("Failed to parse process_started event data:", error);
      }
    }
  );
  unsubscribeFunctions.push(processStartedUnsubscribe);

  // Process output event
  const processOutputUnsubscribe = Events.On(
    "process_output",
    (event: Events.WailsEvent) => {
      try {
        const data: EventData = event.data;

        if (processStatus.value[data.processId]) {
          // Add to outputs
          processStatus.value[data.processId].outputs.push(data.message);

          // Only update status based on message if not locked
          if (
            !processStatus.value[data.processId].statusLocked &&
            data.message.includes("[STATUS]")
          ) {
            const statusMessage = data.message
              .replace("[STATUS]", "")
              .trim()
              .toLowerCase();

            if (statusMessage.includes("loading")) {
              updateProcessStatus(data.processId, "Loading", true, false);
            } else if (
              statusMessage.includes("model loaded") ||
              statusMessage.includes("running")
            ) {
              updateProcessStatus(data.processId, "Running", true, false);
            }
          }
        }
      } catch (error) {
        console.error("Failed to parse process_output event data:", error);
      }
    }
  );
  unsubscribeFunctions.push(processOutputUnsubscribe);

  // Process error event
  const processErrorUnsubscribe = Events.On(
    "process_error",
    (event: Events.WailsEvent) => {
      try {
        const data: EventData = event.data;

        if (processStatus.value[data.processId]) {
          // Add to errors
          processStatus.value[data.processId].errors.push(data.message);

          // Only update status if error is critical and not locked
          if (
            !processStatus.value[data.processId].statusLocked &&
            data.message.includes("[ERROR]") &&
            (data.message.includes("failed") ||
              data.message.includes("unable to start"))
          ) {
            updateProcessStatus(data.processId, "Error", false, false);
          }
        }
      } catch (error) {
        console.error("Failed to parse process_error event data:", error);
      }
    }
  );
  unsubscribeFunctions.push(processErrorUnsubscribe);

  // Process status event
  const processStatusUnsubscribe = Events.On(
    "process_status",
    (event: Events.WailsEvent) => {
      try {
        const data: EventData = event.data;

        if (
          processStatus.value[data.processId] &&
          !processStatus.value[data.processId].statusLocked
        ) {
          if (data.data && data.data.status) {
            const status = data.data.status;

            // Determine running state based on status
            const running = status !== "Stopped" && status !== "Error";

            updateProcessStatus(data.processId, status, running, false);
          }
        }
      } catch (error) {
        console.error("Failed to parse process_status event data:", error);
      }
    }
  );
  unsubscribeFunctions.push(processStatusUnsubscribe);

  // Process completed event
  const processCompletedUnsubscribe = Events.On(
    "process_completed",
    (event: Events.WailsEvent) => {
      try {
        const data: EventData = event.data;
        console.log("Event: process_completed", data);

        if (processStatus.value[data.processId]) {
          // Only update if not locked or sufficient time has passed
          const timeSinceLastChange =
            Date.now() - statusChangeTimes[data.processId];

          if (
            !processStatus.value[data.processId].statusLocked ||
            timeSinceLastChange > 5000
          ) {
            if (!data.success) {
              updateProcessStatus(data.processId, "Error", false, false);
            } else {
              // If process was supposed to keep running, leave as "Running"
              if (processStatus.value[data.processId].status === "Running") {
                // Status remains unchanged
              } else {
                updateProcessStatus(data.processId, "Completed", false, false);
              }
            }

            // Unlock status after completion
            lockProcessStatus(data.processId, false);
          }
        }
      } catch (error) {
        console.error("Failed to parse process_completed event data:", error);
      }
    }
  );
  unsubscribeFunctions.push(processCompletedUnsubscribe);

  // Process stopped event
  const processStoppedUnsubscribe = Events.On(
    "process_stopped",
    (event: Events.WailsEvent) => {
      try {
        const data: EventData = event.data;
        console.log("Event: process_stopped", data);

        if (processStatus.value[data.processId]) {
          // Force update status to Stopped and unlock
          updateProcessStatus(data.processId, "Stopped", false, true);
          lockProcessStatus(data.processId, false);
        }
      } catch (error) {
        console.error("Failed to parse process_stopped event data:", error);
      }
    }
  );
  unsubscribeFunctions.push(processStoppedUnsubscribe);

  handlersRegistered = true;
}

// Function to check process status based on process manager
export function checkRunningProcesses(): Promise<void> {
  registerEventHandlers();

  return ProcessManagerService.CheckRunningProcesses();
}

// Function to get status class for UI
export function getStatusClass(fileName: string): string {
  const process = processStatus.value[fileName];
  if (!process)
    return "bg-gray-100 hover:bg-gray-100 text-gray-800 hover:text-gray-800";

  switch (process.status) {
    case "Initializing":
    case "Starting...":
      return "bg-blue-100 hover:bg-blue-100 text-blue-800 hover:text-blue-800";
    case "Loading":
      return "bg-yellow-100 hover:bg-yellow-100 text-yellow-800 hover:text-yellow-800";
    case "Running":
      return "bg-green-100 hover:bg-green-100 text-green-800 hover:text-green-800";
    case "Error":
      return "bg-red-100 hover:bg-red-100 text-red-800 hover:text-red-800";
    case "Stopping...":
      return "bg-orange-100 hover:bg-orange-100 text-orange-800 hover:text-orange-800";
    case "Stopped":
    case "Completed":
    case "Idle":
      return "bg-gray-100 hover:bg-gray-100 text-gray-800 hover:text-gray-800";
    default:
      return "bg-gray-100 hover:bg-gray-100 text-gray-800 hover:text-gray-800";
  }
}

// Export the reactive state
export { processStatus };
