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
      <div class="section-title">常见目标延迟</div>
      <div class="target-list">
        <div v-for="t in targets" :key="t.name" class="target-item">
          <span class="target-name">{{ t.name }}</span>
          <el-tag :type="tagType(t.rtt)" size="small" effect="light">
            {{ t.rtt === -2 ? "测试中…" : (t.rtt < 0 ? "超时" : t.rtt + " ms") }}
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
              {{ n.rtt === -2 ? "测试中…" : (n.rtt < 0 ? "超时" : n.rtt + " ms") }}
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
  if (rtt === -2) return "info";
  if (rtt < 0) return "danger";
  if (rtt < 100) return "success";
  if (rtt < 300) return "warning";
  return "danger";
};

import { testLocalAll } from "@/utils/ping"

const load = async () => {
  loading.value = true;
  try {
    const { data } = await getNodePing();
    const ts = data?.targets || [];
    ts.forEach((t: any) => {
      t.rtt = t.rtt === -1 ? -1 : -2;
    });
    targets.value = ts;
    
    const ns = data?.nodes || [];
    ns.forEach((n: any) => {
      n.rtt = n.rtt === -1 ? -1 : -2;
    });
    nodes.value = ns;

    triggerDashboardPings();
  } catch {
    targets.value = [];
    nodes.value = [];
  } finally {
    loading.value = false;
  }
};

const triggerDashboardPings = () => {
  const targetsToTest = targets.value.map(t => {
    const parts = t.addr.split(':');
    return {
      server: parts[0],
      port: parseInt(parts[1]) || 443,
      rtt: t.rtt
    };
  });
  testLocalAll(targetsToTest, (index, rtt) => {
    if (targets.value[index]) {
      targets.value[index].rtt = rtt;
    }
  });

  const nodesToTest = nodes.value.map(n => {
    const parts = n.server.split(':');
    return {
      server: parts[0],
      port: parseInt(parts[1]) || 443,
      rtt: n.rtt
    };
  });
  testLocalAll(nodesToTest, (index, rtt) => {
    if (nodes.value[index]) {
      nodes.value[index].rtt = rtt;
    }
  });
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