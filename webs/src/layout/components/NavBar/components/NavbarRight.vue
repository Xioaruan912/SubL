<template>
  <div class="navbar-actions">
    <template v-if="!isMobile">
      <!--测试状态 -->
      <button class="setting-item" title="测试任务" @click="testDialogVisible = true">
        <el-badge is-dot :hidden="!testRunning" class="test-badge">
          <svg-icon icon-class="monitor" />
        </el-badge>
      </button>

      <!--全屏 -->
      <button class="setting-item" title="切换全屏" @click="toggle">
        <svg-icon
          :icon-class="isFullscreen ? 'fullscreen-exit' : 'fullscreen'"
        />
      </button>

      <!-- 布局大小 -->
      <el-tooltip
        :content="$t('sizeSelect.tooltip')"
        effect="dark"
        placement="bottom"
      >
        <size-select class="setting-item" />
      </el-tooltip>

      <!-- 语言选择 -->
      <lang-select class="setting-item" />
    </template>

    <!-- 用户 -->
    <el-dropdown class="setting-item" trigger="click">
      <div class="user-trigger">
        <span class="user-badge">{{ (userStore.user.username || 'U').charAt(0).toUpperCase() }}</span>
        <span class="user-copy"><strong>{{ userStore.user.username }}</strong><small>管理员</small></span>
        <el-icon class="dropdown-arrow"><ArrowDown /></el-icon>
      </div>
      <template #dropdown>
        <el-dropdown-menu>
            <router-link to="/system/user/set">
            <el-dropdown-item>{{ $t("navbar.userset") }}</el-dropdown-item>
            </router-link>
          <el-dropdown-item divided @click="logout">
            {{ $t("navbar.logout") }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <!-- 设置 -->
    <template v-if="defaultSettings.showSettings">
      <button class="setting-item" title="界面设置" @click="settingStore.settingsVisible = true">
        <svg-icon icon-class="setting" />
      </button>
    </template>

    <!-- 测试状态弹窗 -->
    <TestStatusDialog v-model:visible="testDialogVisible" />
  </div>
</template>
<script setup lang="ts">
import {
  useAppStore,
  useTagsViewStore,
  useUserStore,
  useSettingsStore,
} from "@/store";
import defaultSettings from "@/settings";
import { DeviceEnum } from "@/enums/DeviceEnum";
import { GetTestStatus } from "@/api/subcription/node";
import TestStatusDialog from "@/components/TestStatusDialog.vue";
import { ArrowDown } from "@element-plus/icons-vue";

const appStore = useAppStore();
const tagsViewStore = useTagsViewStore();
const userStore = useUserStore();
const settingStore = useSettingsStore();

const testDialogVisible = ref(false);
const testRunning = ref(false);

const route = useRoute();
const router = useRouter();

const isMobile = computed(() => appStore.device === DeviceEnum.MOBILE);

const { isFullscreen, toggle } = useFullscreen();

// 轮询测试状态（是否有测试进行中，用于红点提示）
let statusTimer: any = null;
const pollTestStatus = async () => {
  try {
    const { data } = await GetTestStatus();
    testRunning.value = !!data;
  } catch { testRunning.value = false; }
};
onMounted(() => {
  pollTestStatus();
  statusTimer = setInterval(pollTestStatus, 5000);
});
onBeforeUnmount(() => { if (statusTimer) clearInterval(statusTimer); });

/**
 * 注销
 */
function logout() {
  ElMessageBox.confirm("确定注销并退出系统吗？", "提示", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
    lockScroll: false,
  }).then(() => {
    userStore
      .logout()
      .then(() => {
        tagsViewStore.delAllViews();
      })
      .then(() => {
        router.push(`/login?redirect=${route.fullPath}`);
      });
  });
}
</script>
<style lang="scss" scoped>
.test-badge {
  display: inline-flex;
  align-items: center;

  :deep(.el-badge__content.is-dot) {
    top: 12px;
    right: 4px;
  }
}
.user-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  margin-right: 8px;
  border-radius: 50%;
  background: var(--ui-accent-soft);
  color: var(--ui-accent-strong);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  flex-shrink: 0;
}
.navbar-actions { display:flex; align-items:center; gap:6px; height:100%; }
.user-trigger { display:grid; grid-template-columns:30px minmax(0,1fr) 14px; align-items:center; gap:8px; min-width:136px; height:42px; padding:0 10px; border:1px solid var(--ui-border); border-radius:10px; background:var(--ui-surface); color:var(--ui-text); cursor:pointer; transition:background 140ms ease,border-color 140ms ease,box-shadow 140ms ease; }
.user-trigger:hover { background:var(--ui-hover); border-color:color-mix(in srgb,var(--ui-accent) 26%,var(--ui-border)); }
.user-copy { min-width:0; text-align:left; }
.user-copy strong,.user-copy small { display:block; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.user-copy strong { font-size:12px; line-height:1.2; }.user-copy small { margin-top:2px; color:var(--ui-text-muted); font-size:9px; }
.dropdown-arrow { color:var(--ui-text-muted); transition:transform 160ms ease; }
:deep(.el-dropdown[aria-expanded="true"]) .dropdown-arrow { transform:rotate(180deg); }
.setting-item {
  display: inline-flex;
  align-items:center;
  justify-content:center;
  width:38px;
  min-width:38px;
  height:38px;
  border:1px solid transparent;
  border-radius:9px;
  background:transparent;
  color: var(--ui-text-secondary);
  cursor: pointer;

  &:hover {
    border-color:var(--ui-border);
    background:var(--ui-hover);
    color:var(--ui-text);
  }
}

.layout-top,
.layout-mix {
  .setting-item,
  .el-icon {
    color: var(--el-color-white);
  }
}

.dark .setting-item:hover { background:var(--ui-hover); }
@media (max-width:700px) { .user-trigger { min-width:44px; grid-template-columns:28px; padding:0 7px; }.user-copy,.dropdown-arrow { display:none; } }
</style>
