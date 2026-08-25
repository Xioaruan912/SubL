<template>
  <div class="app-container">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>解锁测试</span>
        </div>
      </template>

      <div class="select-bar">
        <el-select
          v-model="selectedNodeId"
          filterable
          placeholder="选择要测试的节点"
          class="node-select"
        >
          <el-option
            v-for="n in nodes"
            :key="n.ID"
            :label="n.Name"
            :value="n.ID"
          />
        </el-select>
        <el-button type="primary" :loading="loading" @click="startTest">
          开始测试
        </el-button>
      </div>

      <!-- 测试结果 -->
      <template v-if="result">
        <div class="result-summary">
          <el-tag :type="result.ok ? 'success' : 'danger'" effect="dark" size="large">
            {{ result.ok ? "有可解锁服务" : "无可解锁服务" }}
          </el-tag>
        </div>

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

      <el-empty v-else-if="!loading" description="选择节点后点击「开始测试」" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { getNodes, UnlockTest } from "@/api/subcription/node";

defineOptions({
  name: "Unlock",
});

interface NodeItem {
  ID: number;
  Name: string;
  Link: string;
}
interface UnlockCheckResult {
  key: string;
  name: string;
  group: string;
  ok: boolean;
  rtt: number;
  note: string;
}
interface UnlockResult {
  nodeName: string;
  ok: boolean;
  results: UnlockCheckResult[];
  error?: string;
}

const nodes = ref<NodeItem[]>([]);
const selectedNodeId = ref<number | undefined>();
const loading = ref(false);
const result = ref<UnlockResult | null>(null);

const groups = [
  { key: "ai", label: "AI 服务" },
  { key: "video", label: "影视流媒体" },
  { key: "forum", label: "论坛 / 其它" },
];

const resultByGroup = (g: string) =>
  (result.value?.results || []).filter((r) => r.group === g);

const shortNote = (note: string) => {
  if (!note) return "";
  if (note.includes("socks connect")) return "连接失败/超时";
  return note.length > 40 ? note.slice(0, 40) + "..." : note;
};

const startTest = async () => {
  if (!selectedNodeId.value) {
    ElMessage.warning("请先选择节点");
    return;
  }
  loading.value = true;
  result.value = null;
  try {
    const { data } = await UnlockTest({ id: selectedNodeId.value });
    result.value = data;
  } catch (e: any) {
    ElMessage.error(e?.message || "解锁测试失败");
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  try {
    const { data } = await getNodes();
    nodes.value = data || [];
  } catch {
    nodes.value = [];
  }
});
</script>

<style lang="scss" scoped>
.card-header {
  font-size: 15px;
  font-weight: 600;
}

.select-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;

  .node-select {
    width: 320px;
  }
}

.result-summary {
  margin-bottom: 16px;
}

.group-section {
  margin-bottom: 20px;

  .group-title {
    margin-bottom: 10px;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-secondary);
  }
}

.group-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

.unlock-item {
  padding: 12px 14px;
  background: #f8f9fa;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;

  .unlock-item-top {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;

    &.ok { background: #30a46c; }
    &.fail { background: #e5484d; }
  }

  .item-name {
    flex: 1;
    overflow: hidden;
    font-size: 13px;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .item-meta {
    margin-top: 6px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .item-note {
    margin-left: 8px;
    color: var(--el-text-color-placeholder);
  }
}

html.dark .unlock-item {
  background: #202425;
}
</style>