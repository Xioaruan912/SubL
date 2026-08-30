<div align="center">
  <img src="webs/src/assets/logo.png" width="150" height="150" alt="SubLinkX Logo" />
  <h1>SubLinkX</h1>
  <p><strong>代理订阅管理、规则调试、节点质量分析与安全发布平台</strong></p>
  <p>不只是生成订阅，而是在发布前告诉你：配置会怎么走、节点能不能用、目标客户端是否兼容，以及这次修改是否值得上线。</p>
</div>

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8.svg" alt="Go 1.25" />
  <img src="https://img.shields.io/badge/Gin-1.10-008ECF.svg" alt="Gin 1.10" />
  <img src="https://img.shields.io/badge/Vue-3.4-42B883.svg" alt="Vue 3.4" />
  <img src="https://img.shields.io/badge/Element%20Plus-2.6-409EFF.svg" alt="Element Plus 2.6" />
  <img src="https://img.shields.io/badge/SQLite-GORM-003B57.svg" alt="SQLite + GORM" />
  <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="MIT License" />
  <br />
  中文 · <a href="README.en-US.md">English</a>
</div>

---

## 项目定位

SubLinkX 面向自建代理节点、机场订阅和复杂 Clash/Mihomo 分流模板，目标是把传统的“订阅转换面板”升级成一套可验证、可解释、可回滚的代理配置发布系统。

典型工作流：

```text
订阅 / 节点
    ↓
模板与规则
    ↓
语法、引用、循环、协议检查
    ↓
关键网站规则命中模拟
    ↓
按目标场景选择节点
    ↓
真实出口验证
    ↓
与 Last Known Good 对比
    ↓
安全发布 / 失败保持旧版本
```

后端使用 **Go + Gin + GORM + SQLite**，节点真实检测基于 **sing-box**；前端使用 **Vue 3 + Element Plus + ECharts + Monaco Editor**。

## 核心能力

### 1. 一键安全发布

选择“订阅 + 模板 + 目标客户端”后自动执行发布前验证：

- 生成候选配置，不先修改线上绑定
- 校验 YAML / 配置段落、策略组、节点与规则引用
- 检测策略组循环引用
- 校验所有实际引用的 Rule Provider 是否可下载、解析或使用安全缓存
- 执行节点协议兼容性检查
- 运行已保存的分流回归用例
- 检查节点可用性并进行真实出口验证
- 与当前 Last Known Good 版本比较
- 全部通过后才事务发布；失败时保留原模板绑定和原 LKG

### 2. 通用规则解释器

输入域名、IP、端口和 TCP/UDP 上下文，可以看到完整的首条命中路径：

```text
gemini.google.com
  → RULE-SET Google
  → Gemini 策略组
  → JP 自动选择组
  → JP-02
  → 实际出口日本
```

同时展示：

- 命中前有哪些规则未命中，以及原因
- 规则来自本地规则还是远程 Rule Provider
- 策略组如何继续引用下一层策略组
- 哪些节点是候选、为什么被排除
- 最终选择哪个节点及其质量依据
- 模板编辑前 / 编辑后的命中 Diff

支持保存 Regression Case（分流回归用例），可定义预期策略、禁止策略、预期地区、端口和协议。

### 3. 节点 × 测试目标质量矩阵

节点不再只有一个笼统的总分。SubLinkX 会记录真实的：

```text
节点 × 目标网站 × 场景 × 成功/失败 × RTT
```

场景包括网络、AI、社交、内容、金融、工具、开发、媒体等。

模板分流和规则解释器选择节点时按以下顺序使用历史质量：

```text
具体目标历史
  → 同场景历史
  → TCP 总质量兜底
```

系统每 6 小时对当前在线节点进行低负载场景采样，也可在任务中心手动执行完整目标采样。

### 4. 节点协议兼容性矩阵

发布前会针对以下客户端进行能力诊断：

- Clash / Mihomo
- sing-box
- Surge
- Loon
- Quantumult X

诊断会区分：

- 原生支持
- 需要转换
- 完全不支持
- 转换后可能丢失字段

覆盖 `ws`、`grpc`、`httpupgrade`、`reality`、UDP、TFO、MPTCP 等能力，尽量把运行时错误提前变成生成阶段的明确提示。

### 5. 规则中心与更新影响预览

- 支持本地和远程 Clash Rule Provider
- 本地 provider 限制在 `template/` 安全路径内，拒绝目录越权
- 远程 provider 支持缓存回退和请求超时
- 区分 `domain`、`ipcidr`、`classical` 等 behavior
- 更新前显示新增、删除、修改、重复和前序覆盖
- 扫描哪些模板正在引用该规则集
- 对已配置检测目标比较更新前后的分流结果
- 更新前自动保留规则缓存快照，可一键回滚

### 6. 统一后台任务中心

以下长任务统一进入持久化任务中心：

- 节点 / 出口检测
- 质量矩阵采样
- 模板分流验证
- 机场同步
- 规则集同步
- 订阅构建
- 模板验证
- 一键安全发布
- 系统部署任务

支持进度、耗时、取消、重试、错误详情和历史记录。服务重启后未完成任务会被标记为中断，而不是在页面刷新后“消失”。

### 7. 订阅版本与 Last Known Good

订阅后台构建会保存不可变产物快照，包括：

- 输入摘要
- 模板校验值
- 规则校验值
- 产物 SHA
- 语法验证结果
- 分流 / 出口测试报告
- 生成与发布时间

验证成功的版本可以成为 **Last Known Good（最后已知可用版本）**。当实时生成、机场源或远程规则临时故障时，可回退到已经验证过的 LKG。

### 8. 可配置分流检测目标

