/// <reference types="vite/client" />

declare module "virtual:generated-layouts" {
  import type { Router, RouteRecordRaw } from "vue-router";
  export function createGetRoutes(
    router: Router | any,
    withLayout?: boolean
  ): () => RouteRecordRaw[];
  export function setupLayouts(routes: RouteRecordRaw[]): RouteRecordRaw[];
}
