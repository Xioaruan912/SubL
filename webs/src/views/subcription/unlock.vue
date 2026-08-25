<template>
  <div class="app-container">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>解锁测试 / 中国延迟</span>
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
          解锁测试
        </el-button>
        <el-button type="success" :loading="chinaLoading" @click="startChinaPing">
          中国延迟
        </el-button>
      </div>

      <!-- 解锁测试结果 -->
      <template v-if="result">
        <el-divider content-position="left">解锁测试结果</el-divider>
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

      <!-- 中国延迟筛选 + 结果 -->
      <template v-if="chinaResult">
        <el-divider content-position="left">中国各地延迟</el-divider>
        <div class="filter-bar">
          <el-select v-model="filterProvinces" multiple collapse-tags placeholder="省份（默认全部）" class="filter-provinces">
            <el-option v-for="p in provinceList" :key="p" :label="p" :value="p" />
          </el-select>
          <el-select v-model="filterIsps" multiple collapse-tags placeholder="运营商（默认全部）" class="filter-isps">
            <el-option v-for="i in ispList" :key="i" :label="i" :value="i" />
          </el-select>
          <el-select v-model="zstaticPort" class="filter-zstatic">
            <el-option label="zstaticcdn 443" value="443" />
            <el-option label="zstaticcdn 80" value="80" />
          </el-select>
          <el-button size="small" @click="reChinaPing">重新测试</el-button>
        </div>

        <!-- zstaticcdn 延迟 -->
        <div class="zstatic-bar">
          <span class="zstatic-label">字节跳动测速源 (lf3-ips.zstaticcdn.com)</span>
          <el-tag
            v-for="z in chinaResult.zstatic"
            :key="z.port"
            :type="rttType(z.rtt)"
            size="small"
            effect="light"
          >
            :{{ z.port }} → {{ z.rtt < 0 ? "不可达" : z.rtt + " ms" }}
          </el-tag>
        </div>

        <!-- 按省折叠分组 -->
        <el-collapse class="province-collapse">
          <el-collapse-item v-for="g in provinceGroups" :key="g.province" :title="groupTitle(g)">
            <div class="province-grid">
              <div v-for="t in g.items" :key="t.ip + t.port" class="china-item">
                <span class="china-city">{{ t.city }}</span>
                <el-tag :type="rttType(t.rtt)" size="small" effect="light">
                  {{ t.rtt < 0 ? "不可达" : t.rtt + " ms" }}
                </el-tag>
                <span class="china-ip">{{ t.ip }}:{{ t.port }}</span>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </template>

      <el-empty v-else-if="!loading && !chinaLoading" description="选择节点后点击「解锁测试」或「中国延迟」" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { getNodes, UnlockTest, ChinaPingTest } from "@/api/subcription/node";

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
interface ChinaTarget {
  province: string;
  city: string;
  isp: string;
  ip: string;
  port: number;
  rtt: number;
}
interface ChinaResult {
  nodeName: string;
  targets: ChinaTarget[];
  zstatic: { port: number; rtt: number }[];
}

const nodes = ref<NodeItem[]>([]);
const selectedNodeId = ref<number | undefined>();
const loading = ref(false);
const result = ref<UnlockResult | null>(null);
const chinaLoading = ref(false);
const chinaResult = ref<ChinaResult | null>(null);

const filterProvinces = ref<string[]>([]);
const filterIsps = ref<string[]>([]);
const zstaticPort = ref("443");

const provinceList = ["北京","天津","上海","重庆","河北","山西","辽宁","吉林","黑龙江","江苏","浙江","安徽","福建","江西","山东","河南","湖北","湖南","广东","海南","四川","贵州","云南","陕西","甘肃","青海","内蒙古","广西","西藏","宁夏","新疆"];
const ispList = ["电信", "联通", "移动"];

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

const rttType = (rtt: number) => {
  if (rtt < 0) return "danger";
  if (rtt < 100) return "success";
  if (rtt < 300) return "warning";
  return "danger";
};

// 按省折叠分组（本地再过滤前端筛选）
const provinceGroups = computed(() => {
  const byProv: Record<string, ChinaTarget[]> = {};
  for (const t of chinaResult.value?.targets || []) {
    if (filterProvinces.value.length && !filterProvinces.value.includes(t.province)) continue;
    if (filterIsps.value.length && !filterIsps.value.includes(t.isp)) continue;
    if (!byProv[t.province]) byProv[t.province] = [];
    byProv[t.province].push(t);
  }
  return Object.keys(byProv)
    .sort()
    .map((province) => ({ province, items: byProv[province] }));
});

const groupTitle = (g: any) => {
  const total = g.items.length;
  const ok = g.items.filter((i: ChinaTarget) => i.rtt >= 0).length;
  return `${g.province}（${ok}/${total} 可达）`;
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

const doChinaPing = async () => {
  if (!selectedNodeId.value) {
    ElMessage.warning("请先选择节点");
    return;
  }
  chinaLoading.value = true;
  try {
    const payload: any = { id: selectedNodeId.value, zstatic_port: zstaticPort.value };
    if (filterProvinces.value.length) payload.provinces = filterProvinces.value.join(",");
    if (filterIsps.value.length) payload.isps = filterIsps.value.join(",");
    const { data } = await ChinaPingTest(payload);
    chinaResult.value = data;
  } catch (e: any) {
    ElMessage.error(e?.message || "中国延迟测试失败");
  } finally {
    chinaLoading.value = false;
  }
};

const startChinaPing = doChinaPing;
const reChinaPing = doChinaPing;

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

.filter-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;

  .filter-provinces { width: 200px; }
  .filter-isps { width: 140px; }
  .filter-zstatic { width: 160px; }
}

.zstatic-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 8px;

  .zstatic-label {
    font-size: 13px;
    font-weight: 500;
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

.province-collapse {
  .province-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 8px;
    padding: 4px 0;

    .china-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 10px;
      background: #f8f9fa;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;

      .china-city {
        font-size: 13px;
        font-weight: 500;
        flex-shrink: 0;
      }

      .china-ip {
        margin-left: auto;
        font-size: 11px;
        color: var(--el-text-color-placeholder);
      }
    }
  }
}

html.dark .unlock-item,
html.dark .china-item,
html.dark .zstatic-bar {
  background: #202425;
}
</style>