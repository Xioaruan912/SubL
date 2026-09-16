<template>
  <div class="app-container">
    <el-card shadow="never">
      <template #header>
        <div class="card-header"><span>真实出口测速</span></div>
      </template>

      <div class="toolbar">
        <el-select
          v-model="selectedIds"
          multiple
          filterable
          collapse-tags
          collapse-tags-tooltip
          placeholder="选择要测速的节点"
          class="node-select"
          :loading="nodesLoading"
        >
          <el-option v-for="n in nodes" :key="n.ID" :label="n.Name" :value="n.ID">
            <span>{{ n.Name }}</span><small>{{ n.rtt }}ms</small>
          </el-option>
        </el-select>
        <el-input v-model="target" class="target" placeholder="下载测速地址" />
        <el-input-number v-model="sizeMB" :min="1" :max="100" controls-position="right" class="num" />
        <span class="unit">MB</span>
        <el-input-number v-model="timeoutSec" :min="3" :max="120" controls-position="right" class="num" />
        <span class="unit">秒</span>
        <el-button :disabled="nodesLoading" @click="selectAll">全选可达</el-button>
        <el-button type="primary" :loading="loading" @click="startTest">开始测速</el-button>
        <el-button v-if="loading" type="warning" @click="abortTest">停止</el-button>
      </div>

      <el-table :data="results" size="small" height="420" v-loading="loading">
        <el-table-column type="index" width="60" label="#" />
        <el-table-column prop="name" label="节点" min-width="220" show-overflow-tooltip />
        <el-table-column label="TCP" width="100">
          <template #default="{ row }">{{ row.rtt >= 0 ? row.rtt + 'ms' : '超时' }}</template>
        </el-table-column>
        <el-table-column label="下载速率" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.ok" :type="rateType(row.mbps)" size="small">{{ row.mbps }} Mbps</el-tag>
            <el-tag v-else-if="row.error" type="danger" size="small">失败</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="流量" width="110">
          <template #default="{ row }">{{ row.ok ? (row.bytes / 1048576).toFixed(2) + ' MB' : '-' }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="90">
          <template #default="{ row }">{{ row.seconds ? row.seconds + 's' : '-' }}</template>
        </el-table-column>
        <el-table-column prop="error" label="备注" min-width="200" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card shadow="never" style="margin-top:16px">
      <template #header>
        <div class="card-header">
          <span>质量时段趋势</span>
          <span class="muted">按小时聚合（近 7 天），用于观察晚高峰</span>
        </div>
      </template>
      <div id="qualityTrend" class="trend-chart"></div>
      <el-empty v-if="!trendLoading && trendPoints.length === 0" description="暂无质量样本" :image-size="50" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from "vue";
import * as echarts from "echarts/core";
import { TooltipComponent, GridComponent, LegendComponent } from "echarts/components";
import { LineChart } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import { getNodeOverview } from "@/api/subcription/node";

defineOptions({ name: "SpeedTest" });

echarts.use([TooltipComponent, GridComponent, LegendComponent, LineChart, CanvasRenderer]);

interface NodeItem { ID: number; Name: string; Link: string; rtt: number }

const nodes = ref<NodeItem[]>([]);
const nodesLoading = ref(true);
const selectedIds = ref<number[]>([]);
const target = ref("https://speed.cloudflare.com/__down?bytes=5000000");
const sizeMB = ref(5);
const timeoutSec = ref(20);
const loading = ref(false);
const results = ref<any[]>([]);
const abortCtrl = ref<AbortController | null>(null);

const trendPoints = ref<any[]>([]);
const trendLoading = ref(false);
let trendChart: any = null;

const rateType = (mbps: number) => (mbps >= 20 ? "success" : mbps >= 5 ? "warning" : "danger");

const getToken = () => {
  try { return localStorage.getItem("accessToken") || ""; } catch { return ""; }
};

const parseSSE = async (res: Response, onEvent: (type: string, data: any) => void) => {
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
      const type = ev.match(/^event:\s*(\S+)/m)?.[1] || "";
      const dataLine = ev.match(/^data:\s*(.*)$/m);
      if (!dataLine) continue;
      try { onEvent(type, JSON.parse(dataLine[1])); } catch { /* ignore */ }
    }
  }
};

