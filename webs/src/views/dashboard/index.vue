<template>
  <div class="dashboard-page">
    <div class="dashboard-ambient" aria-hidden="true"></div>
    <section class="dashboard-hero">
      <div class="hero-copy">
        <span class="hero-eyebrow">PPEELINK CONTROL CENTER</span>
        <h1>你好，{{ userStore.user.nickname || userStore.user.username || '管理员' }}<i :class="{ live: qualitySummary.total > 0 }">。</i></h1>
        <p>节点状态、订阅服务与网络质量一目了然。</p>
        <div class="hero-actions">
          <router-link to="/subcription/nodes" class="primary-action">管理节点</router-link>
          <router-link to="/subcription/subs" class="ghost-action">查看订阅 <span>→</span></router-link>
        </div>
      </div>
      <div class="survival-panel">
        <div class="panel-top"><span>节点存活率</span><span class="live-badge"><i></i> LIVE</span></div>
        <strong>{{ survivalRate }}%</strong>
        <small>最近一次服务端质量检测</small>
        <div class="ratio-bar" aria-hidden="true">
          <span class="ratio-alive" :style="{ flexGrow: aliveNodes || 0.001 }"></span>
          <span class="ratio-offline" :style="{ flexGrow: qualitySummary.offline || 0.001 }"></span>
        </div>
        <div class="ratio-legend">
          <span><i class="alive"></i>存活 <b>{{ aliveNodes }}</b></span>
          <span><i class="offline"></i>离线 <b>{{ qualitySummary.offline }}</b></span>
        </div>
      </div>
      <svg class="hero-wire" viewBox="0 0 1000 120" preserveAspectRatio="none" aria-hidden="true">
        <defs><linearGradient id="wireFade" x1="0" x2="1"><stop offset="0" stop-color="var(--ui-accent)" stop-opacity="0"/><stop offset=".2" stop-color="var(--ui-accent)" stop-opacity=".25"/><stop offset=".75" stop-color="var(--ui-accent)" stop-opacity=".3"/><stop offset="1" stop-color="var(--ui-accent)" stop-opacity="0"/></linearGradient></defs>
        <path d="M0 82 L120 82 L170 78 L210 86 L260 82 L340 82 L375 70 L405 96 L445 42 L478 104 L512 66 L548 84 L620 82 L680 82 L720 76 L760 86 L810 82 L1000 82" fill="none" stroke="url(#wireFade)" stroke-width="2" vector-effect="non-scaling-stroke" />
      </svg>
    </section>

    <section class="kpi-grid">
      <article style="--tile-accent:#22a06b"><span>健康节点</span><strong>{{ qualitySummary.healthy }}</strong><small>评分达到 80 分</small></article>
      <article style="--tile-accent:#d39b2a"><span>需要关注</span><strong>{{ qualitySummary.warning }}</strong><small>延迟或稳定性异常</small></article>
      <article style="--tile-accent:#d15b4f"><span>当前离线</span><strong>{{ qualitySummary.offline }}</strong><small>最近检测不可达</small></article>
      <article style="--tile-accent:var(--ui-accent)"><span>订阅服务</span><strong>{{ subTotal }}</strong><small>当前已创建订阅</small></article>
    </section>

    <section class="dashboard-section recommendation-section">
      <header class="section-heading section-heading-row">
        <div><span>SMART ROUTING</span><h2>场景节点推荐</h2><p>结合 24 小时可用率、P95 延迟、抖动与最近解锁样本给出理由和置信度。</p></div>
        <el-button plain @click="openAlerts">告警与维护时段</el-button>
      </header>
      <div class="scene-grid">
        <article v-for="scene in recommendations" :key="scene.key" class="scene-card">
          <header><span>{{ scene.name }}</span><small>{{ scene.key.toUpperCase() }}</small></header>
          <el-empty v-if="!scene.nodes?.length" :image-size="36" description="等待质量样本" />
          <div v-for="(item, index) in scene.nodes" :key="item.nodeId" class="recommend-item">
            <b>{{ index + 1 }}</b><div><strong>{{ item.name }}</strong><small>{{ item.reasons.join(' · ') }}</small></div>
            <span>{{ item.score }}<small>分</small></span>
          </div>
        </article>
      </div>
      <div v-if="healthEvents.length" class="event-strip">
        <span class="event-label">最近事件</span>
        <div v-for="event in healthEvents.slice(0, 5)" :key="event.id" class="event-item"><i :class="event.type"></i><b>{{ event.nodeName }}</b><span>{{ event.message }}</span><small>{{ new Date(event.createdAt).toLocaleString() }}</small></div>
      </div>
    </section>

    <section class="dashboard-section">
      <header class="section-heading">
        <span>NETWORK OBSERVABILITY</span><h2>节点网络概览</h2>
        <p>地图显示节点地区分布，右侧展示当前设备的网络延迟参考。</p>
      </header>
      <el-row :gutter="20">
        <el-col :span="16" :xs="24" class="mb-4"><div class="dashboard-panel map-panel"><world-map /></div></el-col>
        <el-col :span="8" :xs="24" class="mb-4"><div class="dashboard-panel"><node-ping /></div></el-col>
      </el-row>
    </section>

    <el-dialog v-model="alertVisible" title="节点告警与维护时段" width="520px">
      <el-form label-position="top">
        <el-form-item><el-switch v-model="alertForm.enabled" active-text="启用 Webhook 告警" /></el-form-item>
        <el-form-item label="Webhook URL"><el-input v-model="alertForm.webhookUrl" placeholder="https://…（兼容接收 JSON POST 的机器人或自动化服务）" /></el-form-item>
        <el-form-item label="连续失败阈值"><el-input-number v-model="alertForm.failureThreshold" :min="1" :max="20" /></el-form-item>
        <el-row :gutter="12"><el-col :span="12"><el-form-item label="维护开始"><el-time-select v-model="alertForm.maintenanceStart" start="00:00" step="00:30" end="23:30" clearable /></el-form-item></el-col><el-col :span="12"><el-form-item label="维护结束"><el-time-select v-model="alertForm.maintenanceEnd" start="00:00" step="00:30" end="23:30" clearable /></el-form-item></el-col></el-row>
        <el-alert type="info" :closable="false" title="维护时段内继续记录检测结果，但不会发送 Webhook。恢复事件同样会通知。" />
      </el-form>
      <template #footer><el-button @click="alertVisible=false">取消</el-button><el-button type="primary" @click="saveAlerts">保存</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: "Dashboard", inheritAttrs: false });
