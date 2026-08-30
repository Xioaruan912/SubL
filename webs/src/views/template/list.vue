<script setup lang='ts'>
import { ref, computed, onMounted } from 'vue'
import { getTemp, AddTemp, UpdateTemp, DelTemp, ValidateTemp, GetTempVersions, GetTempVersion, RollbackTemp } from "@/api/template/temp"
import TemplateMonacoEditor from '@/components/TemplateMonacoEditor.vue'
import TemplateMonacoDiff from '@/components/TemplateMonacoDiff.vue'

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
const editorRef = ref<any>(null)
const outline = ref<{ key: string; line: number; level: number }[]>([])
const validationErrors = ref<string[]>([])
const validating = ref(false)
const versionsVisible = ref(false)
const versions = ref<any[]>([])
const versionLoading = ref(false)
const diffVisible = ref(false)
const oldVersionText = ref('')
const oldVersionTitle = ref('')
const outlineExpanded = ref(false)
const visibleOutline = computed(() => outlineExpanded.value ? outline.value : outline.value.filter(item => item.level === 0))

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
  dialogVisible.value = true
  runValidation()
}

const editorLanguage = computed(() => /\.ya?ml$/i.test(Tempname.value) ? 'yaml' : 'ini')
const loonOutline = (text: string) => {
  const items: { key: string; line: number; level: number }[] = []
  let inSection = false
  text.split(/\r?\n/).forEach((raw, index) => {
    const line = raw.trim()
    if (!line || line.startsWith('#') || line.startsWith(';')) return
    const section = line.match(/^\[([^\]]+)\]$/)
    if (section) {
      items.push({ key: `[${section[1].trim()}]`, line: index + 1, level: 0 })
      inSection = true
      return
    }
    if (!inSection) return
    let key = ''
    const eq = line.indexOf('=')
    if (eq > 0) key = line.slice(0, eq).trim()
    else {
      const parts = line.split(',').map(v => v.trim())
      if (parts.length >= 2 && parts[0] && parts[1]) key = `${parts[0]} · ${parts[1]}`
      else key = line.split(/\s+/)[0] || ''
    }
    if (key) items.push({ key: key.length > 72 ? `${key.slice(0, 69)}...` : key, line: index + 1, level: 1 })
  })
  return items
}
const runValidation = async () => {
  validating.value = true
  try {
    const { data } = await ValidateTemp({ filename: Tempname.value, text: TempText.value })
    outline.value = /\.conf$/i.test(Tempname.value) ? loonOutline(TempText.value) : (data?.outline || [])
    validationErrors.value = data?.errors || []
    if (!validationErrors.value.length) ElMessage.success('模板语法校验通过')
  } finally { validating.value = false }
}
const openVersions = async () => {
  if (!Tempoldname.value) return
  versionsVisible.value = true; versionLoading.value = true
  try { const { data } = await GetTempVersions(Tempoldname.value); versions.value = data || [] } finally { versionLoading.value = false }
}
const compareVersion = async (item: any) => {
  const { data } = await GetTempVersion(item.id)
  oldVersionText.value = data?.content || ''; oldVersionTitle.value = `版本 #${item.id} · ${item.action}`; diffVisible.value = true
}
const rollbackVersion = async (item: any) => {
  await ElMessageBox.confirm(`确定回滚到版本 #${item.id}？当前内容也会自动保留为历史版本。`, '模板回滚', { type: 'warning' })
  await RollbackTemp({ id: item.id }); ElMessage.success('回滚成功'); versionsVisible.value = false; dialogVisible.value = false; gettemps()
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
    <el-dialog v-model="dialogVisible" :title="TempTitle" width="min(1380px, 96vw)" top="3vh" align-center class="workbench-dialog">
      <el-form label-position="top">
        <div class="workbench-toolbar">
          <el-form-item label="模板文件名" class="filename-field"><el-input v-model="Tempname" placeholder="例如 my_clash.yaml / loon.conf" clearable /></el-form-item>
          <div class="workbench-actions">
            <el-button :loading="validating" @click="runValidation">校验语法</el-button>
            <el-button v-if="TempTitle !== '添加模板'" @click="openVersions">版本历史</el-button>
          </div>
        </div>
        <div class="workbench-grid">
          <aside class="outline-panel">
            <div class="panel-title"><span>结构导航 <small>{{ visibleOutline.length }}/{{ outline.length }}</small></span><button v-if="outline.some(item => item.level > 0)" class="outline-toggle" @click="outlineExpanded = !outlineExpanded">{{ outlineExpanded ? '仅一级' : '展开全部' }}</button></div>
            <el-empty v-if="!outline.length" :image-size="44" description="点击校验生成结构" />
            <button v-for="item in visibleOutline" :key="`${item.line}-${item.key}`" class="outline-item" :style="{ paddingLeft: `${12 + item.level * 12}px` }" @click.prevent="editorRef?.revealLine(item.line)">
              <span>{{ item.key }}</span><small>{{ item.line }}</small>
            </button>
          </aside>
          <main class="editor-panel"><TemplateMonacoEditor ref="editorRef" v-model="TempText" :language="editorLanguage" :errors="validationErrors" /></main>
          <aside class="inspect-panel">
            <div class="panel-title">检查结果</div>
            <el-alert v-if="validationErrors.length" v-for="message in validationErrors" :key="message" :title="message" type="error" :closable="false" show-icon />
            <el-result v-else icon="success" title="可以保存" sub-title="服务端语法检查未发现问题" />
            <el-divider />
            <p class="inspect-tip">点击左侧字段会直接定位到对应行；保存前自动创建版本，可随时比较与回滚。</p>
          </aside>
        </div>
      </el-form>
      <template #footer>
        <el-button text @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="addtemp">{{ TempTitle === '添加模板' ? '添加' : '保存修改' }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="versionsVisible" title="模板版本历史" size="460px">
      <div v-loading="versionLoading" class="version-list">
        <div v-for="item in versions" :key="item.id" class="version-item">
          <div><strong>#{{ item.id }}</strong><el-tag size="small" effect="plain">{{ item.action }}</el-tag></div>
          <small>{{ new Date(item.createdAt).toLocaleString() }}</small>
          <div><el-button link type="primary" @click="compareVersion(item)">与当前比较</el-button><el-button link type="warning" @click="rollbackVersion(item)">回滚</el-button></div>
        </div>
        <el-empty v-if="!versionLoading && !versions.length" description="暂无历史版本" />
      </div>
    </el-drawer>
    <el-dialog v-model="diffVisible" :title="oldVersionTitle" width="min(1480px, 96vw)" top="3vh">
      <div class="diff-head"><span>左侧：历史版本</span><span>右侧：当前编辑内容</span><small>绿色为新增，红色为删除，修改行会在两侧直接高亮。</small></div>
      <TemplateMonacoDiff v-if="diffVisible" :original="oldVersionText" :modified="TempText" :language="editorLanguage" />
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
.workbench-toolbar { display:flex; align-items:flex-end; justify-content:space-between; gap:12px; }.filename-field { flex:1; max-width:560px; margin-bottom:12px; }.workbench-actions { display:flex; gap:8px; padding-bottom:12px; }
.workbench-grid { display:grid; grid-template-columns:220px minmax(0,1fr) 260px; gap:12px; min-height:560px; }.outline-panel,.inspect-panel { overflow:auto; max-height:560px; border:1px solid var(--el-border-color-lighter); border-radius:10px; background:var(--el-fill-color-blank); }.panel-title { position:sticky; top:0; z-index:1; display:flex; justify-content:space-between; align-items:center; gap:8px; padding:12px; border-bottom:1px solid var(--el-border-color-lighter); background:var(--el-bg-color); font-weight:700; }.panel-title span { color:var(--el-text-color-secondary); }.panel-title small { margin-left:4px; font-weight:400; }.outline-toggle { border:0; background:transparent; color:var(--el-color-primary); font-size:12px; cursor:pointer; }.outline-item { display:flex; width:100%; justify-content:space-between; gap:8px; padding:8px 10px; border:0; background:transparent; color:var(--el-text-color-regular); text-align:left; cursor:pointer; }.outline-item:hover { background:var(--el-fill-color-light); color:var(--el-color-primary); }.outline-item small { color:var(--el-text-color-placeholder); }.inspect-panel { padding-bottom:12px; }.inspect-panel :deep(.el-alert) { margin:10px; width:auto; }.inspect-panel :deep(.el-result) { padding:28px 10px 12px; }.inspect-tip { padding:0 14px; color:var(--el-text-color-secondary); font-size:12px; line-height:1.7; }.version-item { display:grid; grid-template-columns:1fr auto; gap:7px; padding:12px 0; border-bottom:1px solid var(--el-border-color-lighter); }.version-item > div:first-child { display:flex; gap:8px; align-items:center; }.version-item > div:last-child { grid-column:1/-1; }.version-item small { color:var(--el-text-color-secondary); }.diff-head { display:flex; gap:18px; align-items:center; margin-bottom:10px; color:var(--el-text-color-regular); font-size:13px; }.diff-head small { margin-left:auto; color:var(--el-text-color-secondary); }
@media(max-width:1100px){.workbench-grid{grid-template-columns:180px minmax(0,1fr)}.inspect-panel{display:none}}@media(max-width:700px){.workbench-grid{grid-template-columns:1fr}.outline-panel{display:none}.diff-head{align-items:flex-start;flex-direction:column;gap:4px}.diff-head small{margin-left:0}}
</style>
