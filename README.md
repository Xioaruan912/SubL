<div align="center">
<img src="webs/src/assets/logo.png" width="150px" height="150px" />
</div>

<div align="center">
    <img src="https://img.shields.io/badge/Go-1.25-green.svg"/>
    <img src="https://img.shields.io/badge/Vue-3.4-brightgreen.svg"/>
    <img src="https://img.shields.io/badge/Element Plus-2.6.1-blue.svg"/>
    <img src="https://img.shields.io/badge/ECharts-5.5-purple.svg"/>
    <img src="https://img.shields.io/badge/license-MIT-green.svg"/>
    <div align="center"> 中文 | <a href="README.en-US.md">English</div>
</div>

## [项目简介]

**PPEELink** 基于 [gooaclok819/sublinkX](https://github.com/gooaclok819/sublinkX) 二次开发，在前端 UI、节点管理、解锁检测等方面做了深度定制。

后端采用 **Go + Gin + GORM**，前端采用 **Vue3 + Element Plus + ECharts**。

默认账号 `admin` 密码 `123456`（请自行修改）。

## [二开新增功能]

- 🌍 **首页节点世界地图**：内置 GeoIP 数据库（GeoLite2），自动识别节点所属国家并在地图上打点，可缩放拖拽。
- 📡 **节点延迟检测**：VPS 到常见目标（GitHub/Google/Bing 等）延迟 + 每个节点服务器的 TCP 延迟，首页实时展示。
- 🔓 **解锁测试**：通过 **sing-box** 真实走节点访问 AI（OpenAI/ChatGPT、Claude、Gemini）、影视（Netflix、YouTube、Disney+）、论坛（Google、GitHub、Telegram）等常见服务，检测是否解锁。
- 🎯 **Xboard 风格 filter 正则分组**：Clash 模板中支持 `filter: "(?i)US|USA|..."` 正则，按节点名自动匹配填充分组。
- 🎨 **OpenList 风格 UI**：主色 Ant Design 蓝、浅色侧边栏、大圆角卡片，整体视觉更现代。
- 📚 **规则中心**：同步 ShuntRules 与 ios_rule_script，支持 Clash/Surge/Loon 浏览、规则预览、按需缓存，以及 Clash Rule Provider 导入与模板策略组代理选择。

## [项目特色]

- 自由度和安全性较高，能够记录访问订阅，配置轻松
- 二进制编译无需 Docker 容器
- 目前支持客户端：v2ray clash surge
- v2ray 为 base64 通用格式
- clash 支持协议：ss ssr trojan vmess vless hy hy2 tuic
- surge 支持协议：ss trojan vmess hy2 tuic

## [项目预览]

![1712594176714](webs/src/assets/1.png)
![1712594176714](webs/src/assets/2.png)

## [安装说明]

### Linux 一键安装（源码编译）

> 二开版本需要 `with_utls`（reality 支持）与 `with_quic`（hy2/tuic 支持）编译标签，因此采用源码编译安装，无需管理 release 二进制。

```bash
curl -s -H "Cache-Control: no-cache" -H "Pragma: no-cache" https://raw.githubusercontent.com/Xioaruan912/SubL/main/install.sh | sudo bash
```

需要系统已安装 `git`、`go`（≥1.25）、`Node.js`（≥20，含 corepack）与 `curl`。脚本会自动：

1. 克隆源码并使用 `pnpm-lock.yaml` 安装前端依赖
2. 执行 `pnpm build` 生成 `webs/dist`
3. 使用 `go build -tags "with_utls with_quic"` 编译并嵌入前端
4. 安装二进制到 `/usr/local/bin/ppeelink`
5. 注册 systemd 服务并安装 `ppeelink` 管理菜单

安装后输入 `ppeelink` 呼出管理菜单。

### Docker 方式

仓库自带多阶段 `Dockerfile`，会先构建 Vue 前端，再编译 Go 后端：

```bash
docker build -t ppeelink .
docker run --name ppeelink -p 8000:8000 \
  -v $PWD/db:/app/db \
  -v $PWD/template:/app/template \
  -v $PWD/logs:/app/logs \
  -d ppeelink
```

建议备份 `db/` 与 `template/`。

## [目录结构]

```
api/          后端接口
models/       数据模型（SQLite）
node/         订阅生成与节点解析（clash/surge/vless 等）+ 地理定位/延迟/解锁测试
routers/      路由注册
template/     clash/surge 订阅模板
webs/         Vue3 前端
```

## [开发与构建]

环境要求：Go 1.25+、Node.js 20+、corepack/pnpm。

```bash
# 安装并构建前端
cd webs
corepack enable
pnpm install --frozen-lockfile
pnpm build
cd ..

# 测试后端
go test ./...

# 构建包含前端资源的后端二进制
go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink .
```

也可以直接执行仓库根目录的 `./build.sh` 生成 Linux amd64/arm64 二进制。`webs/dist/` 与二进制均为构建产物，不提交到 Git。

## License

[MIT](LICENSE)