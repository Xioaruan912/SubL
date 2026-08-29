<script setup lang='ts'>
import { ref, reactive, computed, onMounted } from 'vue'
import { getSubs, getSubPreviewNodes, previewSubPipeline, AddSub, DelSub, UpdateSub, ResetToken, SetExpire } from "@/api/subcription/subs"
import { getTemp } from "@/api/subcription/temp"
import { getNodes, getNodeOverview, GetGroupFull } from "@/api/subcription/node"
import QrcodeVue from 'qrcode.vue'
import { VueDraggable } from 'vue-draggable-plus'
import { countryFlag } from "@/utils/flag"

defineOptions({ name: 'SubsPage' })

interface Sub {
  ID: number
  Name: string
  CreatedAt: string
  Config: string
  Token: string
  ExpiresAt: string | null
  Nodes: Node[]
  GroupRefs?: GroupRef[]
  Pipeline?: string
  SourceURLs?: string
  SubLogs: SubLogs[]
}
interface Node { ID: number; Name: string; Link: string; GroupNodes?: { Name: string }[] }
interface GroupRef { ID: number; Name: string; NodeCount: number }
interface Config { clash: string; surge: string; loon: string; udp: boolean; cert: boolean }
interface SubLogs { IP: string; Count: number; Addr: string; Date: string }
interface Temp { file: string }
interface OverviewItem {
  id: number; name: string; server: string; country: string;
  countryCode: string; rtt: number; groups: string[]
}

const tableData = ref<Sub[]>([])
const searchKey = ref('')
const Clash = ref('')
const Surge = ref('')
const Loon = ref('')
const clashMode = ref('1')
const surgeMode = ref('1')
const loonMode = ref('1')
const SubTitle = ref('')
const Subname = ref('')
const oldSubname = ref('')
const airportUrl = ref('')
const isAirportUrl = ref(false)
const dialogVisible = ref(false)
const NodesList = ref<Node[]>([])
const value1 = ref<string[]>([])
const allGroups = ref<GroupRef[]>([])
const selectedGroups = ref<number[]>([])
const openedNodeGroups = ref<string[]>([])
const groupedNodes = computed(() => {
  const by = new Map<string, Node[]>()
  for (const n of NodesList.value) {
    const names = (n.GroupNodes || []).map(g => g.Name).filter(Boolean)
    for (const name of (names.length ? names : ['未分组'])) {
      if (!by.has(name)) by.set(name, [])
      by.get(name)!.push(n)
    }
  }
    return [...by.entries()].filter(([name]) => name !== '未分组').map(([name, nodes]) => ({ name, nodes }))
    .sort((a, b) => a.name === '未分组' ? 1 : b.name === '未分组' ? -1 : a.name.localeCompare(b.name, 'zh-CN'))
})
const udpOn = ref(false)
const certOn = ref(false)
const qrcode = ref('')
const qrTitle = ref('')
const qrDialog = ref(false)
const templist = ref<Temp[]>([])
const editingSubId = ref(0)
const pipeline = reactive({ include: '', exclude: '', renamePattern: '', renameReplacement: '', protocols: [] as string[], sort: 'original', dedupe: true, maxNodes: 0 })
const pipelinePreview = ref<any>(null)
const pipelineLoading = ref(false)
const pipelineJSON = () => JSON.stringify(pipeline)
const resetPipeline = (raw = '') => {
  const value = { include: '', exclude: '', renamePattern: '', renameReplacement: '', protocols: [], sort: 'original', dedupe: true, maxNodes: 0 }
  try { Object.assign(value, JSON.parse(raw || '{}')) } catch { /* use defaults */ }
  Object.assign(pipeline, value); pipelinePreview.value = null
}
const runPipelinePreview = async () => {
  if (!editingSubId.value) { ElMessage.info('新增订阅保存后即可预览真实处理结果'); return }
  pipelineLoading.value = true
  try { const { data } = await previewSubPipeline({ id: editingSubId.value, pipeline: pipelineJSON() }); pipelinePreview.value = data } finally { pipelineLoading.value = false }
}

// 抽屉
const drawerVisible = ref(false)
const drawerLoading = ref(false)
const drawerSub = ref<Sub | null>(null)
const overview = ref<OverviewItem[]>([])
const expirePick = ref('')
const expireClear = ref(false)

// 每卡客户端 tab
const activeClient = ref<Record<number, string>>({})
const clientOptions = [
  { label: '自动识别', value: 'auto' },
  { label: 'V2Ray', value: 'v2ray' },
  { label: 'Clash', value: 'clash' },
  { label: 'Surge', value: 'surge' },
  { label: 'Loon', value: 'loon' },
]

