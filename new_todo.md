# sublinkX 开发待办 (new_todo)

## 已完成 ✅
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