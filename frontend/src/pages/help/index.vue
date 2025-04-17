<script setup>
import {
  AlertCircle,
  ArrowUpRight,
  Bell,
  Book,
  BookOpen,
  Cctv,
  ChevronDown,
  ChevronRight,
  Clock,
  Download,
  FileQuestion,
  FileText,
  HelpCircle,
  Mail,
  MessageSquare,
  Phone,
  RefreshCw,
  Search,
  Shield,
  Smartphone,
  Terminal,
  Users,
  X,
  Youtube,
} from "lucide-vue-next";
import { onBeforeUnmount, onMounted, ref, watch } from "vue";

// Active help category
const activeCategory = ref("getting-started");

// Search functionality
const searchQuery = ref("");
const isSearching = ref(false);
const searchResults = ref([]);

// Help categories
const categories = [
  {
    id: "getting-started",
    icon: Book,
    label: "Getting Started",
    description: "Basic tutorials and guides for new users",
  },
  {
    id: "features",
    icon: FileText,
    label: "Features & Functions",
    description: "Explore Jarvist's capabilities",
  },
  {
    id: "faq",
    icon: HelpCircle,
    label: "Frequently Asked Questions",
    description: "Common questions and answers",
  },
  {
    id: "troubleshooting",
    icon: AlertCircle,
    label: "Troubleshooting",
    description: "Solutions for common issues",
  },
  {
    id: "api",
    icon: Terminal,
    label: "API Documentation",
    description: "For developers and integrations",
  },
  {
    id: "contact",
    icon: Mail,
    label: "Contact Support",
    description: "Get help from our team",
  },
];

// Frequently asked questions
const faqs = [
  {
    question: "How do I set up a new camera?",
    answer:
      "To set up a new camera, go to the Cameras section, click 'Add Camera' and follow the setup wizard. You'll need your camera's IP address and login credentials. For detailed instructions, see our Camera Setup Guide.",
  },
  {
    question: "Is my data secure with Jarvist AI?",
    answer:
      "Yes, Jarvist prioritizes security. All video data is encrypted both in transit and at rest. We use industry-standard security protocols and regular security audits to ensure your data remains protected.",
  },
  {
    question: "Can I access my cameras remotely?",
    answer:
      "Yes, you can access your cameras from anywhere with an internet connection by using our mobile app or web portal. Enable remote access in Settings > Remote Access.",
  },
  {
    question: "What AI features are included?",
    answer:
      "Jarvist AI includes intelligent motion detection, object recognition, facial recognition, unusual activity alerts, and smart notifications. These features can be configured in the Jarvist AI section of the application.",
  },
  {
    question: "How long is footage stored?",
    answer:
      "By default, footage is stored for 30 days. You can adjust retention periods in Settings > Storage. Premium plans offer extended storage options up to 1 year.",
  },
  {
    question: "Does Jarvist work with my existing cameras?",
    answer:
      "Jarvist is compatible with most ONVIF-compliant IP cameras and many major camera brands including Hikvision, Dahua, Axis, and more. Check our compatibility list in the documentation for details.",
  },
];

// Getting started guides
const guides = [
  {
    id: "quick-start",
    title: "Quick Start Guide",
    description: "Get up and running with Jarvist in under 10 minutes",
    icon: RefreshCw,
    type: "article",
    timeToRead: "5 min",
  },
  {
    id: "camera-setup",
    title: "Camera Setup",
    description: "Learn how to add and configure your security cameras",
    icon: Cctv,
    type: "video",
    timeToRead: "8 min",
  },
  {
    id: "ai-features",
    title: "AI Features Overview",
    description: "Discover the power of Jarvist's AI capabilities",
    icon: Shield,
    type: "article",
    timeToRead: "7 min",
  },
  {
    id: "alerts-setup",
    title: "Setting Up Alerts",
    description: "Configure custom notifications and alerts",
    icon: Bell,
    type: "article",
    timeToRead: "6 min",
  },
  {
    id: "mobile-app",
    title: "Mobile App Setup",
    description: "Access your system from anywhere with our mobile app",
    icon: Smartphone,
    type: "video",
    timeToRead: "4 min",
  },
];

// Support options
const supportOptions = [
  {
    title: "Email Support",
    description: "Send us a message and we'll respond within 24 hours",
    icon: Mail,
    action: "support@jarvist.com",
    actionType: "email",
  },
  {
    title: "Phone Support",
    description: "Available Monday-Friday, 9am-5pm EST",
    icon: Phone,
    action: "+1 (555) 123-4567",
    actionType: "phone",
  },
  {
    title: "Live Chat",
    description: "Chat with our support team in real-time",
    icon: MessageSquare,
    action: "Start Chat",
    actionType: "button",
  },
  {
    title: "Community Forum",
    description: "Connect with other Jarvist users",
    icon: Users,
    action: "Visit Forum",
    actionType: "link",
  },
];

