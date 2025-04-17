<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
const route = useRoute();

interface Item {
  title: string;
  href: string;
}

const sidebarNavItems: Item[] = [
  {
    title: "General",
    href: "/settings",
  },
  {
    title: "Scheduler",
    href: "/settings/scheduler",
  },
];

const handleMenuClick = (item: any) => {
  if (item.href) {
    router.push(item.href);
  }
};
</script>

<template>
  <nav class="flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-1">
    <Button
      v-for="item in sidebarNavItems"
      :key="item.title"
      as="a"
      @click="handleMenuClick(item)"
      variant="ghost"
      :class="
        cn(
          'w-full text-left justify-start hover:cursor-pointer',
          route.path === item.href && 'bg-[#e6f3fd] hover:bg-[#e6f3fd]'
        )
      "
    >
      {{ item.title }}
    </Button>
  </nav>
</template>
