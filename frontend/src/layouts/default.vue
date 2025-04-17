<script setup>
import logo from "@/assets/logo-white.svg";
import { Application, Window } from "@wailsio/runtime";
import {
  Cctv,
  ChevronRight,
  Home,
  Menu,
  Minus,
  Settings,
  Shield,
  X,
} from "lucide-vue-next";
import { computed, onMounted, ref, watch } from "vue";

const router = useRouter();
const route = useRoute();
const collapsed = ref(false);
const currentTime = ref(
  new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
);
const currentDate = ref(
  new Date().toLocaleDateString([], {
    weekday: "short",
    month: "short",
    day: "numeric",
  })
);

// Update current time every minute
setInterval(() => {
  currentTime.value = new Date().toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
  currentDate.value = new Date().toLocaleDateString([], {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
}, 60000);

const mainMenuItems = [
  {
    icon: Home,
    label: "Dashboard",
    href: "/dashboard",
  },
  {
    icon: Shield,
    label: "Jarvist AI",
    href: "/jarvist",
  },
  {
    icon: Cctv,
    label: "Cameras",
    href: "/camera",
  },
];

const bottomMenuItems = [
  // {
  //   icon: SquareActivity,
  //   label: "Monitoring",
  //   action: () => handleMonitoringAction(),
  // },
  { icon: Settings, label: "Settings", href: "/settings" },
  // { icon: HelpCircle, label: "Help", href: "/help" },
];

const handleMonitoringAction = () => {
  OpenMonitoring();
};
// State for currently active menu and submenu
const activeMenuItem = computed(() => {
  const currentPath = route.path;

  // First check if we're in a submenu
  for (const item of mainMenuItems) {
    if (item.submenu) {
      for (const subItem of item.submenu) {
        if (
          currentPath === subItem.href ||
          currentPath.startsWith(subItem.href)
        ) {
          return item.label;
        }
      }
    }
  }

  // Then check main menu items
  for (const item of [...mainMenuItems, ...bottomMenuItems]) {
    if (currentPath === item.href || currentPath.startsWith(item.href)) {
      return item.label;
    }
  }

  // Default to Dashboard if no match
  return "Dashboard";
});

const activeSubmenu = ref(null);

// Watch for route changes to update active submenu
watch(
  () => route.path,
  (newPath) => {
    for (const item of mainMenuItems) {
      if (item.submenu) {
        for (const subItem of item.submenu) {
          if (newPath === subItem.href || newPath.startsWith(subItem.href)) {
            activeSubmenu.value = item.label;
            return;
          }
        }
      }
    }
    activeSubmenu.value = null;
  },
  { immediate: true }
);

const handleMenuClick = (item) => {
  if (item.submenu) {
    activeSubmenu.value =
      activeSubmenu.value === item.label ? null : item.label;
  } else if (item.action) {
    item.action();
  } else if (item.href) {
    activeSubmenu.value = null;
    router.push(item.href);
  }
};

const handleSubmenuClick = (submenuItem) => {
  if (submenuItem.href) {
    router.push(submenuItem.href);
  }
};

const toggleSidebar = () => {
  collapsed.value = !collapsed.value;
  // Save preference to localStorage
  localStorage.setItem("sidebarCollapsed", collapsed.value);
};

const windowClose = async () => {
  Application.Quit();
};

const windowMinimise = () => {
  Window.Minimise();
};

onMounted(() => {
  // Restore sidebar state from localStorage
  const savedState = localStorage.getItem("sidebarCollapsed");
  if (savedState !== null) {
    collapsed.value = savedState === "true";
  }

  // Automatically expand submenu if we're on a submenu page
  const currentPath = route.path;
  for (const item of mainMenuItems) {
    if (item.submenu) {
      for (const subItem of item.submenu) {
        if (
          currentPath === subItem.href ||
          currentPath.startsWith(subItem.href)
        ) {
          activeSubmenu.value = item.label;
          break;
        }
      }
    }
  }
});
</script>

<template>
  <div
    class="select-none w-full h-screen bg-white dark:bg-slate-900 rounded-lg shadow-xl overflow-hidden border border-gray-200 dark:border-slate-700 flex flex-col"
  >
    <!-- Title bar -->
    <div
      class="titlebar h-12 bg-white dark:bg-slate-900 flex items-center justify-between select-none drag border-b border-gray-100 dark:border-slate-800"
      style="--wails-draggable: drag"
    >
      <!-- Logo and title -->
      <div class="titlebar-dragarea flex items-center gap-3 h-full px-4">
        <div
          class="flex items-center justify-center w-6 h-6 bg-gradient-to-r from-blue-500 to-indigo-600 rounded"
        >
          <img :src="logo" class="h-4" />
        </div>
        <span class="font-semibold text-gray-800 dark:text-gray-100"
          >JARVIST</span
        >
      </div>

      <!-- Center controls with date and time -->
      <div class="flex items-center gap-4">
        <div class="text-xs text-gray-500 dark:text-gray-400 flex items-center">
          <span>{{ currentTime }}</span>
          <span class="mx-2">•</span>
          <span>{{ currentDate }}</span>
        </div>
      </div>

      <!-- Window controls -->
      <div class="flex h-full">
        <!-- Minimize -->
        <button
          @click="windowMinimise"
          class="h-full px-4 inline-flex text-gray-400 items-center justify-center hover:bg-gray-100 dark:hover:bg-slate-800 transition-colors"
        >
          <Minus class="w-4 h-4" />
        </button>

        <!-- Close -->
        <button
          @click="windowClose"
          class="h-full px-4 inline-flex items-center justify-center text-gray-400 hover:text-white hover:bg-red-500 transition-colors"
        >
          <X class="w-4 h-4" />
        </button>
      </div>
    </div>

    <div class="flex flex-1 h-[calc(100vh-48px)]">
      <!-- Sidebar -->
      <div
        class="h-full transition-all duration-300 border-r border-gray-100 dark:border-slate-800 flex flex-col bg-white dark:bg-slate-900 sidebar-transition"
        :class="collapsed ? 'w-16' : 'w-64'"
      >
        <!-- Sidebar header with collapse button -->
        <div
          class="p-4 border-b border-gray-100 dark:border-slate-800 flex items-center justify-between"
        >
          <h2
            v-if="!collapsed"
            class="font-medium text-gray-800 dark:text-gray-200"
          >
            Navigation
          </h2>
          <button
            @click="toggleSidebar"
            class="p-1.5 rounded-md bg-gray-100 dark:bg-slate-800 text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-slate-700"
            :title="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
          >
            <Menu class="w-4 h-4" />
          </button>
        </div>

        <!-- Main menu -->
        <div class="flex-1 py-4 overflow-y-auto">
          <div class="px-3 mb-6">
            <!-- Main menu items -->
            <div class="space-y-1">
              <div
                v-for="item in mainMenuItems"
                :key="item.label"
                class="relative"
              >
                <div
                  @click="handleMenuClick(item)"
                  class="flex items-center cursor-pointer transition-colors py-2 rounded-md px-2"
                  :class="{
                    'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400':
                      activeMenuItem === item.label ||
                      activeSubmenu === item.label,
                    'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-800 hover:text-gray-900 dark:hover:text-gray-100':
                      activeMenuItem !== item.label &&
                      activeSubmenu !== item.label,
                  }"
                  :title="item.label"
                >
                  <component :is="item.icon" class="w-5 h-5" />
                  <span v-if="!collapsed" class="ml-3 text-sm">{{
                    item.label
                  }}</span>
                  <ChevronRight
                    v-if="!collapsed && item.submenu"
                    class="w-4 h-4 ml-auto transition-transform"
                    :class="{ 'rotate-90': activeSubmenu === item.label }"
                  />
                </div>

                <!-- Submenu -->
                <div
                  v-if="
                    !collapsed && activeSubmenu === item.label && item.submenu
                  "
                  class="mt-1 ml-5 space-y-1 border-l border-gray-200 dark:border-slate-700 pl-3"
                >
                  <div
                    v-for="submenuItem in item.submenu"
                    :key="submenuItem.label"
                    @click="handleSubmenuClick(submenuItem)"
                    class="py-2 px-2 flex cursor-pointer rounded-md hover:bg-gray-100 dark:hover:bg-slate-800"
                    :class="{
                      'bg-blue-50/50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400':
                        route.path === submenuItem.href ||
                        route.path.startsWith(submenuItem.href),
                      'text-gray-600 dark:text-gray-300':
                        route.path !== submenuItem.href &&
                        !route.path.startsWith(submenuItem.href),
                    }"
                  >
                    <component
                      :is="submenuItem.icon"
                      class="w-4 h-4"
                      :class="{
                        'text-blue-500 dark:text-blue-400':
                          route.path === submenuItem.href ||
                          route.path.startsWith(submenuItem.href),
                        'text-gray-500 dark:text-gray-400':
                          route.path !== submenuItem.href &&
                          !route.path.startsWith(submenuItem.href),
                      }"
                    />
                    <div class="ml-3">
                      <div class="text-sm">{{ submenuItem.label }}</div>
                      <div class="text-xs text-gray-500 dark:text-gray-400">
                        {{ submenuItem.description }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Bottom menu items -->
        <div class="py-4 border-t border-gray-100 dark:border-slate-800">
          <div class="px-3 space-y-1">
            <div
              v-for="item in bottomMenuItems"
              :key="item.label"
              @click="handleMenuClick(item)"
              class="flex items-center cursor-pointer transition-colors py-2 rounded-md px-2"
              :class="{
                'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400':
                  activeMenuItem === item.label ||
                  route.path === item.href ||
                  route.path.startsWith(item.href),
                'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-800 hover:text-gray-900 dark:hover:text-gray-100':
                  activeMenuItem !== item.label &&
                  route.path !== item.href &&
                  !route.path.startsWith(item.href),
              }"
              :title="item.label"
            >
              <component :is="item.icon" class="w-5 h-5" />
              <span v-if="!collapsed" class="ml-3 text-sm">{{
                item.label
              }}</span>
            </div>
          </div>
        </div>
      </div>

      <router-view v-slot="{ Component, route }">
        <div
          class="flex-1 flex flex-col bg-[#f0f2f5] overflow-hidden p-4"
          :key="route.path"
        >
          <component :is="Component" :key="route" />
        </div>
      </router-view>
    </div>
  </div>
</template>

<style scoped>
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.2s ease;
}

.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(10px);
}

/* Smooth sidebar transition */
.sidebar-transition {
  transition: width 0.3s ease;
}

/* Custom scrollbar for sidebar */
.overflow-y-auto {
  scrollbar-width: thin;
  scrollbar-color: rgba(156, 163, 175, 0.5) transparent;
}

.overflow-y-auto::-webkit-scrollbar {
  width: 4px;
}

.overflow-y-auto::-webkit-scrollbar-track {
  background: transparent;
}

.overflow-y-auto::-webkit-scrollbar-thumb {
  background-color: rgba(156, 163, 175, 0.5);
  border-radius: 20px;
}

/* Add tooltip for collapsed menu items */
[title] {
  position: relative;
}
</style>
