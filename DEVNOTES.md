# 二开 / 部署踩坑记录 (DEVNOTES)

> 记录 sublinkX 二开与 VPS 部署过程中踩过的坑，避免下次重复犯错。

## 环境
- 上游仓库: https://github.com/gooaclok819/sublinkX
- 二开仓库: https://github.com/Xioaruan912/SubL
- VPS: HostDZire `199.47.242.40` (root, ssh 22)
- 生产目录: `/usr/local/bin/sublink/`（含 db/ template/ logs/ 与可执行文件 `sublink`）
- 生产服务: `sublink.service`（systemd，监听 8001，nginx:80 反代）

## 踩坑记录

### 1. 大文件 scp 上传超时（25MB 二进制）
- 现象：`scp` 单个 25MB 二进制直接超时失败（`shell tool terminated after 120s`）。
- 解决：**分片上传**。本地 `split -b 2m -d file part_`，逐片 scp，VPS 上 `cat part_* > new.bin`，最后 `md5sum` 校验一致。
- 教训：跨低带宽链路传大文件务必分片 + 校验 md5。

### 2. VPS 上已有生产 sublinkX 在跑，不能盲目覆盖
- 现象：`/usr/local/bin/sublink` 目录已有 active 的 `sublink.service`，监听 8001，带真实用户数据（db/sublink.db、config.yaml 含 jwt_secret）。
- 解决：先 `systemctl stop sublink`，**完整备份** `/usr/local/bin/sublink/{db,template,logs}` 和旧二进制到 `/root/backup/sublink_<ts>/`，再替换二进制、重启、验证。
- 教训：**部署前必须探查目标主机现有服务**（`systemctl list-units`, `ss -tlnp`, 目录内容），先备份再操作。

### 3. 新二进制是否兼容旧生产数据？先本地冒烟测试
- 现象：无法确定 main 分支新编译二进制能否直接用旧 sqlite 数据库启动。
- 解决：把备份的 `db/` 拉到本地，用新二进制 `run -port 19999` 启动冒烟，确认 `数据库初始化成功`（AutoMigrate 无报错）、页面 200、API 正常、db 文件大小不变。
- 教训：**数据库/配置升级前，用备份数据在本地先跑通**，确认 schema 兼容再上生产。
- 备注：新编译二进制 md5 `dd177088...`；旧生产二进制 md5 与上游 release 2.1 完全一致 `46bf3962...`，即生产本来就是 2.1。

### 4. 配置与数据目录位置（工作目录相关）
- sublinkX 是**工作目录敏感**程序：`./db/config.yaml`、`./db/sublink.db`、`./template/`、`./logs/` 都在二进制所在工作目录下。
- systemd `ExecStart=/usr/local/bin/sublink/sublink` + `WorkingDirectory=/usr/local/bin/sublink`，所以替换二进制时**只动顶层 `sublink` 文件，绝不能动 db/template/logs**。
- 换端口不要改代码，直接改 `db/config.yaml` 里的 `port`。

### 5. 登录/API 路由
- 登录路由是 `POST /api/v1/auth/login`，需先 `GET /api/v1/auth/captcha` 拿验证码，直接 POST 会返回「请求未携带token」。
- 首页 `/` 返回 200 即为前端正常。

## 部署速查（原地升级）
```bash
# 1) 停止 + 备份
systemctl stop sublink
TS=$(date +%Y%m%d_%H%M%S); mkdir -p /root/backup/sublink_$TS
cp -a /usr/local/bin/sublink/{db,template,logs} /root/backup/sublink_$TS/
cp /usr/local/bin/sublink/sublink /root/backup/sublink_$TS/sublink.old.bin

# 2) 分片上传新二进制（本地）
split -b 2m -d sublink_amd64 /tmp/sublink_part_
# scp 每片到 VPS /tmp 后：
cat /tmp/sublink_part_* > /tmp/sublink.new.bin; md5sum /tmp/sublink.new.bin

# 3) 替换并重启
mv /tmp/sublink.new.bin /usr/local/bin/sublink/sublink; chmod +x ...
systemctl start sublink
systemctl status sublink; ss -tlnp | grep 8001

# 4) 回滚（如需）
systemctl stop sublink
cp /root/backup/sublink_<ts>/sublink.old.bin /usr/local/bin/sublink/sublink
systemctl start sublink
```

