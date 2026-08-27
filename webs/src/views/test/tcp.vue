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
        <el-button v-if="loading" type="warning" @click="abortTest">停止</el-button>
      </div>

      <!-- 颜色图例 -->
      <div class="legend">
        <span>延迟：</span>
        <span v-for="l in legend" :key="l.label" class="legend-item">
          <span class="legend-color" :style="{ background: l.color }" />
          {{ l.label }}
        </span>
      </div>

      <!-- 左地图 + 右网格（始终渲染，页面加载即显示空地图） -->
      <el-row :gutter="16">
        <el-col :span="14" :xs="24">
          <div class="map-card">
            <div id="chinaTcpMap" class="china-map"></div>
            <el-empty v-if="!loading && mapEmpty" description="无测试数据" :image-size="50" />
          </div>
        </el-col>
        <el-col :span="10" :xs="24">
          <el-scrollbar height="560px" class="grid-scroll">
            <div v-for="g in provinceGroups" :key="g.province" class="province-card">
              <div class="province-title">
                {{ g.province }}
                <span class="province-meta">（{{ g.okCount }}/{{ g.items.length }} 可达 · 平均 {{ g.avg }}ms）</span>
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
                </div>
              </div>
            </div>
            <el-empty v-if="!loading && provinceGroups.length === 0" description="暂无结果" :image-size="50" />
          </el-scrollbar>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import * as echarts from "echarts/core";
import { TooltipComponent, GeoComponent, VisualMapComponent } from "echarts/components";
import { MapChart } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import { getNodes } from "@/api/subcription/node";

defineOptions({
  name: "TcpTest",
});

echarts.use([TooltipComponent, GeoComponent, VisualMapComponent, MapChart, CanvasRenderer]);

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
  lat: number;
  lng: number;
}

const nodes = ref<NodeItem[]>([]);
const selectedNodeId = ref<number | undefined>();
const loading = ref(false);
const abortCtrl = ref<AbortController | null>(null);

const targets = ref<ChinaTarget[]>([]);
const zstaticResults = ref<{ port: number; rtt: number }[]>([]);
const mapEmpty = ref(true);
const chart = ref<any>(null);
const chinaJson = ref<any>(null);

const filterProvinces = ref<string[]>([]);
const filterIsps = ref<string[]>([]);
const zstaticPort = ref("443");

const provinceList = ["北京","天津","上海","重庆","河北","山西","辽宁","吉林","黑龙江","江苏","浙江","安徽","福建","江西","山东","河南","湖北","湖南","广东","海南","四川","贵州","云南","陕西","甘肃","青海","内蒙古","广西","西藏","宁夏","新疆"];
const ispList = ["电信", "联通", "移动"];

const legend = [
  { color: "#2ecc71", label: "<50ms" },
  { color: "#a8e063", label: "50-100ms" },
  { color: "#f1c40f", label: "100-200ms" },
  { color: "#e67e22", label: "200-300ms" },
  { color: "#e74c3c", label: ">300ms" },
  { color: "#95a5a6", label: "超时" },
];

// 返回格子背景色（地图也用此色值）
const rttColor = (rtt: number) => {
  if (rtt < 0) return "#95a5a6";
  if (rtt < 50) return "#2ecc71";
  if (rtt < 100) return "#a8e063";
  if (rtt < 200) return "#f1c40f";
  if (rtt < 300) return "#e67e22";
  return "#e74c3c";
};

const rttStyle = (rtt: number) => {
  const c = rttColor(rtt);
  return { background: c, color: rtt < 0 || rtt >= 100 || rtt < 50 ? "#fff" : "#333" };
};

// 按省分组（前端再过滤，聚合平均延迟）
const provinceGroups = computed(() => {
  const byProv: Record<string, ChinaTarget[]> = {};
  for (const t of targets.value) {
    if (filterProvinces.value.length && !filterProvinces.value.includes(t.province)) continue;
    if (filterIsps.value.length && !filterIsps.value.includes(t.isp)) continue;
    if (!byProv[t.province]) byProv[t.province] = [];
    byProv[t.province].push(t);
  }
  return Object.keys(byProv)
    .sort()
    .map((province) => {
      const items = byProv[province];
      const ok = items.filter((i) => i.rtt >= 0);
      const avg = ok.length ? Math.round(ok.reduce((s, i) => s + i.rtt, 0) / ok.length) : -1;
      return { province, items, okCount: ok.length, avg };
    });
});

