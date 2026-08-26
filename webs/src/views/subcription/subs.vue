<script setup lang='ts'>
import { ref, computed, onMounted } from 'vue'
import { getSubs, AddSub, DelSub, UpdateSub, ResetToken, SetExpire } from "@/api/subcription/subs"
import { getTemp } from "@/api/subcription/temp"
import { getNodes, getNodeOverview } from "@/api/subcription/node"
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
  SubLogs: SubLogs[]
}
interface Node { ID: number; Name: string; Link: string }
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
const dialogVisible = ref(false)
const NodesList = ref<Node[]>([])
const value1 = ref<string[]>([])
const udpOn = ref(false)
const certOn = ref(false)
const qrcode = ref('')
const qrTitle = ref('')
const qrDialog = ref(false)
const templist = ref<Temp[]>([])

// 抽屉
const drawerVisible = ref(false)
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

async function getsubs() {
  const { data } = await getSubs()
  tableData.value = data || []
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
})

// ---- 添加/编辑 ----
const addSubs = async () => {
  const config = JSON.stringify({
    "clash": Clash.value.trim(),
    "surge": Surge.value.trim(),
    "loon": Loon.value.trim(),
    "udp": udpOn.value,
    "cert": certOn.value
  })
  if (SubTitle.value === '添加订阅') {
    await AddSub({ config, name: Subname.value.trim(), nodes: value1.value.join(',') })
    ElMessage.success("添加成功")
  } else {
    await UpdateSub({ config, name: Subname.value.trim(), nodes: value1.value.join(','), oldname: oldSubname.value })
    ElMessage.success("更新成功")
  }
  getsubs()
  dialogVisible.value = false
}

const handleAddSub = () => {
  SubTitle.value = '添加订阅'
  Subname.value = ''
  oldSubname.value = ''
  udpOn.value = false
  certOn.value = false
  Clash.value = './template/clash.yaml'
  Surge.value = './template/surge.conf'
  Loon.value = './template/loon.conf'
  clashMode.value = '1'
  surgeMode.value = '1'
  loonMode.value = '1'
  value1.value = []
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
  udpOn.value = !!config.udp
  certOn.value = !!config.cert
  Clash.value = config.clash
  Surge.value = config.surge
  clashMode.value = config.clash.startsWith('./template/') ? '1' : '2'
  surgeMode.value = config.surge.startsWith('./template/') ? '1' : '2'
  loonMode.value = config.loon.startsWith('./template/') ? '1' : '2'
  value1.value = sub.Nodes.map(n => n.Name)
  dialogVisible.value = true
}

// 移除已选节点
const removeNode = (name: string) => {
  value1.value = value1.value.filter(n => n !== name)
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

// ---- 抽屉 ----
const openDrawer = async (sub: Sub) => {
  drawerSub.value = sub
  drawerVisible.value = true
  expireClear.value = !sub.ExpiresAt
  expirePick.value = sub.ExpiresAt || ''
  if (overview.value.length === 0) {
    try {
      const { data } = await getNodeOverview()
      overview.value = data || []
    } catch { /* ignore */ }
  }
}

const drawerNodes = computed(() => {
  const sub = drawerSub.value
  if (!sub) return []
  const names = new Set(sub.Nodes.map(n => n.Name))
  return overview.value.filter(o => names.has(o.name))
})

const rttLabel = (rtt: number) => (rtt < 0 ? '不可达' : `${rtt}ms`)
const rttColor = (rtt: number) => {
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
    <el-empty v-if="!filteredSubs.length" description="暂无订阅，点击右上角「添加订阅」" />
    <div v-else class="card-grid">
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
          <div class="link-row">
            <el-input :model-value="currentUrl(sub)" readonly size="small">
              <template #append>
                <el-button @click="copyUrl(currentUrl(sub))">复制</el-button>
                <el-button @click="handleQrcode(currentUrl(sub), sub.Name)">二维码</el-button>
              </template>
            </el-input>
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
    <el-dialog v-model="dialogVisible" :title="SubTitle" width="720px" align-center>
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

        <!-- 分区三：节点 -->
        <el-divider content-position="left">节点</el-divider>
        <el-form-item label="选择节点">
          <el-select v-model="value1" multiple filterable collapse-tags collapse-tags-tooltip placeholder="搜索并选择节点…" class="full">
            <el-option v-for="item in NodesList" :key="item.Name" :label="item.Name" :value="item.Name" />
          </el-select>
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
        <!-- 节点状态 -->
        <div class="section-title">节点状态（{{ drawerNodes.length }}）</div>
        <div v-if="drawerNodes.length" class="node-list">
          <div v-for="n in drawerNodes" :key="n.id" class="node-item">
            <span class="node-flag">{{ countryFlag(n.countryCode) }}</span>
            <span class="node-name">{{ n.name }}</span>
            <span class="node-country">{{ n.country }}</span>
            <el-tag :type="rttColor(n.rtt)" size="small" effect="light">{{ rttLabel(n.rtt) }}</el-tag>
          </div>
        </div>
        <el-empty v-else description="该订阅暂无节点" :image-size="50" />

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
.order-list { display: flex; flex-direction: column; gap: 6px; }
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
</style>