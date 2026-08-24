<div align="center">
<img src="webs/src/assets/logo.png" width="150px" height="150px" />
</div>

<div align="center">
    <img src="https://img.shields.io/badge/Go-1.22-green.svg"/>
    <img src="https://img.shields.io/badge/Vue-3.4-brightgreen.svg"/>
    <img src="https://img.shields.io/badge/Element Plus-2.6.1-blue.svg"/>
    <img src="https://img.shields.io/badge/ECharts-5.5-purple.svg"/>
    <img src="https://img.shields.io/badge/license-MIT-green.svg"/>
    <div align="center"> 中文 | <a href="README.en-US.md">English</div>
</div>

## [项目简介]

**SubL** 基于 [gooaclok819/sublinkX](https://github.com/gooaclok819/sublinkX) 二次开发，在前端 UI、节点管理、解锁检测等方面做了深度定制。

后端采用 **Go + Gin + GORM**，前端采用 **Vue3 + Element Plus + ECharts**。

默认账号 `admin` 密码 `123456`（请自行修改）。

## [二开新增功能]

- 🌍 **首页节点世界地图**：内置 GeoIP 数据库（GeoLite2），自动识别节点所属国家并在地图上打点，可缩放拖拽。
- 📡 **节点延迟检测**：VPS 到常见目标（GitHub/Google/Bing 等）延迟 + 每个节点服务器的 TCP 延迟，首页实时展示。
- 🔓 **解锁测试**：通过 **sing-box** 真实走节点访问 AI（OpenAI/ChatGPT、Claude、Gemini）、影视（Netflix、YouTube、Disney+）、论坛（Google、GitHub、Telegram）等常见服务，检测是否解锁。
- 🎯 **Xboard 风格 filter 正则分组**：Clash 模板中支持 `filter: "(?i)US|USA|..."` 正则，按节点名自动匹配填充分组。
- 🎨 **OpenList 风格 UI**：主色 Ant Design 蓝、浅色侧边栏、大圆角卡片，整体视觉更现代。

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

需要系统已安装 `git`、`go`（≥1.22）、`curl`。脚本会自动：

1. 克隆源码 → `go build -tags "with_utls with_quic"`
2. 安装二进制到 `/usr/local/bin/sublink`
3. 注册 systemd 服务并开机自启
4. 安装 `sublink` 菜单命令

安装后输入 `sublink` 呼出管理菜单。

### Docker 方式

在自己需要的位置创建一个目录，例如 `mkdir sublinkx`，然后 `cd` 进入该目录，数据会自动挂载：

```bash
docker run --name sublinkx -p 8000:8000 \
-v $PWD/db:/app/db \
-v $PWD/template:/app/template \
-v $PWD/logs:/app/logs \
-d jaaksi/sublinkx
```

需要备份的就是 `db` 和 `template` 目录。

> 注意：Docker 镜像为上游构建版本（`jaaksi/sublinkx`），如需解锁测试等二开功能，建议使用 Linux 源码编译方式。

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

```bash
# 后端（带解锁测试所需标签）
go build -tags "with_utls with_quic" -ldflags="-w -s" -o sublink_amd64 main.go

# 前端
cd webs
npx vite build --mode production
# 构建产物需同步到 static/ 目录（go:embed 使用）
cp -r webs/dist/* ../static/
```

## License

[MIT](LICENSE)