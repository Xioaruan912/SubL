<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { countryFlag } from '@/utils/flag'
import { getSubs, getSubPreviewNodes, subscriptionEgressPlan, subscriptionRuleExplain } from '@/api/subcription/subs'
import { EgressTest, deleteEgressTarget, getEgressTargets, getNodeOverview, getNodeQualityMatrix, saveEgressTarget } from '@/api/subcription/node'
import { startQualityMatrixSample } from '@/api/task'

defineOptions({ name: 'EgressTest' })

type Target = {
  id?: number; key: string; name: string; domain: string; group: string; icon: string;
  path?: string; method?: string; expectedStatus?: string; responseContains?: string;
  requireEgressIp?: boolean; timeoutSeconds?: number; retries?: number; enabled?: boolean; sortOrder?: number
}
type Result = Target & { status: 'pending' | 'testing' | 'done' | 'error'; ip: string; countryCode: string; rtt: number; note: string }

const targets = ref<Target[]>([])
const results = ref<Result[]>([])
const running = ref(false)
const lastTestedAt = ref<Date | null>(null)
const hideIP = ref(false)
const mode = ref<'subscription' | 'local' | 'template' | 'explain'>('subscription')
const subscriptions = ref<any[]>([])
const selectedSubscription = ref<number | null>(null)
const subscriptionNodes = ref<any[]>([])
const selectedNode = ref<number | null>(null)
const nodesLoading = ref(false)
const planLoading = ref(false)
const plan = ref<any | null>(null)
const explainTarget = ref('gemini.google.com')
const explainIP = ref('')
const explainPort = ref<number | undefined>(443)
const explainProtocol = ref('tcp')
const explainLoading = ref(false)
const explainResult = ref<any | null>(null)
const targetDrawer = ref(false)
const matrixDrawer = ref(false)
const matrixLoading = ref(false)
const matrixSampling = ref(false)
const matrixData = ref<any>({ targets:[], nodes:[] })
const targetSaving = ref(false)
const targetForm = ref<Target>({ key:'', name:'', domain:'', group:'ai', icon:'•', path:'/cdn-cgi/trace', method:'GET', expectedStatus:'200-399', responseContains:'', requireEgressIp:true, timeoutSeconds:7, retries:0, enabled:true, sortOrder:100 })
// Changes with each release so browsers do not retain an old immutable chunk.
const splitVerifierBuild = '20260829-rule-check-2'

const groupLabels: Record<string,string> = { network:'网络', ai:'AI', social:'社交', content:'内容', finance:'金融', tools:'工具', developer:'开发', media:'媒体' }
const matrixScenes = computed<string[]>(() => [...new Set<string>((matrixData.value.targets || []).map((item:any) => String(item.group || '')).filter(Boolean))])
const matrixStat = (row:any, scene:string) => row.scenes?.[scene]
const openMatrix = async () => {
  matrixDrawer.value = true; matrixLoading.value = true
  try { const { data } = await getNodeQualityMatrix(24); matrixData.value = data || { targets:[], nodes:[] } }
  finally { matrixLoading.value = false }
}
const sampleMatrix = async (mode:'scene'|'full') => {
  if (matrixSampling.value) return
  if (mode === 'full') await ElMessageBox.confirm('完整采样会对每个当前在线节点检测全部启用目标，耗时和网络请求都明显高于场景采样。继续？','完整质量采样',{type:'warning'})
  matrixSampling.value = true
  try {
    const { data } = await startQualityMatrixSample(mode)
    ElMessage.success(`采样任务已创建 #${data?.taskId || ''}，可在任务中心查看进度`)
  } finally { matrixSampling.value = false }
}
const resetResults = () => { results.value = targets.value.filter(item => item.enabled !== false).map(item => ({ ...item, group:groupLabels[item.group] || item.group, status:'pending', ip:'', countryCode:'', rtt:-1, note:'' })) }
const loadTargets = async () => {
  const { data } = await getEgressTargets()
  targets.value = (data || []).map((item:any) => ({ ...item, icon:item.icon || '•' }))
  resetResults()
}
const newTarget = () => { targetForm.value = { key:'', name:'', domain:'', group:'ai', icon:'•', path:'/cdn-cgi/trace', method:'GET', expectedStatus:'200-399', responseContains:'', requireEgressIp:true, timeoutSeconds:7, retries:0, enabled:true, sortOrder:(targets.value.length + 1) * 10 } }
const editTarget = (item:any) => { targetForm.value = { ...item }; targetDrawer.value = true }
const persistTarget = async () => {
  if (!targetForm.value.key || !targetForm.value.name || !targetForm.value.domain || !targetForm.value.group) { ElMessage.warning('请填写 key、名称、域名和分类'); return }
  targetSaving.value = true
  try { await saveEgressTarget(targetForm.value); ElMessage.success('检测目标已保存'); await loadTargets(); newTarget() } finally { targetSaving.value = false }
}
const removeTarget = async (item:any) => {
  if (!item.id) return
  await ElMessageBox.confirm(`删除检测目标「${item.name}」？`, '确认删除', { type:'warning' })
  await deleteEgressTarget(item.id); ElMessage.success('已删除'); await loadTargets()
}

