<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getRuleSources, getRuleCatalog, getRulePreview, getRuleUpdateImpact, applyRuleUpdate, getRuleSnapshots, rollbackRule, getRuleTemplateGroups, syncRuleCatalog, applyRulesToTemplate } from "@/api/rules";
import { getTemp } from "@/api/template/temp";

type RuleItem = {
  externalId: string;
  sourceKey: string;
  name: string;
  category: string;
  platform: string;
  format: string;
  ruleCount: number;
  checksum?: string;
  localPath?: string;
  remoteUpdate?: string;
  metadataJson?: string;
};

const loading = ref(true);
const syncing = ref(false);
const errorText = ref("");
const rules = ref<RuleItem[]>([]);
const sources = ref<any[]>([]);
const total = ref(0);
const keyword = ref("");
const source = ref("");
const platform = ref("Clash");
const category = ref("");
const categories = ["AI", "流媒体", "社交", "Apple", "Google", "Microsoft", "开发者", "广告/隐私", "其他"];
const selectedIds = ref<string[]>([]);
const previewOpen = ref(false);
const previewLoading = ref(false);
const preview = ref<any>(null);
const impactOpen = ref(false);
const impactLoading = ref(false);
const impactRule = ref<RuleItem | null>(null);
const impact = ref<any>(null);
const snapshots = ref<any[]>([]);
const importOpen = ref(false);
const importLoading = ref(false);
const importItems = ref<RuleItem[]>([]);
const templates = ref<any[]>([]);
const proxyGroups = ref<string[]>([]);
const policyGroups = computed(() => [...proxyGroups.value, "DIRECT"]);
const proxyOptions = computed(() => ["DIRECT", ...proxyGroups.value]);
const importForm = ref({ template: "", policy: "", proxy: "", conflictPolicy: "keep" });

const selectedRules = computed(() => rules.value.filter(r => selectedIds.value.includes(r.externalId)));
const sourceLabel = (key: string) => key === "shunt_rules" ? "ShuntRules" : key === "ios_rule_script" ? "ios_rule_script" : key;
const metadataSummary = (row: RuleItem) => {
  if (!row.metadataJson) return "";
  try {
    return Object.entries(JSON.parse(row.metadataJson)).slice(0, 3).map(([k, v]) => `${k} ${String(v)}`).join(" · ");
  } catch {
    return "";
  }
};

async function loadSources() {
  try {
    const res = await getRuleSources();
    sources.value = Array.isArray(res?.data) ? res.data : [];
  } catch (e: any) {
    errorText.value = `规则来源读取失败：${typeof e === "string" ? e : e?.message || "未知错误"}`;
  }
}

async function loadCatalog() {
  loading.value = true;
  errorText.value = "";
  try {
    const res = await getRuleCatalog({
      source: source.value,
      platform: platform.value,
      category: category.value,
      keyword: keyword.value,
      page: 1,
      pageSize: 100,
    });
    const data = res?.data || {};
    rules.value = Array.isArray(data.items) ? data.items.filter((x: any) => x && x.externalId) : [];
    total.value = Number(data.total || 0);
    selectedIds.value = selectedIds.value.filter(id => rules.value.some(r => r.externalId === id));
  } catch (e: any) {
    rules.value = [];
    total.value = 0;
    errorText.value = `规则目录加载失败：${typeof e === "string" ? e : e?.message || "未知错误"}`;
  } finally {
    loading.value = false;
  }
}

function toggleCategory(value: string) {
  category.value = category.value === value ? "" : value;
  void loadCatalog();
}

function toggleSelected(id: string) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter(x => x !== id)
    : [...selectedIds.value, id];
}

async function doSync() {
  syncing.value = true;
  errorText.value = "";
  try {
    await syncRuleCatalog();
    await Promise.all([loadSources(), loadCatalog()]);
  } catch (e: any) {
    errorText.value = `同步失败：${typeof e === "string" ? e : e?.message || "未知错误"}`;
  } finally {
    syncing.value = false;
  }
}

async function openPreview(row: RuleItem) {
  previewOpen.value = true;
  previewLoading.value = true;
  preview.value = null;
  try {
    const res = await getRulePreview(row.externalId);
    preview.value = res?.data || null;
  } catch (e: any) {
    preview.value = { error: typeof e === "string" ? e : e?.message || "预览失败" };
  } finally {
    previewLoading.value = false;
  }
}

