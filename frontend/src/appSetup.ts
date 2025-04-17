import { registerEventHandlers } from "@/services/processManagerService";

export function initializeApp() {
  document.addEventListener("DOMContentLoaded", () => {
    // EventsOn("navigate", (path) => {
    //   router.push(path);
    // });

    registerEventHandlers();

    VerifyAllProcessStatusConsistency()
      .then(() => {
        console.log("Process status consistency verified on startup");
      })
      .catch((err: Error) => {
        console.error("Failed to verify process status consistency:", err);
      });
  });
}

export function setupAppCleanup() {
  window.addEventListener("beforeunload", () => {
    try {
      ForceRemoveOrphanedStatusFiles();
    } catch (error) {
      console.error("Error during cleanup:", error);
    }
  });
}
