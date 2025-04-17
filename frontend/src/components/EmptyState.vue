<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-vue-next";

defineProps<{
  icon: any;
  title: string;
  description: string;
  showButton?: boolean;
  buttonText?: string;
}>();

const emit = defineEmits<{
  (e: "action"): void;
}>();

const handleAction = () => {
  emit("action");
};
</script>

<template>
  <div
    class="flex flex-col items-center justify-center p-10 rounded-lg border border-dashed border-muted-foreground/20 bg-muted/30"
  >
    <div
      class="w-16 h-16 bg-muted rounded-full flex items-center justify-center mb-4"
    >
      <component :is="icon" class="w-8 h-8 text-muted-foreground" />
    </div>
    <h3 class="text-base font-medium text-gray-800 dark:text-gray-200 mb-1">
      {{ title }}
    </h3>
    <p class="text-sm text-muted-foreground mb-6 text-center max-w-md">
      {{ description }}
    </p>
    <Button v-if="showButton" @click="handleAction">
      <Plus :size="14" class="mr-1.5" /> {{ buttonText }}
    </Button>
  </div>
</template>