async function openImpact(row:RuleItem) {
  impactRule.value = row; impactOpen.value = true; impactLoading.value = true; impact.value = null; snapshots.value = [];
  try {
    const [previewRes, snapshotsRes] = await Promise.all([getRuleUpdateImpact(row.externalId), getRuleSnapshots(row.externalId)]);
    impact.value = previewRes?.data || null; snapshots.value = Array.isArray(snapshotsRes?.data) ? snapshotsRes.data : [];
  } catch (e:any) { impact.value = { error: typeof e === 'string' ? e : e?.message || '更新影响预览失败' }; }
  finally { impactLoading.value = false; }
}
async function applyUpdateNow() {
  if (!impactRule.value) return;
  await ElMessageBox.confirm(`确认更新「${impactRule.value.name}」？系统会先保留当前可回滚快照。`, '应用规则更新', { type:'warning' });
  await applyRuleUpdate(impactRule.value.externalId); ElMessage.success('规则已更新，旧版本已保留'); await loadCatalog(); await openImpact(impactRule.value);
}
async function rollbackSnapshot(snapshotId:number) {
  if (!impactRule.value) return;
  await ElMessageBox.confirm('回滚到这份规则快照？当前版本也会先保留为快照。', '回滚规则', { type:'warning' });
  await rollbackRule(impactRule.value.externalId, snapshotId); ElMessage.success('规则已回滚'); await loadCatalog(); await openImpact(impactRule.value);
}
const ruleText = (r:any) => r ? [r.type, r.value, ...(r.options || [])].join(',') : '';

async function loadTemplateGroups() {
  proxyGroups.value = [];
  importForm.value.proxy = "";
  importForm.value.policy = "";
  if (!importForm.value.template) return;
  try {
    const res = await getRuleTemplateGroups(importForm.value.template);
    proxyGroups.value = Array.isArray(res?.data) ? res.data : [];
    importForm.value.policy = proxyGroups.value[0] || "DIRECT";
  } catch (e: any) {
    errorText.value = `读取模板节点组失败：${typeof e === "string" ? e : e?.message || "未知错误"}`;
  }
}

async function openImport(items: RuleItem[]) {
  if (!items.length) return;
  if (items.some(i => i.platform !== "Clash")) {
    errorText.value = "V1 当前仅支持将 Clash 规则写入模板；Surge / Loon 可浏览和预览。";
    return;
  }
  importItems.value = items;
  importOpen.value = true;
  try {
    const res = await getTemp();
    templates.value = Array.isArray(res?.data) ? res.data.filter((t: any) => /\.ya?ml$/i.test(t.file)) : [];
    if (!importForm.value.template && templates.value.length) importForm.value.template = templates.value[0].file;
    await loadTemplateGroups();
  } catch {
    templates.value = [];
  }
}

async function submitImport() {
  if (!importForm.value.template || !importForm.value.policy.trim()) return;
  importLoading.value = true;
  errorText.value = "";
  try {
    const res: any = await applyRulesToTemplate({
      ruleIds: importItems.value.map(i => i.externalId),
      template: importForm.value.template,
      policy: importForm.value.policy,
      mode: "provider",
      position: "before-match",
      conflictPolicy: importForm.value.conflictPolicy,
      proxy: importForm.value.proxy,
    });
    const importedCount = Array.isArray(res?.data?.results) ? res.data.results.length : importItems.value.length;
    const warnings = Array.isArray(res?.data?.warnings) ? res.data.warnings : [];
    ElMessage.success(`${res?.msg || "规则导入完成"}：${importedCount} 条规则已写入 ${importForm.value.template}`);
    if (warnings.length) ElMessage.warning(`导入完成，但有 ${warnings.length} 条警告，请检查模板内容`);
    importOpen.value = false;
    selectedIds.value = [];
  } catch (e: any) {
    errorText.value = `导入失败：${typeof e === "string" ? e : e?.message || "未知错误"}`;
  } finally {
    importLoading.value = false;
  }
}

onMounted(async () => {
  await loadSources();
  await loadCatalog();
});
</script>