## 二开功能（v2.1 + 二开）

### A. 前端风格 → OpenList manage 风格
- 参考 `/root/openlist-frontend/src/pages/manage/`（SolidJS+Hope UI）与 `src/app/theme.ts`。
- sublinkX 是 **Vue3 + Element Plus**，不重写框架，改走**全局主题变量**复刻观感：
  - `src/styles/index.scss`：主色 `#1890ff`（Ant Design 蓝）、背景 `#f7f8fa`、卡片/按钮/输入框/弹窗大圆角、表格 hover 高亮、filled 输入框。
  - `src/settings.ts`：`themeColor: "#1890ff"`。
  - `src/styles/variables.scss`：侧边栏改为浅色 `#ffffff` + 主色激活项（原深色 `#304156`）。
- 前端构建产物要同步到 Go 的 `static/` 目录（`go:embed` 用），改完 `vite build` 后 `cp -r webs/dist/* static/`，再重新 `go build`。

### B. 模板 filter 正则匹配节点（Xboard 风格）
- **Clash**（`node/clash.go` `DecodeClash`）：proxy-group 里若有 `filter: "(?i)US|USA|..."`，只把**节点名匹配正则**的节点填入该组；无 filter 保持全量填充（兼容旧行为）。支持 `include-all-providers: true` 字段（示意忽略）。
  ```yaml
  - name: AI
    type: select
    include-all-providers: true
    filter: "(?i)US|USA|United States|美国"
    proxies:
      - DIRECT
  ```
- **Surge**（`node/surge.go` `DecodeSurge`）：分组行内支持 `filter("正则")` 标记，只追加匹配节点，并自动移除标记。
- 测试：`node/filter_test.go`、`node/filter_integration_test.go`。

### C. 首页节点地图 + 节点延迟 + 解锁测试（二开第二批）
- **世界地图**（`webs/src/views/dashboard/components/WorldMap.vue`）：
  - 用 ECharts `effectScatter` 展示所有节点，`world.json` 本地打包（`webs/src/assets/world.json`，echarts4.9 标准世界地图，~1MB），零运行时外链。
  - 后端 `GET /api/v1/nodes/map`（`api/map_ping.go`）返回节点国家+坐标。GeoIP 用内置 `node/data/GeoLite2-Country.mmdb`（`go:embed`，从 `github.com/P3TERX/GeoLite.mmdb/releases` 下载，CC BY-SA 4.0），`github.com/oschwald/geoip2-golang` 查询。
  - **地图缩放 bug 修复**：必须用**单一 `geo` 组件**作为底图（`roam: true` + itemStyle），散点绑定 `coordinateSystem: "geo"`；**不能**同时用独立 `map` 系列（否则缩放时散点不跟随）。
- **节点延迟**（`NodePing.vue`）：`GET /api/v1/nodes/ping`，返回 VPS→常见目标（github/google/cloudflare/bing/百度）+ 每个节点服务器 TCP 延迟，60s 缓存。TCP 延迟 <1ms 时记为 1ms。
- **解锁测试**（`webs/src/views/subcription/unlock.vue` + 菜单「解锁测试」）：
  - `POST /api/v1/nodes/unlock`（body: `id` 或 `link`），通过 **sing-box 真实走节点**访问 AI（OpenAI/Claude/Gemini）、影视（Netflix/YouTube/Disney+）、论坛（Google/GitHub/Telegram），60s 缓存 + 全局互斥（同时只测一个）。
  - **必须用 `-tags "with_utls with_quic"` 编译**！否则 reality/hy2 报 `uTLS not included` / 无 QUIC 支持。这是最容易踩的坑。
  - sing-box 版本用 `v1.11.12`（go 1.20 兼容）；`go mod tidy` 可能误升到 v1.13.x（需要 go1.24.7），用 `go mod tidy -compat=1.22` 锁定。