const filteredSubs = computed(() => {
  const kw = searchKey.value.trim().toLowerCase()
  if (!kw) return tableData.value
  return tableData.value.filter(s => s.Name.toLowerCase().includes(kw))
})

const serverAddress = () => {
  return location.protocol + '//' + location.hostname + (location.port ? ':' + location.port : '')
}

const subUrl = (sub: Sub, client: string) => {
  const base = `${serverAddress()}/c/?token=${sub.Token}`
  return client === 'auto' ? base : `${base}&client=${client}`
}

const currentUrl = (sub: Sub) => {
  return subUrl(sub, activeClient.value[sub.ID] || 'auto')
}

const loading = ref(false)

async function getsubs() {
  loading.value = true
  try {
    const { data } = await getSubs()
    tableData.value = data || []
  } finally {
    loading.value = false
  }
}
async function gettemps() {
  const { data } = await getTemp()
  templist.value = data || []
}
onMounted(async () => {
  await getsubs()
  gettemps()
  const { data } = await getNodes()
  NodesList.value = data || []
  try {
    const gp = await GetGroupFull()
    allGroups.value = Array.isArray(gp.data) ? gp.data.map((g: any) => ({ ID: g.id, Name: g.name, NodeCount: g.node_count || 0 })) : []
  } catch { /* ignore */ }
})

// ---- 添加/编辑 ----
const addSubs = async () => {
  if (!Subname.value.trim()) {
    ElMessage.warning("订阅名称不能为空")
    return
  }
  if (!isAirportUrl.value && value1.value.length === 0 && selectedGroups.value.length === 0) {
    ElMessage.warning("请选择至少一个节点或分组")
    return
  }
  if (isAirportUrl.value && !airportUrl.value.trim()) {
    ElMessage.warning("机场订阅链接不能为空")
    return
  }

  const config = JSON.stringify({
    "clash": Clash.value.trim(),
    "surge": Surge.value.trim(),
    "loon": Loon.value.trim(),
    "udp": udpOn.value,
    "cert": certOn.value
  })
  const groupStr = selectedGroups.value.join(',')
  if (SubTitle.value === '添加订阅') {
    await AddSub({ config, name: Subname.value.trim(), nodes: value1.value.join(','), groups: groupStr, airport_url: isAirportUrl.value ? airportUrl.value.trim() : '', pipeline: pipelineJSON() })
    ElMessage.success("添加成功")
  } else {
    await UpdateSub({ config, name: Subname.value.trim(), nodes: value1.value.join(','), groups: groupStr, oldname: oldSubname.value, airport_url: isAirportUrl.value ? airportUrl.value.trim() : '', pipeline: pipelineJSON() })
    ElMessage.success("更新成功")
  }
  getsubs()
  dialogVisible.value = false
}

const handleAddSub = () => {
  SubTitle.value = '添加订阅'
  Subname.value = ''
  oldSubname.value = ''
  airportUrl.value = ''
  isAirportUrl.value = false
  udpOn.value = false
  certOn.value = false
  Clash.value = './template/clash.yaml'
  Surge.value = './template/surge.conf'
  Loon.value = './template/loon.conf'
  clashMode.value = '1'
  surgeMode.value = '1'
  loonMode.value = '1'
  value1.value = []
  selectedGroups.value = []
  editingSubId.value = 0
  resetPipeline()
  dialogVisible.value = true
}

const parseConfig = (raw: string): Config => {
  try {
    const c = JSON.parse(raw || '{}')
    return { clash: c.clash || '', surge: c.surge || '', loon: c.loon || '', udp: !!c.udp, cert: !!c.cert }
  } catch {
    return { clash: '', surge: '', loon: '', udp: false, cert: false }
  }
}

const handleEdit = (sub: Sub) => {
  const config = parseConfig(sub.Config)
  SubTitle.value = '编辑订阅'
  Subname.value = sub.Name
  oldSubname.value = sub.Name
  airportUrl.value = sub.SourceURLs || ''
  isAirportUrl.value = !!sub.SourceURLs
  udpOn.value = !!config.udp
  certOn.value = !!config.cert
  Clash.value = config.clash
  Surge.value = config.surge
  clashMode.value = config.clash.startsWith('./template/') ? '1' : '2'
  surgeMode.value = config.surge.startsWith('./template/') ? '1' : '2'
  loonMode.value = config.loon.startsWith('./template/') ? '1' : '2'
  value1.value = sub.Nodes.map(n => n.Name)
  selectedGroups.value = (sub.GroupRefs || []).map(g => g.ID)
  editingSubId.value = sub.ID
  resetPipeline(sub.Pipeline || '')
  dialogVisible.value = true
}

