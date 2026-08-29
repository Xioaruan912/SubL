<template>
  <div class="logo-container">
    <transition enter-active-class="animate__animated animate__fadeInLeft">
      <router-link v-if="collapse" class="wh-full flex-center" to="/">
        <img v-if="settingsStore.sidebarLogo" :src="logo" class="logo-image" />
      </router-link>

      <router-link v-else class="wh-full logo-link" to="/">
        <img v-if="settingsStore.sidebarLogo" :src="logo" class="logo-image" />
        <span class="logo-copy"><strong>{{ defaultSettings.title }}</strong><small>节点与订阅控制台</small></span>
      </router-link>
    </transition>
  </div>
</template>

<script lang="ts" setup>
import defaultSettings from "@/settings";
import { useSettingsStore } from "@/store";

const settingsStore = useSettingsStore();

defineProps({
  collapse: {
    type: Boolean,
    required: true,
  },
});

const logo = ref(new URL(`../../../../assets/logo.png`, import.meta.url).href);
</script>

<style lang="scss" scoped>
.logo-container {
  width: 100%;
  height: $navbar-height;
  background-color: $sidebar-logo-background;

  .logo-image {
    width: 42px;
    height: 42px;
    border-radius: 12px;
    object-fit: cover;
    box-shadow: 0 6px 16px var(--ui-accent-shadow);
  }

  .logo-title {
    flex-shrink: 0; /* 防止容器在空间不足时缩小 */
    margin-left: 10px;
    font-size: 14px;
    font-weight: bold;
    color: var(--menu-text);
  }
}
.logo-link { display:flex; align-items:center; justify-content:flex-start; gap:11px; padding:9px 18px; text-decoration:none; }
.logo-copy { min-width:0; }
.logo-copy strong,.logo-copy small { display:block; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.logo-copy strong { color:var(--ui-text); font-size:15px; }
.logo-copy small { margin-top:3px; color:var(--ui-text-muted); font-size:11px; font-weight:500; }

.layout-top,
.layout-mix {
  .logo-container {
    width: $sidebar-width;
  }

  &.hideSidebar {
    .logo-container {
      width: $sidebar-width-collapsed;
    }
  }
}
</style>