const loadSubscriptionNodes = async () => {
  subscriptionNodes.value = []; selectedNode.value = null
  if (!selectedSubscription.value) return
  nodesLoading.value = true
  try {
    const [preview, overview] = await Promise.all([getSubPreviewNodes(selectedSubscription.value), getNodeOverview()])
    const quality = new Map((overview.data || []).map((item: any) => [item.id, item]))
    subscriptionNodes.value = (preview.data || []).map((item: any) => ({ ...item, quality: quality.get(item.ID) || {} }))
      .sort((a: any, b: any) => (b.quality.score || 0) - (a.quality.score || 0))
    selectedNode.value = subscriptionNodes.value[0]?.ID || null
  } finally { nodesLoading.value = false }
}

const maskIP = (ip: string) => {
  if (!hideIP.value || !ip) return ip || '--'
  if (ip.includes(':')) { const parts = ip.split(':'); return `${parts.slice(0, 2).join(':')}:****:****` }
  return ip.replace(/\.\d+\.\d+$/, '.*.*')
}

const uniqueEgress = computed(() => {
  const map = new Map<string, { ip: string; countryCode: string; services: string[] }>()
  results.value.filter(item => item.status === 'done' && item.ip).forEach(item => {
    const found = map.get(item.ip)
    if (found) found.services.push(item.name)
    else map.set(item.ip, { ip: item.ip, countryCode: item.countryCode, services: [item.name] })
  })
  return [...map.values()]
})

const statusAllowed = (code:number, spec = '') => {
  if (!spec.trim()) return true
  return spec.split(',').some(raw => {
    const token = raw.trim()
    if (!token) return false
    if (token.includes('-')) {
      const [low, high] = token.split('-', 2).map(Number)
      return Number.isFinite(low) && Number.isFinite(high) && code >= low && code <= high
    }
    return code === Number(token)
  })
}

const detect = async (item: Result) => {
  item.status = 'testing'; item.note = ''; item.rtt = -1
  const attempts = Math.max(1, Math.min(6, (item.retries || 0) + 1))
  for (let attempt = 0; attempt < attempts; attempt++) {
    const controller = new AbortController(); const timer = window.setTimeout(() => controller.abort(), Math.max(1, item.timeoutSeconds || 7) * 1000)
    const started = performance.now()
    try {
      const response = await fetch(`https://${item.domain}${item.path || '/cdn-cgi/trace'}?_=${Date.now()}`, { cache: 'no-store', signal: controller.signal, method:item.method || 'GET' })
      if (!statusAllowed(response.status, item.expectedStatus || '200-399')) throw new Error(`HTTP ${response.status}`)
      const text = item.method === 'HEAD' ? '' : await response.text()
      if (item.responseContains && !text.toLowerCase().includes(item.responseContains.toLowerCase())) throw new Error('响应未包含预期特征')
      const trace = Object.fromEntries(text.trim().split('\n').map(line => { const index = line.indexOf('='); return index > 0 ? [line.slice(0, index), line.slice(index + 1)] : [line, ''] }))
      if (!trace.ip && item.requireEgressIp !== false) throw new Error('目标未返回出口 IP')
      item.ip = trace.ip || ''; item.countryCode = (trace.loc || '').toUpperCase(); item.rtt = Math.max(1, Math.round(performance.now() - started)); item.note = item.ip ? '' : '站点可达，未提供出口 IP'; item.status = 'done'
      window.clearTimeout(timer)
      return
    } catch (error: any) {
      item.status = 'error'; item.note = error?.name === 'AbortError' ? '超时' : (error?.message || '目标未开放跨域检测')
    } finally { window.clearTimeout(timer) }
  }
}

