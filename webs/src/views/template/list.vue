<script setup lang='ts'>
import { ref, computed, onMounted, nextTick } from 'vue'
import { getTemp, AddTemp, UpdateTemp, DelTemp } from "@/api/template/temp"

interface Temp {
  file: string;
  text: string;
  create_date: string;
}

const tableData = ref<Temp[]>([])
const searchKey = ref('')
const Tempoldname = ref('')
const Tempname = ref('')
const TempText = ref('')
const dialogVisible = ref(false)
const TempTitle = ref('')
const locateKey = ref('')
const templateInputRef = ref<any>(null)

// 类型识别：yaml→clash / conf→loon / 其他→generic
const tempType = (file: string) => {
  const f = file.toLowerCase()
  if (f.endsWith('.yaml') || f.endsWith('.yml')) return { label: 'Clash', type: 'primary' as const }
  if (f.endsWith('.conf')) return { label: 'Loon', type: 'warning' as const }
  return { label: 'Generic', type: 'info' as const }
}

const filteredData = computed(() => {
  const kw = searchKey.value.trim().toLowerCase()
  if (!kw) return tableData.value
  return tableData.value.filter(t => t.file.toLowerCase().includes(kw))
})

const fileSize = (text: string) => {
  const bytes = new Blob([text || ""]).size
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / 1024 / 1024).toFixed(2)}MB`
}

const loading = ref(false)

async function gettemps() {
  loading.value = true
  try {
    const { data } = await getTemp()
    tableData.value = data
  } finally {
    loading.value = false
  }
}
onMounted(gettemps)

const handleAddTemp = () => {
  TempTitle.value = '添加模板'
  Tempname.value = ''
  TempText.value = ''
  locateKey.value = ''
  dialogVisible.value = true
}
const addtemp = async () => {
  if (!Tempname.value.trim()) { ElMessage.warning('请填写文件名'); return }
  if (TempTitle.value === '添加模板') {
    await AddTemp({ filename: Tempname.value.trim(), text: TempText.value.trim() })
    ElMessage.success("添加成功")
  } else {
    await UpdateTemp({ filename: Tempname.value.trim(), oldname: Tempoldname.value.trim(), text: TempText.value.trim() })
    ElMessage.success("更新成功")
  }
  gettemps()
  Tempname.value = ''
  TempText.value = ''
  dialogVisible.value = false
}

const handleEdit = (row: Temp) => {
  TempTitle.value = '编辑模板'
  Tempname.value = row.file
  Tempoldname.value = row.file
  TempText.value = row.text
  locateKey.value = ''
  dialogVisible.value = true
}

const locateInTemplate = () => {
  const key = locateKey.value.trim()
  if (!key) return
  const index = TempText.value.indexOf(key)
  if (index < 0) { ElMessage.info('没有找到该内容'); return }
  const textarea = templateInputRef.value?.textarea as HTMLTextAreaElement | undefined
  if (textarea) { textarea.focus(); textarea.setSelectionRange(index, index + key.length); textarea.scrollTop = Math.max(0, (TempText.value.slice(0, index).match(/\n/g) || []).length * 20 - 120) }
}

const handleDel = (row: Temp) => {
  ElMessageBox.confirm(`你是否要删除 ${row.file} ?`, '提示', {
    confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning',
  }).then(async () => {
    await DelTemp({ filename: row.file })
    ElMessage.success('删除成功')
    gettemps()
  })
}

const copyText = async (row: Temp) => {
  try {
    await navigator.clipboard.writeText(row.text)
    ElMessage.success('内容已复制')
  } catch {
    ElMessage.warning('复制失败')
  }
}
</script>

<template>
  <div class="tpl-page">
    <!-- 顶部工具栏 -->
    <div class="toolbar">
      <el-input v-model="searchKey" placeholder="搜索模板文件名…" clearable class="search" />
      <el-button type="primary" @click="handleAddTemp">添加模板</el-button>
    </div>

    <div v-loading="loading" class="tpl-loading-area">
      <el-empty v-if="!loading && !filteredData.length" description="暂无模板，点击右上角「添加模板」" />
      <div v-if="filteredData.length" class="card-grid">
      <el-card v-for="t in filteredData" :key="t.file" shadow="hover" class="tpl-card">
        <template #header>
          <div class="card-head">
            <el-tag :type="tempType(t.file).type" size="small" effect="dark">{{ tempType(t.file).label }}</el-tag>
            <span class="tpl-name" :title="t.file">{{ t.file }}</span>
          </div>
        </template>

        <div class="tpl-meta">
          <span>{{ fileSize(t.text) }}</span>
          <span>修改于 {{ t.create_date }}</span>
        </div>
        <div class="tpl-preview">{{ (t.text || '').slice(0, 200) }}{{ (t.text || '').length > 200 ? '…' : '' }}</div>

        <div class="card-actions">
          <el-button link type="primary" size="small" @click="handleEdit(t)">编辑</el-button>
          <el-button link type="primary" size="small" @click="copyText(t)">复制内容</el-button>
          <el-button link type="danger" size="small" @click="handleDel(t)">删除</el-button>
        </div>
      </el-card>
      </div>
    </div>

    <!-- 添加/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="TempTitle" width="900px" top="5vh" align-center>
      <el-form label-position="top">
        <el-form-item label="模板文件名">
          <el-input v-model="Tempname" placeholder="例如 my_clash.yaml / loon.conf" clearable />
        </el-form-item>
        <el-form-item label="模板内容">
          <div class="editor-tools">
            <el-input v-model="locateKey" placeholder="输入关键词快速定位…" clearable @keyup.enter="locateInTemplate" />
            <el-button @click="locateInTemplate">定位</el-button>
          </div>
          <el-input ref="templateInputRef" v-model="TempText" type="textarea" :rows="22" placeholder="粘贴模板内容" class="template-editor" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button text @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="addtemp">{{ TempTitle === '添加模板' ? '添加' : '保存修改' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.tpl-page { padding: 10px; }
.toolbar { display: flex; gap: 10px; align-items: center; margin-bottom: 14px; }
.toolbar .search { width: 260px; }
.tpl-loading-area { min-height: 400px; }
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
}
.tpl-card { border-radius: 12px; }
.card-head { display: flex; align-items: center; gap: 8px; }
.tpl-name { flex: 1; font-weight: 600; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tpl-meta { display: flex; justify-content: space-between; font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.tpl-preview {
  font-size: 12px; color: var(--el-text-color-regular);
  background: var(--el-fill-color-light); border-radius: 8px;
  padding: 8px 10px; height: 64px; overflow: hidden;
  white-space: pre-wrap; font-family: monospace;
}
.card-actions { display: flex; justify-content: flex-end; gap: 2px; border-top: 1px solid var(--el-border-color-lighter); padding-top: 8px; margin-top: 8px; }
.editor-tools { display: flex; gap: 8px; margin-bottom: 8px; }
.editor-tools .el-input { max-width: 360px; }
.template-editor :deep(textarea) { min-height: 460px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; line-height: 1.55; tab-size: 2; }
</style>
