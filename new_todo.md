# sublinkX 开发待办 (new_todo)

## 已完成 ✅
- [x] EasyCLIProxyAPI 风格前端重构（2026-08-29，本地完成并部署 VPS）
  - 阅读 `router-for-me/EasyCLIProxyAPI` 完整 React/CSS WebUI 源码并迁移其交互原则
  - 全局色彩改为石墨灰 + 青绿色，补齐明暗主题语义变量和旧主题色自动迁移
  - 侧栏升级为 264px 固定工作台：品牌说明、46px 药丸导航、底部明暗分段器与运行状态
  - 统一子菜单、Select、Dropdown、弹窗、抽屉、分段器、表格和 Loading 微动画
  - 页面切换改为 180–220ms 淡入上移，并支持 `prefers-reduced-motion`
  - 顶栏增加页面上下文、紧凑工具按钮、用户信息下拉和箭头旋转
  - 登录页改为半透明控制台面板、柔和网格背景和聚焦动效
  - 仪表盘、节点分组和质量状态移除旧蓝色残留，统一使用语义色
  - 未改动后端接口、路由、请求参数或业务表单逻辑
  - `go test ./...`、`vue-tsc --noEmit`、Vite production build 与本地 HTTP 冒烟均通过
  - 首页进一步参考 `Cli-Proxy-API-Management-Center /management.html#/` 仪表盘重构：移除诗句和时段问候，改为“你好，管理员”+ 节点存活率大数字仪表、存活/离线比例、KPI 卡与网络观测区
- [x] 节点质量中心（2026-08-28）
  - 服务端每 10 分钟检测节点 TCP 可达性并持久化历史，保留 30 天
  - 按最近 24 小时计算可用率、平均延迟、抖动和 0–100 综合评分
  - 新增 `/api/v1/nodes/quality/history` 与 `/api/v1/nodes/quality/summary`
  - 节点卡片/列表显示质量评分、可用率和抖动，支持质量趋势弹窗
  - 首页增加整体健康度、健康/关注/离线节点摘要和行动提示
  - 本地真实数据冒烟：62 个节点，概览/汇总/历史接口均正常
- [x] 修复 SS 解析器错误接受 `noss://` 等非 SS 协议的问题
- [x] 本地生产构建与完整校验：`go test ./...`、`vue-tsc --noEmit`、Vite production build 全部通过
- [x] HostDZire 生产部署（2026-08-28 23:49 CST）
  - 实际服务为 `ppeelink.service`，工作目录 `/usr/local/bin/ppeelink`，监听 8001
  - 备份：`/root/backup/ppeelink_20260828_234955`
- [x] HostDZire 前端重构版发布（2026-08-29 08:49 CST）
  - 备份：`/root/backup/ppeelink_20260829_084938`
  - 发布二进制 SHA256：`78721daee6397e20050bccf04b39542378ca2e7a3a5a0c4b4afaf8e1d6ef4b72`
  - 新二进制 SHA-256：`edbb279f0456205746910c30bc682d411d1af72a75cff3ec6af7887ef7ea4006`
  - 业务数据前后数量一致，数据库完整性检查 `ok`
  - 后端首页、Nginx、静态资源及受保护 API 回归均为 HTTP 200
- [x] 前端视觉重构（Vue3 保留，全局样式深度覆盖 Element Plus）
  - 背景 #f6f7f5 / 暗黑 #111412
  - 内阴影边框替代硬边框、圆角分级、150ms 微交互
  - 移除 TagsView 多标签页、顶栏/主区透明融入底色
  - 侧边栏药丸状高亮菜单
- [x] 订阅弹窗支持「从外部机场订阅导入节点」（勾选后填 URL，提交时后端拉取）
- [x] 后端 `api/sub.go` 支持 `airport_url` 字段，拉取 Base64/明文节点并入库、自动归入同名分组
- [x] 节点解析器容错（vless/trojan/ss 支持原生明文 URL 直解，不再强制 Base64）
- [x] 机场管理模块：
  - `models/airport.go`（Airport 表：Name/URL/AutoCleanup/IsDedicated/LastSync/NodeCount）
  - `api/airport.go`（CRUD + 手动同步接口）
  - `api/airport_sync.go`（并发 TCP 测活、死节点清理、专线免死保留）
  - `cron.go`（每天凌晨 3:00 自动同步全机场）
  - `routers/airport.go` + 前端 `views/subcription/airport.vue` + `api/subcription/airport.ts`
  - 菜单新增「机场管理」
- [x] 修复订阅短链 `/c/` 无法穿透（vite proxy 增加 `/c/`、`/static/`）
- [x] 修复复制/二维码按钮重叠（去掉 el-input append 插槽，改 flex 布局）
- [x] 修复 TCP/解锁测试 404 & "Unexpected end of JSON input"（fetch 缺少 `/dev-api` 前缀，三处已补）

## 进行中 / 待办 🔲
- [ ] 验证机场管理页「测活&同步」按钮完整流程（登录后手测）
- [ ] 验证每日 3:00 定时任务实际触发
- [ ] 生产构建：`pnpm build` 后同步到 `static/`，确认 Nginx/后端直跑时 `/c/` 与 `/dev-api` 路由可用
- [ ] 订阅列表页其余页面（nodes.vue 列表/卡片、dashboard）统一视觉是否完全到位
- [ ] 机场节点过多时「节点列表」折叠分组体验优化
- [ ] 新增机场时可选是否立即同步一次
- [ ] 智能订阅第一版：按用途、国家、最大延迟、最低可用率和解锁能力自动选节点
- [ ] 模板沙盒：保存前预览分组匹配结果、空分组、无效正则和 RULE-SET 引用
- [ ] 节点质量数据积累 24 小时后复核评分阈值（当前首批样本不代表长期稳定性）
- [x] VPS 上线前备份数据库/模板/日志，部署新二进制并验证定时质量任务

## 常用命令
```bash
# 一键重建并启动前后端
/root/sublinkX/rebuild.sh

# 手动分别启动
cd /root/sublinkX && ./ppeelink_amd64          # 后端 8000
cd /root/sublinkX/webs && pnpm run dev         # 前端 8081

# 浏览器
http://localhost:8081
```

## 测试机场 URL
- `https://www.820010.xyz/sub/4afcbb441af72302a45ae6cde75ff48b`（trojan/vless，20 节点，已实测兼容）

## 账号
- 默认 admin / 123456（可改）
