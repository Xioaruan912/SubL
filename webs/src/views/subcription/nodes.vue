<script setup lang='ts'>
import { ref, computed, onMounted, nextTick } from 'vue'
import { getNodes, getNodeOverview, AddNodes, DelNode, UpdateNode, GetGroup, SetGroup, UnbindGroup, DelGroup } from "@/api/subcription/node"
import { countryFlag } from "@/utils/flag"
import NodeUnlockDialog from "@/components/NodeUnlockDialog.vue"
import NodeTcpDialog from "@/components/NodeTcpDialog.vue"

interface OverviewItem {
  id: number
  name: string
  link: string
  server: string
  country: string
  countryCode: string
  rtt: number
  groups: string[]
}
interface NodeItem {
  ID: number
  Name: string
  Link: string
  GroupNodes?: { Name: string }[]
}

// ===== 状态 =====
const loading = ref(false)
const viewMode = ref<'card' | 'list'>('card')   // 卡片/列表
const overviewList = ref<OverviewItem[]>([])     // 概览（含国家/延迟）
const fullNodes = ref<NodeItem[]>([])            // 完整节点（含 Link 供编辑/复制）
const allGroupNames = ref<string[]>([])          // 分组名列表
const activeGroup = ref('全部')                  // 当前选中分组
const searchText = ref('')
const filterCountries = ref<string[]>([])        // 国家筛选（多选）

// 表格选择
const multipleSelection = ref<any[]>([])
const multipleTable = ref<any>(null)

// 卡片视图勾选（跨国家分组全局）
const selectedIds = ref<Set<number>>(new Set())
const selectedNodes = computed(() => filteredNodes.value.filter(n => selectedIds.value.has(n.id)))
const selectedCount = computed(() => selectedIds.value.size)

