<template>
  <div class="app-container">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>TCP 测试</span>
        </div>
      </template>

      <!-- 顶部操作区 -->
      <div class="toolbar">
        <el-select
          v-model="selectedNodeId"
          filterable
          placeholder="选择要测试的节点"
          class="node-select"
        >
          <el-option v-for="n in nodes" :key="n.ID" :label="n.Name" :value="n.ID" />
        </el-select>

        <el-select
          v-model="filterProvinces"
          multiple
          collapse-tags
          collapse-tags-tooltip
          placeholder="省份（默认全部）"
          class="filter-provinces"
        >
          <el-option v-for="p in provinceList" :key="p" :label="p" :value="p" />
        </el-select>

        <el-select
          v-model="filterIsps"
          multiple
          collapse-tags
          placeholder="运营商（默认全部）"
          class="filter-isps"
        >
          <el-option v-for="i in ispList" :key="i" :label="i" :value="i" />
        </el-select>

        <el-select v-model="zstaticPort" class="filter-zstatic">
          <el-option label="zstaticcdn 443" value="443" />
          <el-option label="zstaticcdn 80" value="80" />
        </el-select>

        <el-button type="primary" :loading="loading" @click="startTest">
          开始测试
        </el-button>
      </div>

      <!-- zstaticcdn 延迟 -->
      <div v-if="chinaResult" class="zstatic-bar">
        <span class="zstatic-label">字节跳动测速源 (lf3-ips.zstaticcdn.com)</span>
        <span
          v-for="z in chinaResult.zstatic"
          :key="z.port"
          class="zstatic-tag"
          :style="rttStyle(z.rtt)"
        >
          :{{ z.port }} → {{ z.rtt < 0 ? "不可达" : z.rtt + " ms" }}
        </span>
      </div>

      <!-- 颜色图例 -->
      <div v-if="chinaResult" class="legend">
        <span>延迟：</span>
        <span v-for="l in legend" :key="l.label" class="legend-item">
          <span class="legend-color" :style="{ background: l.color }" />
          {{ l.label }}
        </span>
      </div>

      <!-- itdog 式彩色网格 -->
      <template v-if="chinaResult">
        <div v-for="g in provinceGroups" :key="g.province" class="province-card">
          <div class="province-title">
            {{ g.province }}
            <span class="province-meta">（{{ g.okCount }}/{{ g.items.length }} 可达）</span>
          </div>
          <div class="grid">
            <div
              v-for="t in g.items"
              :key="t.ip + t.port"
              class="cell"
              :style="rttStyle(t.rtt)"
            >
              <div class="cell-city">{{ t.city }} {{ t.isp }}</div>
              <div class="cell-rtt">{{ t.rtt < 0 ? "超时" : t.rtt + "ms" }}</div>
              <div class="cell-ip">{{ t.ip }}:{{ t.port }}</div>
            </div>
          </div>
        </div>
      </template>

      <el-empty v-else-if="!loading" description="选择节点后点击「开始测试」" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { getNodes, ChinaPingTest } from "@/api/subcription/node";

defineOptions({
  name: "TcpTest",
});

interface NodeItem {
  ID: number;
  Name: string;
  Link: string;
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
const chinaResult = ref<ChinaResult | null>(null);

const filterProvinces = ref<string[]>([]);
const filterIsps = ref<string[]>([]);
const zstaticPort = ref("443");

const provinceList = ["北京","天津","上海","重庆","河北","山西","辽宁","吉林","黑龙江","江苏","浙江","安徽","福建","江西","山东","河南","湖北","湖南","广东","海南","四川","贵州","云南","陕西","甘肃","青海","内蒙古","广西","西藏","宁夏","新疆"];
const ispList = ["电信", "联通", "移动"];

// itdog 式延迟颜色区间
const legend = [
  { color: "#2ecc71", label: "<50ms" },
  { color: "#a8e063", label: "50-100ms" },
  { color: "#f1c40f", label: "100-200ms" },
  { color: "#e67e22", label: "200-300ms" },
  { color: "#e74c3c", label: ">300ms" },
  { color: "#95a5a6", label: "超时" },
];

// 返回格子背景色样式 + 文字颜色
const rttStyle = (rtt: number) => {
  if (rtt < 0) return { background: "#95a5a6", color: "#fff" };
  if (rtt < 50) return { background: "#2ecc71", color: "#fff" };
  if (rtt < 100) return { background: "#a8e063", color: "#333" };
  if (rtt < 200) return { background: "#f1c40f", color: "#333" };
  if (rtt < 300) return { background: "#e67e22", color: "#fff" };
  return { background: "#e74c3c", color: "#fff" };
};

// 按省分组（前端再过滤）
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
    .map((province) => {
      const items = byProv[province];
      return {
        province,
        items,
        okCount: items.filter((i) => i.rtt >= 0).length,
      };
    });
});

const startTest = async () => {
  if (!selectedNodeId.value) {
    ElMessage.warning("请先选择节点");
    return;
  }
  loading.value = true;
  chinaResult.value = null;
  try {
    const payload: any = { id: selectedNodeId.value, zstatic_port: zstaticPort.value };
    if (filterProvinces.value.length) payload.provinces = filterProvinces.value.join(",");
    if (filterIsps.value.length) payload.isps = filterIsps.value.join(",");
    const { data } = await ChinaPingTest(payload);
    chinaResult.value = data;
  } catch (e: any) {
    ElMessage.error(e?.message || "TCP 测试失败");
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

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;

  .node-select { width: 240px; }
  .filter-provinces { width: 220px; }
  .filter-isps { width: 150px; }
  .filter-zstatic { width: 160px; }
}

.zstatic-bar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 8px;

  .zstatic-label {
    font-size: 13px;
    font-weight: 500;
  }

  .zstatic-tag {
    padding: 2px 10px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 600;
  }
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);

  .legend-item {
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .legend-color {
    width: 14px;
    height: 14px;
    border-radius: 3px;
  }
}

.province-card {
  margin-bottom: 16px;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;

  .province-title {
    margin-bottom: 10px;
    font-size: 15px;
    font-weight: 600;

    .province-meta {
      font-size: 12px;
      font-weight: 400;
      color: var(--el-text-color-secondary);
    }
  }
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
}

.cell {
  padding: 8px 6px;
  border-radius: 8px;
  text-align: center;
  min-width: 0;

  .cell-city {
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cell-rtt {
    margin-top: 2px;
    font-size: 14px;
    font-weight: 700;
  }

  .cell-ip {
    margin-top: 2px;
    font-size: 10px;
    opacity: 0.8;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

html.dark .zstatic-bar {
  background: #202425;
}
</style>