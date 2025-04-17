type LogFunction = (
  message: string,
  type: "info" | "error" | "success" | "warning"
) => void;
type StatusUpdateFunction = (status: string) => void;
type LoadingFunction = (loading: boolean) => void;

export function useServiceControl(
  updateStatus: StatusUpdateFunction,
  setLoading: LoadingFunction
) {
  /**
   * Start the Windows service
   */
  const startService = async (): Promise<void> => {
    setLoading(true);

    try {
      const result = await StartService();

      // Refresh the status
      const status = await GetServiceStatus();
      updateStatus(status);
    } catch (error: any) {
      const errorMessage = error.message || String(error);
    } finally {
      setLoading(false);
    }
  };

  /**
   * Stop the Windows service
   */
  const stopService = async (): Promise<void> => {
    setLoading(true);

    try {
      const result = await StopService();

      // Refresh the status
      const status = await GetServiceStatus();
      updateStatus(status);
    } catch (error: any) {
      const errorMessage = error.message || String(error);
    } finally {
      setLoading(false);
    }
  };

  /**
   * Restart the Windows service
   */
  const restartService = async (): Promise<void> => {
    setLoading(true);

    try {
      const result = await RestartService();

      // Refresh the status
      const status = await GetServiceStatus();
      updateStatus(status);
    } catch (error: any) {
      const errorMessage = error.message || String(error);
    } finally {
      setLoading(false);
    }
  };

  /**
   * Ensure the service is running
   */
  const ensureRunning = async (): Promise<void> => {
    setLoading(true);

    try {
      const result = await EnsureServiceRunning();

      // Refresh the status
      const status = await GetServiceStatus();
      updateStatus(status);
    } catch (error: any) {
      const errorMessage = error.message || String(error);
    } finally {
      setLoading(false);
    }
  };

  return {
    startService,
    stopService,
    restartService,
    ensureRunning,
  };
}
