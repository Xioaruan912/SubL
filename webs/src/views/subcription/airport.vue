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

        <div class="section-title">节点预览（{{ drawerDetail.nodes?.length || 0 }}）</div>
        <div v-if="drawerDetail.nodes?.length" class="node-list">
          <div v-for="n in drawerDetail.nodes" :key="n.ID" class="node-item">
            <span class="node-name" :title="n.Name">{{ n.Name }}</span>
            <span class="node-link" :title="n.Link">{{ (n.Link || '').slice(0, 40) }}</span>
          </div>
        </div>
        <el-empty v-else description="该机场尚无节点，点击「测活&同步」获取" :image-size="50" />
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAirports, getAirportDetail, AddAirport, UpdateAirport, DelAirport, SyncAirport } from '@/api/subcription/airport'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'

const loading = ref(false)
const tableData = ref<any[]>([])

// 详情抽屉
const drawerVisible = ref(false)
const drawerDetail = ref<any>(null)
const drawerLoading = ref(false)

const openDetail = async (row: any) => {
  drawerVisible.value = true
  drawerLoading.value = true
  drawerDetail.value = null
  try {
    const { data } = await getAirportDetail(row.ID)
    drawerDetail.value = data
  } catch {
    ElMessage.error('获取机场详情失败')
  } finally {
    drawerLoading.value = false
  }
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
  ElMessageBox.confirm(`确定删除机场 [${row.Name}] 吗？`, '提示', { type: 'warning' }).then(async () => {
    await DelAirport({ id: row.ID })
    ElMessage.success('删除成功')
    loadData()
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
.detail-body .node-list { display: flex; flex-direction: column; gap: 6px; max-height: 50vh; overflow-y: auto; }
.detail-body .node-item {
  display: flex; align-items: center; gap: 8px; padding: 6px 10px;
  background: var(--el-fill-color-light); border-radius: 8px;
}
.detail-body .node-name { flex-shrink: 0; max-width: 45%; font-weight: 500; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-body .node-link { font-size: 11px; color: var(--el-text-color-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