<template>
  <div class="page">
    <header class="hero">
      <div>
        <h2>规则中心</h2>
        <p>从 ShuntRules 与 ios_rule_script 选择规则，预览后导入现有模板。</p>
      </div>
      <button class="btn" :disabled="syncing" @click="doSync">{{ syncing ? "同步中…" : "立即同步" }}</button>
    </header>

    <div v-if="errorText" class="alert">{{ errorText }}</div>

    <section class="sources">
      <article v-for="s in sources" :key="s.key" class="source-card">
        <div class="source-head"><strong>{{ s.name }}</strong><span :class="['status', s.status === 'ok' ? 'ok' : 'muted']">{{ s.status === 'ok' ? '已同步' : '尚未同步' }}</span></div>
        <div class="muted-text">{{ s.repo }}</div>
        <div class="source-foot"><span>{{ s.count || 0 }} 条目录 · 已缓存 {{ s.cachedCount || 0 }}</span><span>{{ s.lastSyncAt ? new Date(s.lastSyncAt).toLocaleString() : '-' }}</span></div>
      </article>
    </section>

    <section class="filters">
      <input v-model="keyword" class="input search" placeholder="搜索 OpenAI / Netflix / Telegram…" @keyup.enter="loadCatalog" />
      <select v-model="source" class="input" @change="loadCatalog">
        <option value="">全部来源</option>
        <option value="shunt_rules">ShuntRules</option>
        <option value="ios_rule_script">ios_rule_script</option>
      </select>
      <select v-model="platform" class="input" @change="loadCatalog">
        <option value="Clash">Clash</option>
        <option value="Surge">Surge</option>
        <option value="Loon">Loon</option>
      </select>
      <button class="btn" @click="loadCatalog">搜索</button>
      <button class="btn ghost" @click="keyword='';source='';category='';platform='Clash';loadCatalog()">重置</button>
      <div class="chips">
        <button v-for="c in categories" :key="c" :class="['chip', { active: category === c }]" @click="toggleCategory(c)">{{ c }}</button>
      </div>
    </section>

    <div class="layout">
      <main>
        <div class="list-title"><strong>规则目录 {{ total }}</strong><span>{{ loading ? "正在加载规则目录…" : `已加载 ${rules.length} / 总计 ${total}` }}</span></div>

        <div v-if="loading" class="grid">
          <div v-for="n in 8" :key="n" class="skeleton"><i></i><i></i><i></i><i></i></div>
        </div>
        <div v-else-if="!rules.length" class="empty">没有符合条件的规则。</div>
        <div v-else class="grid">
          <article v-for="r in rules" :key="r.externalId" class="rule-card">
            <div class="rule-head">
              <input type="checkbox" :checked="selectedIds.includes(r.externalId)" @change="toggleSelected(r.externalId)" />
              <div><strong>{{ r.name || '未命名规则' }}</strong><small>{{ r.category || '其他' }}</small></div>
            </div>
            <div class="tags"><span>{{ sourceLabel(r.sourceKey) }}</span><span>{{ r.platform }}</span><span>{{ r.format }}</span></div>
            <div class="meta"><span>{{ r.checksum ? `${r.ruleCount} 条规则` : '自动缓存中…' }}</span><span>{{ r.remoteUpdate || '' }}</span></div>
            <div v-if="metadataSummary(r)" class="summary">{{ metadataSummary(r) }}</div>
            <div class="actions"><button @click="openPreview(r)">预览</button><button @click="openImpact(r)">更新影响</button><button class="primary-link" @click="openImport([r])">导入模板</button></div>
          </article>
        </div>
      </main>

      <aside class="batch">
        <h3>批量导入</h3>
        <p>勾选多个规则后统一绑定策略组。</p>
        <div class="selected">
          <div v-for="r in selectedRules" :key="r.externalId" class="selected-row"><span><strong>{{ r.name }}</strong><small>{{ sourceLabel(r.sourceKey) }} · {{ r.platform }}</small></span><button @click="toggleSelected(r.externalId)">移除</button></div>
          <div v-if="!selectedRules.length" class="empty small">暂未选择</div>
        </div>
        <button class="btn primary full" :disabled="!selectedRules.length" @click="openImport(selectedRules)">导入已选 {{ selectedRules.length ? `(${selectedRules.length})` : '' }}</button>
      </aside>
    </div>

    <div v-if="previewOpen" class="overlay" @click.self="previewOpen=false">
      <section class="modal">
        <header><strong>{{ preview?.name || '规则预览' }}</strong><button @click="previewOpen=false">×</button></header>
        <div class="modal-body">
          <div v-if="previewLoading" class="empty">正在下载并解析规则正文…</div>
          <div v-else-if="preview?.error" class="alert">{{ preview.error }}</div>
          <template v-else-if="preview">
            <div class="preview-meta"><span>来源：{{ sourceLabel(preview.sourceKey) }}</span><span>客户端：{{ preview.platform }}</span><span>规则：{{ preview.ruleCount }}</span></div>
            <pre>{{ (preview.sample || []).map((r:any) => [r.type, r.value, ...(r.options || [])].join(',')).join('\n') }}</pre>
          </template>
        </div>
      </section>
    </div>

    <div v-if="importOpen" class="overlay" @click.self="importOpen=false">
      <section class="modal">
        <header><strong>导入规则</strong><button @click="importOpen=false">×</button></header>
        <div class="modal-body form">
          <div class="notice">将导入：{{ importItems.map(i => i.name).join('、') }}</div>
          <label>目标模板<select v-model="importForm.template" class="input full" @change="loadTemplateGroups"><option v-for="t in templates" :key="t.file" :value="t.file">{{ t.file }}</option></select></label>
          <label>策略组<select v-model="importForm.policy" class="input full"><option v-for="g in policyGroups" :key="g" :value="g">{{ g }}</option></select><small class="field-tip">自动读取当前模板的 proxy-groups，并额外提供 DIRECT。</small></label>
          <label>规则下载代理（proxy）<select v-model="importForm.proxy" class="input full"><option value="">不指定（proxy: ""）</option><option v-for="g in proxyOptions" :key="g" :value="g">{{ g }}</option></select><small class="field-tip">自动读取当前模板的 proxy-groups，并额外提供 DIRECT；用于控制远程规则文件通过哪个策略下载。</small></label>
          <label>Provider 冲突<select v-model="importForm.conflictPolicy" class="input full"><option value="keep">保留现有</option><option value="update-url">更新 URL</option><option value="replace">替换</option></select></label>
          <div class="notice warn">默认使用 Rule Provider，并插入 MATCH 前；写入前会创建模板版本。</div>
        </div>
        <footer><button class="btn" @click="importOpen=false">取消</button><button class="btn primary" :disabled="importLoading" @click="submitImport">{{ importLoading ? '导入中…' : '检查并导入' }}</button></footer>
      </section>
    </div>

    <div v-if="impactOpen" class="overlay" @click.self="impactOpen=false">
      <section class="modal impact-modal">
        <header><strong>{{ impactRule?.name || '规则更新影响' }}</strong><button @click="impactOpen=false">×</button></header>
        <div class="modal-body">
          <div v-if="impactLoading" class="empty">正在拉取远端规则并与当前缓存比较…</div>
          <div v-else-if="impact?.error" class="alert">{{ impact.error }}</div>
          <template v-else-if="impact?.preview">
            <div class="impact-stats">
              <article><small>规则数量</small><b>{{ impact.preview.oldCount }} → {{ impact.preview.newCount }}</b></article>
              <article><small>新增</small><b>+{{ impact.preview.addedCount }}</b></article>
              <article><small>删除</small><b>-{{ impact.preview.deletedCount }}</b></article>
              <article><small>修改</small><b>{{ impact.preview.modifiedCount }}</b></article>
              <article><small>重复</small><b>{{ impact.preview.duplicateCount }}</b></article>
              <article><small>被前序覆盖</small><b>{{ impact.preview.coveredCount }}</b></article>
            </div>
            <div class="notice" :class="{ warn: impact.preview.changed }">{{ impact.preview.changed ? '远端内容与当前缓存不同。应用前请检查下面的模板影响和分流回归。' : '远端内容与当前缓存一致，无需更新。' }}</div>
            <h4>受影响模板</h4><div class="impact-tags"><span v-for="t in impact.affectedTemplates || []" :key="t.template + t.policy">{{ t.template }}{{ t.policy ? ` → ${t.policy}` : '' }}</span><i v-if="!impact.affectedTemplates?.length">暂无模板引用</i></div>
            <h4>分流回归变化</h4>
            <el-table :data="impact.regression || []" size="small" max-height="220"><el-table-column prop="template" label="模板" min-width="140"/><el-table-column prop="target" label="目标" min-width="110"/><el-table-column prop="domain" label="域名" min-width="180"/><el-table-column prop="policy" label="策略" min-width="120"/><el-table-column label="变化" width="140"><template #default="{row}">{{ row.beforeInSet ? '命中' : '不命中' }} → {{ row.afterInSet ? '命中' : '不命中' }}</template></el-table-column></el-table>
            <h4>Diff 样例</h4>
            <div class="diff-grid"><div><b>新增</b><pre>{{ (impact.preview.added || []).slice(0,30).map(ruleText).join('\n') || '--' }}</pre></div><div><b>删除</b><pre>{{ (impact.preview.deleted || []).slice(0,30).map(ruleText).join('\n') || '--' }}</pre></div></div>
            <h4>可回滚快照</h4><div class="snapshot-list"><div v-for="s in snapshots" :key="s.id"><span><b>#{{ s.id }}</b> · {{ s.ruleCount }} 条 · {{ new Date(s.createdAt).toLocaleString() }}</span><button class="btn" @click="rollbackSnapshot(s.id)">回滚</button></div><div v-if="!snapshots.length" class="muted-text">尚无历史快照；第一次应用更新后会自动创建。</div></div>
          </template>
        </div>
        <footer><button class="btn" @click="impactOpen=false">关闭</button><button class="btn primary" :disabled="impactLoading || !impact?.preview?.changed" @click="applyUpdateNow">保存快照并应用更新</button></footer>
      </section>
    </div>
  </div>
