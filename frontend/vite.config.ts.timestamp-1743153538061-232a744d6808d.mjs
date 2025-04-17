// vite.config.ts
import vue from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/@vitejs/plugin-vue/dist/index.mjs";
import autoprefixer from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/autoprefixer/lib/autoprefixer.js";
import tailwind from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/tailwindcss/lib/index.js";
import AutoImport from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/unplugin-auto-import/dist/vite.js";
import Components from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/unplugin-vue-components/dist/vite.js";
import {
  getPascalCaseRouteName,
  VueRouterAutoImports
} from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/unplugin-vue-router/dist/index.js";
import VueRouter from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/unplugin-vue-router/dist/vite.js";
import { fileURLToPath } from "url";
import { defineConfig } from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/vite/dist/node/index.js";
import Pages from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/vite-plugin-pages/dist/index.js";
import Layouts from "file:///D:/Sandbox/jarvist-v2/frontend/node_modules/vite-plugin-vue-layouts/dist/index.mjs";
var __vite_injected_original_import_meta_url = "file:///D:/Sandbox/jarvist-v2/frontend/vite.config.ts";
var vite_config_default = defineConfig({
  css: {
    postcss: {
      plugins: [tailwind(), autoprefixer()]
    }
  },
  plugins: [
    vue(),
    Layouts(),
    Components({
      dirs: ["src/components"],
      extensions: ["vue"],
      resolvers: [
        (componentName) => {
          if (componentName == "Icon") {
            return { name: componentName, from: "@iconify/vue" };
          }
        }
      ]
    }),
    VueRouter({
      getRouteName: (routeNode) => {
        return getPascalCaseRouteName(routeNode).replace(/([a-z\d])([A-Z])/g, "$1-$2").toLowerCase();
      }
    }),
    Pages({
      dirs: ["./src/pages"],
      extendRoute(route) {
        if (route.meta?.layout) {
          return {
            ...route,
            meta: { layout: route.meta.layout }
          };
        }
      },
      exclude: [
        "./src/layouts/*.vue",
        "./src/pages/components/*.vue",
        "./src/pages/**/dto",
        "./src/pages/**/**/*.ts"
      ]
    }),
    AutoImport({
      imports: [
        "vue",
        VueRouterAutoImports,
        "@vueuse/core",
        "@vueuse/math",
        "pinia"
      ],
      dirs: ["src/router", "src/utils", "src/store", "./bindings/jarvist/**"]
    })
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", __vite_injected_original_import_meta_url))
    }
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCJEOlxcXFxTYW5kYm94XFxcXGphcnZpc3QtdjJcXFxcZnJvbnRlbmRcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIkQ6XFxcXFNhbmRib3hcXFxcamFydmlzdC12MlxcXFxmcm9udGVuZFxcXFx2aXRlLmNvbmZpZy50c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vRDovU2FuZGJveC9qYXJ2aXN0LXYyL2Zyb250ZW5kL3ZpdGUuY29uZmlnLnRzXCI7aW1wb3J0IHZ1ZSBmcm9tIFwiQHZpdGVqcy9wbHVnaW4tdnVlXCI7XG5pbXBvcnQgYXV0b3ByZWZpeGVyIGZyb20gXCJhdXRvcHJlZml4ZXJcIjtcbmltcG9ydCB0YWlsd2luZCBmcm9tIFwidGFpbHdpbmRjc3NcIjtcbmltcG9ydCBBdXRvSW1wb3J0IGZyb20gXCJ1bnBsdWdpbi1hdXRvLWltcG9ydC92aXRlXCI7XG5pbXBvcnQgQ29tcG9uZW50cyBmcm9tIFwidW5wbHVnaW4tdnVlLWNvbXBvbmVudHMvdml0ZVwiO1xuaW1wb3J0IHtcbiAgZ2V0UGFzY2FsQ2FzZVJvdXRlTmFtZSxcbiAgVnVlUm91dGVyQXV0b0ltcG9ydHMsXG59IGZyb20gXCJ1bnBsdWdpbi12dWUtcm91dGVyXCI7XG5pbXBvcnQgVnVlUm91dGVyIGZyb20gXCJ1bnBsdWdpbi12dWUtcm91dGVyL3ZpdGVcIjtcbmltcG9ydCB7IGZpbGVVUkxUb1BhdGggfSBmcm9tIFwidXJsXCI7XG5pbXBvcnQgeyBkZWZpbmVDb25maWcgfSBmcm9tIFwidml0ZVwiO1xuaW1wb3J0IFBhZ2VzIGZyb20gXCJ2aXRlLXBsdWdpbi1wYWdlc1wiO1xuaW1wb3J0IExheW91dHMgZnJvbSBcInZpdGUtcGx1Z2luLXZ1ZS1sYXlvdXRzXCI7XG5cbi8vIGh0dHBzOi8vdml0ZWpzLmRldi9jb25maWcvXG5leHBvcnQgZGVmYXVsdCBkZWZpbmVDb25maWcoe1xuICBjc3M6IHtcbiAgICBwb3N0Y3NzOiB7XG4gICAgICBwbHVnaW5zOiBbdGFpbHdpbmQoKSwgYXV0b3ByZWZpeGVyKCldLFxuICAgIH0sXG4gIH0sXG4gIHBsdWdpbnM6IFtcbiAgICB2dWUoKSxcbiAgICBMYXlvdXRzKCksXG4gICAgQ29tcG9uZW50cyh7XG4gICAgICBkaXJzOiBbXCJzcmMvY29tcG9uZW50c1wiXSxcbiAgICAgIGV4dGVuc2lvbnM6IFtcInZ1ZVwiXSxcbiAgICAgIHJlc29sdmVyczogW1xuICAgICAgICAoY29tcG9uZW50TmFtZSkgPT4ge1xuICAgICAgICAgIGlmIChjb21wb25lbnROYW1lID09IFwiSWNvblwiKSB7XG4gICAgICAgICAgICByZXR1cm4geyBuYW1lOiBjb21wb25lbnROYW1lLCBmcm9tOiBcIkBpY29uaWZ5L3Z1ZVwiIH07XG4gICAgICAgICAgfVxuICAgICAgICB9LFxuICAgICAgXSxcbiAgICB9KSxcbiAgICBWdWVSb3V0ZXIoe1xuICAgICAgZ2V0Um91dGVOYW1lOiAocm91dGVOb2RlKSA9PiB7XG4gICAgICAgIHJldHVybiBnZXRQYXNjYWxDYXNlUm91dGVOYW1lKHJvdXRlTm9kZSlcbiAgICAgICAgICAucmVwbGFjZSgvKFthLXpcXGRdKShbQS1aXSkvZywgXCIkMS0kMlwiKVxuICAgICAgICAgIC50b0xvd2VyQ2FzZSgpO1xuICAgICAgfSxcbiAgICB9KSxcbiAgICBQYWdlcyh7XG4gICAgICBkaXJzOiBbXCIuL3NyYy9wYWdlc1wiXSxcbiAgICAgIGV4dGVuZFJvdXRlKHJvdXRlKSB7XG4gICAgICAgIGlmIChyb3V0ZS5tZXRhPy5sYXlvdXQpIHtcbiAgICAgICAgICByZXR1cm4ge1xuICAgICAgICAgICAgLi4ucm91dGUsXG4gICAgICAgICAgICBtZXRhOiB7IGxheW91dDogcm91dGUubWV0YS5sYXlvdXQgfSxcbiAgICAgICAgICB9O1xuICAgICAgICB9XG4gICAgICB9LFxuICAgICAgZXhjbHVkZTogW1xuICAgICAgICBcIi4vc3JjL2xheW91dHMvKi52dWVcIixcbiAgICAgICAgXCIuL3NyYy9wYWdlcy9jb21wb25lbnRzLyoudnVlXCIsXG4gICAgICAgIFwiLi9zcmMvcGFnZXMvKiovZHRvXCIsXG4gICAgICAgIFwiLi9zcmMvcGFnZXMvKiovKiovKi50c1wiLFxuICAgICAgXSxcbiAgICB9KSxcbiAgICBBdXRvSW1wb3J0KHtcbiAgICAgIGltcG9ydHM6IFtcbiAgICAgICAgXCJ2dWVcIixcbiAgICAgICAgVnVlUm91dGVyQXV0b0ltcG9ydHMsXG4gICAgICAgIFwiQHZ1ZXVzZS9jb3JlXCIsXG4gICAgICAgIFwiQHZ1ZXVzZS9tYXRoXCIsXG4gICAgICAgIFwicGluaWFcIixcbiAgICAgIF0sXG4gICAgICBkaXJzOiBbXCJzcmMvcm91dGVyXCIsIFwic3JjL3V0aWxzXCIsIFwic3JjL3N0b3JlXCIsIFwiLi9iaW5kaW5ncy9qYXJ2aXN0LyoqXCJdLFxuICAgIH0pLFxuICBdLFxuICByZXNvbHZlOiB7XG4gICAgYWxpYXM6IHtcbiAgICAgIFwiQFwiOiBmaWxlVVJMVG9QYXRoKG5ldyBVUkwoXCIuL3NyY1wiLCBpbXBvcnQubWV0YS51cmwpKSxcbiAgICB9LFxuICB9LFxufSk7XG4iXSwKICAibWFwcGluZ3MiOiAiO0FBQW9SLE9BQU8sU0FBUztBQUNwUyxPQUFPLGtCQUFrQjtBQUN6QixPQUFPLGNBQWM7QUFDckIsT0FBTyxnQkFBZ0I7QUFDdkIsT0FBTyxnQkFBZ0I7QUFDdkI7QUFBQSxFQUNFO0FBQUEsRUFDQTtBQUFBLE9BQ0s7QUFDUCxPQUFPLGVBQWU7QUFDdEIsU0FBUyxxQkFBcUI7QUFDOUIsU0FBUyxvQkFBb0I7QUFDN0IsT0FBTyxXQUFXO0FBQ2xCLE9BQU8sYUFBYTtBQWJ1SixJQUFNLDJDQUEyQztBQWdCNU4sSUFBTyxzQkFBUSxhQUFhO0FBQUEsRUFDMUIsS0FBSztBQUFBLElBQ0gsU0FBUztBQUFBLE1BQ1AsU0FBUyxDQUFDLFNBQVMsR0FBRyxhQUFhLENBQUM7QUFBQSxJQUN0QztBQUFBLEVBQ0Y7QUFBQSxFQUNBLFNBQVM7QUFBQSxJQUNQLElBQUk7QUFBQSxJQUNKLFFBQVE7QUFBQSxJQUNSLFdBQVc7QUFBQSxNQUNULE1BQU0sQ0FBQyxnQkFBZ0I7QUFBQSxNQUN2QixZQUFZLENBQUMsS0FBSztBQUFBLE1BQ2xCLFdBQVc7QUFBQSxRQUNULENBQUMsa0JBQWtCO0FBQ2pCLGNBQUksaUJBQWlCLFFBQVE7QUFDM0IsbUJBQU8sRUFBRSxNQUFNLGVBQWUsTUFBTSxlQUFlO0FBQUEsVUFDckQ7QUFBQSxRQUNGO0FBQUEsTUFDRjtBQUFBLElBQ0YsQ0FBQztBQUFBLElBQ0QsVUFBVTtBQUFBLE1BQ1IsY0FBYyxDQUFDLGNBQWM7QUFDM0IsZUFBTyx1QkFBdUIsU0FBUyxFQUNwQyxRQUFRLHFCQUFxQixPQUFPLEVBQ3BDLFlBQVk7QUFBQSxNQUNqQjtBQUFBLElBQ0YsQ0FBQztBQUFBLElBQ0QsTUFBTTtBQUFBLE1BQ0osTUFBTSxDQUFDLGFBQWE7QUFBQSxNQUNwQixZQUFZLE9BQU87QUFDakIsWUFBSSxNQUFNLE1BQU0sUUFBUTtBQUN0QixpQkFBTztBQUFBLFlBQ0wsR0FBRztBQUFBLFlBQ0gsTUFBTSxFQUFFLFFBQVEsTUFBTSxLQUFLLE9BQU87QUFBQSxVQUNwQztBQUFBLFFBQ0Y7QUFBQSxNQUNGO0FBQUEsTUFDQSxTQUFTO0FBQUEsUUFDUDtBQUFBLFFBQ0E7QUFBQSxRQUNBO0FBQUEsUUFDQTtBQUFBLE1BQ0Y7QUFBQSxJQUNGLENBQUM7QUFBQSxJQUNELFdBQVc7QUFBQSxNQUNULFNBQVM7QUFBQSxRQUNQO0FBQUEsUUFDQTtBQUFBLFFBQ0E7QUFBQSxRQUNBO0FBQUEsUUFDQTtBQUFBLE1BQ0Y7QUFBQSxNQUNBLE1BQU0sQ0FBQyxjQUFjLGFBQWEsYUFBYSx1QkFBdUI7QUFBQSxJQUN4RSxDQUFDO0FBQUEsRUFDSDtBQUFBLEVBQ0EsU0FBUztBQUFBLElBQ1AsT0FBTztBQUFBLE1BQ0wsS0FBSyxjQUFjLElBQUksSUFBSSxTQUFTLHdDQUFlLENBQUM7QUFBQSxJQUN0RDtBQUFBLEVBQ0Y7QUFDRixDQUFDOyIsCiAgIm5hbWVzIjogW10KfQo=
