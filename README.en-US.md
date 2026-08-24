<div align="center">
<img src="webs/src/assets/logo.png" width="150px" height="150px" />
</div>

<div align="center">
    <img src="https://img.shields.io/badge/Go-1.22-green.svg"/>
    <img src="https://img.shields.io/badge/Vue-3.4-brightgreen.svg"/>
    <img src="https://img.shields.io/badge/Element Plus-2.6.1-blue.svg"/>
    <img src="https://img.shields.io/badge/ECharts-5.5-purple.svg"/>
    <img src="https://img.shields.io/badge/license-MIT-green.svg"/>
    <div align="center"> <a href="README.md">中文</a> | English</div>
</div>

## [Project Information]

**SubL** is a secondary development based on [gooaclok819/sublinkX](https://github.com/gooaclok819/sublinkX), with deep customization in UI, node management, and unlock detection.

Backend: **Go + Gin + GORM** · Frontend: **Vue3 + Element Plus + ECharts**.

Default account `admin` password `123456` (please change it).

## [New Features in This Fork]

- 🌍 **Node World Map on Dashboard**: bundled GeoIP database (GeoLite2), auto-detect node country and plot on a zoomable world map.
- 📡 **Node Latency Detection**: latency from VPS to common targets (GitHub/Google/Bing etc.) plus per-node TCP latency, shown on dashboard.
- 🔓 **Unlock Test**: via **sing-box**, really route through the node to test AI (OpenAI/ChatGPT, Claude, Gemini), video (Netflix, YouTube, Disney+), forum (Google, GitHub, Telegram) services.
- 🎯 **Xboard-style filter regex grouping**: Clash templates support `filter: "(?i)US|USA|..."` to auto-match nodes by name.
- 🎨 **OpenList-style UI**: Ant Design blue primary color, light sidebar, rounded cards.

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

Requires `git`, `go` (≥1.22), `curl`. The script will:

1. Clone source → `go build -tags "with_utls with_quic"`
2. Install binary to `/usr/local/bin/sublink`
3. Register systemd service and enable on boot
4. Install the `sublink` menu command

Run `sublink` to open the management menu after install.

### Docker method

Create a directory where you want it (e.g. `mkdir sublinkx`), then `cd` into it. Data is mounted automatically:

```bash
docker run --name sublinkx -p 8000:8000 \
-v $PWD/db:/app/db \
-v $PWD/template:/app/template \
-v $PWD/logs:/app/logs \
-d jaaksi/sublinkx
```

All you need to back up is `db` and `template`.

> Note: the Docker image is the upstream build (`jaaksi/sublinkx`). For fork features like unlock test, use the Linux source-build method.

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

```bash
# Backend (with unlock-test required tags)
go build -tags "with_utls with_quic" -ldflags="-w -s" -o sublink_amd64 main.go

# Frontend
cd webs
npx vite build --mode production
# Sync build output to static/ (used by go:embed)
cp -r webs/dist/* ../static/
```

## License

[MIT](LICENSE)