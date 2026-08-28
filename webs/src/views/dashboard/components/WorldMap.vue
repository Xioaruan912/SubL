<template>
  <el-card shadow="never">
    <template #header>
      <div class="title">
        <span class="map-title">
          <svg-icon icon-class="monitor" class="mr-1" />
          节点世界地图
        </span>
        <div class="actions">
          <el-tooltip content="刷新节点地图" placement="top">
            <el-button text circle :loading="loading" @click="load">
              <i-ep-refresh class="refresh" />
            </el-button>
          </el-tooltip>
        </div>
      </div>
    </template>
    <div v-loading="loading" class="map-loading-area">
      <div id="nodeWorldMap" class="world-map"></div>
      <el-empty v-if="!loading && points.length === 0" description="暂无节点" :image-size="60" />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import * as echarts from "echarts/core";
import { TooltipComponent, GeoComponent } from "echarts/components";
import { EffectScatterChart } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import { getNodeMap } from "@/api/total";

defineOptions({
  name: "WorldMap",
});

echarts.use([TooltipComponent, GeoComponent, EffectScatterChart, CanvasRenderer]);

interface MapPoint {
  name: string;
  server: string;
  country: string;
  countryCode: string;
  lat: number;
  lng: number;
}

const loading = ref(false);
const points = ref<MapPoint[]>([]);
const chart = ref<any>(null);
const worldJson = ref<any>(null);

// 动态加载世界地图数据（避免打进首屏 bundle）
const loadWorldJson = async () => {
  if (worldJson.value) return worldJson.value;
  try {
    const res = await fetch("/static/world.json");
    worldJson.value = await res.json();
  } catch {
    worldJson.value = null;
  }
  return worldJson.value;
};

const initChart = async () => {
  const geoJson = await loadWorldJson();
  const el = document.getElementById("nodeWorldMap") as HTMLDivElement;
  if (!el) return;
  if (geoJson) {
    echarts.registerMap("world", geoJson);
  }
  chart.value = markRaw(echarts.init(el));
  window.addEventListener("resize", () => chart.value?.resize());
};

// 同坐标（同国家）多个节点环形分散，避免重叠只显示一个
const scatterPoints = () => {
  const groups = new Map<string, MapPoint[]>();
  for (const p of points.value) {
    const key = `${p.lat.toFixed(3)},${p.lng.toFixed(3)}`;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key)!.push(p);
  }
  const out: MapPoint[] = [];
  for (const arr of groups.values()) {
    if (arr.length === 1) {
      out.push(arr[0]);
      continue;
    }
    // 环形分散：半径 0.9 度，按索引均匀分布角度
    arr.forEach((p, i) => {
      const angle = (i / arr.length) * Math.PI * 2;
      out.push({
        ...p,
        lat: p.lat + Math.sin(angle) * 0.9,
        lng: p.lng + Math.cos(angle) * 0.9,
      });
    });
  }
  return out;
};

const setOption = () => {
  if (!chart.value) return;
  const mapData = scatterPoints().map((p) => ({
    name: p.name,
    value: [p.lng, p.lat, 1],
    country: p.country,
    countryCode: p.countryCode,
    server: p.server,
  }));
  chart.value.setOption(
    {
      backgroundColor: "transparent",
      tooltip: {
        trigger: "item",
        formatter: (params: any) => {
          if (params.seriesType === "effectScatter") {
            const d = params.data;
            return `<b>${d.name}</b><br/>国家: ${d.country || "-"} (${d.countryCode || "未知"})<br/>服务器: ${d.server}`;
          }
          return params.name;
        },
      },
      geo: {
        map: "world",
        roam: true,
        scaleLimit: { min: 0.9, max: 8 },
        label: { show: false },
        itemStyle: {
          areaColor: "#e8f2ff",
          borderColor: "#b8d6ff",
          borderWidth: 0.5,
        },
        emphasis: {
          itemStyle: { areaColor: "#d1e6ff" },
          label: { show: false },
        },
        select: { disabled: true },
      },
      series: [
        {
          name: "节点",
          type: "effectScatter",
          coordinateSystem: "geo",
          zlevel: 2,
          rippleEffect: { brushType: "stroke", scale: 3 },
          symbolSize: 8,
          itemStyle: {
            color: "#f97316",
            shadowBlur: 6,
            shadowColor: "rgba(249, 115, 22, .6)",
          },
          data: mapData,
        },
      ],
    },
    true
  );
};

const load = async () => {
  loading.value = true;
  try {
    const { data } = await getNodeMap();
    points.value = data || [];
  } catch {
    points.value = [];
  } finally {
    loading.value = false;
    setOption();
  }
};

onMounted(async () => {
  await initChart();
  load();
});

onActivated(() => {
  chart.value?.resize();
});

onBeforeUnmount(() => {
  chart.value?.dispose();
});
</script>

<style lang="scss" scoped>
.title {
  display: flex;
  align-items: center;
  justify-content: space-between;

  .map-title {
    display: flex;
    align-items: center;
    font-size: 15px;
    font-weight: 600;
  }

  .refresh {
    font-size: 16px;
  }
}

.map-loading-area { position: relative; min-height: 480px; }

.world-map {
  width: 100%;
  height: 480px;
}
</style>