</template>

<style scoped>
.page{padding:10px;color:var(--el-text-color-primary)}.hero{display:flex;justify-content:space-between;gap:16px;align-items:flex-start;margin-bottom:14px}.hero h2{margin:0 0 5px;font-size:24px}.hero p,.muted-text,.batch p{margin:0;color:var(--el-text-color-secondary);font-size:13px}.btn{border:1px solid var(--el-border-color);background:var(--el-bg-color);color:var(--el-text-color-primary);border-radius:8px;padding:9px 13px;cursor:pointer}.btn:disabled{opacity:.55;cursor:not-allowed}.btn.primary{background:var(--el-color-primary);border-color:var(--el-color-primary);color:#fff}.btn.ghost{border-color:transparent}.alert{border:1px solid #fecaca;background:#fef2f2;color:#991b1b;border-radius:9px;padding:11px 13px;margin-bottom:12px}.sources{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;margin-bottom:12px}.source-card,.filters,.batch,.rule-card{border:1px solid var(--el-border-color-lighter);background:var(--el-bg-color);border-radius:12px}.source-card{padding:14px}.source-head,.source-foot,.list-title,.meta{display:flex;justify-content:space-between;gap:10px}.source-foot{margin-top:9px;font-size:12px;color:var(--el-text-color-secondary)}.status{font-size:12px;border-radius:999px;padding:4px 8px}.status.ok{background:#ecfdf5;color:#047857}.status.muted{background:var(--el-fill-color-light);color:var(--el-text-color-secondary)}.filters{display:flex;gap:8px;align-items:center;flex-wrap:wrap;padding:13px;margin-bottom:14px}.input{height:38px;border:1px solid var(--el-border-color);background:var(--el-bg-color);color:var(--el-text-color-primary);border-radius:8px;padding:0 10px}.search{min-width:260px;flex:1}.chips{display:flex;gap:7px;flex-wrap:wrap;width:100%;margin-top:4px}.chip{border:1px solid var(--el-border-color);background:var(--el-bg-color);color:var(--el-text-color-regular);padding:6px 10px;border-radius:999px;cursor:pointer}.chip.active{border-color:var(--el-color-primary);color:var(--el-color-primary);background:var(--el-color-primary-light-9)}.layout{display:grid;grid-template-columns:minmax(0,1fr) 290px;gap:14px}.list-title{font-size:13px;margin-bottom:10px;color:var(--el-text-color-secondary)}.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:12px}.rule-card{padding:14px}.rule-head{display:flex;gap:9px;align-items:flex-start}.rule-head div{display:flex;flex-direction:column;gap:3px}.rule-head small{color:var(--el-text-color-secondary)}.tags{display:flex;gap:6px;flex-wrap:wrap;margin:11px 0}.tags span{font-size:12px;background:var(--el-fill-color-light);padding:4px 7px;border-radius:6px}.meta,.summary{font-size:12px;color:var(--el-text-color-secondary)}.summary{margin-top:7px}.actions{display:flex;justify-content:flex-end;gap:12px;border-top:1px solid var(--el-border-color-lighter);padding-top:9px;margin-top:10px}.actions button,.selected-row button,.modal header button{border:0;background:transparent;color:var(--el-text-color-regular);cursor:pointer}.actions .primary-link{color:var(--el-color-primary)}.batch{padding:14px;height:max-content;position:sticky;top:12px}.batch h3{margin:0 0 5px}.selected{max-height:360px;overflow:auto;margin:12px 0}.selected-row{display:flex;justify-content:space-between;gap:8px;padding:9px 0;border-bottom:1px solid var(--el-border-color-lighter)}.selected-row span{display:flex;flex-direction:column}.selected-row small{color:var(--el-text-color-secondary);margin-top:2px}.full{width:100%}.empty{text-align:center;color:var(--el-text-color-secondary);padding:32px 12px}.empty.small{padding:20px 8px}.skeleton{height:150px;border:1px solid var(--el-border-color-lighter);border-radius:12px;padding:14px}.skeleton i{display:block;height:14px;border-radius:7px;background:var(--el-fill-color);margin-bottom:12px;animation:pulse 1.2s infinite}.skeleton i:nth-child(1){width:45%;height:18px}.skeleton i:nth-child(2){width:70%}.skeleton i:nth-child(3){width:55%}.skeleton i:nth-child(4){width:85%}@keyframes pulse{50%{opacity:.45}}.overlay{position:fixed;inset:0;background:rgba(15,23,42,.45);z-index:3000;display:flex;align-items:center;justify-content:center;padding:20px}.modal{width:min(820px,95vw);max-height:88vh;overflow:auto;background:var(--el-bg-color);border-radius:14px;box-shadow:0 25px 60px rgba(0,0,0,.25)}.modal header,.modal footer{display:flex;justify-content:space-between;gap:10px;align-items:center;padding:15px 18px;border-bottom:1px solid var(--el-border-color-lighter)}.modal footer{justify-content:flex-end;border-bottom:0;border-top:1px solid var(--el-border-color-lighter)}.modal header button{font-size:22px}.modal-body{padding:18px}.preview-meta{display:flex;gap:16px;flex-wrap:wrap;font-size:13px;margin-bottom:12px}.modal pre{background:#0f172a;color:#d1fae5;border-radius:9px;padding:14px;min-height:260px;max-height:50vh;overflow:auto;font-size:12px;line-height:1.55}.form{display:grid;gap:13px}.form label{display:grid;gap:6px;font-size:13px;font-weight:600}.field-tip{font-weight:400;color:var(--el-text-color-secondary);line-height:1.5}.notice{padding:10px 12px;border-radius:8px;background:#ecfdf5;color:#065f46}.notice.warn{background:#fffbeb;color:#92400e}@media(max-width:1000px){.layout{grid-template-columns:1fr}.batch{position:static}.sources{grid-template-columns:1fr}}@media(max-width:700px){.hero{flex-direction:column}.grid{grid-template-columns:1fr}.search{min-width:100%}}
.impact-modal{width:min(1050px,96vw)}.impact-stats{display:grid;grid-template-columns:repeat(6,1fr);gap:8px;margin-bottom:12px}.impact-stats article{padding:10px;border:1px solid var(--el-border-color-lighter);border-radius:9px;display:flex;flex-direction:column}.impact-stats small{color:var(--el-text-color-secondary)}.impact-stats b{font-size:18px}.impact-tags{display:flex;flex-wrap:wrap;gap:7px}.impact-tags span{padding:5px 8px;border-radius:7px;background:var(--el-fill-color-light);font-size:12px}.impact-tags i{color:var(--el-text-color-secondary);font-style:normal}.diff-grid{display:grid;grid-template-columns:1fr 1fr;gap:10px}.diff-grid pre{min-height:140px;max-height:220px}.snapshot-list>div{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:8px 0;border-bottom:1px solid var(--el-border-color-lighter)}@media(max-width:800px){.impact-stats{grid-template-columns:repeat(2,1fr)}.diff-grid{grid-template-columns:1fr}}
</style>