// 卡片勾选切换
const toggleCardSelect = (id: number) => {
  const s = new Set(selectedIds.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selectedIds.value = s
}
// 按国家全选/取消
const toggleCountrySelect = (items: OverviewItem[]) => {
  const ids = items.map(i => i.id)
  const allSelected = ids.every(i => selectedIds.value.has(i))
  const s = new Set(selectedIds.value)
  for (const i of ids) {
    if (allSelected) s.delete(i)
    else s.add(i)
  }
  selectedIds.value = s
}
// 国家是否全选（用于按钮态）
const countryAllSelected = (items: OverviewItem[]) => items.length > 0 && items.every((i: OverviewItem) => selectedIds.value.has(i.id))
// 全局全选/取消
const toggleAllNodes = () => {
  const allIds = filteredNodes.value.map(n => n.id)
  const allSelected = allIds.length > 0 && allIds.every(i => selectedIds.value.has(i))
  const s = new Set(selectedIds.value)
  for (const i of allIds) {
    if (allSelected) s.delete(i)
    else s.add(i)
  }
  selectedIds.value = s
}
// 清空选择
const clearCardSelect = () => { selectedIds.value = new Set() }

// 批量删除选中（卡片视图）
const cardSelectDel = async () => {
  const ids = [...selectedIds.value]
  if (!ids.length) { ElMessage.warning('请选择要删除的节点'); return }
  try {
    await ElMessageBox.confirm(`是否删除选中的 ${ids.length} 条节点 ?`, '提示', { type: 'warning' })
    for (const id of ids) await DelNode({ id })
    ElMessage.success('批量删除成功')
  } catch { /* cancel */ }
  clearCardSelect(); loadAll()
}
// 批量移出当前分组（仅 activeGroup≠全部 时）
const cardSelectUnbind = async () => {
  const ids = [...selectedIds.value]
  if (!ids.length) { ElMessage.warning('请选择节点'); return }
  try {
    await ElMessageBox.confirm(`是否将选中的 ${ids.length} 条节点移出分组「${activeGroup.value}」？仅解除绑定，不会删除节点。`, '提示', { type: 'warning' })
    for (const id of ids) {
      const n = filteredNodes.value.find(x => x.id === id)
      if (n) await UnbindGroup({ name: n.name, group: activeGroup.value })
    }
    ElMessage.success('已移出分组')
  } catch { /* cancel */ }
  clearCardSelect(); loadAll()
}

// 弹窗
const Nodedialog = ref(false)
const Groupdialog = ref(false)
const dialogMode = ref<'add' | 'edit'>('add')
const RadioGroup = ref('1')
const NodeForm = ref({ ID: 0, Title: '', Name: '', Link: '', GroupName: [] as string[] })
const SelectionNodeGroups = ref<string[]>([])
const NodeGroupInput = ref('')
const SelectionNode = ref('')
const delGroupName = ref('')

// ===== 派生数据 =====
// 可用国家列表（用于筛选）
const countryOptions = computed(() => {
  const s = new Set<string>()
  overviewList.value.forEach(n => { if (n.country) s.add(n.country) })
  return [...s].sort()
})

// 按分组+搜索+国家 过滤
const filteredNodes = computed(() => {
  return overviewList.value.filter(n => {
    if (activeGroup.value !== '全部' && !(n.groups || []).includes(activeGroup.value)) return false
    if (searchText.value && !n.name.toLowerCase().includes(searchText.value.toLowerCase())) return false
    if (filterCountries.value.length && !filterCountries.value.includes(n.country)) return false
    return true
  })
})

// 分组树数据
const groupTree = computed(() => {
  const nodes: any[] = []
  for (const g of allGroupNames.value) {
    const count = overviewList.value.filter(n => (n.groups || []).includes(g)).length
    nodes.push({ id: g, label: g, count })
  }
  return nodes
})

// 延迟颜色
const rttColor = (rtt: number) => {
  if (rtt < 0) return "#95a5a6"
  if (rtt < 100) return "#2ecc71"
  if (rtt < 300) return "#f1c40f"
  return "#e74c3c"
}
const rttText = (rtt: number) => rtt < 0 ? "超时" : rtt + "ms"

// ===== 测试弹窗状态 =====
const unlockDialogVisible = ref(false)
const tcpDialogVisible = ref(false)
const testNode = ref<{ id: number; name: string } | null>(null)
const openUnlock = (node: OverviewItem) => { testNode.value = { id: node.id, name: node.name }; unlockDialogVisible.value = true }
const openTcp = (node: OverviewItem) => { testNode.value = { id: node.id, name: node.name }; tcpDialogVisible.value = true }

// ===== 数据加载 =====
const loadAll = async () => {
  loading.value = true
  try {
    const [ov, nd, gp] = await Promise.all([
      getNodeOverview(), getNodes(), GetGroup()
    ])
    overviewList.value = ov?.data || []
    fullNodes.value = nd?.data || []
    allGroupNames.value = Array.isArray(gp?.data) ? gp.data : []
  } catch { /* ignore */ } finally {
    loading.value = false
  }
}

// ===== 添加/编辑节点 =====
const handleAddNode = () => {
  dialogMode.value = 'add'
  Nodedialog.value = true
  NodeForm.value = { ID: 0, Title: '添加节点', Name: '', Link: '', GroupName: [] }
  SelectionNodeGroups.value = []
  NodeGroupInput.value = ''
}
const handleEditNode = (row: any) => {
  const full = fullNodes.value.find(n => n.ID === row.id)
  dialogMode.value = 'edit'
  Nodedialog.value = true
  NodeForm.value = {
    ID: row.id, Title: '编辑节点', Name: row.name, Link: full?.Link || row.link,
    GroupName: (full?.GroupNodes || []).map(g => g.Name),
  }
  SelectionNodeGroups.value = NodeForm.value.GroupName || []
  SelectionNode.value = row.name
}
const SubmitNodeForm = async () => {
  const isAdd = dialogMode.value === 'add'
  const links = NodeForm.value.Link.trim().split(/[\n,]/).map(i => i.trim()).filter(i => i)
  if (isAdd && links.length === 0) { ElMessage.warning('节点链接不能为空'); return }
  try {
    if (isAdd) {
      for (const link of links) {
        await AddNodes({ link, group: RadioGroup.value === '1' ? SelectionNodeGroups.value.join(',') : NodeGroupInput.value })
      }
      ElMessage.success('节点添加成功')
    } else {
      await UpdateNode({
        id: NodeForm.value.ID, name: NodeForm.value.Name, link: NodeForm.value.Link,
        group: RadioGroup.value === '1' ? SelectionNodeGroups.value.join(',') : NodeGroupInput.value,
      })
      ElMessage.success('节点更新成功')
    }
  } catch { ElMessage.error(isAdd ? '添加失败' : '更新失败') }
  loadAll(); ClearInput()
}

// ===== 删除/复制 =====
const copyUrl = (url: string) => {
  if (navigator.clipboard) {
    navigator.clipboard.writeText(url).then(() => ElMessage.success('链接已复制')).catch(() => ElMessage.error('复制失败'))
  } else {
    const ta = document.createElement('textarea'); ta.value = url; document.body.appendChild(ta); ta.select()
    try { document.execCommand('copy'); ElMessage.success('已复制') } catch { ElMessage.warning('复制失败') }
    document.body.removeChild(ta)
  }
}
const copyInfo = (row: any) => {
  const full = fullNodes.value.find(n => n.ID === row.id)
  copyUrl(full?.Link || row.link)
}
const handleDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(`你是否要删除 ${row.name} ?`, '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' })
    await DelNode({ id: row.id })
    ElMessage.success('删除成功')
  } catch { /* cancel */ }
  loadAll(); ClearInput()
}
const selectDel = async () => {
  if (!multipleSelection.value.length) { ElMessage.warning('请选择要删除的节点'); return }
  try {
    await ElMessageBox.confirm(`是否删除选中的 ${multipleSelection.value.length} 条节点 ?`, '提示', { type: 'warning' })
    for (const item of multipleSelection.value) await DelNode({ id: item.id })
    ElMessage.success('批量删除成功')
  } catch { /* cancel */ }
  loadAll()
}
const selectCopy = async () => {
  if (!multipleSelection.value.length) { ElMessage.warning('请选择节点'); return }
  const links = multipleSelection.value.map(i => i.link || i.Link).join('\n')
  copyUrl(links)
}
const selectAll = () => nextTick(() => multipleTable.value?.toggleAllSelection())
const selectClear = () => nextTick(() => multipleTable.value?.clearSelection())

