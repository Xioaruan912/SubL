<template>
  <el-card shadow="never">
    <template #header>
      <div class="title">
        <span class="ping-title">
          <svg-icon icon-class="link" class="mr-1" />
          节点延迟
        </span>
        <div class="actions">
          <el-button text circle :loading="loading" @click="load">
            <i-ep-refresh class="refresh" />
          </el-button>
        </div>
      </div>
    </template>

    <!-- 常见目标延迟 -->
    <div class="section" v-loading="loading">
      <div class="section-title">常见目标延迟（VPS 出口）</div>
      <div class="target-list">
        <div v-for="t in targets" :key="t.name" class="target-item">
          <span class="target-name">{{ t.name }}</span>
          <el-tag :type="tagType(t.rtt)" size="small" effect="light">
            {{ t.rtt < 0 ? "--" : t.rtt + " ms" }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- 节点延迟 -->
    <div class="section" v-loading="loading">
      <div class="section-title">
        当前节点延迟
        <span class="node-count">({{ nodes.length }} 个节点)</span>
      </div>
      <el-scrollbar height="260px" v-if="nodes.length > 0">
        <div class="node-list">
          <div v-for="n in sortedNodes" :key="n.name + n.server" class="node-item">
            <span class="node-name" :title="n.name">{{ n.name }}</span>
            <span class="node-server">{{ n.server }}</span>
            <el-tag :type="tagType(n.rtt)" size="small" effect="light">
              {{ n.rtt < 0 ? "超时" : n.rtt + " ms" }}
            </el-tag>
          </div>
        </div>
      </el-scrollbar>
      <el-empty v-else-if="!loading" description="暂无节点" :image-size="60" />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { getNodePing } from "@/api/total";

defineOptions({
  name: "NodePing",
});

interface PingTarget {
  name: string;
  addr: string;
  rtt: number;
}
interface NodePingItem {
  name: string;
  server: string;
  rtt: number;
}

const loading = ref(false);
const targets = ref<PingTarget[]>([]);
const nodes = ref<NodePingItem[]>([]);

const sortedNodes = computed(() =>
  [...nodes.value].sort((a, b) => {
    const av = a.rtt < 0 ? Number.MAX_SAFE_INTEGER : a.rtt;
    const bv = b.rtt < 0 ? Number.MAX_SAFE_INTEGER : b.rtt;
    return av - bv;
  })
);

const tagType = (rtt: number) => {
  if (rtt < 0) return "danger";
  if (rtt < 100) return "success";
  if (rtt < 300) return "warning";
  return "danger";
};

const load = async () => {
  loading.value = true;
  try {
    const { data } = await getNodePing();
    targets.value = data?.targets || [];
    nodes.value = data?.nodes || [];
  } catch {
    targets.value = [];
    nodes.value = [];
  } finally {
    loading.value = false;
  }
};

onMounted(load);
</script>

<style lang="scss" scoped>
.title {
  display: flex;
  align-items: center;
  justify-content: space-between;

  .ping-title {
    display: flex;
    align-items: center;
    font-size: 15px;
    font-weight: 600;
  }

  .refresh {
    font-size: 16px;
  }
}

.section {
  margin-bottom: 16px;

  .section-title {
    margin-bottom: 10px;
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-secondary);

    .node-count {
      font-weight: 400;
      color: var(--el-text-color-placeholder);
    }
  }
}

.target-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;

  .target-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    background: #f8f9fa;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    .target-name {
      font-size: 13px;
      font-weight: 500;
    }
  }
}

.node-list {
  .node-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 10px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    .node-name {
      flex: 0 0 180px;
      overflow: hidden;
      font-size: 13px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .node-server {
      flex: 1;
      overflow: hidden;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
}

html.dark .target-item {
  background: #202425;
}
</style>