package api

import (
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
	"ppeelink/rulecenter"
)

type publicIncident struct { Type string `json:"type"`; Message string `json:"message"`; CreatedAt time.Time `json:"createdAt"` }

func buildPublicStatus() (gin.H,error) {
	stats,err:=models.GetNodeQualityStats(time.Now().Add(-24*time.Hour));if err!=nil{return nil,err};online,total:=0,0;availabilitySum:=0.0;for _,s:=range stats{total++;availabilitySum+=s.Availability;if s.LastRtt>=0{online++}};avgAvailability:=0.0;if total>0{avgAvailability=availabilitySum/float64(total)}
	var subCount int64;_ = models.DB.Model(&models.Subcription{}).Count(&subCount).Error
	var airports []models.Airport;_ = models.DB.Select("last_sync,node_count").Find(&airports).Error;airportSynced,airportStale:=0,0;latestSync:=time.Time{};for _,a:=range airports{if a.LastSync!=nil{if a.LastSync.After(latestSync){latestSync=*a.LastSync};if time.Since(*a.LastSync)<=36*time.Hour{airportSynced++}else{airportStale++}}else{airportStale++}}
	sources,_:=rulecenter.Sources();ruleOK,ruleError:=0,0;for _,s:=range sources{if s.Status=="ok"{ruleOK++}else if s.Status=="error"{ruleError++}}
	var events []models.NodeHealthEvent;_ = models.DB.Order("created_at desc").Limit(20).Find(&events).Error;incidents:=make([]publicIncident,0,len(events));for _,e:=range events{message:="节点状态发生变化";if e.Type=="recovery"{message="节点组恢复可用"}else if e.Type=="down"{message="节点组出现不可达"};incidents=append(incidents,publicIncident{Type:e.Type,Message:message,CreatedAt:e.CreatedAt})}
	var alert models.AlertSetting;_ = models.DB.First(&alert).Error;overall:="operational";if total>0&&online==0{overall="major_outage"}else if total>0&&float64(online)/float64(total)<0.7{overall="degraded"}
	return gin.H{"generatedAt":time.Now(),"status":overall,"subscriptionCount":subCount,"nodes":gin.H{"monitored":total,"online":online,"offline":total-online,"availability24h":avgAvailability},"airports":gin.H{"total":len(airports),"synced":airportSynced,"stale":airportStale,"latestSync":latestSync},"rules":gin.H{"sources":len(sources),"ok":ruleOK,"error":ruleError},"maintenance":gin.H{"start":alert.MaintenanceStart,"end":alert.MaintenanceEnd},"incidents":incidents},nil
}

func PublicStatus(c *gin.Context){data,err:=buildPublicStatus();if err!=nil{c.JSON(500,gin.H{"status":"unknown","message":"status unavailable"});return};c.JSON(200,data)}

var publicStatusHTML=template.Must(template.New("status").Parse(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>SubLinkX Status</title><style>body{margin:0;background:#f6f7f9;color:#17202a;font:14px system-ui,-apple-system,sans-serif}.wrap{max-width:900px;margin:auto;padding:48px 20px}header{display:flex;justify-content:space-between;align-items:center;margin-bottom:24px}h1{font-size:28px}.badge{padding:8px 12px;border-radius:999px;background:#e7f7ed;color:#18794e;font-weight:700}.badge.bad{background:#fff1f0;color:#c0362c}.grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px}.card,.timeline{background:#fff;border:1px solid #e5e7eb;border-radius:14px;padding:18px}.card b{display:block;font-size:27px;margin:7px 0}.card small,.muted{color:#6b7280}.timeline{margin-top:16px}.event{display:grid;grid-template-columns:12px 1fr auto;gap:9px;padding:10px 0;border-top:1px solid #eef0f2}.dot{width:8px;height:8px;border-radius:50%;background:#d05248;margin-top:5px}.dot.recovery{background:#239b64}@media(max-width:700px){.grid{grid-template-columns:1fr}header{align-items:flex-start;flex-direction:column}}</style></head><body><main class="wrap"><header><div><div class="muted">SUBLINKX PUBLIC STATUS</div><h1>服务状态</h1></div><span id="status" class="badge">加载中</span></header><section id="grid" class="grid"></section><section class="timeline"><h3>最近状态变化</h3><div id="events" class="muted">加载中…</div></section></main><script>const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));fetch('/api/v1/status/public',{cache:'no-store'}).then(r=>r.json()).then(d=>{const st=document.getElementById('status');const names={operational:'运行正常',degraded:'部分降级',major_outage:'严重故障',unknown:'未知'};st.textContent=names[d.status]||d.status;if(d.status!=='operational')st.classList.add('bad');const n=d.nodes||{},a=d.airports||{},r=d.rules||{};const cells=[['节点在线',String(n.online||0)+'/'+String(n.monitored||0),'24h 平均可用率 '+Number(n.availability24h||0).toFixed(1)+'%'],['机场同步',String(a.synced||0)+'/'+String(a.total||0),String(a.stale||0)+' 个需要关注'],['规则源',String(r.ok||0)+'/'+String(r.sources||0),String(r.error||0)+' 个同步异常']];document.getElementById('grid').innerHTML=cells.map(x=>'<article class="card"><small>'+esc(x[0])+'</small><b>'+esc(x[1])+'</b><span class="muted">'+esc(x[2])+'</span></article>').join('');const ev=d.incidents||[];document.getElementById('events').innerHTML=ev.length?ev.map(e=>'<div class="event"><i class="dot '+(e.type==='recovery'?'recovery':'')+'"></i><span>'+esc(e.message)+'</span><small>'+new Date(e.createdAt).toLocaleString()+'</small></div>').join(''):'近期无状态变化';}).catch(()=>{document.getElementById('status').textContent='状态不可用';document.getElementById('status').classList.add('bad')})</script></body></html>`))

func PublicStatusPage(c *gin.Context){c.Header("Cache-Control","no-store");c.Status(http.StatusOK);_ = publicStatusHTML.Execute(c.Writer,nil)}
