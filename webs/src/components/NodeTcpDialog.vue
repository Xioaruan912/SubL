<template>
  <el-dialog
    :model-value="visible"
    :title="'代理链路 TCP · ' + nodeName"
    width="85%"
    top="5vh"
    :close-on-click-modal="false"
    @update:model-value="(v: boolean) => emit('update:visible', v)"
    @open="onOpen"
    @closed="onClosed"
  >
    <!-- 操作栏 -->
    <div class="toolbar">
      <el-button :loading="loading" type="primary" size="small" @click="stopOtherTestAndStart">重新测速</el-button>
      <el-button v-if="loading" size="small" type="warning" @click="abortTest">停止</el-button>
      <el-select v-model="zstaticPort" size="small" class="zstatic">
        <el-option label="zstaticcdn 443" value="443" />
        <el-option label="zstaticcdn 80" value="80" />
      </el-select>
      <span class="method-hint">VPS → 节点 → 国内目标 · 双样本均值</span>
    </div>

    <!-- 图例 -->
    <div class="legend">
      <span>延迟：</span>
      <span v-for="l in legend" :key="l.label" class="legend-item">
        <span class="legend-color" :style="{ background: l.color }" />
        {{ l.label }}
      </span>
    </div>

    <el-row :gutter="12" v-loading="loading">
      <!-- 左地图 -->
      <el-col :span="14" :xs="24">
        <div id="chinaTcpMapDialog" class="china-map"></div>
      </el-col>
      <!-- 右网格 -->
      <el-col :span="10" :xs="24">
        <el-scrollbar height="420px" class="grid-scroll">
          <div v-for="g in provinceGroups" :key="g.province" class="province-card">
            <div class="province-title">
              {{ g.province }}
              <span class="province-meta">（{{ g.okCount }}/{{ g.items.length }} 可达 · 平均 {{ g.avg }}ms）</span>
            </div>
            <div class="grid">
              <div v-for="t in g.items" :key="t.ip + t.port" class="cell" :style="rttStyle(t.rtt)">
                <div class="cell-city">{{ t.city }} {{ t.isp }}</div>
                <div class="cell-rtt">{{ t.rtt < 0 ? "超时" : t.rtt + "ms" }}</div>
              </div>
            </div>
          </div>
          <el-empty v-if="!loading && provinceGroups.length === 0" description="暂无结果" :image-size="40" />
        </el-scrollbar>
      </el-col>
    </el-row>

    <template #footer>
      <el-button @click="emit('update:visible', false)">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from "vue";
import * as echarts from "echarts/core";
import { TooltipComponent, GeoComponent, VisualMapComponent } from "echarts/components";
import { MapChart } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import { saveTcp, getTcp } from "@/utils/nodeTestStorage";
import { GetTestStatus, CancelTest } from "@/api/subcription/node";

defineOptions({ name: "NodeTcpDialog" });

echarts.use([TooltipComponent, GeoComponent, VisualMapComponent, MapChart, CanvasRenderer]);

const props = defineProps<{ visible: boolean; nodeId: number; nodeName: string }>();
const emit = defineEmits<{ (e: "update:visible", v: boolean): void }>();

interface ChinaTarget {
  province: string; city: string; isp: string; ip: string;
  port: number; rtt: number; lat: number; lng: number;
}

const loading = ref(false);
const abortCtrl = ref<AbortController | null>(null);
const targets = ref<ChinaTarget[]>([]);
const zstaticResults = ref<{ port: number; rtt: number }[]>([]);
const mapEmpty = ref(true);
const chart = ref<any>(null);
const chinaJson = ref<any>(null);
const zstaticPort = ref("443");

const legend = [
  { color: "#2ecc71", label: "<50ms" }, { color: "#a8e063", label: "50-100ms" },
  { color: "#f1c40f", label: "100-200ms" }, { color: "#e67e22", label: "200-300ms" },
  { color: "#e74c3c", label: ">300ms" }, { color: "#95a5a6", label: "超时" },
];

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

const provinceGroups = computed(() => {
  const byProv: Record<string, ChinaTarget[]> = {};
  for (const t of targets.value) {
    if (!byProv[t.province]) byProv[t.province] = [];
    byProv[t.province].push(t);
  }
  return Object.keys(byProv).sort().map((province) => {
    const items = byProv[province];
    const ok = items.filter((i) => i.rtt >= 0);
    const avg = ok.length ? Math.round(ok.reduce((s, i) => s + i.rtt, 0) / ok.length) : -1;
    return { province, items, okCount: ok.length, avg };
  });
});

const loadChinaJson = async () => {
  if (chinaJson.value) return chinaJson.value;
  try {
    const res = await fetch("/static/china.json");
    chinaJson.value = await res.json();
  } catch { chinaJson.value = null; }
  return chinaJson.value;
};