// ===== 分组绑定 =====
const AddGroup = async () => {
  if (RadioGroup.value === '1' && !SelectionNodeGroups.value.length) { ElMessage.warning('请选择分组'); return }
  if (RadioGroup.value === '2' && !NodeGroupInput.value.trim()) { ElMessage.warning('分组名不能为空'); return }
  if (!SelectionNode.value) { ElMessage.warning('请选择节点'); return }
  try {
    await SetGroup({ name: SelectionNode.value, group: RadioGroup.value === '1' ? SelectionNodeGroups.value.join(',') : NodeGroupInput.value })
    ElMessage.success('分组绑定成功')
  } catch { ElMessage.error('绑定失败') }
  loadAll(); ClearInput()
}

// ===== 分组删除 =====
const delGroup = async (group: string) => {
  if (!group) { ElMessage.warning('请选择分组'); return }
  try {
    await ElMessageBox.confirm(`是否删除分组「${group}」？该分组下节点不会被删除，仅解除绑定。`, '提示', { type: 'warning' })
    await DelGroup({ name: group })
    ElMessage.success('分组已删除')
    if (activeGroup.value === group) activeGroup.value = '全部'
    loadAll()
  } catch { /* cancel */ }
}

const ClearInput = () => {
  SelectionNode.value = ''
  NodeForm.value = { ID: 0, Title: '', Name: '', Link: '', GroupName: [] }
  NodeGroupInput.value = ''
  SelectionNodeGroups.value = []
  Nodedialog.value = false
  Groupdialog.value = false
}