// 移除已选节点
const removeNode = (name: string) => {
  value1.value = value1.value.filter(n => n !== name)
}

// 全选全部节点
const selectAllNodes = () => {
  const s = new Set(value1.value)
  for (const n of NodesList.value) s.add(n.Name)
  value1.value = [...s]
}
const selectGroupNodes = (nodes: Node[]) => {
  const s = new Set(value1.value)
  nodes.forEach(n => s.add(n.Name))
  value1.value = [...s]
}

// 一键清空已选节点
const clearAllNodes = () => {
  value1.value = []
}

// ---- 删除 ----
const handleDel = (sub: Sub) => {
  ElMessageBox.confirm(`你是否要删除 ${sub.Name} ?`, '提示', {
    confirmButtonText: 'OK', cancelButtonText: 'Cancel', type: 'warning',
  }).then(async () => {
    await DelSub({ id: sub.ID })
    getsubs()
    ElMessage.success('删除成功')
  })
}

// ---- 链接 ----
const copyUrl = (url: string) => {
  const textarea = document.createElement('textarea')
  textarea.value = url
  document.body.appendChild(textarea)
  textarea.select()
  try {
    const successful = document.execCommand('copy')
    ElMessage({ type: successful ? 'success' : 'warning', message: successful ? '复制成功！' : '复制失败！' })
  } catch {
    ElMessage({ type: 'warning', message: '复制失败！' })
  } finally {
    document.body.removeChild(textarea)
  }
}

const handleQrcode = (url: string, title: string) => {
  qrcode.value = url
  qrTitle.value = title
  qrDialog.value = true
}

const openUrl = (url: string) => {
  window.open(url)
}

// ---- 重置链接 ----
const handleReset = (sub: Sub) => {
  ElMessageBox.confirm(`重置后 ${sub.Name} 的旧订阅链接将立即失效，确定重置？`, '重置订阅链接', {
    confirmButtonText: '重置', cancelButtonText: '取消', type: 'warning',
  }).then(async () => {
    const { data } = await ResetToken({ id: sub.ID })
    sub.Token = data?.token || sub.Token
    ElMessage.success('订阅链接已重置')
  })
}

// ---- 过期 ----
const expireInfo = (sub: Sub) => {
  if (!sub.ExpiresAt) return { label: '永不过期', type: 'info' as const }
  const t = new Date(sub.ExpiresAt).getTime()
  if (isNaN(t) || t < Date.now()) return { label: '已过期', type: 'danger' as const }
  const days = Math.ceil((t - Date.now()) / 86400000)
  return { label: days <= 1 ? '今日过期' : `${days} 天后过期`, type: 'warning' as const }
}

