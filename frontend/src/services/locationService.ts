import type { Location } from "bindings/jarvist/internal/common/models/models";
import { ref } from "vue";

export interface LocationResponse {
  success: boolean;
  message: string;
  data?: Location | null;
  error?: string;
  timestamp: string;
}

// State management
export const locationsState = ref<Location[]>([]);

/**
 * Creates a new location
 * @param input Location input data
 * @returns Response with created location
 */
export async function createLocation(
  input: LocationInput
): Promise<LocationResponse> {
  try {
    const location = await LocationService.CreateLocation(input);

    if (location) {
      locationsState.value.push(location);

      return {
        success: true,
        message: "Location created successfully",
        data: location,
        timestamp: new Date().toISOString(),
      };
    }

    return {
      success: false,
      message: "Error creating location",
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error creating location:", error);
    return {
      success: false,
      message: "Error creating location",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

/**
 * Fetches all locations
 * @returns Response with locations list
 */
export async function listLocations(): Promise<LocationResponse> {
  try {
    const locations = await LocationService.ListLocations();

    locationsState.value = locations;

    return {
      success: true,
      message: "Locations retrieved successfully",
      data: locations.length > 0 ? locations[0] : null,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error listing locations:", error);
    return {
      success: false,
      message: "Error retrieving locations",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

/**
 * Updates an existing location
 * @param id Location ID
 * @param input Updated location data
 * @returns Response with updated location
 */
export async function updateLocation(
  id: string,
  input: LocationInput
): Promise<LocationResponse> {
  try {
    const updatedLocation = await LocationService.UpdateLocation(id, input);

    const index = locationsState.value.findIndex((p) => p.id === id);
    if (index !== -1 && updatedLocation) {
      locationsState.value[index] = updatedLocation;
    }

    return {
      success: true,
      message: "Location updated successfully",
      data: updatedLocation,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error updating location:", error);
    return {
      success: false,
      message: "Error updating location",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

/**
 * Deletes a location
 * @param id Location ID to delete
 * @returns Response indicating success or failure
 */
export async function deleteLocation(id: string): Promise<LocationResponse> {
  try {
    await LocationService.DeleteLocation(id);

    // Update local state
    locationsState.value = locationsState.value.filter((p) => p.id !== id);

    return {
      success: true,
      message: "Location deleted successfully",
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error deleting location:", error);
    return {
      success: false,
      message: "Error deleting location",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

/**
 * Gets a location by ID from local state
 * @param id Location ID
 * @returns Location object or undefined if not found
 */
export function getLocationById(id: string): Location | undefined {
  return locationsState.value.find((p) => p.id === id);
}