// 卡片视图分组（按国家分组展示）
const cardGroups = computed(() => {
  const byCountry: Record<string, OverviewItem[]> = {}
  for (const n of filteredNodes.value) {
    const key = n.country || '未知'
    if (!byCountry[key]) byCountry[key] = []
    byCountry[key].push(n)
  }
  return Object.keys(byCountry).sort().map(c => ({ country: c, items: byCountry[c] }))
})

const handleSelectionChange = (val: any[]) => { multipleSelection.value = val }

onMounted(loadAll)
</script>

<template>
  <div class="p-4 md:p-6 lg:p-8 h-full flex flex-col gap-6">
    <el-row :gutter="24">
      <!-- 左侧分组树 -->
      <el-col :span="5" :xs="24">
        <div class="bg-white dark:bg-[#1a1d1b] rounded-xl shadow-[inset_0_0_0_1px_rgba(0,0,0,0.06)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)] p-5 sticky top-6">
          <div class="flex justify-between items-center mb-4">
            <span class="font-semibold text-gray-700 dark:text-gray-200">分组</span>
            <el-button link type="primary" size="small" @click="Groupdialog = true">管理</el-button>
          </div>
          <div class="flex flex-col gap-1">
            <div
              class="px-3 py-2 rounded-lg cursor-pointer transition-all duration-150 flex justify-between items-center"
              :class="activeGroup === '全部' ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 font-medium' : 'hover:bg-gray-50 dark:hover:bg-white/5 text-gray-600 dark:text-gray-400'"
              @click="activeGroup = '全部'"
            >
              <span>全部</span>
              <span class="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-0.5 rounded-full text-gray-600 dark:text-gray-300">{{ overviewList.length }}</span>
            </div>
            <div
              v-for="g in groupTree"
              :key="g.id"
              class="px-3 py-2 rounded-lg cursor-pointer transition-all duration-150 flex justify-between items-center"
              :class="activeGroup === g.id ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 font-medium' : 'hover:bg-gray-50 dark:hover:bg-white/5 text-gray-600 dark:text-gray-400'"
              @click="activeGroup = g.id"
            >
              <span class="truncate pr-2">{{ g.label }}</span>
              <span class="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-0.5 rounded-full text-gray-600 dark:text-gray-300">{{ g.count }}</span>
            </div>
          </div>
        </div>
      </el-col>

      <!-- 主区 -->
      <el-col :span="19" :xs="24">
        <div class="bg-white dark:bg-[#1a1d1b] rounded-xl shadow-[inset_0_0_0_1px_rgba(0,0,0,0.06)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.08)] flex flex-col overflow-hidden min-h-[500px]">
          <!-- 工具条 -->
          <div class="p-5 border-b border-gray-100 dark:border-white/5 flex flex-wrap gap-3 items-center">
            <el-input v-model="searchText" placeholder="搜索节点名" clearable class="w-64" />
            <el-select v-model="filterCountries" multiple collapse-tags placeholder="国家筛选" class="w-48">
              <el-option v-for="c in countryOptions" :key="c" :label="c" :value="c" />
            </el-select>
            <el-button-group>
              <el-button :type="viewMode === 'card' ? 'primary' : ''" @click="viewMode = 'card'">卡片</el-button>
              <el-button :type="viewMode === 'list' ? 'primary' : ''" @click="viewMode = 'list'">列表</el-button>
            </el-button-group>
            <template v-if="viewMode === 'card'">
              <el-button size="small" @click="toggleAllNodes">
                {{ filteredNodes.length && filteredNodes.every(n => selectedIds.has(n.id)) ? '取消全选' : '全选' }}
              </el-button>
              <el-button size="small" @click="clearCardSelect" :disabled="!selectedCount">取消选择</el-button>
              <el-button v-if="activeGroup !== '全部'" size="small" type="warning" @click="cardSelectUnbind" :disabled="!selectedCount">移出分组</el-button>
              <el-button size="small" type="danger" @click="cardSelectDel" :disabled="!selectedCount">删除选中({{ selectedCount }})</el-button>
            </template>
            <div class="flex-1"></div>
            <el-button :loading="loading" @click="loadAll">刷新</el-button>
            <el-button type="primary" @click="handleAddNode">添加节点</el-button>
          </div>

          <div class="p-5 bg-gray-50/30 dark:bg-black/10 flex-1">
          <!-- 卡片视图 -->
          <template v-if="viewMode === 'card'">
            <div v-for="g in cardGroups" :key="g.country" class="mb-6 last:mb-0">
              <div class="flex items-center gap-2 text-sm font-semibold text-gray-500 dark:text-gray-400 mb-3 ml-1">
                <el-checkbox
                  :model-value="countryAllSelected(g.items)"
                  @change="() => toggleCountrySelect(g.items)"
                  size="small"
                />
                <span>{{ countryFlag(g.items[0].countryCode) }} {{ g.country }}（{{ g.items.length }}）</span>
                <el-button link type="primary" size="small" @click="toggleCountrySelect(g.items)">
                  {{ countryAllSelected(g.items) ? '取消全选' : '全选' }}
                </el-button>
              </div>
              <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                <div v-for="n in g.items" :key="n.id" class="bg-white dark:bg-[#202322] rounded-xl p-4 shadow-[inset_0_0_0_1px_rgba(0,0,0,0.05),0_1px_2px_rgba(0,0,0,0.02)] dark:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.05)] hover:shadow-md transition-all duration-150 group" :class="selectedIds.has(n.id) ? 'node-card-selected' : ''">
                  <div class="flex justify-between items-start mb-2">
                    <div class="flex items-center gap-2 overflow-hidden flex-1 pr-2">
                      <el-checkbox :model-value="selectedIds.has(n.id)" @change="() => toggleCardSelect(n.id)" size="small" class="node-check" />
                      <span class="text-lg">{{ countryFlag(n.countryCode) }}</span>
                      <span class="font-medium text-gray-800 dark:text-gray-200 truncate" :title="n.name">{{ n.name }}</span>
                    </div>
                    <span class="text-xs px-2 py-0.5 rounded-full whitespace-nowrap" :style="{ background: rttColor(n.rtt), color: '#fff' }">{{ rttText(n.rtt) }}</span>
                  </div>
                  <div class="text-xs text-gray-500 dark:text-gray-400 mb-4 truncate">{{ n.country }} · {{ n.server }}</div>
                  <div class="flex items-center justify-end gap-1 opacity-60 group-hover:opacity-100 transition-opacity">
                    <el-button link type="primary" size="small" @click="openUnlock(n)">解锁</el-button>
                    <el-button link type="success" size="small" @click="openTcp(n)">TCP</el-button>
                    <el-button link type="primary" size="small" @click="handleEditNode(n)">编辑</el-button>
                    <el-button link type="primary" size="small" @click="copyInfo(n)">复制</el-button>
                    <el-button link type="danger" size="small" @click="handleDel(n)">删除</el-button>
                  </div>
                </div>
              </div>
            </div>
            <el-empty v-if="!loading && filteredNodes.length === 0" description="无匹配节点" />
          </template>

          <!-- 列表视图 -->
          <template v-else>
            <el-table
              ref="multipleTable"
              :data="filteredNodes"
              stripe
              style="width: 100%"
              @selection-change="handleSelectionChange"
            >
              <el-table-column type="selection" width="45" />
              <el-table-column label="节点" min-width="200">
                <template #default="{ row }">
                  <span class="row-flag">{{ countryFlag(row.countryCode) }}</span>
                  <span class="row-name">{{ row.name }}</span>
                </template>
              </el-table-column>
              <el-table-column label="国家" width="90">
                <template #default="{ row }">{{ row.country }}</template>
              </el-table-column>
              <el-table-column label="延迟" width="90">
                <template #default="{ row }">
                  <el-tag :type="row.rtt < 0 ? 'danger' : row.rtt < 100 ? 'success' : row.rtt < 300 ? 'warning' : 'danger'" size="small">
                    {{ rttText(row.rtt) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="分组" min-width="120">
                <template #default="{ row }">
                  <el-tag v-for="g in (row.groups || [])" :key="g" size="small" effect="plain">{{ g }}</el-tag>
                  <span v-if="!(row.groups || []).length" class="muted">未分组</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="180">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="handleEditNode(row)">编辑</el-button>
                  <el-button link type="primary" size="small" @click="copyInfo(row)">复制</el-button>
                  <el-button link type="danger" size="small" @click="handleDel(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
            <div class="mt-4 flex gap-2">
              <el-button size="small" @click="selectAll">全选</el-button>
              <el-button size="small" @click="selectClear">取消</el-button>
              <el-button size="small" type="primary" @click="selectCopy">复制选中</el-button>
              <el-button size="small" type="danger" @click="selectDel">删除选中</el-button>
            </div>
          </template>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- 添加/编辑节点弹窗 -->
    <el-dialog v-model="Nodedialog" :title="NodeForm.Title" width="560px" align-center>
      <el-form label-position="top" class="node-form">
        <el-divider content-position="left">基本信息</el-divider>
        <el-form-item label="节点链接">
          <el-input v-model="NodeForm.Link" placeholder="节点链接，支持多行（回车/逗号分隔）" type="textarea" :autosize="{ minRows: 3, maxRows: 8 }" />
        </el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" label="节点名称">
          <el-input v-model="NodeForm.Name" placeholder="节点名称（编辑时）" />
        </el-form-item>

        <el-divider content-position="left">分组</el-divider>
        <el-radio-group v-model="RadioGroup" size="small" class="group-tabs">
          <el-radio-button v-if="allGroupNames.length" value="1">选择已有分组</el-radio-button>
          <el-radio-button value="2">创建新分组</el-radio-button>
        </el-radio-group>
        <div v-if="RadioGroup === '1' && allGroupNames.length" class="group-field">
          <el-select v-model="SelectionNodeGroups" multiple placeholder="选择分组" class="full">
            <el-option v-for="g in allGroupNames" :key="g" :label="g" :value="g" />
          </el-select>
        </div>
        <div v-else class="group-field">
          <el-input v-model="NodeGroupInput" placeholder="输入新分组名" />
        </div>
      </el-form>
      <template #footer>
        <el-button text @click="Nodedialog = false">取消</el-button>
        <el-button type="primary" @click="SubmitNodeForm">{{ dialogMode === 'add' ? '添加节点' : '保存修改' }}</el-button>
      </template>
    </el-dialog>

    <!-- 分组管理弹窗 -->
    <el-dialog v-model="Groupdialog" title="分组管理" width="560px" align-center>
      <el-form label-position="top" class="node-form">
        <el-divider content-position="left">绑定节点到分组</el-divider>
        <el-form-item label="选择节点">
          <el-select v-model="SelectionNode" filterable placeholder="搜索并选择节点…" class="full">
            <el-option v-for="n in overviewList" :key="n.id" :label="n.name" :value="n.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="绑定到">
          <el-radio-group v-model="RadioGroup" size="small" class="group-tabs">
            <el-radio-button v-if="allGroupNames.length" value="1">已有分组</el-radio-button>
            <el-radio-button value="2">新分组</el-radio-button>
          </el-radio-group>
          <div v-if="RadioGroup === '1' && allGroupNames.length" class="group-field">
            <el-select v-model="SelectionNodeGroups" multiple placeholder="选择分组" class="full">
              <el-option v-for="g in allGroupNames" :key="g" :label="g" :value="g" />
            </el-select>
          </div>
          <div v-else class="group-field">
            <el-input v-model="NodeGroupInput" placeholder="输入新分组名" />
          </div>
        </el-form-item>
        <div class="group-actions">
          <el-button type="primary" @click="AddGroup">绑定</el-button>
        </div>

        <div class="del-group-card">
          <div class="del-group-title">删除分组</div>
          <div class="del-group-row">
            <el-select v-model="delGroupName" placeholder="选择要删除的分组" class="full">
              <el-option v-for="g in allGroupNames" :key="g" :label="g" :value="g" />
            </el-select>
            <el-button type="danger" @click="delGroup(delGroupName)">删除所选</el-button>
          </div>
          <div class="text-xs text-gray-400 mt-1">删除分组仅解除节点绑定，不会删除节点本身。</div>
        </div>
      </el-form>
      <template #footer>
        <el-button text @click="Groupdialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 解锁 / TCP 测试弹窗 -->
    <NodeUnlockDialog
      v-if="testNode"
      v-model:visible="unlockDialogVisible"
      :node-id="testNode.id"
      :node-name="testNode.name"
    />
    <NodeTcpDialog
      v-if="testNode"
      v-model:visible="tcpDialogVisible"
      :node-id="testNode.id"
      :node-name="testNode.name"
    />
  </div>
</template>

<style scoped>
.nodes-container { padding: 4px; }
.group-card { min-height: 70vh; }
.group-header { display: flex; justify-content: space-between; align-items: center; }
.group-tree { max-height: 65vh; overflow-y: auto; }
.tree-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 10px; margin-bottom: 4px; border-radius: 8px; cursor: pointer;
  color: var(--el-text-color-regular);
}
.tree-item:hover { background: var(--el-fill-color-light); }
.tree-item.active { background: #f97316; color: #fff; }
.toolbar { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; margin-bottom: 16px; }
.toolbar .search { width: 200px; }
.toolbar .country { width: 200px; }
.flex-1 { flex: 1; }
.card-group { margin-bottom: 18px; }
.card-group-title { font-size: 15px; font-weight: 600; margin-bottom: 10px; }
.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 12px; }
.node-card {
  padding: 12px 14px; border: 1px solid var(--el-border-color-lighter); border-radius: 12px;
  transition: box-shadow .2s;
}
.node-card:hover { box-shadow: 0 2px 12px rgba(0,0,0,.1); }
.card-top { display: flex; align-items: center; gap: 8px; }
.card-flag { font-size: 20px; }
.card-name { flex: 1; font-weight: 600; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rtt-badge { padding: 2px 8px; border-radius: 6px; font-size: 12px; font-weight: 600; }
.card-country { margin-top: 6px; font-size: 12px; color: var(--el-text-color-secondary); }
.unlock-row { display: flex; gap: 8px; margin-top: 8px; flex-wrap: wrap; }
.unlock-item { font-size: 12px; color: var(--el-text-color-regular); }
.card-actions { margin-top: 10px; display: flex; gap: 4px; }
.row-flag { margin-right: 6px; }
.row-name { font-weight: 500; }
.muted { color: var(--el-text-color-placeholder); font-size: 12px; }
.list-actions { margin-top: 12px; display: flex; gap: 8px; }
.dialog-actions { margin-top: 12px; }
.del-group-row { display: flex; gap: 10px; align-items: center; }
.node-form .el-divider { margin: 6px 0 16px; }
.node-form .el-form-item { margin-bottom: 16px; }
.group-tabs { display: block; margin-bottom: 10px; }
.group-field { margin-bottom: 4px; }
.full { width: 100%; }
.node-card-selected {
  box-shadow: inset 0 0 0 2px var(--el-color-primary), 0 1px 2px rgba(0,0,0,0.02) !important;
}
.node-check { margin-right: 0; }
.group-actions { margin-top: 4px; }
.del-group-card {
  margin-top: 4px;
  padding: 12px 14px;
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: 10px;
}
html.dark .del-group-card { background: rgba(245, 108, 108, 0.08); border-color: rgba(245, 108, 108, 0.2); }
.del-group-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; color: var(--el-color-danger); }
.del-group-row .el-select { flex: 1; }
</style>