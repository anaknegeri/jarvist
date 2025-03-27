import vue from "@vitejs/plugin-vue";
import autoprefixer from "autoprefixer";
import tailwind from "tailwindcss";
import { defineConfig } from "vite";
import VueRouter from "unplugin-vue-router/vite";
import Components from "unplugin-vue-components/vite";
import Layouts from "vite-plugin-vue-layouts";
import {
  getPascalCaseRouteName,
  VueRouterAutoImports,
} from "unplugin-vue-router";
import Pages from "vite-plugin-pages";

// https://vitejs.dev/config/
export default defineConfig({
  css: {
    postcss: {
      plugins: [tailwind(), autoprefixer()],
    },
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
        },
      ],
    }),
    VueRouter({
      getRouteName: (routeNode) => {
        return getPascalCaseRouteName(routeNode)
          .replace(/([a-z\d])([A-Z])/g, "$1-$2")
          .toLowerCase();
      },
    }),
    Pages({
      dirs: ["./src/pages"],
      extendRoute(route) {
        if (route.meta?.layout) {
          return {
            ...route,
            meta: { layout: route.meta.layout },
          };
        }
      },
      exclude: [
        "./src/layouts/*.vue",
        "./src/pages/components/*.vue",
        "./src/pages/**/dto",
        "./src/pages/**/**/*.ts",
      ],
    }),
  ],
});
