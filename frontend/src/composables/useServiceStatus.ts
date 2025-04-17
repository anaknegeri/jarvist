type StatusUpdateFunction = (status: string) => void;
type LoadingFunction = (loading: boolean) => void;

export function useServiceStatus(
  updateStatus: StatusUpdateFunction,
  setLoading: LoadingFunction
) {
  /**
   * Get the current status of the Windows service
   * @param silent If true, don't set loading state (for background refreshes)
   */
  const getServiceStatus = async (silent: boolean = false): Promise<void> => {
    if (!silent) {
      setLoading(true);
    }

    try {
      const status = await GetServiceStatus();
      updateStatus(status);
    } catch (error: any) {
      console.error("Error getting service status:", error);
      updateStatus("Error");
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  };

  /**
   * Check if the service is installed
   */
  const isServiceInstalled = async (): Promise<boolean> => {
    try {
      return await IsServiceInstalled();
    } catch (error: any) {
      console.error("Error checking if service is installed:", error);
      return false;
    }
  };

  /**
   * Check if the service is running
   */
  const isServiceRunning = async (): Promise<boolean> => {
    try {
      return await IsServiceRunning();
    } catch (error: any) {
      console.error("Error checking if service is running:", error);
      return false;
    }
  };

  /**
   * Get detailed information about the service
   */
  const getServiceDetails = async (): Promise<any> => {
    try {
      return await GetServiceDetails();
    } catch (error: any) {
      console.error("Error getting service details:", error);
      return null;
    }
  };

  return {
    getServiceStatus,
    isServiceInstalled,
    isServiceRunning,
    getServiceDetails,
  };
}
