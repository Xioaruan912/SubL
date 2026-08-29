<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { countryFlag } from '@/utils/flag'
import { getSubs, getSubPreviewNodes } from '@/api/subcription/subs'
import { EgressTest, getNodeOverview } from '@/api/subcription/node'

defineOptions({ name: 'EgressTest' })

type Target = { name: string; domain: string; group: string; icon: string; tracePath?: string; ipOptional?: boolean }
type Result = Target & { status: 'pending' | 'testing' | 'done' | 'error'; ip: string; countryCode: string; rtt: number; note: string }

const targets: Target[] = [
  { name: 'Cloudflare', domain: 'www.cloudflare.com', group: '网络', icon: '☁️' },
  { name: 'ChatGPT', domain: 'chatgpt.com', group: 'AI', icon: '◉' },
  { name: 'OpenAI', domain: 'openai.com', group: 'AI', icon: '◎' },
  { name: 'Gemini', domain: 'gemini.google.com', group: 'AI', icon: '✨', tracePath: '/', ipOptional: true },
  { name: 'Claude', domain: 'claude.ai', group: 'AI', icon: '◌' },
  { name: 'Anthropic', domain: 'anthropic.com', group: 'AI', icon: '△' },
  { name: 'Discord', domain: 'gateway.discord.gg', group: '社交', icon: '♬' },
  { name: 'X', domain: 'x.com', group: '社交', icon: '𝕏' },
  { name: 'Medium', domain: 'medium.com', group: '内容', icon: 'M' },
  { name: 'Perplexity', domain: 'www.perplexity.ai', group: 'AI', icon: '✦' },
  { name: 'Coinbase', domain: 'coinbase.com', group: '金融', icon: '₿' },
  { name: 'Notion', domain: 'notion.so', group: '工具', icon: 'N' },
  { name: 'Cloudflare CDN', domain: 'cdnjs.cloudflare.com', group: '开发', icon: '☁️' },
  { name: 'npm Registry', domain: 'registry.npmjs.org', group: '开发', icon: 'npm' },
  { name: 'GitLab', domain: 'gitlab.com', group: '开发', icon: '◈' },
  { name: 'Crunchyroll', domain: 'crunchyroll.com', group: '媒体', icon: '▶' },
]

const results = ref<Result[]>(targets.map(item => ({ ...item, status: 'pending', ip: '', countryCode: '', rtt: -1, note: '' })))
const running = ref(false)
const lastTestedAt = ref<Date | null>(null)
const hideIP = ref(false)
const mode = ref<'subscription' | 'local'>('subscription')
const subscriptions = ref<any[]>([])
const selectedSubscription = ref<number | null>(null)
const subscriptionNodes = ref<any[]>([])
const selectedNode = ref<number | null>(null)
const nodesLoading = ref(false)

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

const detect = async (item: Result) => {
  item.status = 'testing'; item.note = ''; item.rtt = -1
  const controller = new AbortController(); const timer = window.setTimeout(() => controller.abort(), 6000)
  const started = performance.now()
  try {
    const response = await fetch(`https://${item.domain}${item.tracePath || '/cdn-cgi/trace'}?_=${Date.now()}`, { cache: 'no-store', signal: controller.signal })
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const text = await response.text()
    const trace = Object.fromEntries(text.trim().split('\n').map(line => { const index = line.indexOf('='); return index > 0 ? [line.slice(0, index), line.slice(index + 1)] : [line, ''] }))
    if (!trace.ip && !item.ipOptional) throw new Error('目标未返回出口 IP')
    item.ip = trace.ip || ''; item.countryCode = (trace.loc || '').toUpperCase(); item.rtt = Math.max(1, Math.round(performance.now() - started)); item.note = item.ip ? '' : '站点可达，未提供出口 IP'; item.status = 'done'
  } catch (error: any) {
    item.status = 'error'; item.note = error?.name === 'AbortError' ? '超时' : '目标未开放跨域检测'
  } finally { window.clearTimeout(timer) }
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
  results.value = targets.map(item => ({ ...item, status: 'pending', ip: '', countryCode: '', rtt: -1, note: '' }))
  try {
    const { data } = await EgressTest({ id: selectedNode.value })
    const names: Record<string,string> = { network:'网络', ai:'AI', social:'社交', content:'内容', finance:'金融', tools:'工具', developer:'开发', media:'媒体' }
    results.value = (data?.results || []).map((item: any) => ({ ...(targets.find(target => target.domain === item.domain) || { icon: '•' }), name:item.name, domain:item.domain, group:names[item.group] || item.group, status:item.status === 'available' || item.status === 'reachable' ? 'done' : 'error', ip:item.ip || '', countryCode:item.countryCode || '', rtt:item.rtt ?? -1, note:item.note || '未获取出口 IP' }))
    lastTestedAt.value = new Date()
  } finally { running.value = false }
}

