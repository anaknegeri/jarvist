export interface RTSPResponse {
  success: boolean;
  message: string;
  url?: string;
  timestamp: string;
  error?: string;
  data?: any;
  screenshotPath?: string;
}

// State for RTSP connections
export const rtspStatus = ref<Record<string, RTSPResponse>>({});

// Check RTSP connection with URL
export async function checkRTSPConnection(
  rtspURL: string,
  takeScreenshot: boolean = false
): Promise<RTSPResponse> {
  try {
    // Call backend function
    const jsonResponse = await CheckConnection(rtspURL, takeScreenshot);

    // Parse response
    const response: RTSPResponse = JSON.parse(jsonResponse);

    // Store in state
    if (response.url) {
      rtspStatus.value[response.url] = response;
    }

    return response;
  } catch (error) {
    console.error("Error checking RTSP connection:", error);
    return {
      success: false,
      message: "Error checking RTSP connection",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

// Check RTSP connection with config
export async function checkRTSPWithConfig(
  config: RTSPConfig,
  takeScreenshot: boolean = false
): Promise<RTSPResponse> {
  try {
    // Call backend function
    const jsonResponse = await CheckConnectionWithConfig(config, takeScreenshot);

    // Parse response
    const response: RTSPResponse = JSON.parse(jsonResponse);

    // Store in state
    if (response.url) {
      rtspStatus.value[response.url] = response;
    }

    return response;
  } catch (error) {
    return {
      success: false,
      message: "Error checking RTSP connection",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}

// Generate RTSP URL from config
export async function generateRTSPURL(config: RTSPConfig): Promise<string> {
  try {
    return await GenerateRTSPURL(config);
  } catch (error) {
    console.error("Error generating RTSP URL:", error);
    throw error;
  }
}

// Convert config to RTSP URL (client-side function)
export function convertToRtspUrl(config: RTSPConfig): string {
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

// Helper function to format password
function formatPassword(password: string, encode: boolean): string {
  return encode ? encodeURIComponent(password) : password;
}
