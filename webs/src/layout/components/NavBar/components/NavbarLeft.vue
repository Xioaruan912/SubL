<template>
  <div class="navbar-left">
    <hamburger
      :is-active="appStore.sidebar.opened"
      @toggle-click="toggleSideBar"
    />
    <div class="route-heading">
      <strong>{{ routeTitle }}</strong>
      <span>{{ routeDescription }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAppStore } from "@/store";

const appStore = useAppStore();
const route = useRoute();
const routeTitle = computed(() => String(route.meta?.title || "控制台"));
const descriptions: Record<string, string> = {
  "/": "运行状态与节点分布",
  "/dashboard": "运行状态与节点分布",
  "/subcription": "管理节点、机场与订阅",
  "/template": "构建和维护客户端配置",
};
const routeDescription = computed(() => {
  const key = Object.keys(descriptions).find(k => k !== "/" && route.path.startsWith(k));
  return descriptions[key || "/"];
});

function toggleSideBar() {
  appStore.toggleSidebar();
}
</script>

<style scoped>
.navbar-left { display:flex; align-items:center; min-width:0; height:100%; }
.route-heading { min-width:0; padding-left:4px; }
.route-heading strong,.route-heading span { display:block; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.route-heading strong { color:var(--ui-text); font-size:14px; line-height:1.2; }
.route-heading span { margin-top:3px; color:var(--ui-text-muted); font-size:10px; }
@media (max-width:700px) { .route-heading span { display:none; } }
</style>