管理员可直接配置测试目标，无需修改代码：

- 名称、Key、域名、分类和图标
- GET / HEAD 与请求路径
- 期望 HTTP 状态码
- 响应内容特征
- 是否要求出口 IP
- 超时与重试
- 启停和排序

检测目标本身不硬编码“应该走哪个国家”；期望地区只从当前模板的真实规则和策略组约束中推导。

### 9. 节点与分组管理

- 所有节点强制至少属于一个分组，历史无分组节点自动进入“默认”组
- 支持一个节点属于多个分组
- 机场同步后自动维护对应分组
- 支持单节点和整组**全局隐藏**，隐藏不会删除数据
- 被隐藏节点不会进入正常列表、订阅下发、质量采样、推荐、分流选节点和安全发布
- 隐藏状态可在分组管理中恢复

### 10. 安全、审计与状态页

- 用户密码使用 bcrypt；历史明文密码在成功登录后自动迁移
- 登录失败限流
- API Token 仅保存 SHA-256 摘要，支持 `read` / `write` / `admin` 权限和有效期
- 安全发布、系统部署、备份恢复、审计等高风险操作要求 admin scope
- 节点链接、订阅源、密码、Token、Webhook 等敏感日志脱敏
- 默认不信任任意反向代理；仅通过 `SUBLINKX_TRUSTED_PROXIES` 显式配置可信代理
- 管理操作审计记录操作者、认证类型、IP、方法、路径、状态和时间，不保存请求体或敏感凭据
- 安全配置导出默认排除密码、JWT Secret、API Token、订阅 Token、节点链接、机场 URL 和 Webhook
- `/status` 提供不泄露节点地址的公开只读状态页和事故时间线

## 订阅输出与客户端

当前订阅产物支持：

- Clash / Mihomo
- Surge
- Loon
- V2Ray / Base64 通用节点订阅

协议兼容检查的覆盖范围比订阅输出范围更广，用于提前判断同一批节点在 sing-box、Quantumult X 等目标客户端中的可表达程度。

## 节点检测

SubLinkX 不把“TCP 端口能连通”等同于“节点可用”。系统可以分别记录：

- TCP 可达性与 RTT
- 24h / 7d / 30d 可用率
- 平均延迟、P95、抖动和连续失败
- AI / 流媒体等解锁观察
- 按具体目标和场景的真实代理请求结果
- 实际出口 IP / 地区（目标提供时）

因此首页“当前离线”、TCP 质量、场景质量和解锁能力是不同维度，不会用单一“可信度 100%”冒充节点一定可用。

## 项目预览

![SubLinkX Preview 1](webs/src/assets/1.png)
![SubLinkX Preview 2](webs/src/assets/2.png)

> 部分截图可能落后于最新 UI，以实际运行版本为准。

## Linux 安装

### 一键安装（源码构建）

SubLinkX 使用 `with_utls` 与 `with_quic` 编译标签，以支持 Reality、Hysteria2、TUIC 等能力。

```bash
curl -s -H "Cache-Control: no-cache" -H "Pragma: no-cache" \
  https://raw.githubusercontent.com/Xioaruan912/SubL/main/install.sh | sudo bash
```

需要：

- `git`
- Go 1.25+
- Node.js 20+
- corepack / pnpm
- `curl`

安装脚本会完成前端构建、Go 编译、systemd 注册和管理命令安装。当前兼容管理命令 / 服务二进制名称仍为：

```bash
ppeelink
```

默认 Web 端口为 `8000`。

首次安装会创建管理员账号：

```text
用户名：admin
密码：123456
```

**首次登录后请立即修改默认密码。**

### Docker

仓库包含多阶段 `Dockerfile`：

```bash
docker build -t sublinkx .

docker run --name sublinkx \
  -p 8000:8000 \
  -v $PWD/db:/app/db \
  -v $PWD/template:/app/template \
  -v $PWD/logs:/app/logs \
  -d sublinkx
```

升级前建议至少备份 `db/` 与 `template/`。

## 开发与构建

```bash
# 前端
cd webs
corepack enable
pnpm install --frozen-lockfile
pnpm exec vue-tsc --noEmit
pnpm build
cd ..

# Go 测试
go test ./...

# Linux 生产构建
GOOS=linux GOARCH=amd64 \
  go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink .
```

也可以使用仓库中的 `build.sh` 生成 Linux amd64 / arm64 产物。

## 目录结构

```text
api/          HTTP API、规则解释、安全发布、任务中心等业务逻辑
models/       SQLite / GORM 数据模型与质量、任务、版本数据
node/         节点解析、sing-box 检测、出口与协议能力
routers/      Gin 路由注册
rulecenter/   规则集解析、缓存与规则中心
template/     Clash / Surge / Loon 模板与本地规则
webs/         Vue 3 管理前端
db/           SQLite、运行配置和规则缓存（运行时数据）
```

## 数据与升级建议

- 不要把 `db/` 当作可随意删除的缓存目录，其中包含主要业务数据。
- 升级前备份 `db/`、`template/` 和当前运行二进制。
- 生产替换建议使用“备份 → 上传临时文件 → SHA256 校验 → 原子替换 → 重启 → HTTP / 日志验证”的流程。
- 不要把密码、API Token、机场订阅地址或节点链接写入公开日志和 Git 仓库。

## License

[MIT](LICENSE)

## 致谢

感谢 [gooaclok819/sublinkX](https://github.com/gooaclok819/sublinkX) 早期项目提供的基础思路与代码参考。SubLinkX 在后续开发中围绕节点质量、规则解释、Rule Provider、任务中心、版本回滚、安全发布、安全审计与运维体系进行了持续扩展与重构。
