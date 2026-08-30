<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { cancelTask, getTasks, retryTask } from '@/api/task'

defineOptions({ name:'TaskCenter' })
const tasks = ref<any[]>([])
const loading = ref(false)
let timer:number | undefined

const load = async (quiet = false) => {
  if (!quiet) loading.value = true
  try { const { data } = await getTasks(150); tasks.value = data || [] }
  finally { loading.value = false }
}
const active = computed(() => tasks.value.filter(item => item.status === 'running' || item.status === 'queued').length)
const elapsed = (item:any) => {
  if (!item.startedAt) return '--'
  const end = item.finishedAt ? new Date(item.finishedAt).getTime() : Date.now()
  const sec = Math.max(0, Math.round((end - new Date(item.startedAt).getTime()) / 1000))
  return sec < 60 ? `${sec}s` : `${Math.floor(sec/60)}m ${sec%60}s`
}
const statusType = (status:string) => status === 'success' ? 'success' : status === 'failed' ? 'danger' : status === 'cancelled' ? 'info' : 'warning'
const statusName:Record<string,string> = { queued:'排队中', running:'运行中', success:'成功', failed:'失败', cancelled:'已取消' }
const cancel = async (id:number) => { await cancelTask(id); ElMessage.success('已请求取消'); await load(true) }
const retry = async (id:number) => { await retryTask(id); ElMessage.success('已创建重试任务'); await load(true) }

onMounted(async () => { await load(); timer = window.setInterval(() => load(true), 3000) })
onUnmounted(() => { if (timer) window.clearInterval(timer) })
</script>

<template>
  <div class="task-page">
    <section class="task-hero"><div><span>TASK ORCHESTRATION</span><h1>后台任务中心</h1><p>统一查看节点/分流检测、机场同步、规则同步和后续订阅构建任务。刷新页面不会丢失历史。</p></div><div><b>{{ active }}</b><small>正在运行</small><el-button :loading="loading" @click="load()">刷新</el-button></div></section>
    <section class="task-card">
      <el-table v-loading="loading" :data="tasks" row-key="id" size="small">
        <el-table-column prop="id" label="#" width="72" />
        <el-table-column label="任务" min-width="220"><template #default="{ row }"><div class="task-name"><b>{{ row.name }}</b><small>{{ row.type }}</small></div></template></el-table-column>
        <el-table-column label="状态" width="100"><template #default="{ row }"><el-tag :type="statusType(row.status)" size="small">{{ statusName[row.status] || row.status }}</el-tag></template></el-table-column>
        <el-table-column label="进度" min-width="180"><template #default="{ row }"><el-progress :percentage="row.progress || 0" :status="row.status === 'failed' ? 'exception' : row.status === 'success' ? 'success' : undefined" /><small>{{ row.message }}</small></template></el-table-column>
        <el-table-column label="耗时" width="100"><template #default="{ row }">{{ elapsed(row) }}</template></el-table-column>
        <el-table-column label="错误" min-width="220"><template #default="{ row }"><span class="task-error">{{ row.error || '--' }}</span></template></el-table-column>
        <el-table-column label="操作" width="140" fixed="right"><template #default="{ row }"><el-button v-if="row.status === 'running' || row.status === 'queued'" link type="danger" @click="cancel(row.id)">取消</el-button><el-button v-else link type="primary" @click="retry(row.id)">重试</el-button></template></el-table-column>
      </el-table>
    </section>
  </div>
</template>

<style scoped>
.task-page{padding:28px}.task-hero,.task-card{border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color);box-shadow:var(--el-box-shadow-light)}.task-hero{display:flex;justify-content:space-between;align-items:flex-end;padding:26px;margin-bottom:16px}.task-hero span{color:var(--el-color-primary);font:800 11px ui-monospace;letter-spacing:.14em}.task-hero h1{margin:7px 0;font-size:28px}.task-hero p{margin:0;color:var(--el-text-color-secondary);font-size:12px}.task-hero>div:last-child{display:flex;align-items:center;gap:10px}.task-hero>div:last-child b{font:700 26px ui-monospace}.task-hero>div:last-child small{color:var(--el-text-color-secondary)}.task-card{padding:18px}.task-name{display:flex;flex-direction:column;gap:2px}.task-name small,.task-card td small{color:var(--el-text-color-secondary);font-size:10px}.task-error{color:var(--el-color-danger);font-size:11px;word-break:break-word}@media(max-width:720px){.task-page{padding:14px}.task-hero{align-items:flex-start;flex-direction:column;gap:14px}}
</style>
