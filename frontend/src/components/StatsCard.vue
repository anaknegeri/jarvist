<!-- StatsCard.vue -->
<script setup lang="ts">
import { Card, CardContent } from "@/components/ui/card";
import { formatNumber } from "@/lib/common";

defineProps<{
  title: string;
  value: number;
  labelText?: string;
  indicatorColor: string;
  gradientFrom: string;
  gradientTo: string;
  icon: any;
  iconColorClass: string;
  iconBgClass: string;
  additionalInfo?: { text: string; color: string; dotColor: string }[];
}>();
</script>

<template>
  <Card
    class="shadow-sm"
    :class="`bg-gradient-to-br ${gradientFrom} ${gradientTo}`"
  >
    <CardContent class="p-4 flex justify-between items-center">
      <div>
        <p class="text-sm text-muted-foreground mb-1">{{ title }}</p>
        <h3 class="text-2xl font-bold text-gray-800 dark:text-gray-100">
          {{ formatNumber(value) }}
        </h3>

        <p
          v-if="labelText"
          class="text-xs mt-1 flex items-center mb-1"
          :class="indicatorColor"
        >
          <span
            class="inline-block w-1.5 h-1.5 rounded-full mr-1"
            :class="
              indicatorColor.includes('blue')
                ? 'bg-blue-500'
                : indicatorColor.includes('indigo')
                  ? 'bg-indigo-500'
                  : indicatorColor.includes('emerald')
                    ? 'bg-emerald-500'
                    : indicatorColor.includes('red')
                      ? 'bg-red-500'
                      : 'bg-gray-500'
            "
          ></span>
          {{ labelText }}
        </p>

        <div v-if="additionalInfo" class="flex items-center space-x-4">
          <div
            v-for="(info, index) in additionalInfo"
            :key="index"
            class="text-xs mt-1 flex items-center"
            :class="info.color"
          >
            <span
              class="inline-block w-1.5 h-1.5 rounded-full mr-1"
              :class="info.dotColor"
            ></span>
            <span>{{ info.text }}</span>
          </div>
        </div>
      </div>

      <div
        class="w-12 h-12 rounded-full flex items-center justify-center"
        :class="iconBgClass"
      >
        <component :is="icon" class="w-6 h-6" :class="iconColorClass" />
      </div>
    </CardContent>
  </Card>
</template>
