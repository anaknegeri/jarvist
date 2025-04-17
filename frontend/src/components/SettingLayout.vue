<script setup lang="ts"></script>
<template>
  <div class="space-y-2 h-full flex flex-col">
    <div class="grid gap-4">
      <div class="flex items-center justify-between space-y-2">
        <h2 class="text-xl font-bold tracking-tight">Settings</h2>
      </div>
    </div>
    <Separator class="my-6" />
    <div
      class="flex flex-1 flex-col space-y-8 lg:flex-row lg:space-x-4 lg:space-y-0 h-full"
    >
      <aside class="lg:w-1/6">
        <SidebarSetting />
      </aside>
      <div class="flex-1 flex">
        <div class="flex flex-1 overflow-hidden">
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>