## 新踩坑

### 6. 前端构建命令坑（pnpm build 会失败）
- 现象：`pnpm build` 报错（husky prepare + preinstall 触发 pnpm install 死循环，且克隆目录无 `.git`）。
- 解决：直接 `npx vite build --mode production` 绕过 preinstall/prepare 脚本。
- 教训：克隆/二开仓库改前端时，用 `npx vite build` 而非 `pnpm build`。

### 7. yaml 输出会把非 ASCII 节点名转义成 \U...
- 现象：`yaml.Marshal` 把 emoji 节点名转义为 `"\U0001F1FA\U0001F1F8"` 形式，测试里用 `strings.Contains(输出, "🇺🇸 US-01")` 匹配失败。
- 解决：测试断言按 ASCII 后缀（如 `US-01`）匹配，或解析出 group 的 proxies 段再断言。
- 教训：写 YAML 相关断言时，不要直接匹配含 emoji 的原始字符串。

### 8. 大二进制必须分片上传 + sing-box 体积
- 现象：`go:embed` mmdb + sing-box 后二进制约 **48MB**，scp 单文件必超时。
- 解决：`split -b 2m -d` 分片 → 逐片 scp → `cat` 合并 → `md5sum` 校验。
- 教训：二进制超 30MB 一律分片。

### 9. sing-box 集成要点
- **build tags**：reality 需要 `with_utls`，hy2/tuic 需要 `with_quic`，缺一不可，否则启动报错。
- **版本锁定**：`go mod tidy` 会把 sing-box 解析到 latest（v1.13.19 要 go1.24.7），必须 `go mod tidy -compat=1.22`。
- **box 包位置**：v1.11.x 的 `box` 包在模块根目录（`box.go`），导入用 `github.com/sagernet/sing-box`（别名 `singbox`），不是 `.../box` 子目录。
- **解锁测试耗时**：每个目标 5-8s，9 个目标串行约 90s；必须 60s 缓存，否则并发访问会打爆网络。部分节点 reality 握手会被本地网络干扰（`reality verification failed` / `EOF`），在 VPS 上正常。

### 10. README / 安装脚本个性化
- `.gitignore` 原本忽略 `install.sh`！不删掉它，`curl .../SubL/main/install.sh` 会 404。二开仓库必须从 `.gitignore` 移除 `install.sh`。
- 安装方式改为**源码编译**（方案A）：clone → `go build -tags "with_utls with_quic"` → systemd。无需管理 release 二进制。
- `menu.sh` 的更新逻辑同样改为 clone+编译；`./sublink --version` 应为 `-version`（main.go 是 `-version`），且 `-version` 需要 `./db` 目录存在（服务启动后自动创建）。
- `build.sh` 三条命令都必须带 `with_utls with_quic` 标签，否则产物不支持 reality/hy2。
- Docker 部分保留上游镜像 `jaaksi/sublinkx`（用户决定）；Dockerfile 的 `go build` 未带标签，容器内解锁测试不可用，README 已注明用源码编译方式。
- README 中 Stargazers / ZMTO 感谢信已移除，改为二开功能清单 + 目录结构 + 构建说明。

### 11. 解锁检测判定（关键修复：不能只看 HTTP 状态码）
- **错误做法**：原来只要 `200 <= 状态码 < 500` 就判定"解锁"。但很多服务的检测 URL 本身就会返回非 5xx：
  - `generativelanguage.googleapis.com/` 根路径恒 **404** → 任何节点都判 Gemini"解锁"（误判，HK 不支持的也显示解锁）
  - `api.anthropic.com/v1/messages` 无 key 返回 **405** → 恒"解锁"
