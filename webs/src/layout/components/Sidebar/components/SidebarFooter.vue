<template>
  <div class="sidebar-footer">
    <div class="theme-selector" role="group" aria-label="界面主题">
      <button :class="{ active: theme === ThemeEnum.LIGHT }" @click="setTheme(ThemeEnum.LIGHT)">浅色</button>
      <button :class="{ active: theme === ThemeEnum.DARK }" @click="setTheme(ThemeEnum.DARK)">深色</button>
    </div>
    <div class="sidebar-version">
      <span class="status-dot"></span>
      <span>PPEELink 控制台</span>
      <small>v{{ defaultSettings.version }}</small>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useSettingsStore } from "@/store";
import { ThemeEnum } from "@/enums/ThemeEnum";
import defaultSettings from "@/settings";

const settingsStore = useSettingsStore();
const theme = computed(() => settingsStore.theme);
const setTheme = (value: ThemeEnum) => settingsStore.changeTheme(value);
</script>

<style lang="scss" scoped>
.sidebar-footer { padding: 14px 16px 16px; border-top: 1px solid var(--ui-border); }
.theme-selector { display:grid; grid-template-columns:1fr 1fr; gap:3px; padding:3px; border:1px solid var(--ui-border); border-radius:9px; background:var(--ui-surface); }
.theme-selector button { height:32px; border:0; border-radius:6px; background:transparent; color:var(--ui-text-muted); font-size:12px; font-weight:700; transition:background 140ms ease,color 140ms ease,box-shadow 140ms ease; }
.theme-selector button:hover { color:var(--ui-text); background:var(--ui-hover); }
.theme-selector button.active { color:var(--ui-text); background:var(--ui-surface-strong); box-shadow:inset 0 0 0 1px var(--ui-border),0 1px 2px rgba(24,40,31,.06); }
.sidebar-version { display:flex; align-items:center; gap:7px; margin-top:11px; padding:0 3px; color:var(--ui-text-muted); font-size:11px; }
.sidebar-version small { margin-left:auto; }
.status-dot { width:7px; height:7px; border-radius:50%; background:#22a06b; box-shadow:0 0 0 3px rgba(34,160,107,.13); }
.hideSidebar & { display:none; }
</style>
