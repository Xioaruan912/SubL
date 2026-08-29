<div align="center">
<img src="webs/src/assets/logo.png" width="150px" height="150px" />
</div>

<div align="center">
    <img src="https://img.shields.io/badge/Go-1.25-green.svg"/>
    <img src="https://img.shields.io/badge/Vue-3.4-brightgreen.svg"/>
    <img src="https://img.shields.io/badge/Element Plus-2.6.1-blue.svg"/>
    <img src="https://img.shields.io/badge/ECharts-5.5-purple.svg"/>
    <img src="https://img.shields.io/badge/license-MIT-green.svg"/>
    <div align="center"> <a href="README.md">中文</a> | English</div>
</div>

## [Project Information]

**PPEELink** is a secondary development based on [gooaclok819/sublinkX](https://github.com/gooaclok819/sublinkX), with deep customization in UI, node management, and unlock detection.

Backend: **Go + Gin + GORM** · Frontend: **Vue3 + Element Plus + ECharts**.

Default account `admin` password `123456` (please change it).

## [New Features in This Fork]

- 🌍 **Node World Map on Dashboard**: bundled GeoIP database (GeoLite2), auto-detect node country and plot on a zoomable world map.
- 📡 **Node Latency Detection**: latency from VPS to common targets (GitHub/Google/Bing etc.) plus per-node TCP latency, shown on dashboard.
- 🔓 **Unlock Test**: via **sing-box**, really route through the node to test AI (OpenAI/ChatGPT, Claude, Gemini), video (Netflix, YouTube, Disney+), forum (Google, GitHub, Telegram) services.
- 🎯 **Xboard-style filter regex grouping**: Clash templates support `filter: "(?i)US|USA|..."` to auto-match nodes by name.
- 🎨 **OpenList-style UI**: Ant Design blue primary color, light sidebar, rounded cards.
- 📚 **Rule Center**: sync ShuntRules and ios_rule_script, browse Clash/Surge/Loon rules, preview and cache payloads on demand, and import Clash Rule Providers with a proxy-group selector from the target template.

## [Project Features]

- High degree of freedom and security, records subscription access, easy configuration
- Binary compilation, no Docker container required
- Supported clients: v2ray clash surge
- v2ray is a base64 universal format
- clash supported protocols: ss ssr trojan vmess vless hy hy2 tuic
- surge supported protocols: ss trojan vmess hy2 tuic

## [Project Preview]

![1712594176714](webs/src/assets/1.png)
![1712594176714](webs/src/assets/2.png)

## [Installation]

### Linux one-click install (source build)

> This fork requires `with_utls` (reality) and `with_quic` (hy2/tuic) build tags, so we install via source compilation (no release binaries needed).

```bash
curl -s -H "Cache-Control: no-cache" -H "Pragma: no-cache" https://raw.githubusercontent.com/Xioaruan912/SubL/main/install.sh | sudo bash
```

Requires `git`, `go` (≥1.25), `Node.js` (≥20 with corepack), and `curl`. The script will:

1. Clone the source and install frontend dependencies from `pnpm-lock.yaml`
2. Run `pnpm build` to generate `webs/dist`
3. Compile Go with `with_utls` and `with_quic`, embedding the frontend
4. Install the binary to `/usr/local/bin/ppeelink`
5. Register systemd and install the `ppeelink` management command

Run `ppeelink` to open the management menu after install.

### Docker method

The repository includes a multi-stage `Dockerfile` that builds the Vue frontend first and then compiles the Go backend:

```bash
docker build -t ppeelink .
docker run --name ppeelink -p 8000:8000 \
  -v $PWD/db:/app/db \
  -v $PWD/template:/app/template \
  -v $PWD/logs:/app/logs \
  -d ppeelink
```

Back up `db/` and `template/`.

## [Directory Structure]

```
api/          Backend APIs
models/       Data models (SQLite)
node/         Subscription generation & node parsing (clash/surge/vless etc.) + geo/latency/unlock test
routers/      Route registration
template/     clash/surge subscription templates
webs/         Vue3 frontend
```

## [Development & Build]

Requirements: Go 1.25+, Node.js 20+, corepack/pnpm.

```bash
cd webs
corepack enable
pnpm install --frozen-lockfile
pnpm build
cd ..

go test ./...
go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink .
```

You can also run `./build.sh` to build Linux amd64/arm64 binaries. `webs/dist/` and compiled binaries are build artifacts and are not committed to Git.

## License

[MIT](LICENSE)