const runCurrent = () => mode.value === 'subscription' ? runSelected() : runLocal()

const statusType = (item: any) => item.status === 'done' ? 'success' : item.status === 'error' ? 'danger' : 'info'
const statusText = (item: any) => item.status === 'done' ? '已获取' : item.status === 'testing' ? '检测中' : item.status === 'error' ? item.note : '等待'

onMounted(async () => {
  const { data } = await getSubs(); subscriptions.value = data || []
  if (subscriptions.value.length) { selectedSubscription.value = subscriptions.value[0].ID; await loadSubscriptionNodes() }
})
</script>

<template>
  <div class="egress-page">
    <section class="hero-card">
      <div><span class="eyebrow">SPLIT ROUTING INSPECTOR</span><h1>订阅分流与出口检测</h1><p>选择订阅及其中的具体节点，服务端会通过该节点访问各目标并返回真实出口 IP。也可切换为本机模式，检查当前浏览器的分流规则。</p></div>
      <div class="hero-actions"><el-switch v-model="hideIP" active-text="隐藏 IP" /><el-button type="primary" :loading="running" @click="runCurrent">开始检测</el-button></div>
    </section>

    <section class="source-card">
      <el-segmented v-model="mode" :options="[{label:'订阅节点检测',value:'subscription'},{label:'本机浏览器检测',value:'local'}]" />
      <div v-if="mode === 'subscription'" class="source-selectors">
        <el-select v-model="selectedSubscription" placeholder="选择订阅" filterable @change="loadSubscriptionNodes"><el-option v-for="sub in subscriptions" :key="sub.ID" :label="sub.Name" :value="sub.ID"><span>{{ sub.Name }}</span><small>{{ sub.Nodes?.length || 0 }} 个固定节点</small></el-option></el-select>
        <span class="arrow">→</span>
        <el-select v-model="selectedNode" placeholder="选择具体节点" filterable :loading="nodesLoading"><el-option v-for="item in subscriptionNodes" :key="item.ID" :label="item.Name" :value="item.ID"><span>{{ item.Name }}</span><small>质量 {{ item.quality?.score || 0 }} · {{ item.quality?.averageRtt >= 0 ? item.quality.averageRtt + 'ms' : '无样本' }}</small></el-option></el-select>
        <el-tag v-if="subscriptionNodes.length" type="success" effect="plain">默认质量最优</el-tag>
      </div>
      <el-alert v-else type="info" :closable="false" title="本机模式由浏览器直连目标网站；它反映当前设备的代理分流，不经过 SubLinkX 节点。" />
    </section>

    <section class="summary-card">
      <header><div><b>出口 IP 汇总</b><small>{{ uniqueEgress.length }} 个不同出口</small></div><small v-if="lastTestedAt">更新于 {{ lastTestedAt.toLocaleTimeString() }}</small></header>
      <div v-if="uniqueEgress.length" class="egress-grid">
        <article v-for="item in uniqueEgress" :key="item.ip"><span class="flag">{{ countryFlag(item.countryCode) }}</span><div><strong>{{ maskIP(item.ip) }}</strong><small>{{ item.countryCode || '未知地区' }} · {{ item.services.length }} 个目标</small></div></article>
      </div>
      <el-empty v-else :image-size="48" description="正在收集出口信息" />
    </section>

    <section class="result-card">
      <header><div><b>网站分流结果</b><small>{{ mode === 'subscription' ? '通过所选节点检测，最多 4 项并发' : '浏览器直连检测，最多 4 项并发' }}</small></div><el-tag type="info" effect="plain">失败不等于网站不可用</el-tag></header>
      <el-table :data="results" row-key="domain" class="result-table">
        <el-table-column label="目标" min-width="190"><template #default="{ row }"><div class="target-cell"><span class="target-title"><i class="target-icon">{{ row.icon }}</i>{{ row.name }}</span><small>{{ row.domain }}</small></div></template></el-table-column>
        <el-table-column prop="group" label="类型" width="90"><template #default="{ row }"><el-tag size="small" effect="plain">{{ row.group }}</el-tag></template></el-table-column>
        <el-table-column label="地区" width="100"><template #default="{ row }"><span v-if="row.countryCode">{{ countryFlag(row.countryCode) }} {{ row.countryCode }}</span><span v-else>--</span></template></el-table-column>
        <el-table-column label="出口 IP" min-width="170"><template #default="{ row }"><span class="mono">{{ maskIP(row.ip) }}</span></template></el-table-column>
        <el-table-column label="响应" width="90"><template #default="{ row }">{{ row.rtt >= 0 ? `${row.rtt}ms` : '--' }}</template></el-table-column>
        <el-table-column label="状态" min-width="190"><template #default="{ row }"><el-tag :type="statusType(row)" size="small" :effect="row.status === 'testing' ? 'plain' : 'light'">{{ statusText(row) }}</el-tag><small v-if="row.note" class="status-note">{{ row.note }}</small></template></el-table-column>
      </el-table>
      <p class="privacy-note">检测结果仅用于当前页面展示，SubLinkX 不持久化出口 IP。本机模式下部分目标关闭 CORS 时会显示“未开放跨域检测”。</p>
    </section>
  </div>
