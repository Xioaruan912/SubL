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
    <div id="nodeWorldMap" class="world-map"></div>
    <el-empty v-if="!loading && points.length === 0" description="暂无节点" :image-size="60" />
  </el-card>
</template>

<script setup lang="ts">
import * as echarts from "echarts";
import worldJson from "@/assets/world.json";
import { getNodeMap } from "@/api/total";

defineOptions({
  name: "WorldMap",
});

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

const initChart = () => {
  echarts.registerMap("world", worldJson as any);
  const el = document.getElementById("nodeWorldMap") as HTMLDivElement;
  if (!el) return;
  chart.value = markRaw(echarts.init(el));
  window.addEventListener("resize", () => chart.value?.resize());
};

const setOption = () => {
  if (!chart.value) return;
  const mapData = points.value.map((p) => ({
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
            color: "#1890ff",
            shadowBlur: 6,
            shadowColor: "rgba(24, 144, 255, .6)",
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

onMounted(() => {
  initChart();
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

.world-map {
  width: 100%;
  height: 480px;
}
</style>