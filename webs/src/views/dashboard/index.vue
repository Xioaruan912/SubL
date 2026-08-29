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
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: "Dashboard", inheritAttrs: false });
import { useUserStore } from "@/store/modules/user";
import { getSubTotal } from "@/api/total";
import { getNodeQualitySummary } from "@/api/subcription/node";
const WorldMap = defineAsyncComponent(() => import("./components/WorldMap.vue"));
const NodePing = defineAsyncComponent(() => import("./components/NodePing.vue"));
const userStore = useUserStore();
const subTotal = ref(0);
const qualitySummary = ref({ total: 0, healthy: 0, warning: 0, offline: 0, averageScore: 0 });
const aliveNodes = computed(() => Math.max(qualitySummary.value.total - qualitySummary.value.offline, 0));
const survivalRate = computed(() => qualitySummary.value.total ? Math.round(aliveNodes.value * 1000 / qualitySummary.value.total) / 10 : 0);
onMounted(async () => {
  const [subs, quality] = await Promise.allSettled([getSubTotal(), getNodeQualitySummary()]);
  if (subs.status === "fulfilled") subTotal.value = subs.value?.data || 0;
  if (quality.status === "fulfilled") qualitySummary.value = { ...qualitySummary.value, ...(quality.value?.data || {}) };
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
@keyframes period-breath { 50% { opacity:.38; } } @keyframes live-ping { 70%,100% { box-shadow:0 0 0 7px rgba(34,160,107,0); } }
@media (max-width:900px) { .dashboard-page { padding-inline:22px; }.dashboard-hero { grid-template-columns:1fr; padding-bottom:70px; }.kpi-grid { grid-template-columns:repeat(2,minmax(0,1fr)); } }
@media (max-width:520px) { .dashboard-page { padding:24px 14px 44px; }.hero-copy h1 { font-size:40px; }.survival-panel { padding:20px; }.kpi-grid { grid-template-columns:1fr 1fr; gap:10px; }.kpi-grid article { min-height:126px; padding:16px; }.dashboard-section { margin-top:48px; } }
@media (prefers-reduced-motion:reduce) { .hero-copy h1 i.live,.live-badge i { animation:none; } }
</style>
