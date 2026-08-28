<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from "vue";
import { getClientList, checkClient } from "@/api/subcription/clients";

defineOptions({ name: "ClientsPage" });

interface PlatformItem {
  key: string;
  label: string;
  version: string;
  size: number;
  status: string;
  errMsg: string;
  updatedAt: number;
}
interface ClientItem {
  type: string;
  name: string;
  icon: string;
  owner: string;
  repo: string;
  platforms: PlatformItem[];
  // App Store 外链
  appStoreUrl?: string;
  price?: string;
  region?: string;
  desc?: string;
}

const items = ref<ClientItem[]>([]);
const lastCheck = ref(0);
const checking = ref(false);
const downloading = ref<Record<string, boolean>>({});
const timer = ref<any>(null);

const sizeLabel = (s: number) => {
  if (!s) return "";
  const mb = s / 1024 / 1024;
  return mb >= 1024 ? `${(mb / 1024).toFixed(1)}GB` : `${mb.toFixed(0)}MB`;
};

const statusMeta = (p: PlatformItem) => {
  switch (p.status) {
    case "ready":
      return { label: p.version ? `已就绪 · v${p.version.replace(/^v/, "")}` : "已就绪", type: "success" as const, show: !!p.version };
    case "downloading":
      return { label: "下载中…", type: "warning" as const, show: true };
    case "failed":
      return { label: "失败", type: "danger" as const, show: true };
    default:
      return { label: "待更新", type: "info" as const, show: true };
  }
};

const fmtTime = (t: number) => {
  if (!t) return "";
  const d = new Date(t * 1000);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
};

const loading = ref(false);

const load = async (showLoading = false) => {
  if (showLoading) loading.value = true;
  try {
    const { data } = await getClientList();
    items.value = data?.items || [];
    lastCheck.value = data?.lastCheck || 0;
  } catch { /* ignore */ } finally {
    if (showLoading) loading.value = false;
  }
};

const doCheck = async () => {
  checking.value = true;
  await checkClient();
  // 轮询等待下载完成
  await poll();
  checking.value = false;
};

const poll = async () => {
  clearTimeout(timer.value);
  await load(false);
  const anyBusy = items.value.some(c => c.platforms.some(p => p.status === "downloading"));
  if (anyBusy) {
    timer.value = setTimeout(poll, 3000);
  }
};

// 下载（原生 fetch + token，blob 触发浏览器保存）
const download = async (client: string, p: PlatformItem) => {
  const key = `${client}_${p.key}`;
  if (downloading.value[key] || p.status !== "ready") return;
  downloading.value[key] = true;
  try {
    const token = localStorage.getItem("accessToken") || "";
    const url = `/api/v1/clients/download?client=${encodeURIComponent(client)}&platform=${p.key}`;
    const resp = await fetch(url, { headers: { Authorization: token } });
    if (!resp.ok) {
      const txt = await resp.text();
      ElMessage.error("下载失败: " + (txt.slice(0, 120) || resp.status));
      return;
    }
    const blob = await resp.blob();
    const cd = resp.headers.get("Content-Disposition") || "";
    const match = cd.match(/filename\*=utf-8''([^;]+)/) || cd.match(/filename="?([^";]+)"?/);
    const name = match ? decodeURIComponent(match[1]) : `${client}_${p.key}.bin`;
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = name;
    a.click();
    URL.revokeObjectURL(a.href);
  } catch (e: any) {
    ElMessage.error("下载失败: " + (e?.message || "未知错误"));
  } finally {
    downloading.value[key] = false;
  }
};

const goRepo = (c: ClientItem) => window.open(c.type === "appstore" ? c.appStoreUrl : `https://github.com/${c.owner}/${c.repo}`);

onMounted(() => {
  load(true);
  timer.value = setTimeout(poll, 3000);
});
onBeforeUnmount(() => clearTimeout(timer.value));
</script>