const initChart = async () => {
  const geoJson = await loadChinaJson();
  const el = document.getElementById("chinaTcpMapDialog") as HTMLDivElement;
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
          const byIsp: Record<string, ChinaTarget[]> = {};
          for (const t of items) { if (!byIsp[t.isp]) byIsp[t.isp] = []; byIsp[t.isp].push(t); }
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
      type: "piecewise", show: true, left: 8, bottom: 8, textStyle: { fontSize: 10 },
      pieces: [
        { gt: 0, lte: 50, color: "#2ecc71", label: "<50ms" },
        { gt: 50, lte: 100, color: "#a8e063", label: "50-100ms" },
        { gt: 100, lte: 200, color: "#f1c40f", label: "100-200ms" },
        { gt: 200, lte: 300, color: "#e67e22", label: "200-300ms" },
        { gt: 300, color: "#e74c3c", label: ">300ms" },
      ],
    },
    series: [{
      name: "TCP 延迟", type: "map", map: "china", roam: true,
      scaleLimit: { min: 0.9, max: 6 },
      label: { show: false },
      emphasis: { label: { show: true }, itemStyle: { areaColor: "#ffd666" } },
      itemStyle: { areaColor: "#eef1f5", borderColor: "#b8d6ff", borderWidth: 0.5 },
      data: [],
    }],
  }, true);
  window.addEventListener("resize", () => chart.value?.resize());
};

const updateMap = () => {
  if (!chart.value) return;
  const byProv: Record<string, ChinaTarget[]> = {};
  for (const t of targets.value) {
    if (!byProv[t.province]) byProv[t.province] = [];
    byProv[t.province].push(t);
  }
  const data = Object.keys(byProv).map((prov) => {
    const items = byProv[prov];
    const ok = items.filter((t) => t.rtt >= 0);
    const avg = ok.length ? Math.round(ok.reduce((s, t) => s + t.rtt, 0) / ok.length) : -1;
    return { name: prov, value: avg, targets: items };
  });
  chart.value.setOption({ series: [{ data }] }, false);
  mapEmpty.value = data.length === 0;
};

const parseSSE = async (res: Response) => {
  const reader = res.body?.getReader();
  if (!reader) return;
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
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

const getToken = () => {
  try { return localStorage.getItem("accessToken") || ""; } catch { return ""; }
};

const startTest = async () => {
  if (!chart.value) { await nextTick(); await initChart(); }
  loading.value = true;
  targets.value = [];
  zstaticResults.value = [];
  mapEmpty.value = true;
  updateMap();
  abortCtrl.value?.abort();
  abortCtrl.value = new AbortController();
  const payload = new URLSearchParams();
  payload.append("id", String(props.nodeId));
  payload.append("zstatic_port", zstaticPort.value);
  try {
    const res = await fetch(import.meta.env.VITE_APP_BASE_API + "/api/v1/nodes/chinaping/stream", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded", Authorization: getToken() },
      body: payload.toString(),
      signal: abortCtrl.value.signal,
    });
    if (!res.ok) {
      const body = await res.text();
      try { const j = JSON.parse(body); ElMessage.error(j.msg || "请求失败"); }
      catch { ElMessage.error("请求失败: " + res.status); }
      return;
    }
    await parseSSE(res);
    // 持久化
    saveTcp(props.nodeId, { targets: targets.value, zstatic: zstaticResults.value });
  } catch (e: any) {
    if (e?.name !== "AbortError") ElMessage.error(e?.message || "测试失败");
  } finally {
    loading.value = false;
    abortCtrl.value = null;
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
  abortCtrl.value?.abort();
  loading.value = false;
};

const applyCached = (cached: any) => {
  targets.value = cached?.targets || [];
  zstaticResults.value = cached?.zstatic || [];
  mapEmpty.value = targets.value.length === 0;
  updateMap();
};

const onOpen = async () => {
  await nextTick();
  if (!chart.value) await initChart();
  // 只读取 localStorage 持久化结果，不自动测速
  const cached = getTcp(props.nodeId);
  if (cached) applyCached(cached);
  else { targets.value = []; zstaticResults.value = []; mapEmpty.value = true; updateMap(); }
};

const onClosed = () => {
  // 不 abort：让测试继续跑完，结果写入 localStorage，重开时恢复
};

watch(() => props.visible, (v) => { if (v) onOpen(); }, { immediate: true });
watch(() => props.nodeId, () => { if (props.visible) onOpen(); });
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; align-items: center; margin-bottom: 12px; }
.toolbar .zstatic { width: 160px; }
.method-hint { color: var(--el-text-color-secondary); font-size: 12px; }
.legend {
  display: flex; flex-wrap: wrap; gap: 12px; align-items: center;
  margin-bottom: 12px; font-size: 12px; color: var(--el-text-color-secondary);
}
.legend-item { display: flex; gap: 4px; align-items: center; }
.legend-color { width: 14px; height: 14px; border-radius: 3px; }
.china-map { width: 100%; height: 420px; }
.grid-scroll { border: 1px solid var(--el-border-color-lighter); border-radius: 10px; padding: 8px; }
.province-card { margin-bottom: 10px; padding: 8px; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; }
.province-title { margin-bottom: 6px; font-size: 13px; font-weight: 600; }
.province-meta { font-size: 11px; font-weight: 400; color: var(--el-text-color-secondary); }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(80px, 1fr)); gap: 5px; }
.cell { padding: 5px 3px; border-radius: 5px; text-align: center; min-width: 0; }
.cell-city { font-size: 10px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cell-rtt { margin-top: 2px; font-size: 12px; font-weight: 700; }
</style>