- **正确做法**（参考 `lmc999/RegionRestrictionCheck`），每个服务定制 `Check func(c *http.Client)(bool,string)`（`node/unlock_checks.go`）：
  - **OpenAI/ChatGPT**：请求 `api.openai.com/compliance/cookie_requirements` + `ios.chat.openai.com/`，响应体含 `unsupported_country` / `vpn` 则未解锁。
  - **Claude**：访问 `claude.ai/`，若 `CheckRedirect` 捕获重定向到 `app-unavailable-in-region` 则未解锁。
  - **Gemini**：访问 `gemini.google.com`，响应体含标记 `45631641,null,true` 则解锁（带地区码 `,2,1,200,"JPN"`）。
  - **Netflix**：请求两个 title 页（81280792 / 70143836），均含 `Oh no!` = 仅原创可看；任一无则解锁。
  - **Disney+**：POST `disney.api.edge.bamgrid.com/devices`，含 `forbidden-location`/`403 ERROR` 则未解锁，含 `assertion` 则解锁。
  - **YouTube/Google/GitHub/Telegram**：基础可达性（200/204）。
- 统一 UA 用浏览器 UA（部分服务如 claude.ai/gemini.google.com 对非浏览器 UA 会异常）。
- 验证：HK 节点实测 OpenAI `区域不支持`❌、Claude `区域不支持`❌、Gemini ✅ —— 与真实情况一致。

### 12. 模板管理重构：独立「模板」大类 + 「操作模板」构建器
- **菜单**：`api/mentu.go` 新增顶层 `/template`（模板）大类，下设 `list`（模板列表）+ `builder`（操作模板）；从「订阅管理」移除原 template 子项。
- **文件迁移**：`webs/src/views/subcription/template.vue` → `webs/src/views/template/list.vue`（组件路径要对应 `Component: "template/list"`），原 `src/api/subcription/temp.ts` 复制为 `src/api/template/temp.ts`；`subs.vue` 仍引用 `@/api/subcription/temp`，保留原文件即可。
- **操作模板**（`POST /api/v1/template/build`，`api/template_builder.go`）：表单构建 clash 配置 → 保存到 `template/` 目录。分组支持 `select`（手动选择）/`url-test`（自动测速）/`fallback`（故障转移）三种，`filter` 正则 + `include-all-providers` 复用 Xboard filter 逻辑。保存后自动出现在订阅的 Clash 模板下拉框。
- **关键点**：构建器生成的模板 `proxies: ~` 占位（与现有 `template/clash.yaml` 一致），订阅时 `DecodeClash` 会填充节点并按 filter 分组。`yaml.Marshal` 会把 emoji 转义为 `\Uxxxx`，clash 可正常解析（等价）。
- 默认规则 `defaultClashRules`（直连内网 + GEOIP CN + MATCH）在 `api/template_builder.go`。
- **验证**：build API 生成含 filter 的模板 → `DecodeClash` 实测 AI 组只填充 US 节点、无 filter 组填充全部 —— 端到端正确。

### 13. 操作模板扩展（完整 mihomo 配置 + 规则集 + 端口联动）
- **需求**：`/template/builder` 支持完整 mihomo 配置（端口/geodata/tun/sniffer/dns/listeners）、规则集(rule-providers)、规则(rules)自定义，并能编辑现有模板。
- **后端** `api/template_builder.go` 重构：
  - `BuilderRequest` 扩展：完整端口（port/socks/mixed/redir/tproxy/dns_listen）、高级配置（unified-delay/geodata/tun/sniffer/dns）、`RuleProvider` 规则集、`Rules` 规则、`EditOldname`（编辑改名）、`Source`（default/custom）。
  - `GET /api/v1/template/default`：返回默认 mihomo 配置（默认分组 日常使用+AI filter、默认规则集 LAN/Direct/Proxy/AI/ESET_China、默认规则 RULE-SET 引用）。
  - **端口偏移联动** `applyPortOffsets`：`port_offset=true` 时，主端口改动 → mixed/redir/tproxy/socks 端口基于基准 7890 偏移同步；dns listen/external-controller 是独立端口不联动。
  - **关键点**：`applyBuilderDefaults` 的 default 分支**不能无条件覆盖用户显式传入的端口**，只填充默认分组/规则集/规则（否则用户改 port 会被重置为 7890）。
  - `DecodeClash` 只处理 proxies/proxy-groups，但 rule-providers/rules 因整体 map 解析会原样保留 → 订阅时 RULE-SET 引用正常。