<template>
  <div class="clients-page">
    <!-- 顶部工具栏 -->
    <div class="toolbar">
      <div class="toolbar-info">
        <span>客户端下载中心</span>
        <span v-if="lastCheck" class="muted">上次检查：{{ fmtTime(lastCheck) }}</span>
      </div>
      <el-button type="primary" :loading="checking" @click="doCheck">
        <svg-icon icon-class="refresh" /> 检查更新
      </el-button>
    </div>

    <div v-loading="loading" class="client-loading-area">
      <el-empty v-if="!loading && !items.length" description="暂无客户端" />
      <div v-if="items.length" class="card-grid">
      <el-card v-for="c in items" :key="c.name" shadow="hover" class="client-card">
        <template #header>
          <div class="card-head">
            <span class="client-icon">{{ c.icon }}</span>
            <span class="client-name">{{ c.name }}</span>
            <el-tag v-if="c.type === 'appstore'" type="danger" size="small" effect="dark">付费</el-tag>
            <el-tag v-if="c.type === 'appstore'" type="info" size="small" effect="plain">{{ c.region }}</el-tag>
            <el-button link type="primary" size="small" @click="goRepo(c)">
              {{ c.type === 'appstore' ? 'App Store' : 'GitHub' }}
            </el-button>
          </div>
        </template>

        <!-- App Store 外链客户端 -->
        <div v-if="c.type === 'appstore'" class="appstore-body">
          <div class="appstore-desc">{{ c.desc }}</div>
          <div class="appstore-price">{{ c.price }}</div>
          <el-button type="primary" @click="goRepo(c)">
            <svg-icon icon-class="download" /> 前往 App Store
          </el-button>
        </div>

        <!-- GitHub 下载客户端 -->
        <div v-else class="platform-list">
          <div v-for="p in c.platforms" :key="p.key" class="platform-row">
            <div class="platform-info">
              <span class="platform-label">{{ p.label }}</span>
              <el-tag :type="statusMeta(p).type" size="small" effect="light">{{ statusMeta(p).label }}</el-tag>
              <span v-if="p.size" class="platform-size">{{ sizeLabel(p.size) }}</span>
            </div>
            <div class="platform-action">
              <span v-if="p.errMsg" class="platform-err">{{ p.errMsg.slice(0, 24) }}</span>
              <el-button
                size="small"
                :type="p.status === 'ready' ? 'primary' : 'default'"
                :loading="downloading[`${c.name}_${p.key}`]"
                :disabled="p.status !== 'ready'"
                @click="download(c.name, p)"
              >下载</el-button>
            </div>
          </div>
        </div>
      </el-card>
      </div>
    </div>
  </div>
</template>

<style scoped>
.client-loading-area { min-height: 400px; }
.clients-page { padding: 10px; }
.toolbar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 14px;
}
.toolbar-info { display: flex; align-items: center; gap: 12px; font-size: 15px; font-weight: 600; }
.toolbar-info .muted { font-size: 12px; font-weight: 400; color: var(--el-text-color-secondary); }
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
  gap: 14px;
}
.client-card { border-radius: 12px; }
.card-head { display: flex; align-items: center; gap: 10px; }
.client-icon { font-size: 22px; }
.client-name { flex: 1; font-weight: 600; font-size: 15px; }
.platform-list { display: flex; flex-direction: column; gap: 8px; }
.platform-row {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 12px; border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px; background: var(--el-fill-color-light);
}
.platform-info { display: flex; align-items: center; gap: 8px; }
.platform-label { font-size: 13px; font-weight: 500; }
.platform-size { font-size: 12px; color: var(--el-text-color-secondary); }
.platform-action { display: flex; align-items: center; gap: 8px; }
.platform-err { max-width: 140px; font-size: 12px; color: var(--el-color-danger); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.appstore-body {
  display: flex; flex-direction: column; align-items: center; gap: 12px;
  padding: 16px 0;
}
.appstore-desc { font-size: 13px; color: var(--el-text-color-regular); text-align: center; }
.appstore-price { font-size: 28px; font-weight: 700; color: var(--el-color-primary); }
</style>