</template>

<style scoped>
.target-title{display:flex;align-items:center;gap:7px}.target-icon{display:grid;place-items:center;width:24px;height:24px;border-radius:7px;background:var(--el-fill-color-light);font-size:14px;font-style:normal}.status-note{display:block;margin-left:8px;color:var(--el-text-color-secondary);font-size:10px}
.target-title{display:flex;align-items:center;gap:7px}.target-icon{display:grid;place-items:center;width:24px;height:24px;border-radius:7px;background:var(--el-fill-color-light);font-size:14px;font-style:normal}.status-note{display:block;margin-left:8px;color:var(--el-text-color-secondary);font-size:10px}
.source-card{padding:16px 20px;margin-bottom:16px;border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color);box-shadow:var(--el-box-shadow-light)}.source-selectors{display:grid;grid-template-columns:minmax(220px,1fr) 24px minmax(260px,1.4fr) auto;align-items:center;gap:10px;margin-top:14px}.source-selectors .arrow{text-align:center;color:var(--el-text-color-placeholder)}.source-selectors :deep(.el-select-dropdown__item){display:flex;justify-content:space-between}.source-selectors small{margin-left:14px;color:var(--el-text-color-secondary)}
.egress-page{width:min(1120px,100%);margin:0 auto;padding:28px}.hero-card,.summary-card,.result-card{border:1px solid var(--el-border-color-lighter);border-radius:16px;background:var(--el-bg-color);box-shadow:var(--el-box-shadow-light)}.hero-card{display:flex;align-items:flex-end;justify-content:space-between;gap:30px;padding:28px;margin-bottom:16px;background:radial-gradient(circle at 90% 0,color-mix(in srgb,var(--el-color-primary) 12%,transparent),transparent 42%),var(--el-bg-color)}.eyebrow{color:var(--el-color-primary);font-family:monospace;font-size:11px;font-weight:800;letter-spacing:.14em}.hero-card h1{margin:8px 0 7px;font-size:30px;letter-spacing:-.03em}.hero-card p{max-width:700px;margin:0;color:var(--el-text-color-secondary);font-size:13px;line-height:1.7}.hero-actions{display:flex;align-items:center;gap:16px;flex-shrink:0}.summary-card,.result-card{padding:20px;margin-bottom:16px}.summary-card>header,.result-card>header{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:14px}.summary-card header>div,.result-card header>div{display:flex;flex-direction:column;gap:3px}.summary-card header small,.result-card header small{color:var(--el-text-color-secondary);font-size:11px}.egress-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px}.egress-grid article{display:flex;align-items:center;gap:11px;padding:13px;border:1px solid var(--el-border-color-lighter);border-radius:11px;background:var(--el-fill-color-extra-light)}.egress-grid .flag{font-size:23px}.egress-grid article div,.target-cell{display:flex;min-width:0;flex-direction:column}.egress-grid strong,.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace}.egress-grid small,.target-cell small{overflow:hidden;color:var(--el-text-color-secondary);font-size:11px;text-overflow:ellipsis;white-space:nowrap}.target-cell span{font-weight:650}.privacy-note{margin:14px 0 0;color:var(--el-text-color-secondary);font-size:11px;line-height:1.6}.result-table{width:100%}@media(max-width:720px){.egress-page{padding:14px}.hero-card{align-items:flex-start;flex-direction:column;padding:20px}.hero-actions{width:100%;justify-content:space-between}.result-card{padding:12px}.summary-card>header,.result-card>header{align-items:flex-start;flex-direction:column}}
@media(max-width:720px){.source-selectors{grid-template-columns:1fr}.source-selectors .arrow{display:none}}
</style>