import { useUserStore } from "@/store/modules/user";
import { getSubTotal } from "@/api/total";
import { getNodeQualitySummary, getNodeRecommendations, getNodeHealthEvents, getAlertSetting, updateAlertSetting } from "@/api/subcription/node";
const WorldMap = defineAsyncComponent(() => import("./components/WorldMap.vue"));
const NodePing = defineAsyncComponent(() => import("./components/NodePing.vue"));
const userStore = useUserStore();
const subTotal = ref(0);
const qualitySummary = ref({ total: 0, healthy: 0, warning: 0, offline: 0, averageScore: 0 });
const recommendations = ref<any[]>([]);
const healthEvents = ref<any[]>([]);
const alertVisible = ref(false);
const alertForm = ref({ enabled:false, webhookUrl:'', failureThreshold:3, maintenanceStart:'', maintenanceEnd:'' });
const openAlerts = async () => { const { data } = await getAlertSetting(); alertForm.value = { ...alertForm.value, ...(data || {}) }; alertVisible.value = true; };
const saveAlerts = async () => { await updateAlertSetting(alertForm.value); ElMessage.success('告警设置已保存'); alertVisible.value = false; };
const aliveNodes = computed(() => Math.max(qualitySummary.value.total - qualitySummary.value.offline, 0));
const survivalRate = computed(() => qualitySummary.value.total ? Math.round(aliveNodes.value * 1000 / qualitySummary.value.total) / 10 : 0);
onMounted(async () => {
  const [subs, quality, recommends, events] = await Promise.allSettled([getSubTotal(), getNodeQualitySummary(), getNodeRecommendations(), getNodeHealthEvents(20)]);
  if (subs.status === "fulfilled") subTotal.value = subs.value?.data || 0;
  if (quality.status === "fulfilled") qualitySummary.value = { ...qualitySummary.value, ...(quality.value?.data || {}) };
  if (recommends.status === "fulfilled") recommendations.value = recommends.value?.data || [];
  if (events.status === "fulfilled") healthEvents.value = events.value?.data || [];
});
</script>

