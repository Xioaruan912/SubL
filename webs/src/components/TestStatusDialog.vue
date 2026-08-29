<template>
  <el-dialog
    :model-value="visible"
    title="测试状态"
    width="400px"
    append-to-body
    destroy-on-close
    :close-on-click-modal="false"
    @update:model-value="(v: boolean) => emit('update:visible', v)"
  >
    <!-- 有测试进行中 -->
    <template v-if="status">
      <div class="test-item">
        <div class="test-row">
          <span class="dot running" />
          <span class="label">测试进行中</span>
          <el-tag size="small" :type="status.type === 'unlock' ? 'primary' : 'success'" effect="dark">
            {{ status.type === 'unlock' ? '解锁' : 'TCP' }}
          </el-tag>
        </div>
        <div class="test-row">
          <span class="label">节点</span>
          <span class="value">{{ status.nodeName }}</span>
        </div>
        <div class="test-row">
          <span class="label">开始时间</span>
          <span class="value">{{ fmtTime(status.startedAt) }}</span>
        </div>
        <div class="test-row">
          <span class="label">已进行</span>
          <span class="value">{{ elapsed }}</span>
        </div>
      </div>
    </template>
    <!-- 空闲 -->
    <template v-else>
      <el-empty description="当前无测试进行中" :image-size="60" />
    </template>

    <template #footer>
      <el-button @click="emit('update:visible', false)">关闭</el-button>
      <el-button v-if="status" type="danger" :loading="canceling" @click="cancel">停止测试</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from "vue";
import { GetTestStatus, CancelTest } from "@/api/subcription/node";

defineProps<{ visible: boolean }>();
const emit = defineEmits<{ (e: "update:visible", v: boolean): void }>();

interface TestStatus {
  nodeName: string;
  nodeId: number;
  type: string;
  startedAt: string;
}

const status = ref<TestStatus | null>(null);
const canceling = ref(false);
const now = ref(Date.now());

const fmtTime = (s: string) => {
  if (!s) return "-";
  const d = new Date(s);
  return d.toLocaleString();
};

const elapsed = ref("");

const refresh = async () => {
  try {
    const { data } = await GetTestStatus();
    status.value = data || null;
    if (status.value) {
      const start = new Date(status.value.startedAt).getTime();
      const sec = Math.floor((Date.now() - start) / 1000);
      elapsed.value = sec < 60 ? sec + " 秒" : Math.floor(sec / 60) + " 分 " + (sec % 60) + " 秒";
    }
  } catch { status.value = null; }
};

const cancel = async () => {
  canceling.value = true;
  try {
    await CancelTest();
    ElMessage.success("已停止测试");
    status.value = null;
  } catch (e: any) {
    ElMessage.error(e?.message || "停止失败");
  } finally {
    canceling.value = false;
  }
};

let timer: any = null;
onMounted(() => { refresh(); timer = setInterval(refresh, 3000); });
onBeforeUnmount(() => { if (timer) clearInterval(timer); });
</script>

<style scoped>
.test-item { padding: 4px 0; }
.test-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 14px;
}
.dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.dot.running { background: #30a46c; animation: pulse 1.5s infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.3; } }
.label { color: var(--el-text-color-secondary); width: 70px; flex-shrink: 0; }
.value { font-weight: 500; word-break: break-all; }
</style>