// Search functionality
const performSearch = () => {
  if (searchQuery.value.trim() === "") {
    isSearching.value = false;
    searchResults.value = [];
    return;
  }

  isSearching.value = true;

  // Mock search implementation (would be replaced with real search logic)
  const query = searchQuery.value.toLowerCase();
  const results = [];

  // Search through FAQs
  faqs.forEach((faq) => {
    if (
      faq.question.toLowerCase().includes(query) ||
      faq.answer.toLowerCase().includes(query)
    ) {
      results.push({
        type: "faq",
        title: faq.question,
        content: faq.answer,
        category: "FAQ",
      });
    }
  });

  // Search through guides
  guides.forEach((guide) => {
    if (
      guide.title.toLowerCase().includes(query) ||
      guide.description.toLowerCase().includes(query)
    ) {
      results.push({
        type: "guide",
        title: guide.title,
        content: guide.description,
        category: "Getting Started",
        icon: guide.icon,
      });
    }
  });

  searchResults.value = results.slice(0, 5); // Limit to top 5 results
};

// Clear search
const clearSearch = () => {
  searchQuery.value = "";
  isSearching.value = false;
  searchResults.value = [];
};

// Handle category selection
const selectCategory = (categoryId) => {
  activeCategory.value = categoryId;
  clearSearch();
};

// Open links in external browser
const openExternalLink = (url) => {
  // In a desktop app, this would use the system's default browser
  console.log("Opening external link:", url);
  // Example implementation for Electron app:
  // if (window.electron) window.electron.shell.openExternal(url);
};

// Handle opening a guide
const openGuide = (guideId) => {
  console.log("Opening guide:", guideId);
  // This would navigate to or open the specific guide
};

// Start the live chat service
const startLiveChat = () => {
  console.log("Starting live chat");
  // This would open the built-in chat interface
};

// Handle key events for search
const handleKeyDown = (event) => {
  if (event.key === "Escape") {
    clearSearch();
  } else if (event.key === "F1") {
    // F1 typically opens help in desktop apps
    event.preventDefault();
    selectCategory("getting-started");
  } else if ((event.ctrlKey || event.metaKey) && event.key === "f") {
    // Ctrl+F or Cmd+F for search
    event.preventDefault();
    document.getElementById("help-search").focus();
  }
};

// Register and unregister global keyboard shortcuts
onMounted(() => {
  window.addEventListener("keydown", handleKeyDown);
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleKeyDown);
});

// Watch for search input changes
watch(searchQuery, (newVal) => {
  if (newVal.trim().length > 2) {
    performSearch();
  } else if (newVal.trim() === "") {
    clearSearch();
  }
});

// FAQ disclosure state management
const openFaqs = ref([]);

const toggleFaq = (index) => {
  if (openFaqs.value.includes(index)) {
    openFaqs.value = openFaqs.value.filter((i) => i !== index);
  } else {
    openFaqs.value.push(index);
  }
};

