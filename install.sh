#!/bin/bash
set -e

# 仓库信息（二开仓库）
REPO="Xioaruan912/SubL"
REPO_URL="https://github.com/${REPO}.git"
REPO_RAW="https://raw.githubusercontent.com/${REPO}/main"

# 检查用户是否为root
if [ "$(id -u)" != "0" ]; then
    echo "该脚本必须以root身份运行。"
    exit 1
fi

# 检查必要工具
for cmd in git go curl node corepack; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "缺少依赖: $cmd。请先安装 Git、Go 1.25+、Node.js 20+（含 corepack）与 curl。"
        exit 1
    fi
done

# 创建一个程序目录
INSTALL_DIR="/usr/local/bin/ppeelink"
mkdir -p "$INSTALL_DIR"

# 克隆源码（浅克隆）
TMP_DIR=$(mktemp -d)
echo "==> 克隆源码 ${REPO} ..."
git clone --depth 1 "$REPO_URL" "$TMP_DIR/ppeelink"
cd "$TMP_DIR/ppeelink"

# 构建前端（Go 二进制会嵌入 webs/dist）
echo "==> 构建前端..."
corepack enable >/dev/null 2>&1 || true
cd webs
pnpm install --frozen-lockfile
pnpm build
cd ..

# 编译后端（带 with_utls / with_quic 标签）
echo "==> 编译后端（with_utls with_quic）..."
go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink .

# 安装二进制
chmod +x ppeelink
mv ppeelink "$INSTALL_DIR/ppeelink"
rm -rf "$TMP_DIR"

# 创建systemctl服务
cat > /etc/systemd/system/ppeelink.service <<EOF
[Unit]
Description=PPEELink Service

[Service]
ExecStart=$INSTALL_DIR/ppeelink
WorkingDirectory=$INSTALL_DIR
Restart=always
[Install]
WantedBy=multi-user.target
EOF

# 重新加载systemd守护进程
systemctl daemon-reload

# 启动并启用服务
systemctl start ppeelink
systemctl enable ppeelink
echo "服务已启动并已设置为开机启动"
echo "默认账号admin 密码123456 默认端口8000（可在 /usr/local/bin/ppeelink/db/config.yaml 修改）"

# 下载menu.sh并设置权限
curl -o /usr/bin/ppeelink -H "Cache-Control: no-cache" -H "Pragma: no-cache" "$REPO_RAW/menu.sh"
chmod 755 "/usr/bin/ppeelink"
echo "安装完成，输入 ppeelink 呼出菜单"