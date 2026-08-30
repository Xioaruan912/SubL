<template>
  <div class="p-4 md:p-6 lg:p-8 h-full flex flex-col gap-6">
    <div class="bg-white dark:bg-[#1a1d1b] rounded-xl shadow-[inset_0_0_0_1px_rgba(0,0,0,0.06)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)] flex flex-col overflow-hidden min-h-[500px]">
      
      <!-- Toolbar -->
      <div class="p-5 border-b border-gray-100 dark:border-white/5 flex gap-3 items-center justify-between">
        <div class="text-lg font-medium text-gray-800 dark:text-gray-200">机场数据源管理</div>
        <el-button type="primary" @click="handleAdd"><el-icon class="mr-1"><Plus /></el-icon>添加机场</el-button>
      </div>

      <!-- Table -->
      <div class="p-5 bg-gray-50/30 dark:bg-black/10 flex-1">
        <el-table :data="tableData" v-loading="loading" class="w-full">
          <el-table-column prop="ID" label="ID" width="80" />
          <el-table-column prop="Name" label="机场名称" min-width="150">
            <template #default="{ row }">
              <span class="font-medium text-gray-800 dark:text-gray-200">{{ row.Name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="URL" label="订阅链接" min-width="250">
            <template #default="{ row }">
              <div class="truncate text-xs text-gray-500" :title="row.URL">{{ row.URL }}</div>
            </template>
          </el-table-column>
          <el-table-column label="自动清理死节点" width="130" align="center">
            <template #default="{ row }">
              <el-tag :type="row.AutoCleanup ? 'success' : 'info'" size="small">{{ row.AutoCleanup ? '是' : '否' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="专线(测活免死)" width="130" align="center">
            <template #default="{ row }">
              <el-tag :type="row.IsDedicated ? 'warning' : 'info'" size="small">{{ row.IsDedicated ? '专线免死' : '常规丢弃' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="NodeCount" label="存活节点" width="100" align="center" />
          <el-table-column label="上次同步" width="160">
            <template #default="{ row }">
              <span class="text-xs text-gray-500">{{ row.LastSync ? new Date(row.LastSync).toLocaleString() : '从未同步' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="280" fixed="right" align="center">
            <template #default="{ row }">
              <el-button link type="success" size="small" @click="handleSync(row)">测活&同步</el-button>
              <el-button link type="primary" size="small" @click="openDetail(row)">详情</el-button>
              <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button link type="danger" size="small" @click="handleDel(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- Form Dialog -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px" align-center>
      <el-form label-position="top">
        <el-form-item label="机场名称">
          <el-input v-model="form.Name" placeholder="例如：Nexitally" />
          <div class="text-xs text-gray-400 mt-1">同步后，所有节点将自动归类到此名称的分组中。</div>
        </el-form-item>
        <el-form-item label="机场订阅链接">
          <el-input v-model="form.URL" placeholder="https://..." />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="自动清理死节点">
              <el-switch v-model="form.AutoCleanup" />
              <div class="text-xs text-gray-400 mt-1">每次同步时进行TCP测活，丢弃连不上的节点。</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="是否为国内专线">
              <el-switch v-model="form.IsDedicated" />
              <div class="text-xs text-gray-400 mt-1">勾选此项，则测活失败时也会强制保留该节点。</div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">{{ dialogTitle === '添加机场' ? '添加' : '保存' }}</el-button>
      </template>
    </el-dialog>

    <!-- 详情抽屉 -->
    <el-drawer v-model="drawerVisible" :title="drawerDetail?.name || '机场详情'" size="560px" v-loading="drawerLoading">
      <div v-if="drawerDetail" class="detail-body">
        <div class="info-grid">
          <div class="info-item"><span class="info-label">订阅链接</span><span class="info-val truncate" :title="drawerDetail.url">{{ drawerDetail.url }}</span></div>
          <div class="info-item"><span class="info-label">清理死节点</span><span class="info-val">{{ drawerDetail.auto_cleanup ? '是' : '否' }}</span></div>
          <div class="info-item"><span class="info-label">专线免死</span><span class="info-val">{{ drawerDetail.is_dedicated ? '是' : '否' }}</span></div>
          <div class="info-item"><span class="info-label">上次同步</span><span class="info-val">{{ drawerDetail.last_sync ? new Date(drawerDetail.last_sync).toLocaleString() : '从未同步' }}</span></div>
          <div class="info-item"><span class="info-label">节点总数</span><span class="info-val">{{ drawerDetail.node_count }}</span></div>
          <div class="info-item"><span class="info-label">当前存活</span><span class="info-val">{{ drawerDetail.nodes?.length || 0 }}</span></div>
        </div>

        <div class="section-title node-select-head">
          <span>节点选择（已选 {{ selectedNodeNames.length }} / 共 {{ drawerDetail.nodes?.length || 0 }}）</span>
          <div class="node-select-actions">
            <el-button link type="primary" size="small" @click="toggleAllNodes">
              {{ selectedNodeNames.length === (drawerDetail.nodes?.length || 0) && drawerDetail.nodes?.length ? '取消全选' : '全选' }}
            </el-button>
          </div>
        </div>
        <div v-if="drawerDetail.nodes?.length" class="node-check-list">
          <el-checkbox
            v-for="n in drawerDetail.nodes"
            :key="n.ID"
            :model-value="selectedNodeNames.includes(n.Name)"
            :value="n.Name"
            class="node-check-item"
            @update:model-value="(v: any) => {
              if (v) { if (!selectedNodeNames.includes(n.Name)) selectedNodeNames.push(n.Name) }
              else { selectedNodeNames = selectedNodeNames.filter(x => x !== n.Name) }
            }"
          >
            <div class="node-cell">
              <div class="node-cell-main">
                <span class="node-name" :title="n.Name">{{ n.Name }}</span>
                <span class="node-link" :title="n.Link">{{ (n.Link || '').slice(0, 36) }}</span>
              </div>
              <div class="node-cell-meta">
                <span v-if="nodeRtt(n.Name) !== -2" class="node-rtt" :style="{ background: rttColor(nodeRtt(n.Name)), color: '#fff' }">{{ rttText(nodeRtt(n.Name)) }}</span>
                <span v-else class="node-rtt node-rtt-loading">…</span>
                <el-tag v-if="geminiBadge(n.Name)" :type="geminiBadge(n.Name)!.type" size="small" effect="light" :class="geminiBadge(n.Name)!.cls">
                  {{ geminiBadge(n.Name)!.text }}
                </el-tag>
                <el-button link type="primary" size="small" @click.stop.prevent="testGemini(n)" :disabled="!!geminiTesting">测AI</el-button>
              </div>
            </div>
          </el-checkbox>
        </div>
        <el-empty v-else description="该机场尚无节点，点击「测活&同步」获取" :image-size="50" />
        <div class="node-select-foot">
          <span class="text-xs text-gray-400">未勾选任何节点时，订阅引用该机场将默认全量导入。</span>
          <el-button type="primary" size="small" :loading="savingNodes" @click="saveNodeSelection">保存选择</el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getAirports, getAirportDetail, selectAirportNodes, AddAirport, UpdateAirport, DelAirport, SyncAirport } from '@/api/subcription/airport'
import { getNodeOverview } from '@/api/subcription/node'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'

const loading = ref(false)
const tableData = ref<any[]>([])

// 详情抽屉
const drawerVisible = ref(false)
const drawerDetail = ref<any>(null)
const drawerLoading = ref(false)
const selectedNodeNames = ref<string[]>([])
const savingNodes = ref(false)
const rttMap = ref<Record<string, number>>({}) // 节点名 -> 延迟
const geminiStatus = ref<Record<string, string>>({}) // 节点名 -> idle/testing/ok/fail
const geminiTesting = ref('') // 当前测试中的节点名

// 延迟徽标
const rttColor = (rtt: number) => {
  if (rtt === -2) return '#95a5a6'
  if (rtt < 0) return '#95a5a6'
  if (rtt < 100) return '#2ecc71'
  if (rtt < 300) return '#f1c40f'
  return '#e74c3c'
}
const rttText = (rtt: number) => rtt === -2 ? '测试中…' : (rtt < 0 ? '超时' : rtt + 'ms')
const nodeRtt = (name: string) => {
  const v = rttMap.value[name]
  return v === undefined ? -2 : v // -2 表示未知(加载中)
}

const openDetail = async (row: any) => {
  drawerVisible.value = true
  drawerLoading.value = true
  drawerDetail.value = null
  selectedNodeNames.value = []
  geminiStatus.value = {}
  geminiTesting.value = ''
  try {
    const { data } = await getAirportDetail(row.ID)
    drawerDetail.value = data
    selectedNodeNames.value = data?.selected_nodes || []
    // 并行拉取节点概览（含延迟）
    loadNodeRtt()
  } catch {
    ElMessage.error('获取机场详情失败')
  } finally {
    drawerLoading.value = false
  }
}

import { testLocalAll } from "@/utils/ping"

const loadNodeRtt = async () => {
  try {
    const { data } = await getNodeOverview()
    const nodes = data || []
    const map: Record<string, number> = {}
    for (const n of nodes) {
      n.rtt = n.rtt === -1 ? -1 : -2;
      map[n.name] = n.rtt
    }
    rttMap.value = map

    testLocalAll(nodes, (index, rtt) => {
      const node = nodes[index];
      if (node && rttMap.value[node.name] !== undefined) {
        rttMap.value[node.name] = rtt;
      }
    })
  } catch { /* ignore */ }
}

// 测试某节点的 Gemini 解锁（走解锁接口，只测 google-gemini）
const testGemini = async (node: any) => {
  if (geminiTesting.value) {
    ElMessage.warning(`已有节点测试中（${geminiTesting.value}），请稍候`)
    return
  }
  geminiTesting.value = node.Name
  geminiStatus.value[node.Name] = 'testing'
  try {
    const token = localStorage.getItem('accessToken') || ''
    const payload = new URLSearchParams()
    payload.append('link', node.Link)
    payload.append('service', 'google-gemini')
    const res = await fetch('/api/v1/nodes/unlock', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', Authorization: token },
      body: payload.toString(),
    })
    const json = await res.json()
    if (json?.code === '00000') {
      const r = (json.data?.results || []).find((x: any) => x.key === 'google-gemini')
      geminiStatus.value[node.Name] = r?.ok ? 'ok' : 'fail'
    } else if (json?.code === '42900') {
      geminiStatus.value[node.Name] = 'idle'
      ElMessage.warning(json.msg || '已有测试进行中')
    } else {
      geminiStatus.value[node.Name] = 'fail'
      ElMessage.error(json?.msg || '测试失败')
    }
  } catch {
    geminiStatus.value[node.Name] = 'fail'
  } finally {
    geminiTesting.value = ''
  }
}

const geminiBadge = (name: string) => {
  const s = geminiStatus.value[name]
  if (s === 'testing') return { type: 'info' as const, text: '测试中', cls: 'gemini-testing' }
  if (s === 'ok') return { type: 'success' as const, text: 'AI ✓', cls: '' }
  if (s === 'fail') return { type: 'danger' as const, text: 'AI ✗', cls: '' }
  return null
}

const saveNodeSelection = async () => {
  const d = drawerDetail.value
  if (!d) return
  savingNodes.value = true
  try {
    await selectAirportNodes({ id: d.id, nodes: selectedNodeNames.value.join(',') })
    ElMessage.success('节点选择已保存')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.msg || '保存失败')
  } finally {
    savingNodes.value = false
  }
}

const toggleAllNodes = () => {
  const d = drawerDetail.value
  if (!d?.nodes?.length) return
  selectedNodeNames.value = selectedNodeNames.value.length === d.nodes.length
    ? []
    : d.nodes.map((n: any) => n.Name)
}

const dialogVisible = ref(false)
const dialogTitle = ref('添加机场')
const form = ref({
  ID: 0,
  Name: '',
  URL: '',
  AutoCleanup: true,
  IsDedicated: false
})

const loadData = async () => {
  loading.value = true
  try {
    const { data } = await getAirports()
    tableData.value = data || []
  } finally {
    loading.value = false
  }
}

const handleAdd = () => {
  dialogTitle.value = '添加机场'
  form.value = { ID: 0, Name: '', URL: '', AutoCleanup: true, IsDedicated: false }
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  dialogTitle.value = '编辑机场'
  form.value = { ...row }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (!form.value.Name || !form.value.URL) {
    ElMessage.warning('名称和URL不能为空')
    return
  }
  try {
    if (dialogTitle.value === '添加机场') {
      await AddAirport(form.value)
      ElMessage.success('添加成功')
    } else {
      await UpdateAirport(form.value)
      ElMessage.success('修改成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.msg || '操作失败')
  }
}

const handleDel = (row: any) => {
  ElMessageBox.confirm(
    `确定删除机场 [${row.Name}] 吗？\n是否同时删除该机场所属的所有节点？`,
    '确认删除',
    {
      distinguishCancelAndClose: true,
      confirmButtonText: '删除机场和节点',
      cancelButtonText: '仅删除机场',
      type: 'warning'
    }
  ).then(async () => {
    await DelAirport({ id: row.ID, delete_nodes: true })
    ElMessage.success('机场及所属节点已删除')
    loadData()
  }).catch(async (action) => {
    if (action === 'cancel') {
      await DelAirport({ id: row.ID, delete_nodes: false })
      ElMessage.success('仅删除机场成功，节点已保留')
      loadData()
    }
  })
}

const handleSync = async (row: any) => {
  ElMessage.info(`后台已开始同步测活 [${row.Name}]，这可能需要几十秒。请稍后刷新页面查看最新状态。`)
  try {
    await SyncAirport({ id: row.ID })
  } catch (err: any) {
    ElMessage.error(err.response?.data?.msg || '同步请求失败')
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.detail-body .info-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 10px 16px; margin-bottom: 18px;
}
.detail-body .info-item {
  display: flex; flex-direction: column; gap: 2px;
  padding: 8px 10px; background: var(--el-fill-color-light); border-radius: 8px; min-width: 0;
}
.detail-body .info-label { font-size: 12px; color: var(--el-text-color-secondary); }
.detail-body .info-val { font-size: 13px; font-weight: 500; color: var(--el-text-color-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-body .section-title { font-weight: 600; margin: 14px 0 8px; color: var(--el-text-color-primary); }
.detail-body .node-select-head { display: flex; align-items: center; justify-content: space-between; }
.detail-body .node-select-actions { font-weight: 400; }
.detail-body .node-check-list { display: flex; flex-direction: column; gap: 6px; max-height: 45vh; overflow-y: auto; }
.detail-body .node-check-item {
  display: flex; align-items: center; gap: 8px; padding: 6px 10px;
  background: var(--el-fill-color-light); border-radius: 8px; margin-right: 0; width: 100%;
}
.detail-body .node-check-item:hover { background: var(--el-fill-color); }
.detail-body .node-cell { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; }
.detail-body .node-cell-main { display: flex; flex-direction: column; gap: 1px; flex: 1; min-width: 0; }
.detail-body .node-name { font-weight: 500; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-body .node-link { font-size: 11px; color: var(--el-text-color-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-body .node-cell-meta { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.detail-body .node-rtt { font-size: 11px; padding: 1px 6px; border-radius: 6px; font-weight: 600; }
.detail-body .node-rtt-loading { background: var(--el-fill-color); color: var(--el-text-color-secondary); }
.gemini-testing { animation: geminiPulse 1s ease-in-out infinite; }
@keyframes geminiPulse { 0%, 100% { opacity: 0.4; } 50% { opacity: 1; } }
.detail-body .node-select-foot { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-top: 12px; }
@media (max-width: 720px) {
  :deep(.el-dialog .el-col) { max-width: 100%; flex: 0 0 100%; }
  .detail-body .info-grid { grid-template-columns: 1fr; gap: 8px; }
  .detail-body .node-select-head,
  .detail-body .node-select-foot { align-items: flex-start; flex-direction: column; }
  .detail-body .node-cell { align-items: flex-start; }
  .detail-body .node-cell-meta { align-items: flex-end; flex-direction: column; gap: 3px; }
  .detail-body .node-select-foot :deep(.el-button) { width: 100%; margin-left: 0; }
}
</style>