const isFaqOpen = (index) => {
  return openFaqs.value.includes(index);
};
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Help header section with search -->
    <div
      class="flex items-center justify-between py-2 border-b border-gray-200 dark:border-slate-700 mb-4"
    >
      <h1 class="text-xl font-semibold text-gray-800 dark:text-gray-200">
        Help Center
      </h1>
      <div class="relative w-96">
        <Search
          class="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400"
        />
        <input
          id="help-search"
          v-model="searchQuery"
          type="text"
          placeholder="Search help topics..."
          class="w-full pl-10 pr-10 py-2 rounded-md bg-white dark:bg-slate-800 border border-gray-200 dark:border-slate-700 focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
          @keyup.enter="performSearch"
        />
        <button
          v-if="searchQuery"
          @click="clearSearch"
          class="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
        >
          <X class="h-4 w-4" />
        </button>
      </div>
      <div class="text-xs text-gray-500 dark:text-gray-400">
        <span class="mr-1">Tip: Press</span>
        <kbd
          class="px-1.5 py-0.5 bg-gray-100 dark:bg-slate-700 rounded border border-gray-300 dark:border-slate-600 font-mono text-xs"
          >Ctrl+F</kbd
        >
        <span class="mx-1">to search,</span>
        <kbd
          class="px-1.5 py-0.5 bg-gray-100 dark:bg-slate-700 rounded border border-gray-300 dark:border-slate-600 font-mono text-xs"
          >Esc</kbd
        >
        <span class="ml-1">to clear</span>
      </div>
    </div>

    <div class="flex flex-1 gap-4 overflow-hidden">
      <!-- Left sidebar with categories -->
      <div
        class="w-64 shrink-0 border-r border-gray-200 dark:border-slate-700 overflow-auto pr-2"
      >
        <h2 class="font-medium text-gray-800 dark:text-gray-200 mb-3 px-2">
          Topics
        </h2>
        <div class="space-y-0.5">
          <div
            v-for="category in categories"
            :key="category.id"
            @click="selectCategory(category.id)"
            class="flex items-center px-2 py-2 rounded-md cursor-pointer transition-colors"
            :class="{
              'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400':
                activeCategory === category.id,
              'hover:bg-gray-100 dark:hover:bg-slate-800':
                activeCategory !== category.id,
              'text-gray-800 dark:text-gray-200':
                activeCategory !== category.id,
            }"
          >
            <component :is="category.icon" class="w-5 h-5 mr-3 shrink-0" />
            <div class="truncate">
              <div class="text-sm font-medium">{{ category.label }}</div>
            </div>
          </div>
        </div>

        <!-- Quick links -->
        <div class="mt-6">
          <h2 class="font-medium text-gray-800 dark:text-gray-200 mb-3 px-2">
            Quick Links
          </h2>
          <div class="space-y-0.5">
            <a
              href="#"
              class="flex items-center px-2 py-2 rounded-md hover:bg-gray-100 dark:hover:bg-slate-800 text-gray-700 dark:text-gray-300"
            >
              <Youtube class="w-4 h-4 mr-3 text-red-500 shrink-0" />
              <span class="text-sm">Video Tutorials</span>
            </a>
            <a
              href="#"
              class="flex items-center px-2 py-2 rounded-md hover:bg-gray-100 dark:hover:bg-slate-800 text-gray-700 dark:text-gray-300"
            >
              <BookOpen class="w-4 h-4 mr-3 text-blue-500 shrink-0" />
              <span class="text-sm">Documentation</span>
            </a>
            <a
              href="#"
              class="flex items-center px-2 py-2 rounded-md hover:bg-gray-100 dark:hover:bg-slate-800 text-gray-700 dark:text-gray-300"
            >
              <Download class="w-4 h-4 mr-3 text-green-500 shrink-0" />
              <span class="text-sm">Download Manuals</span>
            </a>
            <a
              href="#"
              class="flex items-center px-2 py-2 rounded-md hover:bg-gray-100 dark:hover:bg-slate-800 text-gray-700 dark:text-gray-300"
            >
              <FileQuestion class="w-4 h-4 mr-3 text-purple-500 shrink-0" />
              <span class="text-sm">Release Notes</span>
            </a>
          </div>
        </div>
      </div>

      <!-- Main content area -->
      <div class="flex-1 overflow-auto pr-1">
        <!-- Search results -->
        <div
          v-if="isSearching && searchResults.length > 0"
          class="bg-white dark:bg-slate-900 rounded-md border border-gray-200 dark:border-slate-700 p-4 mb-4"
        >
          <h2
            class="text-md font-medium text-gray-800 dark:text-gray-200 mb-3 flex items-center"
          >
            <Search class="w-4 h-4 mr-2 text-gray-500 dark:text-gray-400" />
            Search Results for "{{ searchQuery }}"
          </h2>

          <div class="space-y-3">
            <div
              v-for="(result, index) in searchResults"
              :key="`search-${index}`"
              class="p-3 border border-gray-100 dark:border-slate-800 rounded-md hover:bg-gray-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
            >
              <div class="flex items-start">
                <div
                  class="bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 p-1.5 rounded-full mr-3 shrink-0"
                >
                  <component
                    :is="
                      result.type === 'guide' && result.icon
                        ? result.icon
                        : FileText
                    "
                    class="w-4 h-4"
                  />
                </div>
                <div>
                  <div class="flex items-center">
                    <h3 class="font-medium text-gray-800 dark:text-gray-200">
                      {{ result.title }}
                    </h3>
                    <span
                      class="ml-2 text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-slate-800 px-2 py-0.5 rounded"
                      >{{ result.category }}</span
                    >
                  </div>
                  <p
                    class="text-sm text-gray-600 dark:text-gray-400 mt-1 line-clamp-2"
                  >
                    {{ result.content }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div
          v-else-if="isSearching && searchResults.length === 0"
          class="bg-white dark:bg-slate-900 rounded-md border border-gray-200 dark:border-slate-700 p-4 mb-4"
        >
          <div class="text-center py-6">
            <AlertCircle class="w-10 h-10 text-gray-400 mx-auto mb-2" />
            <h3
              class="text-md font-medium text-gray-800 dark:text-gray-200 mb-1"
            >
              No results found
            </h3>
            <p class="text-sm text-gray-600 dark:text-gray-400">
              We couldn't find any matching results for "{{ searchQuery }}"
            </p>
            <button
              @click="clearSearch"
              class="mt-3 px-3 py-1 text-sm text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-800 rounded-md hover:bg-blue-50 dark:hover:bg-blue-900/30"
            >
              Clear Search
            </button>
          </div>
        </div>

        <!-- Getting Started content -->
        <div
          v-else-if="activeCategory === 'getting-started'"
          class="overflow-auto"
        >
          <div
            class="bg-white dark:bg-slate-900 rounded-md border border-gray-200 dark:border-slate-700 p-4 mb-4"
          >
            <div class="flex items-center mb-3">
              <Book class="w-5 h-5 mr-2 text-blue-600 dark:text-blue-400" />
              <h2 class="text-lg font-medium text-gray-800 dark:text-gray-200">
                Getting Started with Jarvist
              </h2>
            </div>
            <p class="text-gray-600 dark:text-gray-400 mb-4">
              Welcome to Jarvist! This guide will help you get started with your
              security system. Follow these guides to set up your system quickly
              and efficiently.
            </p>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-3 mt-4">
              <div
                v-for="guide in guides"
                :key="guide.id"
                @click="openGuide(guide.id)"
                class="flex border border-gray-100 dark:border-slate-800 rounded-md p-3 hover:bg-gray-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors"
              >
                <div
                  class="bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 p-2 rounded-full h-fit mr-3 shrink-0"
                >
                  <component :is="guide.icon" class="w-4 h-4" />
                </div>
                <div>
                  <div class="flex items-center">
                    <h3
                      class="font-medium text-gray-800 dark:text-gray-200 text-sm"
                    >
                      {{ guide.title }}
                    </h3>
                    <div
                      class="ml-2 text-xs px-2 py-0.5 rounded-sm"
                      :class="{
                        'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400':
                          guide.type === 'article',
                        'bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400':
                          guide.type === 'video',
                      }"
                    >
                      {{ guide.type }}
                    </div>
                  </div>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    {{ guide.description }}
                  </p>
                  <div
                    class="flex items-center mt-2 text-xs text-gray-500 dark:text-gray-400"
                  >
                    <Clock class="w-3.5 h-3.5 mr-1" />
                    <span>{{ guide.timeToRead }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div
            class="bg-white dark:bg-slate-900 rounded-md border border-gray-200 dark:border-slate-700 p-4"
          >
            <div class="flex items-center mb-3">
              <ChevronRight
                class="w-5 h-5 mr-2 text-blue-600 dark:text-blue-400"
              />
              <h2 class="text-lg font-medium text-gray-800 dark:text-gray-200">
                Recommended Next Steps
              </h2>
            </div>

            <div class="space-y-2">
              <div
                class="flex items-start p-3 border border-gray-100 dark:border-slate-800 rounded-md hover:bg-gray-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
              >
                <div
                  class="bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 p-1.5 rounded-full mr-3 shrink-0"
                >
                  <Cctv class="w-4 h-4" />
                </div>
                <div>
                  <h3
                    class="font-medium text-gray-800 dark:text-gray-200 text-sm"
                  >
                    Connect Your First Camera
                  </h3>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    Learn how to add and configure your security cameras.
                  </p>
                  <button
                    class="mt-2 text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 flex items-center"
                  >
                    View tutorial <ChevronRight class="w-3 h-3 ml-1" />
                  </button>
                </div>
              </div>

              <div
                class="flex items-start p-3 border border-gray-100 dark:border-slate-800 rounded-md hover:bg-gray-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
              >
                <div
                  class="bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 p-1.5 rounded-full mr-3 shrink-0"
                >
                  <Shield class="w-4 h-4" />
                </div>
                <div>
                  <h3
                    class="font-medium text-gray-800 dark:text-gray-200 text-sm"
                  >
                    Set Up Jarvist AI
                  </h3>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    Configure AI features like motion detection and object
                    recognition.
                  </p>
                  <button
                    class="mt-2 text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 flex items-center"
                  >
                    View tutorial <ChevronRight class="w-3 h-3 ml-1" />
                  </button>
                </div>
              </div>

              <div
                class="flex items-start p-3 border border-gray-100 dark:border-slate-800 rounded-md hover:bg-gray-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
              >
                <div
                  class="bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 p-1.5 rounded-full mr-3 shrink-0"
                >
                  <Bell class="w-4 h-4" />
                </div>
                <div>
                  <h3
                    class="font-medium text-gray-800 dark:text-gray-200 text-sm"
                  >
                    Configure Alerts
                  </h3>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    Set up custom notifications and alerts for important events.
                  </p>
                  <button
                    class="mt-2 text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 flex items-center"
                  >
                    View tutorial <ChevronRight class="w-3 h-3 ml-1" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- FAQ content -->
        <div v-else-if="activeCategory === 'faq'">
          <div
            class="bg-white dark:bg-slate-900 rounded-md border border-gray-200 dark:border-slate-700 p-4"
          >
            <div class="flex items-center mb-3">
              <HelpCircle
                class="w-5 h-5 mr-2 text-blue-600 dark:text-blue-400"
              />
              <h2 class="text-lg font-medium text-gray-800 dark:text-gray-200">
                Frequently Asked Questions
              </h2>
            </div>

            <div class="space-y-2">
              <div
                v-for="(faq, index) in faqs"
                :key="index"
                class="border border-gray-100 dark:border-slate-800 rounded-md overflow-hidden"
              >
                <div
                  class="flex justify-between w-full px-4 py-3 text-left text-gray-800 dark:text-gray-200 font-medium bg-gray-50 dark:bg-slate-800/50 hover:bg-gray-100 dark:hover:bg-slate-800 transition-colors cursor-pointer"
                  @click="toggleFaq(index)"
                >
                  <span>{{ faq.question }}</span>
                  <ChevronDown
                    :class="isFaqOpen(index) ? 'rotate-180 transform' : ''"
                    class="w-5 h-5 text-gray-500 dark:text-gray-400 transition-transform"
                  />
                </div>
                <div
                  v-if="isFaqOpen(index)"
                  class="px-4 py-3 text-sm text-gray-600 dark:text-gray-400 bg-white dark:bg-slate-900"
                >
                  {{ faq.answer }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Contact support content -->
        <div v-else-if="activeCategory === 'contact'">
          <div
            class="bg-white dark:bg-slate-900 rounded-md border border-gray-200 dark:border-slate-700 p-4 mb-4"
          >
            <div class="flex items-center mb-3">
              <Mail class="w-5 h-5 mr-2 text-blue-600 dark:text-blue-400" />
              <h2 class="text-lg font-medium text-gray-800 dark:text-gray-200">
                Contact Support
              </h2>
            </div>
            <p class="text-gray-600 dark:text-gray-400 mb-4">
              Our support team is here to help. Choose your preferred method of
              contact below.
            </p>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div
                v-for="option in supportOptions"
                :key="option.title"
                class="flex flex-col border border-gray-100 dark:border-slate-800 rounded-md p-3 hover:bg-gray-50 dark:hover:bg-slate-800/50 transition-colors cursor-pointer"
              >
                <div class="flex items-center mb-2">
                  <div
                    class="bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 p-1.5 rounded-full mr-2 shrink-0"
                  >
                    <component :is="option.icon" class="w-4 h-4" />
                  </div>
                  <h3
                    class="font-medium text-gray-800 dark:text-gray-200 text-sm"
                  >
                    {{ option.title }}
                  </h3>
                </div>
                <p class="text-xs text-gray-600 dark:text-gray-400 mb-3">
                  {{ option.description }}
                </p>

                <div class="mt-auto">
                  <a
                    v-if="option.actionType === 'email'"
                    :href="`mailto:${option.action}`"
                    class="inline-flex items-center text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
                  >
                    {{ option.action }} <ArrowUpRight class="w-3 h-3 ml-1" />
                  </a>
                  <a
                    v-else-if="option.actionType === 'phone'"
                    :href="`tel:${option.action.replace(/[^0-9+]/g, '')}`"
                    class="inline-flex items-center text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
                  >
                    {{ option.action }} <ArrowUpRight class="w-3 h-3 ml-1" />
                  </a>
                  <button
                    v-else-if="option.actionType === 'button'"
                    @click="startLiveChat"
                    class="inline-flex items-center justify-center px-3 py-1 rounded bg-blue-600 hover:bg-blue-700 text-white text-xs font-medium transition-colors"
                  >
                    {{ option.action }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