const runLocal = async () => {
  if (running.value) return
  running.value = true
  results.value.forEach(item => { item.status = 'pending'; item.ip = ''; item.countryCode = ''; item.note = '' })
  let cursor = 0
  const worker = async () => { while (cursor < results.value.length) { const item = results.value[cursor++]; await detect(item) } }
  await Promise.all(Array.from({ length: 4 }, worker))
  lastTestedAt.value = new Date(); running.value = false
}

const runSelected = async () => {
  if (!selectedSubscription.value) { ElMessage.warning('请先选择订阅'); return }
  if (!selectedNode.value) { ElMessage.warning('该订阅没有可检测节点'); return }
  running.value = true
  resetResults()
  try {
    const { data } = await EgressTest({ id: selectedNode.value })
    results.value = (data?.results || []).map((item: any) => ({ ...(targets.value.find(target => target.domain === item.domain) || { icon: '•' }), key:item.key, name:item.name, domain:item.domain, group:groupLabels[item.group] || item.group, status:item.status === 'available' || item.status === 'reachable' ? 'done' : 'error', ip:item.ip || '', countryCode:item.countryCode || '', rtt:item.rtt ?? -1, note:item.note || '未获取出口 IP' }))
    lastTestedAt.value = new Date()
  } finally { running.value = false }
}

const runCurrent = () => mode.value === 'subscription' ? runSelected() : mode.value === 'template' ? runPlan() : mode.value === 'explain' ? runExplain() : runLocal()
const runPlan = async () => {
  if (!selectedSubscription.value) { ElMessage.warning('请先选择订阅'); return }
  planLoading.value = true
  try { const { data } = await subscriptionEgressPlan(selectedSubscription.value); plan.value = data; if (data?.items) results.value = data.items.map((item:any) => { const target = targets.value.find(t => t.domain === item.domain) || { key:item.key, icon:'•' }; const check = item.result || {}; return { ...target, name:item.name, domain:item.domain, group:groupLabels[item.group] || item.group, status: check.status === 'available' || check.status === 'reachable' ? 'done' : check.status ? 'error' : 'pending', ip:check.ip || '', countryCode:check.countryCode || '', rtt:check.rtt ?? -1, note:check.note || (item.fallback ? `未找到 ${item.expectedCountry}，已使用质量最优节点` : '') } }) ; lastTestedAt.value = new Date() } finally { planLoading.value = false }
}
const runExplain = async () => {
  if (!selectedSubscription.value) { ElMessage.warning('请先选择订阅'); return }
  if (!explainTarget.value.trim() && !explainIP.value.trim() && !explainPort.value) { ElMessage.warning('至少输入域名、IP 或端口之一'); return }
  explainLoading.value = true; explainResult.value = null
  try {
    const { data } = await subscriptionRuleExplain({ subscriptionId:selectedSubscription.value, target:explainTarget.value.trim(), ip:explainIP.value.trim(), port:explainPort.value || 0, protocol:explainProtocol.value })
    explainResult.value = data
  } finally { explainLoading.value = false }
}
const faviconUrl = (domain:string) => {
  // npm registry 没有稳定的 favicon，使用 npm 官方站点图标。
  const iconDomain = domain === 'registry.npmjs.org' ? 'www.npmjs.com' : domain
  return `https://${iconDomain}/favicon.ico`
}
const iconFallback = (event: Event, icon: string) => { const image = event.target as HTMLImageElement; image.style.display = 'none'; const next = image.nextElementSibling as HTMLElement | null; if (next) { next.textContent = icon; next.style.display = 'grid' } }