- **前端** `builder.vue` 重构：模板来源下拉（内置默认 mihomo/新建空白/编辑已有文件）、分区块折叠面板（端口/基础/高级）、规则集动态行、rules 文本框。
- **gzip 优化（前端性能）**：
  - WorldMap 改用 `echarts/core` tree-shaken（从 1MB→437KB），world.json 移到 `public/` 运行时 fetch（不进 bundle）。
  - 删除未用代码：`src/views/demo/`、dashboard 的 BarChart/PieChart/RadarChart/FunnelChart、`src/components/WangEditor`（含 @wangeditor 依赖）。
  - **后端 gzip**：`middlewares/static.go` 自定义 StaticFS（`gin-contrib/gzip` 对 `r.StaticFS` 无效！），拦截静态文本响应 gzip + 正确 Content-Encoding，删除 Content-Length 用 chunked。图片排除。

### 14. 部署断点续传（rsync）
- 分片上传易因单个分片损坏/漏传导致 md5 不匹配（踩过 part_06 不完整）。
- **方案**：VPS `apt install rsync` + 本地 `rsync -avz --partial --append-verify -e "sshpass ..."` 上传，中断后再跑一次从断点继续，md5 校验。
- 本地是容器内网 IP，VPS 无法直连本地 HTTP，故用 rsync over ssh（VPS 需装 rsync）。

### 15. 中国各地延迟测试（解锁测试页新增）
- **需求**：在解锁测试页选择节点后，走节点测到**中国各省市运营商**（电信/联通/移动）的 TCP 延迟 + 字节跳动测速源 `lf3-ips.zstaticcdn.com` 延迟。
- **数据**：用户提供全量清单（`/mnt/c/Users/win/Desktop/新建 文本文档.txt`，WSL 桌面文件），约 2428 条 `省 市 运营商 纬度 经度 IP 端口`。用 Python 脚本解析生成 `node/china_data.go` 的 `chinaTargets` 全量切片。
- **筛选**：`FilterChinaTargets(provinces, isps)` 支持按省/运营商过滤；为空时默认**每省每运营商取 1 条**（`defaultChinaTargets`，去重）。
- **核心**：`node/china_ping.go` `RunChinaPing` 复用 sing-box socks 代理（`buildOutboundConfig` + `singbox.New` + socks 入站），用 `golang.org/x/net/proxy.SOCKS5` dialer 对每个目标 TCP connect 计时；**并发限 12**，`tcpPingVia` 带超时（goroutine + select），不可达/超时返回 -1。
- **API**：`POST /api/v1/nodes/chinaping`（body: id/link + 可选 provinces/isps/zstatic_port），60s 缓存 + 复用 `UnlockTestBusy` 互斥。
- **前端** `unlock.vue`：新增「中国延迟」按钮 + 省份/运营商筛选 + zstatic 端口选择，结果**按省折叠**（el-collapse）展示，延迟着色。
- **数据可用性坑**：中国运营商 IP:端口**并非全部可达**（如 `101.71.101.121:80`、`211.92.8.8:22` 不可达——端口关闭/被拦）。代码必须处理不可达（返回 -1），前端展示"不可达"。测试结果会有相当比例失败属正常。