// 加载中国地图 GeoJSON
const loadChinaJson = async () => {
  if (chinaJson.value) return chinaJson.value;
  try {
    const res = await fetch("/static/china.json");
    chinaJson.value = await res.json();
  } catch {
    chinaJson.value = null;
  }
  return chinaJson.value;
};

// 初始化地图
const initChart = async () => {
  const geoJson = await loadChinaJson();
  const el = document.getElementById("chinaTcpMap") as HTMLDivElement;
  if (!el) return;
  if (geoJson) echarts.registerMap("china", geoJson);
  chart.value = markRaw(echarts.init(el));
  chart.value.setOption({
    backgroundColor: "transparent",
    tooltip: {
      trigger: "item",
      formatter: (p: any) => {
        if (p.seriesType === "map") {
          const items: ChinaTarget[] = p.data?.targets || [];
          if (!items.length) return p.name;
          // 按运营商聚合
          const byIsp: Record<string, ChinaTarget[]> = {};
          for (const t of items) {
            if (!byIsp[t.isp]) byIsp[t.isp] = [];
            byIsp[t.isp].push(t);
          }
          let html = `<b>${p.name}</b><br/>`;
          for (const isp of Object.keys(byIsp)) {
            const list = byIsp[isp];
            const ok = list.filter((t) => t.rtt >= 0);
            const avg = ok.length ? Math.round(ok.reduce((s, t) => s + t.rtt, 0) / ok.length) : -1;
            const cities = [...new Set(list.map((t) => t.city))].join("/");
            html += `${isp}：${avg >= 0 ? avg + "ms" : "超时"}（${cities}）<br/>`;
          }
          return html;
        }
        return p.name;
      },
    },
    visualMap: {
      type: "piecewise",
      show: true,
      left: 8,
      bottom: 8,
      textStyle: { fontSize: 11 },
      pieces: [
        { gt: 0, lte: 50, color: "#2ecc71", label: "<50ms" },
        { gt: 50, lte: 100, color: "#a8e063", label: "50-100ms" },
        { gt: 100, lte: 200, color: "#f1c40f", label: "100-200ms" },
        { gt: 200, lte: 300, color: "#e67e22", label: "200-300ms" },
        { gt: 300, color: "#e74c3c", label: ">300ms" },
      ],
    },
    series: [{
      name: "TCP 延迟",
      type: "map",
      map: "china",
      roam: true,
      scaleLimit: { min: 0.9, max: 6 },
      label: { show: false },
      emphasis: { label: { show: true }, itemStyle: { areaColor: "#ffd666" } },
      itemStyle: {
        areaColor: "#eef1f5",
        borderColor: "#b8d6ff",
        borderWidth: 0.5,
      },
      data: [],
    }],
  }, true);
  window.addEventListener("resize", () => chart.value?.resize());
};

// 更新地图省份颜色（按平均延迟着色），data 附带该省运营商明细供 tooltip 展示
const updateMap = () => {
  if (!chart.value) return;
  const byProv: Record<string, ChinaTarget[]> = {};
  for (const t of targets.value) {
    if (filterProvinces.value.length && !filterProvinces.value.includes(t.province)) continue;
    if (filterIsps.value.length && !filterIsps.value.includes(t.isp)) continue;
    if (!byProv[t.province]) byProv[t.province] = [];
    byProv[t.province].push(t);
  }
  const data = Object.keys(byProv).map((prov) => {
    const items = byProv[prov];
    const ok = items.filter((t) => t.rtt >= 0);
    const avg = ok.length ? Math.round(ok.reduce((s, t) => s + t.rtt, 0) / ok.length) : -1;
    return { name: prov, value: avg, targets: items };
  });
  chart.value.setOption({
    series: [{ data }],
  }, false);
  mapEmpty.value = data.length === 0;
};

