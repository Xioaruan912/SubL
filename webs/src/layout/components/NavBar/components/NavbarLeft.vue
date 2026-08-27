<template>
  <div class="flex">
    <hamburger
      :is-active="appStore.sidebar.opened"
      @toggle-click="toggleSideBar"
    />
    <!-- 极简模式：移除生硬的面包屑，使顶栏更像客户端工具栏 -->
    <!-- <breadcrumb /> -->
  </div>
</template>

<script setup lang="ts">
import { useAppStore } from "@/store";

const appStore = useAppStore();

function toggleSideBar() {
  appStore.toggleSidebar();
}
</script>
