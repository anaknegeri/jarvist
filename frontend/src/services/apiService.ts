import type { SiteCategory } from "bindings/jarvist/internal/wails/services/site";

export interface CategoryResponse {
  success: boolean;
  message: string;
  data?: SiteCategory;
  error?: string;
  timestamp: string;
}

export const siteCategoryState = ref<SiteCategory[]>([]);

export async function siteCategories(): Promise<CategoryResponse> {
  try {
    const categories = await GetSiteCetegories();

    siteCategoryState.value = categories;

    return {
      success: true,
      message: "Categories retrieved successfully",
      data: categories[0],
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    console.error("Error listing categories:", error);
    return {
      success: false,
      message: "Error retrieving categories",
      error: error instanceof Error ? error.message : String(error),
      timestamp: new Date().toISOString(),
    };
  }
}