<style lang="scss" scoped>
.dashboard-page { position:relative; width:min(1180px,100%); margin:0 auto; padding:36px 36px 60px; color:var(--ui-text); }
.dashboard-ambient { position:absolute; inset:0 -30px auto; height:520px; overflow:hidden; pointer-events:none; background:radial-gradient(ellipse 65% 60% at 54% 12%,color-mix(in srgb,var(--ui-accent) 10%,transparent),transparent 72%); mask-image:linear-gradient(to bottom,transparent,#000 18%,#000 64%,transparent); }
.dashboard-ambient::after { position:absolute; inset:0; opacity:.22; background-image:linear-gradient(to right,var(--ui-border) 1px,transparent 1px),linear-gradient(to bottom,var(--ui-border) 1px,transparent 1px); background-size:68px 68px; mask-image:radial-gradient(ellipse 72% 60% at 50% 0,#000,transparent 78%); content:""; }
.dashboard-hero { position:relative; display:grid; grid-template-columns:minmax(0,1.15fr) minmax(310px,.85fr); align-items:center; gap:clamp(28px,5vw,64px); min-height:360px; padding:24px 0 92px; }
.hero-copy,.survival-panel { position:relative; z-index:1; }.hero-copy { display:flex; flex-direction:column; align-items:flex-start; }
.hero-eyebrow,.section-heading > span { color:var(--ui-accent-strong); font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:11px; font-weight:800; letter-spacing:.15em; }
.hero-copy h1 { margin:14px 0 12px; color:var(--ui-text); font-size:clamp(42px,6vw,72px); font-weight:720; line-height:1.05; letter-spacing:-.045em; }
.hero-copy h1 i { color:var(--ui-text-muted); font-style:normal; }.hero-copy h1 i.live { color:#22a06b; animation:period-breath 3.2s ease-in-out infinite; }.hero-copy p { margin:0; color:var(--ui-text-secondary); font-size:14px; }
.hero-actions { display:flex; align-items:center; gap:14px; margin-top:26px; }.primary-action,.ghost-action { display:inline-flex; align-items:center; gap:6px; min-height:42px; padding:0 20px; border-radius:999px; font-size:13px; font-weight:700; text-decoration:none; transition:transform 180ms cubic-bezier(.22,1,.36,1),box-shadow 180ms ease,color 140ms ease; }
.primary-action { background:var(--ui-text); color:var(--ui-canvas); box-shadow:0 10px 24px color-mix(in srgb,var(--ui-text) 18%,transparent); }.primary-action:hover { transform:translateY(-2px); }.ghost-action { padding-inline:6px; color:var(--ui-text-secondary); }.ghost-action span { transition:transform 160ms ease; }.ghost-action:hover { color:var(--ui-text); }.ghost-action:hover span { transform:translateX(3px); }
.survival-panel { display:flex; flex-direction:column; gap:7px; padding:26px; border:1px solid color-mix(in srgb,var(--ui-border) 82%,transparent); border-radius:20px; background:linear-gradient(145deg,color-mix(in srgb,var(--ui-surface-strong) 92%,transparent),color-mix(in srgb,var(--ui-surface) 76%,transparent)); box-shadow:0 20px 50px rgba(24,40,31,.12); backdrop-filter:blur(14px); }
.panel-top { display:flex; align-items:center; justify-content:space-between; color:var(--ui-text-muted); font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:11px; font-weight:800; letter-spacing:.11em; }.live-badge { display:inline-flex; align-items:center; gap:6px; padding:4px 9px; border:1px solid rgba(34,160,107,.28); border-radius:999px; background:rgba(34,160,107,.08); color:#178052; font-size:9px; }.live-badge i { width:6px; height:6px; border-radius:50%; background:#22a06b; animation:live-ping 2s ease-out infinite; }
.survival-panel > strong { margin-top:5px; font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:clamp(48px,6vw,66px); font-weight:650; line-height:1; letter-spacing:-.04em; }.survival-panel > small { color:var(--ui-text-muted); }.ratio-bar { display:flex; gap:2px; height:7px; margin-top:14px; }.ratio-bar span { min-width:4px; border-radius:99px; }.ratio-alive { background:#22a06b; }.ratio-offline { background:#d15b4f; }
.ratio-legend { display:flex; gap:20px; padding-top:12px; border-top:1px solid var(--ui-border); color:var(--ui-text-secondary); font-size:12px; }.ratio-legend span { display:flex; align-items:center; gap:6px; }.ratio-legend i { width:8px; height:8px; border-radius:2px; }.ratio-legend i.alive { background:#22a06b; }.ratio-legend i.offline { background:#d15b4f; }.ratio-legend b { color:var(--ui-text); font-family:ui-monospace,SFMono-Regular,Menlo,monospace; }
.hero-wire { position:absolute; right:0; bottom:0; left:0; width:100%; height:130px; pointer-events:none; }
.kpi-grid { position:relative; z-index:1; display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:16px; margin-top:2px; }.kpi-grid article { display:flex; flex-direction:column; gap:6px; min-height:142px; padding:20px; border:1px solid var(--ui-border); border-radius:14px; background:color-mix(in srgb,var(--ui-surface-strong) 84%,transparent); transition:border-color 150ms ease; }.kpi-grid article::before { width:28px; height:3px; margin-bottom:3px; border-radius:99px; background:var(--tile-accent); content:""; }.kpi-grid article:hover { border-color:color-mix(in srgb,var(--tile-accent) 42%,var(--ui-border)); }
.kpi-grid span { color:var(--ui-text-secondary); font-size:12px; font-weight:700; }.kpi-grid strong { color:var(--ui-text); font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:34px; line-height:1.15; }.kpi-grid small { margin-top:auto; color:var(--ui-text-muted); font-size:11px; }
.dashboard-section { margin-top:70px; }.section-heading { margin-bottom:22px; }.section-heading h2 { margin:8px 0 7px; color:var(--ui-text); font-size:26px; letter-spacing:-.025em; }.section-heading p { margin:0; color:var(--ui-text-muted); font-size:13px; }.dashboard-panel { height:100%; min-height:420px; padding:22px; border:1px solid var(--ui-border); border-radius:16px; background:var(--ui-surface-strong); box-shadow:var(--ui-panel-shadow); }.map-panel { overflow:hidden; }
.section-heading-row { display:flex; align-items:flex-end; justify-content:space-between; gap:20px; }.scene-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:14px; }.scene-card { padding:18px; border:1px solid var(--ui-border); border-radius:15px; background:var(--ui-surface-strong); }.scene-card > header { display:flex; justify-content:space-between; margin-bottom:12px; }.scene-card > header span { font-weight:750; }.scene-card > header small { color:var(--ui-accent-strong); font-family:monospace; font-size:10px; letter-spacing:.12em; }.recommend-item { display:grid; grid-template-columns:24px minmax(0,1fr) auto; align-items:center; gap:9px; padding:10px 0; border-top:1px solid var(--ui-border); }.recommend-item > b { display:grid; place-items:center; width:22px; height:22px; border-radius:7px; background:color-mix(in srgb,var(--ui-accent) 12%,transparent); color:var(--ui-accent-strong); font-size:11px; }.recommend-item > div { min-width:0; display:flex; flex-direction:column; gap:3px; }.recommend-item strong { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-size:13px; }.recommend-item div small { color:var(--ui-text-muted); font-size:10px; }.recommend-item > span { font-family:monospace; font-size:20px; font-weight:700; }.recommend-item > span small { font-size:9px; color:var(--ui-text-muted); }.event-strip { margin-top:14px; padding:14px 18px; border:1px solid var(--ui-border); border-radius:14px; background:var(--ui-surface-strong); }.event-label { display:block; margin-bottom:8px; color:var(--ui-text-muted); font-size:11px; font-weight:800; letter-spacing:.1em; }.event-item { display:grid; grid-template-columns:8px 120px minmax(0,1fr) auto; align-items:center; gap:9px; min-height:30px; font-size:12px; }.event-item i { width:7px; height:7px; border-radius:50%; background:#d15b4f; }.event-item i.recovery { background:#22a06b; }.event-item > span,.event-item > small { color:var(--ui-text-muted); }.event-item > small { font-size:10px; }
@keyframes period-breath { 50% { opacity:.38; } } @keyframes live-ping { 70%,100% { box-shadow:0 0 0 7px rgba(34,160,107,0); } }
@media (max-width:900px) { .dashboard-page { padding-inline:22px; }.dashboard-hero { grid-template-columns:1fr; padding-bottom:70px; }.kpi-grid { grid-template-columns:repeat(2,minmax(0,1fr)); }.scene-grid{grid-template-columns:1fr}.event-item{grid-template-columns:8px 90px 1fr}.event-item>small{display:none} }
@media (max-width:520px) { .dashboard-page { padding:24px 14px 44px; }.hero-copy h1 { font-size:40px; }.survival-panel { padding:20px; }.kpi-grid { grid-template-columns:1fr 1fr; gap:10px; }.kpi-grid article { min-height:126px; padding:16px; }.dashboard-section { margin-top:48px; } }
@media (prefers-reduced-motion:reduce) { .hero-copy h1 i.live,.live-badge i { animation:none; } }
</style>
