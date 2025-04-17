<script setup lang="ts">
import logo from "@/assets/logo-white.svg";
import { CheckCircle2 } from "lucide-vue-next";
import { onMounted, ref } from "vue";

const progress = ref(0);
const loading = ref(true);
const loadingText = ref("Initializing...");
const loadComplete = ref(false);
const appVersion = ref<string>("1.0.0");
const copyright = ref<string>("© 2025, Pitjarus Teknologi");

// Loading steps simulation
const loadingSteps = [
  { text: "Initializing system...", duration: 800 },
  { text: "Loading resources...", duration: 600 },
  { text: "Checking license...", duration: 700 },
  { text: "Connecting to services...", duration: 900 },
  { text: "Starting application...", duration: 1000 },
];

const loadConfig = async () => {
  appVersion.value = await GetProductVersion();
  copyright.value = await GetCopyright();
};

// Simulate loading process
onMounted(async () => {
  loadConfig();

  const totalDuration = loadingSteps.reduce(
    (acc, step) => acc + step.duration,
    0
  );
  const progressIncrement = 100 / totalDuration;

  let elapsedTime = 0;

  for (const step of loadingSteps) {
    loadingText.value = step.text;

    await new Promise((resolve) => {
      let startTime = performance.now();
      let stepProgress = 0;

      const interval = setInterval(() => {
        const currentTime = performance.now();
        const elapsed = currentTime - startTime;

        if (elapsed >= step.duration) {
          clearInterval(interval);
          elapsedTime += step.duration;
          progress.value = Math.min(
            Math.floor(elapsedTime * progressIncrement),
            100
          );
          resolve(null);
        } else {
          stepProgress = (elapsed / step.duration) * 100;
          progress.value = Math.min(
            Math.floor((elapsedTime + elapsed) * progressIncrement),
            100
          );
        }
      }, 16);
    });
  }

  // Loading complete
  loading.value = false;
  loadComplete.value = true;
});
</script>

<template>
  <div
    class="splash-screen min-h-screen w-full bg-gradient-to-br from-indigo-900 via-slate-900 to-gray-900 flex flex-col items-center justify-center overflow-hidden"
  >
    <!-- Logo Animation Container -->
    <div class="relative mb-12">
      <!-- Glowing background effect -->
      <div
        class="absolute inset-0 bg-indigo-600/20 blur-3xl rounded-full animate-pulse"
      ></div>

      <!-- Logo with pulsing animation -->
      <div class="relative z-10 flex items-center justify-center">
        <div class="logo-container relative">
          <!-- Spinning outer circle -->
          <div
            class="absolute inset-0 rounded-full border-8 border-indigo-600/30 animate-[spin_8s_linear_infinite]"
          ></div>

          <!-- Pulsing middle circle -->
          <div
            class="absolute inset-2 rounded-full border-4 border-indigo-500/40 animate-[pulse_4s_cubic-bezier(0.4,0,0.6,1)_infinite]"
          ></div>

          <!-- Logo background -->
          <div
            class="h-32 w-32 bg-gradient-to-br from-indigo-600 to-indigo-800 rounded-full flex items-center justify-center shadow-lg shadow-indigo-700/50"
          >
            <img :src="logo" class="h-14" />
          </div>

          <!-- Orbit particles -->
          <div class="orbit">
            <div class="particle bg-blue-400"></div>
            <div
              class="particle bg-indigo-400"
              style="animation-delay: -2s"
            ></div>
            <div
              class="particle bg-purple-400"
              style="animation-delay: -4s"
            ></div>
            <div class="particle bg-sky-400" style="animation-delay: -6s"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- App name with text reveal animation -->
    <div class="mb-10 text-center">
      <h1
        class="text-4xl font-bold text-white tracking-tight mb-2 animate-fade-in"
      >
        <span
          class="text-transparent bg-clip-text bg-gradient-to-r from-indigo-300 to-blue-300"
          >JARVIST</span
        >
        <span
          class="text-transparent bg-clip-text bg-gradient-to-r from-blue-300 to-sky-300 ml-2"
          >AI</span
        >
      </h1>
      <p class="text-indigo-300/80 animate-fade-in animation-delay-300">
        Intelligent Vision System
      </p>
    </div>

    <!-- Loading progress -->
    <div class="w-72 mb-2">
      <div class="relative h-1 w-full bg-gray-700 rounded-full overflow-hidden">
        <div
          class="absolute top-0 left-0 h-full bg-gradient-to-r from-indigo-500 to-blue-500 rounded-full transition-all duration-300 ease-out"
          :style="{ width: `${progress}%` }"
        ></div>
      </div>
    </div>

    <!-- Loading or complete status -->
    <div class="h-6 flex items-center justify-center text-sm">
      <div
        v-if="loading"
        class="text-indigo-200/70 flex items-center animate-pulse"
      >
        <span>{{ loadingText }}</span>
        <span class="ml-1 flex">
          <span class="animate-[bounce_1s_infinite_0ms]">.</span>
          <span class="animate-[bounce_1s_infinite_200ms]">.</span>
          <span class="animate-[bounce_1s_infinite_400ms]">.</span>
        </span>
      </div>
      <div v-else class="text-green-300 flex items-center animate-fade-in">
        <CheckCircle2 class="w-4 h-4 mr-1.5" />
        <span>Ready to launch</span>
      </div>
    </div>

    <!-- Footer -->
    <div class="absolute bottom-4 flex flex-col items-center justify-center">
      <p class="text-xs text-gray-400 mb-1">Version v{{ appVersion }}</p>
      <p class="text-xs text-gray-500">{{ copyright }}</p>
    </div>
  </div>
</template>

<style scoped>
/* Orbit animation */
.logo-container {
  position: relative;
  height: 8rem;
  width: 8rem;
}

.orbit {
  position: absolute;
  top: 50%;
  left: 50%;
  width: 160px;
  height: 160px;
  margin-top: -80px;
  margin-left: -80px;
  border-radius: 100%;
}

.particle {
  position: absolute;
  width: 10px;
  height: 10px;
  border-radius: 100%;
  animation: orbit 8s linear infinite;
}

@keyframes orbit {
  0% {
    transform: rotate(0deg) translateX(80px) rotate(0deg);
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
  100% {
    transform: rotate(360deg) translateX(80px) rotate(-360deg);
    opacity: 1;
  }
}

/* Text fade-in animation */
.animate-fade-in {
  animation: fadeIn 1s ease-out forwards;
  opacity: 0;
}

.animation-delay-300 {
  animation-delay: 300ms;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Modern loading dots animation */
@keyframes bounce {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-3px);
  }
}
</style>

<route lang="yaml">
meta:
  layout: blank
</route>
