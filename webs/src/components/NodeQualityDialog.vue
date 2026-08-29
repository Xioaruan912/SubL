<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { getNodeQualityHistory } from "@/api/subcription/node";

const props = defineProps<{
  visible: boolean;
  node: any | null;
}>();
const emit = defineEmits(["update:visible"]);
const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit("update:visible", value),
});
const loading = ref(false);
const samples = ref<any[]>([]);

const load = async () => {
  if (!props.node?.id) return;
  loading.value = true;
  try {
    const response: any = await getNodeQualityHistory(props.node.id, 24);
    samples.value = response?.data || [];
  } finally {
    loading.value = false;
  }
};

watch(() => props.visible, (value) => { if (value) load(); });

const points = computed(() => {
  const values = samples.value.slice(-40);
  if (!values.length) return "";
  const width = 560, height = 110, maxRtt = Math.max(500, ...values.filter(x => x.success).map(x => x.rtt));
  return values.map((item, index) => {
    const x = values.length === 1 ? width / 2 : index * width / (values.length - 1);
    const value = item.success ? item.rtt : maxRtt;
    const y = height - Math.min(value, maxRtt) / maxRtt * (height - 12) - 6;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(" ");
});

const scoreType = computed(() => props.node?.score >= 80 ? "success" : props.node?.score >= 60 ? "warning" : "danger");
const formatTime = (value: string) => value ? new Date(value).toLocaleString() : "--";
</script>

<template>
  <el-dialog v-model="dialogVisible" :title="`${node?.name || ''} · 质量详情`" width="680px" align-center>
    <div v-if="node" v-loading="loading">
      <div class="quality-metrics">
        <div class="metric score"><span>综合评分</span><strong>{{ node.score || 0 }}</strong><el-tag :type="scoreType" size="small">{{ node.score >= 80 ? '优秀' : node.score >= 60 ? '一般' : '需关注' }}</el-tag></div>
        <div class="metric"><span>24h 可用率</span><strong>{{ node.availability ?? 0 }}%</strong></div>
        <div class="metric"><span>平均延迟</span><strong>{{ node.averageRtt >= 0 ? `${node.averageRtt}ms` : '--' }}</strong></div>
        <div class="metric"><span>延迟抖动</span><strong>{{ node.jitter ?? 0 }}ms</strong></div>
      </div>
      <div class="trend-card">
        <div class="trend-title"><span>最近 24 小时服务端检测</span><small>{{ samples.length }} 个样本</small></div>
        <svg v-if="points" viewBox="0 0 560 110" preserveAspectRatio="none" class="trend-chart">
          <defs><linearGradient id="qualityFill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#409eff" stop-opacity=".3"/><stop offset="1" stop-color="#409eff" stop-opacity="0"/></linearGradient></defs>
          <polyline :points="points" fill="none" stroke="#409eff" stroke-width="3" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
        </svg>
        <el-empty v-else description="质量样本正在积累，约 10 分钟后可看到趋势" :image-size="64" />
      </div>
      <el-table :data="samples.slice(-10).reverse()" size="small" max-height="260">
        <el-table-column label="检测时间" min-width="170"><template #default="{ row }">{{ formatTime(row.checkedAt) }}</template></el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.success ? '正常' : '超时' }}</el-tag></template></el-table-column>
        <el-table-column label="延迟" width="100"><template #default="{ row }">{{ row.rtt >= 0 ? `${row.rtt}ms` : '--' }}</template></el-table-column>
      </el-table>
    </div>
  </el-dialog>
</template>

<style scoped>
.quality-metrics { display:grid; grid-template-columns:repeat(4,1fr); gap:10px; margin-bottom:16px; }
.metric { padding:14px; border-radius:12px; background:var(--el-fill-color-light); display:flex; flex-direction:column; gap:5px; }
.metric span { font-size:12px; color:var(--el-text-color-secondary); }
.metric strong { font-size:21px; color:var(--el-text-color-primary); }
.metric.score { background:var(--el-color-primary-light-9); }
.trend-card { padding:14px; margin-bottom:16px; border:1px solid var(--el-border-color-lighter); border-radius:12px; }
.trend-title { display:flex; justify-content:space-between; margin-bottom:8px; font-weight:600; }
.trend-title small { color:var(--el-text-color-secondary); font-weight:400; }
.trend-chart { width:100%; height:110px; border-bottom:1px solid var(--el-border-color-lighter); }
@media (max-width:640px) { .quality-metrics { grid-template-columns:repeat(2,1fr); } }
</style>