const statusType = (item: any) => item.status === 'done' ? 'success' : item.status === 'error' ? 'danger' : 'info'
const statusText = (item: any) => item.status === 'done' ? '已获取' : item.status === 'testing' ? '检测中' : item.status === 'error' ? item.note : '等待'

onMounted(async () => {
  await loadTargets()
  const { data } = await getSubs(); subscriptions.value = data || []
  if (subscriptions.value.length) { selectedSubscription.value = subscriptions.value[0].ID; await loadSubscriptionNodes() }
})
</script>

<template>
  <div class="egress-page">
    <section class="hero-card">
      <div><span class="eyebrow">SPLIT ROUTING INSPECTOR</span><h1>订阅分流与出口检测</h1><p>选择订阅及其中的具体节点，服务端会通过该节点访问各目标并返回真实出口 IP。也可切换为本机模式，检查当前浏览器的分流规则。</p></div>
      <div class="hero-actions"><el-switch v-model="hideIP" active-text="隐藏 IP" /><el-button @click="openMatrix">质量矩阵</el-button><el-button @click="targetDrawer = true">检测目标</el-button><el-button type="primary" :loading="running" @click="runCurrent">开始检测</el-button></div>
    </section>

    <section class="source-card">
      <el-segmented v-model="mode" :options="[{label:'订阅节点检测',value:'subscription'},{label:'模板规则验证',value:'template'},{label:'规则解释器',value:'explain'},{label:'本机浏览器检测',value:'local'}]" />
      <div v-if="mode === 'subscription' || mode === 'template' || mode === 'explain'" class="source-selectors" :class="{ 'template-selectors': mode === 'template' || mode === 'explain' }">
        <el-select v-model="selectedSubscription" placeholder="选择订阅" filterable @change="loadSubscriptionNodes"><el-option v-for="sub in subscriptions" :key="sub.ID" :label="sub.Name" :value="sub.ID"><span>{{ sub.Name }}</span><small>{{ sub.Nodes?.length || 0 }} 个固定节点</small></el-option></el-select>
        <template v-if="mode === 'subscription'">
          <span class="arrow">→</span>
          <el-select v-model="selectedNode" placeholder="选择具体节点" filterable :loading="nodesLoading"><el-option v-for="item in subscriptionNodes" :key="item.ID" :label="item.Name" :value="item.ID"><span>{{ item.Name }}</span><small>质量 {{ item.quality?.score || 0 }} · {{ item.quality?.averageRtt >= 0 ? item.quality.averageRtt + 'ms' : '无样本' }}</small></el-option></el-select>
          <el-tag v-if="subscriptionNodes.length" type="success" effect="plain">默认质量最优</el-tag>
        </template>
        <el-alert v-else-if="mode === 'template'" type="info" :closable="false" title="将读取该订阅绑定的 Clash/Surge/Loon 模板，并按实际规则自动选择节点" />
        <el-alert v-else type="info" :closable="false" title="读取该订阅绑定的 Clash/Mihomo 模板，逐条解释为什么命中或未命中" />
      </div>
      <el-alert v-else type="info" :closable="false" title="本机模式由浏览器直连目标网站；它反映当前设备的代理分流，不经过 SubLinkX 节点。" />
    </section>

    <section v-if="mode === 'explain'" class="plan-card explain-card">
      <header><div><b>规则为什么这样走</b><small>按 Clash/Mihomo 首条命中语义逐条解释，不修改任何配置</small></div><el-button type="primary" :loading="explainLoading" @click="runExplain">开始解释</el-button></header>
      <div class="explain-query">
        <el-input v-model="explainTarget" placeholder="域名，例如 gemini.google.com" />
        <el-input v-model="explainIP" placeholder="目标 IP（可选）" />
        <el-input-number v-model="explainPort" :min="1" :max="65535" placeholder="端口" />
        <el-select v-model="explainProtocol"><el-option label="TCP" value="tcp" /><el-option label="UDP" value="udp" /></el-select>
      </div>
      <template v-if="explainResult">
        <div class="explain-flow">
          <span>{{ explainResult.target || explainResult.ip || '请求' }}</span><b>→</b>
          <span>{{ explainResult.matchedRule || '未命中' }}</span><b>→</b>
          <span v-for="item in explainResult.chain || []" :key="item">{{ item }}</span>
          <template v-if="explainResult.selectedNode"><b>→</b><span class="selected">{{ explainResult.selectedNode.name }}</span></template>
        </div>
        <el-descriptions :column="3" border size="small" class="explain-meta">
          <el-descriptions-item label="模板">{{ explainResult.template }}</el-descriptions-item>
          <el-descriptions-item label="命中位置">第 {{ explainResult.ruleIndex || '--' }} 条</el-descriptions-item>
          <el-descriptions-item label="候选节点">{{ explainResult.candidateCount || 0 }}</el-descriptions-item>
          <el-descriptions-item label="策略">{{ explainResult.policy || '--' }}</el-descriptions-item>
          <el-descriptions-item label="期望地区">{{ explainResult.expectedCountry || '不限' }}</el-descriptions-item>
          <el-descriptions-item label="实际节点">{{ explainResult.selectedNode?.name || '未选择' }}</el-descriptions-item>
        </el-descriptions>
        <header class="subhead"><b>命中前未命中规则</b><small>共评估 {{ explainResult.evaluatedCount || 0 }} 条</small></header>
        <el-table :data="explainResult.previous || []" size="small" max-height="360">
          <el-table-column prop="index" label="#" width="64" />
          <el-table-column prop="rule" label="规则" min-width="320"><template #default="{ row }"><span class="rule-text">{{ row.rule }}</span></template></el-table-column>
          <el-table-column prop="reason" label="为什么没命中" min-width="240" />
          <el-table-column prop="source" label="来源" min-width="140" />
        </el-table>
        <el-alert v-if="explainResult.warnings?.length" class="plan-warning" type="warning" :closable="false" :title="explainResult.warnings.join('；')" />
      </template>
      <el-empty v-else-if="!explainLoading" :image-size="54" description="输入请求上下文后查看完整规则路径" />
    </section>

    <section v-if="plan && mode === 'template'" class="plan-card">
      <header><div><b>模板分流验证</b><small>{{ plan.template || '未读取到本地模板' }} · 已按规则选择质量最优节点 · {{ splitVerifierBuild }}</small></div><el-tag type="success" effect="plain">{{ plan.items?.length || 0 }} 个目标</el-tag></header>
      <el-table :data="plan.items" size="small">
        <el-table-column label="网站" width="140"><template #default="{ row }"><span class="target-title"><img class="site-icon" :src="faviconUrl(row.domain)" @error="iconFallback($event, (targets.find(t => t.domain === row.domain) || { icon: '•' }).icon)">{{ row.name }}</span></template></el-table-column>
        <el-table-column label="命中规则" min-width="220"><template #default="{ row }"><span class="rule-text">{{ row.matchedRule || '默认策略' }}</span></template></el-table-column>
        <el-table-column label="期望地区" width="100"><template #default="{ row }">{{ row.expectedCountry || '不限' }}</template></el-table-column>
        <el-table-column label="实际节点" min-width="180"><template #default="{ row }">{{ row.selectedNode?.name || '无可用节点' }}<small v-if="row.selectedNode"> · {{ row.selectedNode.countryCode || '未知' }} · 质量 {{ row.selectedNode.score || 0 }}</small></template></el-table-column>
        <el-table-column label="结果" width="110"><template #default="{ row }"><el-tag :type="row.selectedNode?.id > 0 && (row.result?.status === 'available' || row.result?.status === 'reachable') ? 'success' : 'danger'" size="small">{{ row.selectedNode?.id > 0 && (row.result?.status === 'available' || row.result?.status === 'reachable') ? '分流成功' : '失败' }}</el-tag></template></el-table-column>
      </el-table>
      <el-alert v-if="plan.warnings?.length" class="plan-warning" type="warning" :closable="false" :title="plan.warnings.join('；')" />
    </section>

    <section v-if="mode !== 'explain'" class="summary-card">
      <header><div><b>出口 IP 汇总</b><small>{{ uniqueEgress.length }} 个不同出口</small></div><small v-if="lastTestedAt">更新于 {{ lastTestedAt.toLocaleTimeString() }}</small></header>
      <div v-if="uniqueEgress.length" class="egress-grid">
        <article v-for="item in uniqueEgress" :key="item.ip"><span class="flag">{{ countryFlag(item.countryCode) }}</span><div><strong>{{ maskIP(item.ip) }}</strong><small>{{ item.countryCode || '未知地区' }} · {{ item.services.length }} 个目标</small></div></article>
      </div>
      <el-empty v-else :image-size="48" description="正在收集出口信息" />
    </section>

    <section v-if="mode !== 'explain'" class="result-card">
      <header><div><b>网站分流结果</b><small>{{ mode === 'template' ? '按订阅模板规则选择节点并验证' : mode === 'subscription' ? '通过所选节点检测，最多 4 项并发' : '浏览器直连检测，最多 4 项并发' }}</small></div><el-tag type="info" effect="plain">失败不等于网站不可用</el-tag></header>
      <el-table :data="results" row-key="domain" class="result-table">
        <el-table-column label="目标" min-width="190"><template #default="{ row }"><div class="target-cell"><span class="target-title"><img class="site-icon" :src="faviconUrl(row.domain)" @error="iconFallback($event, row.icon)"><i class="target-icon" style="display:none">{{ row.icon }}</i>{{ row.name }}</span><small>{{ row.domain }}</small></div></template></el-table-column>
        <el-table-column prop="group" label="类型" width="90"><template #default="{ row }"><el-tag size="small" effect="plain">{{ row.group }}</el-tag></template></el-table-column>
        <el-table-column label="地区" width="100"><template #default="{ row }"><span v-if="row.countryCode">{{ countryFlag(row.countryCode) }} {{ row.countryCode }}</span><span v-else>--</span></template></el-table-column>
        <el-table-column label="出口 IP" min-width="170"><template #default="{ row }"><span class="mono">{{ maskIP(row.ip) }}</span></template></el-table-column>
        <el-table-column label="响应" width="90"><template #default="{ row }">{{ row.rtt >= 0 ? `${row.rtt}ms` : '--' }}</template></el-table-column>
        <el-table-column label="状态" min-width="190"><template #default="{ row }"><el-tag :type="statusType(row)" size="small" :effect="row.status === 'testing' ? 'plain' : 'light'">{{ statusText(row) }}</el-tag><small v-if="row.note" class="status-note">{{ row.note }}</small></template></el-table-column>
      </el-table>
      <p class="privacy-note">检测结果仅用于当前页面展示，SubLinkX 不持久化出口 IP。本机模式下部分目标关闭 CORS 时会显示“未开放跨域检测”。</p>
    </section>

    <el-drawer v-model="targetDrawer" title="分流检测目标" size="720px">
      <el-alert type="info" :closable="false" title="目标只定义如何检测，不配置期望国家；期望地区始终由当前模板规则与策略组推导。" />
      <div class="target-admin-form">
        <el-input v-model="targetForm.key" placeholder="key，例如 youtube" :disabled="!!targetForm.id" />
        <el-input v-model="targetForm.name" placeholder="名称" />
        <el-input v-model="targetForm.domain" placeholder="域名，例如 www.youtube.com" />
        <el-input v-model="targetForm.group" placeholder="分类，例如 media" />
        <el-input v-model="targetForm.icon" placeholder="图标/Emoji" />
        <el-input v-model="targetForm.path" placeholder="检测路径，例如 /cdn-cgi/trace" />
        <el-select v-model="targetForm.method"><el-option label="GET" value="GET" /><el-option label="HEAD" value="HEAD" /></el-select>
        <el-input v-model="targetForm.expectedStatus" placeholder="期望状态，如 200-399 或 200,204" />
        <el-input v-model="targetForm.responseContains" placeholder="响应必须包含（可选）" />
        <el-input-number v-model="targetForm.timeoutSeconds" :min="1" :max="60" controls-position="right" />
        <el-input-number v-model="targetForm.retries" :min="0" :max="5" controls-position="right" />
        <el-input-number v-model="targetForm.sortOrder" :min="0" :max="9999" controls-position="right" />
        <el-switch v-model="targetForm.requireEgressIp" active-text="要求出口 IP" />
        <el-switch v-model="targetForm.enabled" active-text="启用" />
      </div>
      <div class="target-admin-actions"><el-button @click="newTarget">新建</el-button><el-button type="primary" :loading="targetSaving" @click="persistTarget">保存目标</el-button></div>
      <el-table :data="targets" size="small" class="target-admin-table">
        <el-table-column prop="name" label="名称" min-width="110" />
        <el-table-column prop="domain" label="域名" min-width="180" />
        <el-table-column prop="group" label="分类" width="90" />
        <el-table-column label="状态" width="76"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="120"><template #default="{ row }"><el-button link type="primary" @click="editTarget(row)">编辑</el-button><el-button link type="danger" @click="removeTarget(row)">删除</el-button></template></el-table-column>
      </el-table>
    </el-drawer>
    <el-drawer v-model="matrixDrawer" title="节点 × 场景质量矩阵（24h）" size="88%">
      <el-alert type="info" :closable="false" title="真实目标检测会沉淀为矩阵样本；模板选节点优先使用具体目标历史，其次同场景历史，最后回退 TCP 总质量。" />
      <div class="matrix-actions"><span>自动：每 6 小时对在线节点轮换采样每个场景的代表目标。</span><el-button :loading="matrixSampling" @click="sampleMatrix('scene')">立即场景采样</el-button><el-button :loading="matrixSampling" @click="sampleMatrix('full')">完整采样 16 个目标</el-button><el-button @click="openMatrix">刷新矩阵</el-button></div>
      <el-table v-loading="matrixLoading" :data="matrixData.nodes || []" size="small" height="calc(100vh - 180px)" class="matrix-table">
        <el-table-column prop="name" label="节点" fixed min-width="190" />
        <el-table-column v-for="scene in matrixScenes" :key="scene" :label="groupLabels[scene] || scene" min-width="150">
          <template #default="{ row }">
            <div v-if="matrixStat(row, scene)?.sampleCount" class="matrix-cell">
              <b>{{ matrixStat(row, scene).score }} 分</b>
              <span>{{ matrixStat(row, scene).availability }}% · {{ matrixStat(row, scene).averageRtt >= 0 ? matrixStat(row, scene).averageRtt + 'ms' : '--' }}</span>
              <small>{{ matrixStat(row, scene).sampleCount }} 样本 · 置信 {{ matrixStat(row, scene).confidence }}%</small>
            </div>
            <span v-else class="matrix-empty">暂无样本</span>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>
  </div>