const fmtTime = (s: string) => {
  if (!s) return ''
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const previewNodesList = ref<Node[]>([])

import { testLocalAll } from "@/utils/ping"

// ---- 抽屉 ----
const openDrawer = async (sub: Sub) => {
  drawerSub.value = sub
  drawerVisible.value = true
  drawerLoading.value = true
  expireClear.value = !sub.ExpiresAt
  expirePick.value = sub.ExpiresAt || ''
  previewNodesList.value = []

  try {
    const [ovRes, previewRes] = await Promise.all([
      overview.value.length === 0 ? getNodeOverview() : Promise.resolve({ data: overview.value }),
      getSubPreviewNodes(sub.ID)
    ])
    if (overview.value.length === 0) overview.value = ovRes.data || []
    previewNodesList.value = previewRes.data || []

    const names = new Set(previewNodesList.value.map(n => n.Name))
    overview.value.forEach(o => {
      if (names.has(o.name)) {
        o.rtt = o.rtt === -1 ? -1 : -2;
      }
    })

    const nodesToTest = overview.value.filter(o => names.has(o.name));
    testLocalAll(nodesToTest, (index, rtt) => {
      const targetNode = nodesToTest[index];
      const ovIndex = overview.value.findIndex(o => o.name === targetNode.name);
      if (ovIndex !== -1) {
        overview.value[ovIndex].rtt = rtt;
      }
    });

  } catch { /* ignore */ } finally {
    drawerLoading.value = false
  }
}

// 将拉取到的所有真实节点按国家分组
const drawerNodesByCountry = computed(() => {
  if (!previewNodesList.value.length) return []
  // 取所有的名字
  const names = new Set(previewNodesList.value.map(n => n.Name))
  
  // 从 overview 中找到对应的数据
  const mapped = overview.value.filter(o => names.has(o.name))
  
  // 按国家聚合
  const byCountry: Record<string, OverviewItem[]> = {}
  for (const n of mapped) {
    const key = n.country || '未知'
    if (!byCountry[key]) byCountry[key] = []
    byCountry[key].push(n)
  }
  return Object.keys(byCountry).sort().map(c => ({ country: c, items: byCountry[c] }))
})

const drawerGroupRefs = computed(() => {
  const sub = drawerSub.value
  if (!sub) return []
  return (sub.GroupRefs || []).map(g => ({ ...g }))
})

const rttLabel = (rtt: number) => rtt === -2 ? '测试中…' : (rtt < 0 ? '不可达' : `${rtt}ms`)
const rttColor = (rtt: number) => {
  if (rtt === -2) return 'info' as const
  if (rtt < 0) return 'danger' as const
  if (rtt < 100) return 'success' as const
  if (rtt < 200) return 'warning' as const
  return 'danger' as const
}

const saveExpire = async () => {
  const sub = drawerSub.value
  if (!sub) return
  const expire = expireClear.value ? '' : String(Math.floor(new Date(expirePick.value).getTime() / 1000))
  await SetExpire({ id: sub.ID, expire })
  sub.ExpiresAt = expireClear.value ? null : new Date(expirePick.value).toISOString()
  ElMessage.success('过期时间已更新')
}
</script>

<template>
  <div class="subs-page">
    <!-- 顶部工具栏 -->
    <div class="toolbar">
      <el-input v-model="searchKey" placeholder="搜索订阅名称…" clearable class="search" />
      <el-button type="primary" @click="handleAddSub">添加订阅</el-button>
    </div>

    <!-- 订阅卡片 -->
    <div v-loading="loading" class="subs-loading-area">
      <el-empty v-if="!loading && !filteredSubs.length" description="暂无订阅，点击右上角「添加订阅」" />
      <div v-if="filteredSubs.length" class="card-grid">
      <el-card v-for="sub in filteredSubs" :key="sub.ID" shadow="hover" class="sub-card">
        <template #header>
          <div class="card-head">
            <div class="card-title">
              <span class="sub-name" @click="openDrawer(sub)">{{ sub.Name }}</span>
              <el-tag :type="expireInfo(sub).type" size="small" effect="plain">{{ expireInfo(sub).label }}</el-tag>
            </div>
            <div class="card-stats">
              <el-tag size="small" type="info">{{ sub.Nodes.length }} 节点</el-tag>
              <el-tag size="small" type="primary">{{ (sub.SubLogs || []).reduce((a, l) => a + l.Count, 0) }} 次访问</el-tag>
            </div>
          </div>
        </template>

        <!-- 链接区 -->
        <div class="link-area">
          <el-segmented v-model="activeClient[sub.ID]" :options="clientOptions" size="small" class="client-tabs" />
          <div class="flex items-center gap-2">
            <el-input :model-value="currentUrl(sub)" readonly size="small" class="flex-1" />
            <el-button size="small" class="shrink-0" @click="copyUrl(currentUrl(sub))">复制</el-button>
            <el-button size="small" class="shrink-0" @click="handleQrcode(currentUrl(sub), sub.Name)">二维码</el-button>
          </div>
        </div>

        <!-- 操作 -->
        <div class="card-actions">
          <el-button link type="warning" size="small" @click="handleReset(sub)">重置链接</el-button>
          <el-button link type="primary" size="small" @click="openDrawer(sub)">详情</el-button>
          <el-button link type="primary" size="small" @click="handleEdit(sub)">编辑</el-button>
          <el-button link type="danger" size="small" @click="handleDel(sub)">删除</el-button>
        </div>
      </el-card>
      </div>
    </div>

    <!-- 二维码弹窗 -->
    <el-dialog v-model="qrDialog" width="300px" align-center :title="qrTitle">
      <div class="qr-body">
        <QrcodeVue :value="qrcode" :size="200" level="H" />
        <el-input :model-value="qrcode" readonly size="small" />
        <div class="qr-actions">
          <el-button size="small" @click="copyUrl(qrcode)">复制</el-button>
          <el-button size="small" @click="openUrl(qrcode)">打开</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 添加/编辑订阅弹窗 -->
    <el-dialog v-model="dialogVisible" :title="SubTitle" width="min(920px, 96vw)" align-center>
      <el-form label-position="top" class="sub-form">
        <!-- 分区一：基本信息 -->
        <el-divider content-position="left">基本信息</el-divider>
        <el-form-item label="订阅名称">
          <el-input v-model="Subname" placeholder="如：自建节点 · 港美日" clearable />
        </el-form-item>

        <!-- 分区二：模板与参数 -->
        <el-divider content-position="left">模板与参数</el-divider>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="Clash 模板">
              <el-radio-group v-model="clashMode" class="src-tabs" size="small">
                <el-radio-button value="1">本地</el-radio-button>
                <el-radio-button value="2">URL</el-radio-button>
              </el-radio-group>
              <el-select v-if="clashMode === '1'" v-model="Clash" placeholder="选择本地 clash 模板" class="full">
                <el-option v-for="t in templist" :key="t.file" :label="t.file" :value="'./template/' + t.file" />
              </el-select>
              <el-input v-else v-model="Clash" placeholder="粘贴远程模板链接 https://…" class="full" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Surge 模板">
              <el-radio-group v-model="surgeMode" class="src-tabs" size="small">
                <el-radio-button value="1">本地</el-radio-button>
                <el-radio-button value="2">URL</el-radio-button>
              </el-radio-group>
              <el-select v-if="surgeMode === '1'" v-model="Surge" placeholder="选择本地 surge 模板" class="full">
                <el-option v-for="t in templist" :key="t.file" :label="t.file" :value="'./template/' + t.file" />
              </el-select>
              <el-input v-else v-model="Surge" placeholder="粘贴远程模板链接 https://…" class="full" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Loon 模板">
              <el-radio-group v-model="loonMode" class="src-tabs" size="small">
                <el-radio-button value="1">本地</el-radio-button>
                <el-radio-button value="2">URL</el-radio-button>
              </el-radio-group>
              <el-select v-if="loonMode === '1'" v-model="Loon" placeholder="选择本地 loon 模板" class="full">
                <el-option v-for="t in templist" :key="t.file" :label="t.file" :value="'./template/' + t.file" />
              </el-select>
              <el-input v-else v-model="Loon" placeholder="粘贴远程模板链接 https://…" class="full" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <div class="switch-item">
              <div class="switch-main">
                <span class="switch-label">UDP</span>
                <el-switch v-model="udpOn" />
              </div>
              <div class="switch-hint">提升 UDP 转发支持</div>
            </div>
          </el-col>
          <el-col :span="12">
            <div class="switch-item">
              <div class="switch-main">
                <span class="switch-label">跳过证书</span>
                <el-switch v-model="certOn" />
              </div>
              <div class="switch-hint">关闭 TLS 证书校验</div>
            </div>
          </el-col>
        </el-row>

        <!-- 分区三：节点与机场订阅 -->
        <el-divider content-position="left">节点来源</el-divider>
        <div class="mb-4">
          <el-radio-group v-model="isAirportUrl" class="w-full" style="display: flex;">
            <el-radio-button :value="false" style="flex: 1;" class="text-center">本地配置 (手动选择/关联分组)</el-radio-button>
            <el-radio-button :value="true" style="flex: 1;" class="text-center">机场订阅导入</el-radio-button>
          </el-radio-group>
        </div>

        <template v-if="isAirportUrl">
          <el-form-item label="机场订阅源（支持主源 + 备用源）">
            <el-input v-model="airportUrl" type="textarea" :rows="3" placeholder="每行一个订阅链接；主源失败时自动尝试下一行" clearable />
            <div class="text-xs text-gray-500 mt-1">按顺序尝试，首个可用的 2xx 非空响应生效。</div>
          </el-form-item>
        </template>
        
        <template v-else>
          <el-row :gutter="24">
            <el-col :span="12">
              <el-form-item label="选择具体节点">
                <div class="node-pick-tools">
                  <el-button link type="primary" size="small" @click="selectAllNodes">全选节点</el-button>
                  <el-button link type="danger" size="small" @click="clearAllNodes" :disabled="!value1.length">清空已选</el-button>
                </div>
                <el-collapse v-model="openedNodeGroups" class="node-group-picker">
                  <el-collapse-item v-for="group in groupedNodes" :key="group.name" :name="group.name">
                    <template #title>
                      <span class="picker-group-title">{{ group.name }}</span><span class="picker-group-count">{{ group.nodes.length }} 个节点</span>
                    </template>
                    <div class="picker-node-list">
                      <el-checkbox v-for="item in group.nodes" :key="item.Name" :model-value="value1.includes(item.Name)" @change="(checked: any) => checked ? value1.push(item.Name) : removeNode(item.Name)">
                        {{ item.Name }}
                      </el-checkbox>
                    </div>
                    <el-button link type="primary" size="small" @click="selectGroupNodes(group.nodes)">选择本组全部</el-button>
                  </el-collapse-item>
                </el-collapse>
              </el-form-item>
              <div class="field-label">已选节点（可拖拽排序）</div>
              <VueDraggable v-model="value1" :animation="150" ghost-class="ghost" class="order-list">
                <div v-for="(nodeName, index) in value1" :key="nodeName" class="order-item">
                  <span class="order-badge">{{ index + 1 }}</span>
                  <span class="order-name">{{ nodeName }}</span>
                  <span class="order-drag">☰</span>
                  <el-icon class="order-remove" @click="removeNode(nodeName)"><svg viewBox="0 0 1024 1024"><path fill="currentColor" d="M764.288 214.592 512 466.88 259.712 214.592a31.936 31.936 0 0 0-45.12 45.12L466.752 512 214.528 764.224a31.936 31.936 0 1 0 45.12 45.184L512 557.184l252.288 252.288a31.936 31.936 0 0 0 45.12-45.12L557.12 512.064l252.288-252.352a31.936 31.936 0 1 0-45.12-45.184z"/></svg></el-icon>
                </div>
              </VueDraggable>
              <div v-if="!value1.length" class="order-empty">尚未选择节点，该订阅将不含节点</div>
            </el-col>

            <el-col :span="12">
              <el-form-item label="关联节点分组">
                <div class="group-pick" style="width: 100%;">
                  <el-checkbox-group v-model="selectedGroups" class="group-checkbox-list">
                    <el-checkbox v-for="g in allGroups" :key="g.ID" :value="g.ID" class="group-checkbox">
                      <span class="gc-name">{{ g.Name }}</span>
                      <span class="gc-count">{{ g.NodeCount }} 节点</span>
                    </el-checkbox>
                  </el-checkbox-group>
                  <el-empty v-if="!allGroups.length" description="暂无分组" :image-size="40" />
                  <div v-if="selectedGroups.length" class="text-xs text-orange-500 mt-2">
                    已关联 {{ selectedGroups.length }} 个分组，订阅拉取时自动展开其全部节点（随机场重新同步后自动更新）。
                  </div>
                </div>
              </el-form-item>
            </el-col>
          </el-row>
        </template>

        <el-divider content-position="left">节点处理链</el-divider>
        <div class="pipeline-card">
          <el-row :gutter="12">
            <el-col :span="12" :xs="24"><el-form-item label="包含名称（正则）"><el-input v-model="pipeline.include" placeholder="例如 香港|日本" clearable /></el-form-item></el-col>
            <el-col :span="12" :xs="24"><el-form-item label="排除名称（正则）"><el-input v-model="pipeline.exclude" placeholder="例如 剩余|官网" clearable /></el-form-item></el-col>
            <el-col :span="12" :xs="24"><el-form-item label="重命名正则"><el-input v-model="pipeline.renamePattern" placeholder="例如 ^\[机场A\]\s*" clearable /></el-form-item></el-col>
            <el-col :span="12" :xs="24"><el-form-item label="替换为"><el-input v-model="pipeline.renameReplacement" placeholder="留空表示删除匹配内容" clearable /></el-form-item></el-col>
            <el-col :span="12" :xs="24"><el-form-item label="协议过滤"><el-select v-model="pipeline.protocols" multiple clearable placeholder="全部协议" class="full"><el-option v-for="p in ['ss','ssr','vmess','vless','trojan','hysteria2','tuic']" :key="p" :label="p" :value="p" /></el-select></el-form-item></el-col>
            <el-col :span="8" :xs="16"><el-form-item label="排序"><el-select v-model="pipeline.sort" class="full"><el-option label="保留原顺序" value="original"/><el-option label="名称" value="name"/><el-option label="国家/地区" value="country"/><el-option label="低延迟优先" value="latency"/><el-option label="质量分优先" value="quality"/></el-select></el-form-item></el-col>
            <el-col :span="4" :xs="8"><el-form-item label="最多节点"><el-input-number v-model="pipeline.maxNodes" :min="0" :max="9999" controls-position="right" /></el-form-item></el-col>
          </el-row>
          <div class="pipeline-footer"><el-checkbox v-model="pipeline.dedupe">按节点链接去重</el-checkbox><el-button :loading="pipelineLoading" @click="runPipelinePreview">预览处理结果</el-button></div>
          <el-alert v-if="pipelinePreview" type="success" :closable="false" show-icon><template #title>处理前 {{ pipelinePreview.before }} 个 → 处理后 {{ pipelinePreview.after }} 个</template><template #default><span v-for="(count, reason) in pipelinePreview.rejected" :key="reason" class="reject-stat">{{ reason }} {{ count }}</span></template></el-alert>
        </div>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button text @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="addSubs">{{ SubTitle === '添加订阅' ? '添加订阅' : '保存修改' }}</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 抽屉详情 -->
    <el-drawer v-model="drawerVisible" :title="drawerSub?.Name" size="620px">
      <div v-if="drawerSub" class="drawer-body">
        <!-- 关联分组 -->
        <div v-if="drawerGroupRefs.length" class="section-title">关联分组（机场同步自动更新节点）</div>
        <div v-if="drawerGroupRefs.length" class="group-tags">
          <el-tag v-for="g in drawerGroupRefs" :key="g.ID" type="warning" effect="plain" size="small">
            {{ g.Name }} · {{ g.NodeCount }} 节点
          </el-tag>
        </div>

        <!-- 节点状态 -->
        <div class="section-title">包含节点详情（共 {{ previewNodesList.length }} 个）</div>
        <div v-loading="drawerLoading" style="min-height: 100px;">
          <div v-if="drawerNodesByCountry.length" class="country-node-groups">
            <div v-for="g in drawerNodesByCountry" :key="g.country" class="country-group">
              <div class="country-group-title">
                <span class="country-group-flag">{{ countryFlag(g.items[0].countryCode) }}</span>
                <span class="country-group-name">{{ g.country }}</span>
                <span class="country-group-count">（{{ g.items.length }}）</span>
              </div>
              <div class="node-list">
                <div v-for="n in g.items" :key="n.id" class="node-item">
                  <span class="node-name">{{ n.name }}</span>
                  <span class="node-country">{{ n.server }}</span>
                  <el-tag :type="rttColor(n.rtt)" size="small" effect="light">{{ rttLabel(n.rtt) }}</el-tag>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else-if="!drawerLoading" description="该订阅暂无节点" :image-size="50" />
        </div>

        <!-- 模板配置 -->
        <div class="section-title">模板配置</div>
        <div class="cfg-grid">
          <div class="cfg-item"><span class="cfg-label">Clash</span><span class="cfg-val">{{ parseConfig(drawerSub.Config).clash || '未配置' }}</span></div>
          <div class="cfg-item"><span class="cfg-label">Surge</span><span class="cfg-val">{{ parseConfig(drawerSub.Config).surge || '未配置' }}</span></div>
          <div class="cfg-item"><span class="cfg-label">Loon</span><span class="cfg-val">{{ parseConfig(drawerSub.Config).loon || '未配置' }}</span></div>
          <div class="cfg-item"><span class="cfg-label">强制</span><span class="cfg-val">{{ parseConfig(drawerSub.Config).udp ? 'UDP ' : '' }}{{ parseConfig(drawerSub.Config).cert ? '跳过证书' : '' }}</span></div>
        </div>

        <!-- 过期设置 -->
        <div class="section-title">过期设置</div>
        <div class="expire-row">
          <el-checkbox v-model="expireClear" @change="() => { if (expireClear) expirePick = '' }">永不过期</el-checkbox>
          <el-date-picker v-model="expirePick" type="datetime" :disabled="expireClear" placeholder="选择过期时间" style="width: 220px" />
          <el-button type="primary" size="small" @click="saveExpire">保存</el-button>
        </div>

        <!-- 访问记录 -->
        <div class="section-title">访问记录（{{ drawerSub.SubLogs?.length || 0 }}）</div>
        <el-table :data="drawerSub.SubLogs || []" border size="small">
          <el-table-column prop="IP" label="IP" />
          <el-table-column prop="Count" label="次数" width="70" />
          <el-table-column prop="Addr" label="来源" />
          <el-table-column prop="Date" label="最近时间" />
        </el-table>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.subs-page { padding: 10px; }
.toolbar { display: flex; gap: 10px; align-items: center; margin-bottom: 14px; }
.toolbar .search { width: 260px; }
.subs-loading-area { min-height: 400px; }
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 14px;
}
.sub-card { border-radius: 12px; }
.card-head { display: flex; flex-direction: column; gap: 6px; }
.card-title { display: flex; align-items: center; gap: 8px; }
.sub-name { font-weight: 600; font-size: 15px; cursor: pointer; }
.sub-name:hover { color: var(--el-color-primary); }
.card-stats { display: flex; gap: 6px; }
.link-area { margin-bottom: 6px; }
.client-tabs { margin-bottom: 8px; }
.link-row .el-input { width: 100%; }
.card-actions { display: flex; justify-content: flex-end; gap: 2px; border-top: 1px solid var(--el-border-color-lighter); padding-top: 8px; }
.qr-body { display: flex; flex-direction: column; align-items: center; gap: 10px; }
.qr-actions { display: flex; gap: 8px; }
.drawer-body .section-title { font-weight: 600; margin: 16px 0 8px; color: var(--el-text-color-primary); }
.group-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 4px; }
.node-list { display: flex; flex-direction: column; gap: 6px; }
.node-item { display: flex; align-items: center; gap: 8px; padding: 6px 10px; background: var(--el-fill-color-light); border-radius: 8px; }
.node-flag { font-size: 18px; }
.node-name { flex: 1; font-weight: 500; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.node-country { font-size: 12px; color: var(--el-text-color-secondary); }
.cfg-grid { display: flex; flex-direction: column; gap: 6px; }
.cfg-item { display: flex; gap: 8px; font-size: 13px; }
.cfg-label { color: var(--el-text-color-secondary); width: 48px; flex-shrink: 0; }
.cfg-val { word-break: break-all; }
.expire-row { display: flex; align-items: center; gap: 10px; }
.mt { margin-top: 12px; }
.field-label { font-size: 13px; color: var(--el-text-color-secondary); margin-bottom: 6px; }
.sub-form .el-divider { margin: 6px 0 16px; }
.sub-form .el-form-item { margin-bottom: 16px; }
.src-tabs { margin-bottom: 8px; display: block; }
.full { width: 100%; }
.switch-item {
  display: flex; flex-direction: column; gap: 4px;
  padding: 12px 14px; border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px; background: var(--el-fill-color-light); margin-bottom: 16px;
}
.switch-main { display: flex; align-items: center; justify-content: space-between; }
.switch-label { font-size: 13px; font-weight: 500; }
.switch-hint { font-size: 12px; color: var(--el-text-color-secondary); }
.order-list { display: flex; flex-direction: column; gap: 6px; max-height: 240px; overflow-y: auto; padding-right: 4px; }
.order-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 12px; background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter); border-radius: 8px;
  transition: background-color .15s;
}
.order-item:hover { background: var(--el-fill-color); }
.order-badge {
  width: 22px; height: 22px; border-radius: 50%; flex-shrink: 0;
  background: var(--el-color-primary-light-8); color: var(--el-color-primary);
  font-size: 12px; font-weight: 600; display: flex; align-items: center; justify-content: center;
}
html.dark .order-badge { background: var(--el-color-primary-light-3); color: #fff; }
.order-name { flex: 1; font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.order-drag { cursor: grab; color: var(--el-text-color-placeholder); font-size: 14px; }
.order-remove { cursor: pointer; color: var(--el-text-color-placeholder); font-size: 14px; }
.order-remove:hover { color: var(--el-color-danger); }
.order-empty { padding: 14px 0; text-align: center; font-size: 12px; color: var(--el-text-color-placeholder); }
.ghost { opacity: 0.5; background: var(--el-color-primary-light-8); }
.group-checkbox-list { display: flex; flex-direction: column; gap: 4px; max-height: 240px; overflow-y: auto; padding-right: 4px; }
.group-checkbox { margin-right: 0; width: 100%; padding: 4px 8px; border-radius: 6px; }
.group-checkbox:hover { background: var(--el-fill-color-light); }
.node-pick-tools { display: flex; align-items: center; gap: 8px; width: 100%; }
.node-pick-tools .el-select { flex: 1; }
.node-group-picker { width: 100%; margin-top: 8px; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; overflow: hidden; }
.node-group-picker :deep(.el-collapse-item__header) { padding: 0 10px; height: 38px; line-height: 38px; }
.picker-group-title { font-weight: 600; }
.picker-group-count { margin-left: 8px; color: var(--el-text-color-secondary); font-size: 12px; }
.picker-node-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 4px 8px; margin-bottom: 8px; }
.picker-node-list .el-checkbox { min-width: 0; margin-right: 0; overflow: hidden; }
.picker-node-list :deep(.el-checkbox__label) { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.country-node-groups { display: flex; flex-direction: column; gap: 16px; max-height: 400px; overflow-y: auto; padding-right: 4px; }
.country-group-title { display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 600; margin-bottom: 6px; color: var(--el-text-color-primary); }
.country-group-count { color: var(--el-text-color-secondary); font-size: 12px; font-weight: normal; }
.gc-name { font-size: 13px; }
.gc-count { margin-left: 8px; font-size: 12px; color: var(--el-text-color-secondary); }
.pipeline-card { padding:14px; border:1px solid var(--el-border-color-lighter); border-radius:10px; background:var(--el-fill-color-extra-light); }.pipeline-card .el-form-item { margin-bottom:12px; }.pipeline-footer { display:flex; justify-content:space-between; align-items:center; margin-bottom:10px; }.reject-stat { margin-right:12px; font-size:12px; }
</style>
