export interface TimezoneResponse {
  success: boolean;
  message: string;
  data?: TimeZone;
  error?: string;
  timestamp: string;
}

export const timezoneState = ref<TimeZone[]>([]);

export async function listTimezones(): Promise<TimezoneResponse> {
  try {
    const timezones = await GetTimeZones();

    timezoneState.value = timezones;

    return {
      success: true,
      message: "Timezones retrieved successfully",
      data: timezones[0],
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error listing timezones:", error);
    return {
      success: false,
      message: "Error retrieving timezones",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}