</template>

<style scoped>
.target-title{display:flex;align-items:center;gap:7px}.target-icon{display:grid;place-items:center;width:24px;height:24px;border-radius:7px;background:var(--el-fill-color-light);font-size:14px;font-style:normal}.status-note{display:block;margin-left:8px;color:var(--el-text-color-secondary);font-size:10px}
.site-icon{width:24px;height:24px;border-radius:7px;object-fit:contain;background:var(--el-fill-color-light);padding:3px}.plan-card{padding:20px;margin-bottom:16px;border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color);box-shadow:var(--el-box-shadow-light)}.plan-card>header{display:flex;align-items:center;justify-content:space-between;margin-bottom:14px}.plan-card header div{display:flex;flex-direction:column;gap:3px}.plan-card header small,.plan-card td small{color:var(--el-text-color-secondary);font-size:11px}.rule-text{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:11px}.plan-warning{margin-top:12px}
.target-title{display:flex;align-items:center;gap:7px}.target-icon{display:grid;place-items:center;width:24px;height:24px;border-radius:7px;background:var(--el-fill-color-light);font-size:14px;font-style:normal}.status-note{display:block;margin-left:8px;color:var(--el-text-color-secondary);font-size:10px}
.source-card{padding:16px 20px;margin-bottom:16px;border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color);box-shadow:var(--el-box-shadow-light)}.source-selectors{display:grid;grid-template-columns:minmax(220px,1fr) 24px minmax(260px,1.4fr) auto;align-items:center;gap:10px;margin-top:14px}.source-selectors .arrow{text-align:center;color:var(--el-text-color-placeholder)}.source-selectors :deep(.el-select-dropdown__item){display:flex;justify-content:space-between}.source-selectors small{margin-left:14px;color:var(--el-text-color-secondary)}
.egress-page{width:min(1120px,100%);margin:0 auto;padding:28px}.hero-card,.summary-card,.result-card{border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color);box-shadow:var(--el-box-shadow-light)}.hero-card{display:flex;align-items:flex-end;justify-content:space-between;gap:30px;padding:28px;margin-bottom:16px;background:radial-gradient(circle at 90% 0,color-mix(in srgb,var(--el-color-primary) 12%,transparent),transparent 42%),var(--el-bg-color)}.eyebrow{color:var(--el-color-primary);font-family:monospace;font-size:11px;font-weight:800;letter-spacing:.14em}.hero-card h1{margin:8px 0 7px;font-size:30px;letter-spacing:-.03em}.hero-card p{max-width:700px;margin:0;color:var(--el-text-color-secondary);font-size:13px;line-height:1.7}.hero-actions{display:flex;align-items:center;gap:16px;flex-shrink:0}.summary-card,.result-card{padding:20px;margin-bottom:16px}.summary-card>header,.result-card>header{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:14px}.summary-card header>div,.result-card header>div{display:flex;flex-direction:column;gap:3px}.summary-card header small,.result-card header small{color:var(--el-text-color-secondary);font-size:11px}.egress-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px}.egress-grid article{display:flex;align-items:center;gap:11px;padding:13px;border:1px solid var(--el-border-color-lighter);border-radius:11px;background:var(--el-fill-color-extra-light)}.egress-grid .flag{font-size:23px}.egress-grid article div,.target-cell{display:flex;min-width:0;flex-direction:column}.egress-grid strong,.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.egress-grid small,.target-cell small{overflow:hidden;color:var(--el-text-color-secondary);font-size:11px;text-overflow:ellipsis;white-space:nowrap}.target-cell span{font-weight:650}.privacy-note{margin:14px 0 0;color:var(--el-text-color-secondary);font-size:11px;line-height:1.6}.result-table{width:100%}@media(max-width:720px){.egress-page{padding:14px}.hero-card{align-items:flex-start;flex-direction:column;padding:20px}.hero-actions{width:100%;justify-content:space-between}.result-card{padding:12px}.summary-card>header,.result-card>header{align-items:flex-start;flex-direction:column}}
@media(max-width:720px){.source-selectors{grid-template-columns:1fr}.source-selectors .arrow{display:none}}
.template-selectors{grid-template-columns:minmax(260px,420px) 1fr}.template-selectors :deep(.el-alert){margin:0}
.explain-query{display:grid;grid-template-columns:2fr 1.4fr 150px 120px;gap:10px;margin-bottom:14px}.explain-flow{display:flex;align-items:center;flex-wrap:wrap;gap:8px;padding:14px;margin-bottom:14px;border:1px solid var(--el-border-color-lighter);border-radius:12px;background:var(--el-fill-color-extra-light)}.explain-flow span{padding:6px 9px;border-radius:8px;background:var(--el-bg-color);font-size:12px}.explain-flow .selected{color:var(--el-color-success);font-weight:700}.explain-flow b{color:var(--el-text-color-placeholder)}.explain-meta{margin-bottom:14px}.subhead{display:flex;align-items:center;justify-content:space-between;margin:16px 0 8px}.subhead small{color:var(--el-text-color-secondary)}@media(max-width:900px){.explain-query{grid-template-columns:1fr 1fr}}@media(max-width:600px){.explain-query{grid-template-columns:1fr}}
.target-admin-form{display:grid;grid-template-columns:1fr 1fr;gap:10px;margin:16px 0}.target-admin-actions{display:flex;justify-content:flex-end;gap:8px;margin-bottom:16px}.target-admin-table{margin-top:8px}@media(max-width:720px){.target-admin-form{grid-template-columns:1fr}}
.matrix-table{margin-top:14px}.matrix-cell{display:flex;flex-direction:column;gap:2px}.matrix-cell b{font-size:13px}.matrix-cell span,.matrix-cell small,.matrix-empty{color:var(--el-text-color-secondary);font-size:10px}
.matrix-actions{display:flex;align-items:center;justify-content:flex-end;gap:8px;margin-top:12px}.matrix-actions span{margin-right:auto;color:var(--el-text-color-secondary);font-size:11px}
</style>
