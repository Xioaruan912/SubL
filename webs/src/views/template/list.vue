<script setup lang='ts'>
import { ref, computed, onMounted } from 'vue'
import { getTemp, AddTemp, UpdateTemp, DelTemp, ValidateTemp, PreflightTemp, GetTempVersions, GetTempVersion, RollbackTemp } from "@/api/template/temp"
import TemplateMonacoEditor from '@/components/TemplateMonacoEditor.vue'
import TemplateMonacoDiff from '@/components/TemplateMonacoDiff.vue'
import { compareRegression, deleteRegressionCase, getRegressionCases, saveRegressionCase } from '@/api/regression'
import { startTemplateValidationTask } from '@/api/task'

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
const preflightVisible = ref(false)
const preflightLoading = ref(false)
const preflightReport = ref<any>(null)
const preflightDomains = ref('gemini.google.com\nchatgpt.com\nopenai.com\nclaude.ai')
const lastPreflightSignature = ref('')
const baselineText = ref('')
const regressionVisible = ref(false)
const regressionCases = ref<any[]>([])
const regressionDiff = ref<any[]>([])
const regressionResults = ref<any[]>([])
const regressionForm = ref<any>({ name:'', domain:'', expectedPolicy:'', expectedCountry:'', forbiddenPolicy:'DIRECT', protocol:'tcp', port:443, enabled:true })

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
  baselineText.value = ''
  preflightReport.value = null
  lastPreflightSignature.value = ''
  dialogVisible.value = true
}
const addtemp = async () => {
  if (!Tempname.value.trim()) { ElMessage.warning('请填写文件名'); return }
  const canSave = await runPreflight(false)
  if (!canSave) {
    preflightVisible.value = true
    ElMessage.error('发布前预检发现错误，请修复后再保存')
    return
  }
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
  baselineText.value = row.text
  preflightReport.value = null
  lastPreflightSignature.value = ''
  dialogVisible.value = true
  runValidation()
}
const loadRegressionCases = async () => { const { data } = await getRegressionCases(); regressionCases.value = data || [] }
const openRegression = async () => { regressionVisible.value = true; await loadRegressionCases(); if (baselineText.value && TempText.value) { const { data } = await compareRegression({ filename:Tempname.value, before:baselineText.value, after:TempText.value }); regressionDiff.value = data?.diff || []; regressionResults.value = data?.after || [] } }
const persistRegression = async () => { if (!regressionForm.value.name || !regressionForm.value.domain) { ElMessage.warning('请填写名称和域名'); return }; await saveRegressionCase(regressionForm.value); regressionForm.value={ name:'',domain:'',expectedPolicy:'',expectedCountry:'',forbiddenPolicy:'DIRECT',protocol:'tcp',port:443,enabled:true }; await loadRegressionCases(); await openRegression() }
const removeRegression = async (row:any) => { await deleteRegressionCase(row.id); await loadRegressionCases(); await openRegression() }
const editRegression = (row:any) => { regressionForm.value = { ...row } }
const queueTemplateValidation = async () => { if (!Tempoldname.value) { ElMessage.info('请先保存模板，再加入后台验证任务'); return }; const {data}=await startTemplateValidationTask(Tempoldname.value);ElMessage.success(`模板验证任务已创建 #${data?.taskId||''}`) }

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
const runPreflight = async (show = true) => {
  if (!Tempname.value.trim() || !TempText.value.trim()) {
    ElMessage.warning('请先填写模板文件名和内容')
    return false
  }
  const signature = `${Tempname.value.trim()}\u0000${TempText.value}\u0000${preflightDomains.value}`
  if (!show && signature === lastPreflightSignature.value && preflightReport.value?.valid) return true
  if (show) preflightVisible.value = true
  preflightLoading.value = true
  try {
    const { data } = await PreflightTemp({ filename: Tempname.value.trim(), text: TempText.value, domains: preflightDomains.value })
    preflightReport.value = data
    lastPreflightSignature.value = signature
    if (data?.valid) {
      if (show) ElMessage.success(data?.summary?.warnings ? '预检通过，存在需要确认的警告' : '发布前预检通过')
      return true
    }
    return false
  } catch {
    return false
  } finally {
    preflightLoading.value = false
  }
}
const severityLabel = (value: string) => value === 'error' ? '错误' : value === 'warning' ? '警告' : '提示'
const severityType = (value: string) => value === 'error' ? 'danger' : value === 'warning' ? 'warning' : 'info'
const routeLabel = (value: string) => value === 'matched' ? '已确认' : value === 'partial' ? '需复核' : '未命中'
const routeType = (value: string) => value === 'matched' ? 'success' : value === 'partial' ? 'warning' : 'danger'
const compatibilityLabel = (value: string) => value === 'pass' ? '通过' : value === 'error' ? '阻止发布' : '需确认'
const revealIssue = (line?: number) => {
  if (!line) return
  preflightVisible.value = false
  requestAnimationFrame(() => editorRef.value?.revealLine(line))
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
            <el-button type="primary" plain :loading="preflightLoading" @click="runPreflight(true)">发布前预检</el-button>
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

    <el-dialog v-model="preflightVisible" title="发布前预检" width="min(1120px, 96vw)" top="4vh" class="preflight-dialog">
      <div class="preflight-query">
        <div><strong>规则解释目标</strong><small>每行一个域名；只解释模板规则，不会从浏览器直连目标网站。</small></div>
        <el-input v-model="preflightDomains" type="textarea" :rows="2" placeholder="gemini.google.com&#10;chatgpt.com" />
        <div class="preflight-actions"><el-button @click="openRegression">回归用例</el-button><el-button @click="queueTemplateValidation">后台验证</el-button><el-button type="primary" :loading="preflightLoading" @click="runPreflight(true)">重新预检</el-button></div>
      </div>

      <div v-loading="preflightLoading" class="preflight-body">
        <template v-if="preflightReport">
          <div class="preflight-hero" :class="preflightReport.valid ? 'is-pass' : 'is-error'">
            <div class="preflight-mark">{{ preflightReport.valid ? '✓' : '!' }}</div>
            <div>
              <strong>{{ preflightReport.valid ? '可以安全保存' : '发现阻止保存的问题' }}</strong>
              <small>{{ String(preflightReport.format || '').toUpperCase() }} · {{ preflightReport.coverage === 'full' ? '完整语义检查' : '基础结构检查' }}</small>
            </div>
            <div class="preflight-stats">
              <span><b>{{ preflightReport.summary?.errors || 0 }}</b> 错误</span>
              <span><b>{{ preflightReport.summary?.warnings || 0 }}</b> 警告</span>
              <span><b>{{ preflightReport.summary?.groups || 0 }}</b> 策略组</span>
              <span><b>{{ preflightReport.summary?.rules || 0 }}</b> 规则</span>
            </div>
          </div>

          <section class="preflight-section">
            <header><strong>客户端与测试兼容性</strong><small>协议 {{ preflightReport.protocols?.length || 0 }} 类</small></header>
            <div class="compat-grid">
              <article v-for="item in preflightReport.compatibility" :key="item.target">
                <div><b>{{ item.target }}</b><el-tag size="small" :type="severityType(item.status === 'error' ? 'error' : item.status === 'warning' ? 'warning' : 'info')">{{ compatibilityLabel(item.status) }}</el-tag></div>
                <p>{{ item.detail }}</p>
              </article>
            </div>
            <div v-if="preflightReport.protocols?.length" class="protocol-row">
              <el-tag v-for="item in preflightReport.protocols" :key="`${item.type}-${item.network}`" effect="plain">
                {{ item.type || '未知' }}{{ item.network ? ` / ${item.network}` : '' }} × {{ item.count }}
              </el-tag>
            </div>
            <el-table v-if="preflightReport.capabilityMatrix?.length" :data="preflightReport.capabilityMatrix" size="small" class="capability-table">
              <el-table-column prop="client" label="目标客户端" width="140" />
              <el-table-column label="结论" width="100"><template #default="{row}"><el-tag size="small" :type="row.status==='error'?'danger':row.status==='warning'?'warning':'success'">{{ compatibilityLabel(row.status) }}</el-tag></template></el-table-column>
              <el-table-column label="原生支持" min-width="180"><template #default="{row}">{{ row.native?.join('、') || '--' }}</template></el-table-column>
              <el-table-column label="需要转换" min-width="150"><template #default="{row}">{{ row.converted?.join('、') || '--' }}</template></el-table-column>
              <el-table-column label="不支持" min-width="160"><template #default="{row}">{{ row.unsupported?.join('、') || '--' }}</template></el-table-column>
              <el-table-column label="可能丢失字段" min-width="180"><template #default="{row}">{{ row.lostFields?.join('、') || '--' }}</template></el-table-column>
            </el-table>
          </section>

          <section class="preflight-section">
            <header><strong>问题清单</strong><small>{{ preflightReport.issues?.length || 0 }} 项</small></header>
            <el-empty v-if="!preflightReport.issues?.length" :image-size="48" description="没有发现结构或引用问题" />
            <button v-for="(item, index) in preflightReport.issues" :key="`${item.code}-${index}`" class="issue-row" :class="{ clickable: item.line }" @click="revealIssue(item.line)">
              <el-tag size="small" :type="severityType(item.severity)">{{ severityLabel(item.severity) }}</el-tag>
              <span>{{ item.message }}</span>
              <small v-if="item.line">第 {{ item.line }} 行</small>
            </button>
          </section>

          <section class="preflight-section">
            <header><strong>规则命中解释</strong><small>按模板顺序首次命中</small></header>
            <el-table :data="preflightReport.routes || []" class="route-table">
              <el-table-column label="目标" min-width="150"><template #default="{ row }"><b class="mono">{{ row.domain }}</b></template></el-table-column>
              <el-table-column label="状态" width="88"><template #default="{ row }"><el-tag size="small" :type="routeType(row.status)">{{ routeLabel(row.status) }}</el-tag></template></el-table-column>
              <el-table-column label="命中规则" min-width="250"><template #default="{ row }"><div class="route-rule"><span>{{ row.matchedRule || '没有可确认的匹配' }}</span><small v-if="row.ruleIndex">规则 #{{ row.ruleIndex }}</small></div></template></el-table-column>
              <el-table-column label="策略链" min-width="230"><template #default="{ row }"><div class="route-chain"><span v-for="(part, index) in row.chain" :key="`${part}-${index}`"><i v-if="index">→</i>{{ part }}</span><small v-if="row.candidates?.length" :title="row.candidates.join('、')">候选：{{ row.candidates.join('、') }}</small><small v-if="row.notes?.length" :title="row.notes.join('\n')">{{ row.notes.join('；') }}</small></div></template></el-table-column>
            </el-table>
          </section>
        </template>
        <el-empty v-else description="点击重新预检生成报告" />
      </div>
      <template #footer><el-button @click="preflightVisible = false">关闭</el-button></template>
    </el-dialog>

    <el-drawer v-model="regressionVisible" title="分流回归用例" size="860px">
      <el-alert type="info" :closable="false" title="回归用例可以明确预期策略/地区或禁止策略；编辑模板时会比较修改前后的命中变化。" />
      <div class="regression-form">
        <el-input v-model="regressionForm.name" placeholder="名称，例如 Gemini 必须走 AI" />
        <el-input v-model="regressionForm.domain" placeholder="域名，例如 gemini.google.com" />
        <el-input v-model="regressionForm.expectedPolicy" placeholder="预期策略（可选）" />
        <el-input v-model="regressionForm.expectedCountry" placeholder="预期地区 JP/US（可选）" />
        <el-input v-model="regressionForm.forbiddenPolicy" placeholder="禁止策略，例如 DIRECT" />
        <el-select v-model="regressionForm.protocol"><el-option label="TCP" value="tcp"/><el-option label="UDP" value="udp"/></el-select>
        <el-input-number v-model="regressionForm.port" :min="1" :max="65535" />
        <el-switch v-model="regressionForm.enabled" active-text="启用" />
      </div>
      <div class="regression-actions"><el-button type="primary" @click="persistRegression">保存用例</el-button></div>
      <el-table :data="regressionCases" size="small">
        <el-table-column prop="name" label="名称" min-width="130"/><el-table-column prop="domain" label="域名" min-width="170"/><el-table-column prop="expectedPolicy" label="预期策略" min-width="120"/><el-table-column prop="expectedCountry" label="地区" width="70"/><el-table-column prop="forbiddenPolicy" label="禁止" width="90"/>
        <el-table-column label="操作" width="120"><template #default="{row}"><el-button link type="primary" @click="editRegression(row)">编辑</el-button><el-button link type="danger" @click="removeRegression(row)">删除</el-button></template></el-table-column>
      </el-table>
      <template v-if="regressionDiff.length">
        <h4>当前编辑内容与打开时版本的命中差异</h4>
        <el-table :data="regressionDiff" size="small"><el-table-column prop="domain" label="域名" min-width="170"/><el-table-column prop="beforePolicy" label="修改前" min-width="120"/><el-table-column prop="afterPolicy" label="修改后" min-width="120"/><el-table-column label="变化" width="90"><template #default="{row}"><el-tag :type="row.changed?'warning':'success'" size="small">{{ row.changed?'变化':'一致' }}</el-tag></template></el-table-column></el-table>
      </template>
      <template v-if="regressionResults.length"><h4>修改后回归结果</h4><el-table :data="regressionResults" size="small"><el-table-column label="用例" min-width="140"><template #default="{row}">{{ row.case?.name }}</template></el-table-column><el-table-column prop="policy" label="实际策略" min-width="120"/><el-table-column label="结果" width="90"><template #default="{row}"><el-tag :type="row.passed?'success':'danger'" size="small">{{ row.passed?'通过':'失败' }}</el-tag></template></el-table-column><el-table-column prop="reason" label="原因" min-width="180"/></el-table></template>
    </el-drawer>
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
.preflight-query { display:grid; grid-template-columns:190px minmax(0,1fr) auto; align-items:center; gap:14px; padding:14px; border:1px solid var(--el-border-color-lighter); border-radius:12px; background:var(--el-fill-color-extra-light); }
.preflight-actions{display:flex;gap:8px}.capability-table{border-top:1px solid var(--el-border-color-lighter)}.regression-form{display:grid;grid-template-columns:1fr 1fr;gap:10px;margin:16px 0}.regression-actions{display:flex;justify-content:flex-end;margin-bottom:14px}@media(max-width:720px){.regression-form{grid-template-columns:1fr}}
.preflight-query>div { display:flex; flex-direction:column; gap:4px; }.preflight-query small { color:var(--el-text-color-secondary); font-size:11px; line-height:1.5; }.preflight-body { min-height:300px; margin-top:14px; }
.preflight-hero { display:flex; align-items:center; gap:13px; padding:16px 18px; border:1px solid color-mix(in srgb,var(--el-color-success) 35%,var(--el-border-color)); border-radius:13px; background:color-mix(in srgb,var(--el-color-success) 7%,var(--el-bg-color)); }.preflight-hero.is-error { border-color:color-mix(in srgb,var(--el-color-danger) 35%,var(--el-border-color)); background:color-mix(in srgb,var(--el-color-danger) 7%,var(--el-bg-color)); }
.preflight-mark { display:grid; width:36px; height:36px; place-items:center; border-radius:10px; background:var(--el-color-success); color:white; font-size:21px; font-weight:800; }.is-error .preflight-mark { background:var(--el-color-danger); }.preflight-hero>div:nth-child(2) { display:flex; flex-direction:column; gap:3px; }.preflight-hero>div:nth-child(2) small { color:var(--el-text-color-secondary); }
.preflight-stats { display:flex; gap:18px; margin-left:auto; color:var(--el-text-color-secondary); font-size:12px; }.preflight-stats span { display:flex; align-items:baseline; gap:4px; }.preflight-stats b { color:var(--el-text-color-primary); font-size:18px; }
.preflight-section { margin-top:14px; overflow:hidden; border:1px solid var(--el-border-color-lighter); border-radius:12px; }.preflight-section>header { display:flex; justify-content:space-between; padding:12px 14px; background:var(--el-fill-color-extra-light); }.preflight-section>header small { color:var(--el-text-color-secondary); }
.compat-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(250px,1fr)); gap:10px; padding:12px; }.compat-grid article { padding:12px; border:1px solid var(--el-border-color-lighter); border-radius:10px; }.compat-grid article>div { display:flex; justify-content:space-between; gap:8px; }.compat-grid p { margin:7px 0 0; color:var(--el-text-color-secondary); font-size:12px; line-height:1.55; }.protocol-row { display:flex; flex-wrap:wrap; gap:7px; padding:0 12px 12px; }
.issue-row { display:grid; width:100%; grid-template-columns:64px 1fr auto; align-items:center; gap:10px; padding:10px 13px; border:0; border-top:1px solid var(--el-border-color-lighter); background:transparent; color:var(--el-text-color-regular); text-align:left; }.issue-row.clickable { cursor:pointer; }.issue-row.clickable:hover { background:var(--el-fill-color-light); }.issue-row small { color:var(--el-color-primary); }
.route-rule,.route-chain { display:flex; flex-direction:column; gap:3px; }.route-rule span { font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:12px; }.route-rule small,.route-chain small { overflow:hidden; color:var(--el-text-color-secondary); font-size:11px; text-overflow:ellipsis; white-space:nowrap; }.route-chain>span { display:inline; font-size:12px; }.route-chain i { margin:0 5px; color:var(--el-text-color-placeholder); font-style:normal; }
@media(max-width:1100px){.workbench-grid{grid-template-columns:180px minmax(0,1fr)}.inspect-panel{display:none}}
@media(max-width:700px){
  .tpl-page{padding:6px}
  .toolbar{flex-wrap:wrap}.toolbar .search{width:100%;flex:1 1 100%}.toolbar :deep(.el-button){flex:1;margin-left:0}
  .card-grid{grid-template-columns:minmax(0,1fr);gap:10px}.tpl-card{min-width:0}.card-head,.card-actions{flex-wrap:wrap}.card-actions{justify-content:flex-start}
  .workbench-toolbar{align-items:stretch;flex-direction:column}.filename-field{width:100%;max-width:none;margin-bottom:0}.workbench-actions{flex-wrap:wrap;padding-bottom:0}.workbench-actions :deep(.el-button){flex:1 1 calc(50% - 4px);margin-left:0}
  .workbench-grid{grid-template-columns:1fr;min-height:420px}.outline-panel{display:none}.editor-panel{min-width:0;overflow:hidden}
  .diff-head{align-items:flex-start;flex-direction:column;gap:4px}.diff-head small{margin-left:0}
  .preflight-query{grid-template-columns:1fr}.preflight-actions{flex-wrap:wrap}.preflight-actions :deep(.el-button){flex:1;margin-left:0}
  .preflight-stats{display:grid;grid-template-columns:1fr 1fr;margin-left:0;width:100%}.preflight-hero{align-items:flex-start;flex-wrap:wrap}.preflight-mark{flex:0 0 auto}
  .issue-row{grid-template-columns:58px 1fr}.issue-row small{grid-column:2}.compat-grid{grid-template-columns:1fr;padding:8px}
  .regression-actions :deep(.el-button){width:100%;margin-left:0}
}
</style>