// SSE 流式解析（fetch + ReadableStream）
const parseSSE = async (res: Response) => {
  const reader = res.body?.getReader();
  if (!reader) return;
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    // SSE 事件以空行分隔
    const events = buf.split("\n\n");
    buf = events.pop() || "";
    for (const ev of events) {
      const eventType = ev.match(/^event:\s*(\S+)/m)?.[1];
      const dataMatch = ev.match(/^data:\s*(.*)$/m);
      if (!dataMatch) continue;
      try {
        const data = JSON.parse(dataMatch[1]);
        handleSSEEvent(eventType || "", data);
      } catch { /* ignore */ }
    }
  }
};

const handleSSEEvent = (type: string, data: any) => {
  if (type === "province" && data?.targets) {
    // 追加该省目标（去重）
    for (const t of data.targets) {
      const idx = targets.value.findIndex((x) => x.ip === t.ip && x.port === t.port);
      if (idx >= 0) targets.value[idx] = t;
      else targets.value.push(t);
    }
    updateMap();
  } else if (type === "zstatic" && data?.zstatic) {
    zstaticResults.value = data.zstatic;
  } else if (type === "error") {
    ElMessage.error(data?.msg || "测试失败");
  }
};

const startTest = async () => {
  if (!selectedNodeId.value) {
    ElMessage.warning("请先选择节点");
    return;
  }
  // 初始化地图（若尚未初始化，确保 DOM 就绪）
  if (!chart.value) {
    await nextTick();
    await initChart();
  }
  loading.value = true;
  targets.value = [];
  zstaticResults.value = [];
  mapEmpty.value = true;
  updateMap();

  abortCtrl.value = new AbortController();
  const payload = new URLSearchParams();
  payload.append("id", String(selectedNodeId.value));
  payload.append("zstatic_port", zstaticPort.value);
  if (filterProvinces.value.length) payload.append("provinces", filterProvinces.value.join(","));
  if (filterIsps.value.length) payload.append("isps", filterIsps.value.join(","));

  try {
    const res = await fetch(import.meta.env.VITE_APP_BASE_API + "/api/v1/nodes/chinaping/stream", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded", Authorization: "Bearer " + getToken() },
      body: payload.toString(),
      signal: abortCtrl.value.signal,
    });
    if (!res.ok) {
      const body = await res.text();
      try {
        const j = JSON.parse(body);
        ElMessage.error(j.msg || "请求失败");
      } catch {
        ElMessage.error("请求失败: " + res.status);
      }
      return;
    }
    await parseSSE(res);
  } catch (e: any) {
    if (e?.name !== "AbortError") ElMessage.error(e?.message || "测试失败");
  } finally {
    loading.value = false;
    abortCtrl.value = null;
  }
};

// 从 localStorage 获取 token（与 request 工具一致，已含 Bearer 前缀）
const getToken = () => {
  try {
    return localStorage.getItem("accessToken") || "";
  } catch {
    return "";
  }
};

const abortTest = () => {
  abortCtrl.value?.abort();
  loading.value = false;
};

onMounted(async () => {
  try {
    const { data } = await getNodes();
    nodes.value = data || [];
  } catch {
    nodes.value = [];
  }
  // 确保 DOM 渲染完成后再初始化地图（容器始终存在）
  await nextTick();
  await initChart();
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
  margin-bottom: 14px;

  .node-select { width: 220px; }
  .filter-provinces { width: 210px; }
  .filter-isps { width: 140px; }
  .filter-zstatic { width: 160px; }
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-bottom: 14px;
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

.map-card {
  position: relative;
  min-height: 400px;
}

.china-map {
  width: 100%;
  height: 560px;
}

.grid-scroll {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  padding: 8px;
}

.province-card {
  margin-bottom: 10px;
  padding: 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;

  .province-title {
    margin-bottom: 8px;
    font-size: 14px;
    font-weight: 600;

    .province-meta {
      font-size: 11px;
      font-weight: 400;
      color: var(--el-text-color-secondary);
    }
  }
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
  gap: 6px;
}

.cell {
  padding: 6px 4px;
  border-radius: 6px;
  text-align: center;
  min-width: 0;

  .cell-city {
    font-size: 11px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cell-rtt {
    margin-top: 2px;
    font-size: 13px;
    font-weight: 700;
  }
}
</style>