const selectAll = () => { selectedIds.value = nodes.value.filter((n) => n.rtt >= 0).map((n) => n.ID); };

const startTest = async () => {
  if (!selectedIds.value.length) { ElMessage.warning("请先选择节点"); return; }
  loading.value = true;
  results.value = [];
  abortCtrl.value = new AbortController();
  const payload = new URLSearchParams();
  payload.append("ids", selectedIds.value.join(","));
  payload.append("target", target.value);
  payload.append("size", String(sizeMB.value * 1048576));
  payload.append("timeout", String(timeoutSec.value));
  try {
    const res = await fetch(import.meta.env.VITE_APP_BASE_API + "/api/v1/nodes/speedtest/stream", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded", Authorization: getToken() },
      body: payload.toString(),
      signal: abortCtrl.value.signal,
    });
    if (!res.ok) {
      const text = await res.text();
      let msg = "请求失败: " + res.status;
      try { msg = JSON.parse(text).msg || msg; } catch { /* ignore */ }
      ElMessage.error(msg);
      return;
    }
    await parseSSE(res, (type, data) => {
      if (type === "node") results.value.push(data);
      else if (type === "done") ElMessage.success("测速完成");
    });
  } catch (e: any) {
    if (e?.name !== "AbortError") ElMessage.error(e?.message || "测速失败");
  } finally {
    loading.value = false;
    abortCtrl.value = null;
  }
};

const abortTest = () => { abortCtrl.value?.abort(); loading.value = false; };

const loadTrend = async () => {
  trendLoading.value = true;
  try {
    const res = await fetch(import.meta.env.VITE_APP_BASE_API + "/api/v1/nodes/quality/trend?hours=168", {
      headers: { Authorization: getToken() },
    });
    const json = await res.json();
    trendPoints.value = json?.data || [];
  } catch {
    trendPoints.value = [];
  } finally {
    trendLoading.value = false;
  }
  await nextTick();
  renderTrend();
};

const renderTrend = () => {
  const el = document.getElementById("qualityTrend") as HTMLDivElement;
  if (!el) return;
  if (!trendChart) trendChart = echarts.init(el);
  const hours = trendPoints.value.map((p) => p.hour);
  const availability = trendPoints.value.map((p) => (p.samples ? Math.round((p.successes / p.samples) * 1000) / 10 : 0));
  const rtt = trendPoints.value.map((p) => Math.round(p.avgRtt || 0));
  trendChart.setOption({
    tooltip: { trigger: "axis" },
    legend: { data: ["可用率(%)", "平均延迟(ms)"] },
    grid: { left: 50, right: 50, top: 40, bottom: 30 },
    xAxis: { type: "category", data: hours, name: "小时" },
    yAxis: [
      { type: "value", name: "可用率%", min: 0, max: 100 },
      { type: "value", name: "ms" },
    ],
    series: [
      { name: "可用率(%)", type: "line", smooth: true, data: availability, areaStyle: { opacity: 0.1 } },
      { name: "平均延迟(ms)", type: "line", smooth: true, yAxisIndex: 1, data: rtt },
    ],
  }, true);
  window.addEventListener("resize", () => trendChart?.resize());
};

onMounted(async () => {
  try {
    const { data } = await getNodeOverview();
    nodes.value = (data || []).filter((n: NodeItem) => n.rtt >= 0).sort((a: NodeItem, b: NodeItem) => a.rtt - b.rtt);
  } catch {
    nodes.value = [];
  } finally {
    nodesLoading.value = false;
  }
  await loadTrend();
});
</script>

<style lang="scss" scoped>
.card-header { display: flex; align-items: baseline; gap: 10px; font-size: 15px; font-weight: 600; }
.card-header .muted { font-size: 12px; font-weight: 400; color: var(--el-text-color-secondary); }
.toolbar { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; margin-bottom: 14px; }
.toolbar .node-select { width: 320px; }
.toolbar .target { width: 320px; }
.toolbar .num { width: 130px; }
.toolbar .unit { color: var(--el-text-color-secondary); font-size: 12px; }
.trend-chart { width: 100%; height: 320px; }
@media (max-width: 720px) {
  .toolbar .node-select, .toolbar .target, .toolbar .num { width: 100%; }
}
</style>
