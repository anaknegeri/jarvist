<script setup lang="ts">
import { cn } from "@/lib/utils";
import type { HTMLAttributes } from "vue";

const props = defineProps<{
  class?: HTMLAttributes["class"];
}>();
</script>

<template>
  <div
    :class="
      cn(
        'rounded-md border border-gray-200 dark:border-gray-700 bg-white text-zinc-950 dark:bg-zinc-950 dark:text-zinc-50',
        props.class
      )
    "
  >
    <slot />
  </div>
</template>
