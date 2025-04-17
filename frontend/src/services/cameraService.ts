import { Events } from "@wailsio/runtime";
import { reactive, ref } from "vue";

export interface LineData {
  start: CoordPoint;
  end: CoordPoint;
  direction: string;
  color?: string;
}

export interface CoordPoint {
  x: number;
  y: number;
}

export interface CameraResponse {
  success: boolean;
  message: string;
  data?: any;
  lines?: LineData[];
  error?: string;
  timestamp: string;
}

export interface CameraStatus {
  camera_id: number;
  camera_uuid: string;
  is_connected: boolean;
  last_checked: string;
  status_message?: string;
  error?: string;
}

// State management
export const camerasState = ref<any[]>([]);
export const cameraStatusesState = reactive<Record<string, CameraStatus>>({});
export const isLoading = ref(false);

// Get all cameras
export async function listCameras(): Promise<CameraResponse> {
  isLoading.value = true;
  try {
    const cameras = await CameraService.ListCamera();

    // Update local state
    camerasState.value = cameras;
    // Fetch camera statuses
    await getConnectionStatuses();

    return {
      success: true,
      message: "Cameras retrieved successfully",
      data: cameras.length > 0 ? cameras[0] : undefined,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error listing cameras:", error);
    return {
      success: false,
      message: "Error retrieving cameras",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  } finally {
    isLoading.value = false;
  }
}

// Create a new camera
export async function createCamera(input: any): Promise<CameraResponse> {
  isLoading.value = true;
  try {
    const camera = await CameraService.CreateCamera(input);

    // Add to local state
    if (camera) {
      camerasState.value.push(camera);

      // Check connection for the new camera
      checkCameraConnection(camera.ID);

      return {
        success: true,
        message: "Camera created successfully",
        data: camera,
        timestamp: new Date().toISOString(),
      };
    }

    return {
      success: false,
      message: "Failed to create camera - no data returned",
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error creating camera:", error);
    return {
      success: false,
      message: "Error creating camera",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  } finally {
    isLoading.value = false;
  }
}

// Get a specific camera
export async function getCamera(id: number): Promise<CameraResponse> {
  isLoading.value = true;
  try {
    const camera = await CameraService.GetCameraByID(id);
    const { 1: lines } = await CameraService.GetCameraWithLines(id);

    // Check connection for this camera if not already checked
    if (camera && !cameraStatusesState[camera.uuid]) {
      checkCameraConnection(id);
    }

    return {
      success: true,
      message: "Camera retrieved successfully",
      data: camera,
      lines: lines,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error getting camera:", error);
    return {
      success: false,
      message: "Error retrieving camera",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  } finally {
    isLoading.value = false;
  }
}

// Update an existing camera
export async function updateCamera(
  id: number,
  input: any
): Promise<CameraResponse> {
  isLoading.value = true;
  try {
    const updatedCamera = await CameraService.UpdateCamera(id, input);

    const index = camerasState.value.findIndex((c) => c.ID === id);
    if (index !== -1) {
      camerasState.value[index] = updatedCamera;
    }

    // Check connection for the updated camera
    checkCameraConnection(id);

    return {
      success: true,
      message: "Camera updated successfully",
      data: updatedCamera,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error updating camera:", error);
    return {
      success: false,
      message: "Error updating camera",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  } finally {
    isLoading.value = false;
  }
}

// Delete a camera
export async function deleteCamera(id: number): Promise<CameraResponse> {
  isLoading.value = true;
  try {
    await CameraService.DeleteCamera(id);

    // Find the camera to get its UUID
    const camera = camerasState.value.find((c) => c.ID === id);

    // Remove from local states
    camerasState.value = camerasState.value.filter((c) => c.ID !== id);
    if (camera && camera.UUID) {
      delete cameraStatusesState[camera.UUID];
    }

    return {
      success: true,
      message: "Camera deleted successfully",
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error deleting camera:", error);
    return {
      success: false,
      message: "Error deleting camera",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  } finally {
    isLoading.value = false;
  }
}

// Helper function to get a camera by ID
export function getCameraById(id: number): any | undefined {
  return camerasState.value.find((c) => c.ID === id);
}

// Check connection status for a specific camera
export async function checkCameraConnection(
  id: number
): Promise<CameraResponse> {
  try {
    const status = await CameraService.CheckCameraConnectionNow(id);

    // Update the status in the status state
    cameraStatusesState[status.camera_uuid] = status;

    // Also update connection info in the camera object
    const cameraIndex = camerasState.value.findIndex((c) => c.ID === id);
    if (cameraIndex !== -1) {
      const camera = camerasState.value[cameraIndex];
      camera.is_connected = status.is_connected;
      camera.last_checked = status.last_checked;
      camera.status_message = status.status_message;
    }

    return {
      success: true,
      message: "Connection checked successfully",
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error checking connection:", error);
    return {
      success: false,
      message: "Error checking connection",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

// Get connection status for all cameras
export async function getConnectionStatuses(): Promise<CameraResponse> {
  try {
    const statuses = await CameraService.GetAllConnectionStatuses();

    // Update all statuses in our state
    Object.assign(cameraStatusesState, statuses);

    // Update camera objects with connection info
    for (const [uuid, status] of Object.entries(statuses)) {
      const cameraIndex = camerasState.value.findIndex((c) => c.uuid === uuid);
      if (cameraIndex !== -1) {
        const camera = camerasState.value[cameraIndex];
        camera.is_connected = status.is_connected;
        camera.last_checked = status.last_checked;
        camera.status_message = status.status_message;
      }
    }

    return {
      success: true,
      message: "Retrieved connection statuses successfully",
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error getting connection statuses:", error);
    return {
      success: false,
      message: "Error getting connection statuses",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

// Set up event listener for connection status updates - menggunakan Wails v3 API yang benar
export function setupConnectionStatusListener() {
  const unsubscribe = Events.On(
    "camera:status-update",
    (event: Events.WailsEvent) => {
      try {
        const data = event.data as Record<string, CameraStatus>;

        // Update our status state
        Object.assign(cameraStatusesState, data);

        // Update camera objects with connection info
        for (const [camera_uuid, status] of Object.entries(data)) {
          const cameraIndex = camerasState.value.findIndex(
            (c) => c.uuid === camera_uuid
          );
          if (cameraIndex !== -1) {
            const camera = camerasState.value[cameraIndex];
            camera.is_connected = status.is_connected;
            camera.last_checked = status.last_checked;
            camera.status_message = status.status_message;
          }
        }
      } catch (e) {
        console.error("Error processing camera status update:", e);
      }
    }
  );

  // Return unsubscribe function for cleanup
  return unsubscribe;
}

// Clean up listeners - menggunakan Wails v3 API yang benar
export function cleanupConnectionStatusListener() {
  Events.Off("camera:status-update");
}
