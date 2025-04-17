/**
 * Format number with commas as thousands separators
 */
export const formatNumber = (num: number): string => {
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
};

/**
 * Get current date in localized format
 */
export const getCurrentDate = (): string => {
  return new Date().toLocaleDateString("en-US", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
};

/**
 * Format the last checked time
 */
export const formatLastChecked = (lastChecked?: string): string => {
  if (!lastChecked) return "Never checked";

  try {
    const date = new Date(lastChecked);
    return date.toLocaleString();
  } catch (e) {
    console.error(e);
    return lastChecked;
  }
};

/**
 * Get current time as a string
 */
export const getCurrentTime = (): string => {
  return new Date().toLocaleTimeString();
};
