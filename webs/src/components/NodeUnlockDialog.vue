<template>
  <el-dialog
    :model-value="visible"
    :title="'解锁测试 · ' + nodeName"
    width="70%"
    :close-on-click-modal="false"
    @update:model-value="(v: boolean) => emit('update:visible', v)"
    @open="onOpen"
    @closed="onClosed"
  >
    <!-- 操作栏 -->
    <div class="toolbar">
      <el-button type="primary" size="small" :loading="loading" @click="stopOtherTestAndStart">
        重新测速
      </el-button>
      <el-button v-if="loading" size="small" type="warning" @click="abortTest">停止</el-button>
    </div>

    <!-- 结果摘要 -->
    <div v-if="result" class="result-summary">
      <el-tag :type="result.ok ? 'success' : 'danger'" effect="dark" size="large">
        {{ result.ok ? "有可解锁服务" : "无可解锁服务" }}
      </el-tag>
      <span v-if="loading" class="loading-hint">正在测试，请稍候…</span>
    </div>

    <!-- 测试结果 -->
    <template v-if="result">
      <div v-for="group in groups" :key="group.key" class="group-section">
        <div class="group-title">{{ group.label }}</div>
        <div class="group-grid">
          <div v-for="r in resultByGroup(group.key)" :key="r.key" class="unlock-item">
            <div class="unlock-item-top">
              <span class="status-dot" :class="r.ok ? 'ok' : 'fail'" />
              <span class="item-name">{{ r.name }}</span>
              <el-tag :type="r.ok ? 'success' : 'danger'" size="small" effect="light">
                {{ r.ok ? "解锁" : "未解锁" }}
              </el-tag>
            </div>
            <div class="item-meta">
              <span v-if="r.rtt > 0">延迟 {{ r.rtt }}ms</span>
              <span v-if="!r.ok && r.note" class="item-note">{{ shortNote(r.note) }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <div v-else-if="loading" class="loading-hint">正在测试，请稍候…</div>
    <el-empty v-else description="点击「开始测试」进行解锁检测" />

    <template #footer>
      <el-button @click="emit('update:visible', false)">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { UnlockTest, GetTestStatus, CancelTest } from "@/api/subcription/node";
import { saveUnlock, getUnlock } from "@/utils/nodeTestStorage";

const emit = defineEmits<{ (e: "update:visible", v: boolean): void }>();

interface UnlockCheckResult {
  key: string; name: string; group: string; ok: boolean; rtt: number; note: string;
}
interface UnlockResult {
  nodeName: string; ok: boolean; results: UnlockCheckResult[]; error?: string;
}

const loading = ref(false);
const result = ref<UnlockResult | null>(null);
let abortCtrl: AbortController | null = null;

const groups = [
  { key: "ai", label: "AI 服务" },
  { key: "video", label: "影视流媒体" },
  { key: "forum", label: "论坛 / 其它" },
];

const props = defineProps<{ visible: boolean; nodeId: number; nodeName: string }>();

const resultByGroup = (g: string) =>
  (result.value?.results || []).filter((r) => r.group === g);

const shortNote = (note: string) => {
  if (!note) return "";
  if (note.includes("socks connect")) return "连接失败/超时";
  return note.length > 40 ? note.slice(0, 40) + "..." : note;
};

const startTest = async () => {
  loading.value = true;
  result.value = null;
  abortCtrl?.abort();
  abortCtrl = new AbortController();
  try {
    const payload = new URLSearchParams();
    payload.append("id", String(props.nodeId));
    const res = await fetch(import.meta.env.VITE_APP_BASE_API + "/api/v1/nodes/unlock", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded", Authorization: "Bearer " + getToken() },
      body: payload.toString(),
      signal: abortCtrl.signal,
    });
    const json = await res.json();
    if (json?.code === "00000") {
      result.value = json.data;
      saveUnlock(props.nodeId, json.data);
    } else {
      ElMessage.error(json?.msg || "解锁测试失败");
    }
  } catch (e: any) {
    if (e?.name !== "AbortError") ElMessage.error(e?.message || "解锁测试失败");
  } finally {
    loading.value = false;
  }
};

// 停止正在进行的其它测试（若存在），等待锁释放后再测速
const stopOtherTestAndStart = async () => {
  let cur = null;
  try { cur = (await GetTestStatus())?.data || null; } catch { /* ignore */ }
  if (cur) {
    ElMessage.warning(`正在停止当前测试（${cur.type === 'unlock' ? '解锁' : 'TCP'} · ${cur.nodeName}）`);
    try { await CancelTest(); } catch { /* ignore */ }
    // 等待锁释放（最多 3s）
    for (let i = 0; i < 6; i++) {
      await new Promise(r => setTimeout(r, 500));
      let after = null;
      try { after = (await GetTestStatus())?.data || null; } catch { /* ignore */ }
      if (!after) break;
    }
  }
  startTest();
};

const abortTest = () => {
  abortCtrl?.abort();
  abortCtrl = null;
  loading.value = false;
};

const getToken = () => {
  try { return localStorage.getItem("accessToken") || ""; } catch { return ""; }
};

const onOpen = () => {
  // 只读取 localStorage 持久化结果，不自动测速
  const cached = getUnlock(props.nodeId);
  result.value = cached || null;
};

const onClosed = () => {
  // 不 abort：让测试继续跑完，结果写入 localStorage，重开时恢复
  result.value = null;
};

watch(() => props.visible, (v) => { if (v) onOpen(); }, { immediate: true });
watch(() => props.nodeId, () => { if (props.visible) onOpen(); });
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; align-items: center; margin-bottom: 12px; }
.result-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
.group-section { margin-bottom: 16px; }
.group-title {
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.group-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
}
.unlock-item {
  padding: 10px 12px;
  background: #f8f9fa;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
}
.unlock-item-top { display: flex; align-items: center; gap: 8px; }
.status-dot {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
}
.status-dot.ok { background: #30a46c; }
.status-dot.fail { background: #e5484d; }
.item-name {
  flex: 1; overflow: hidden; font-size: 13px; font-weight: 500;
  text-overflow: ellipsis; white-space: nowrap;
}
.item-meta { margin-top: 6px; font-size: 12px; color: var(--el-text-color-secondary); }
.item-note { margin-left: 8px; color: var(--el-text-color-placeholder); }
.loading-hint { padding: 30px 0; text-align: center; color: var(--el-text-color-secondary); }
html.dark .unlock-item { background: #202425; }
</style>