#!/bin/bash
# 二开仓库
REPO="Xioaruan912/SubL"
REPO_URL="https://github.com/${REPO}.git"

INSTALL_DIR="/usr/local/bin/ppeelink"

function check_tools {
    for cmd in git go node corepack; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            echo "缺少依赖: $cmd。请先安装 Git、Go 1.25+、Node.js 20+（含 corepack）。"
            return 1
        fi
    done
    return 0
}

function Up {
    check_tools || return 1
    echo "==> 克隆源码 ${REPO} ..."
    TMP_DIR=$(mktemp -d)
    git clone --depth 1 "$REPO_URL" "$TMP_DIR/ppeelink" || { echo "克隆失败"; rm -rf "$TMP_DIR"; return 1; }
    cd "$TMP_DIR/ppeelink"
    echo "==> 构建前端..."
    corepack enable >/dev/null 2>&1 || true
    cd webs
    pnpm install --frozen-lockfile || { echo "前端依赖安装失败"; rm -rf "$TMP_DIR"; return 1; }
    pnpm build || { echo "前端构建失败"; rm -rf "$TMP_DIR"; return 1; }
    cd ..
    echo "==> 编译后端（with_utls with_quic）..."
    go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink . || { echo "编译失败"; rm -rf "$TMP_DIR"; return 1; }
    # 停止服务后替换
    if systemctl is-active --quiet ppeelink; then
        systemctl stop ppeelink
    fi
    chmod +x ppeelink
    mv ppeelink "$INSTALL_DIR/ppeelink"
    rm -rf "$TMP_DIR"
    systemctl start ppeelink
    echo "更新完成"
}

function Select {
    # 获取服务状态
    cd "$INSTALL_DIR"
    status=$(systemctl is-active ppeelink)
    version=$(./ppeelink -version 2>/dev/null | head -1)
    echo "当前版本:$version"
    echo "当前运行状态: $status"
    echo "1. 启动服务"
    echo "2. 停止服务"
    echo "3. 卸载安装"
    echo "4. 查看服务状态"
    echo "5. 查看运行目录"
    echo "6. 修改端口"
    echo "7. 更新"
    echo "8. 重置账号密码"
    echo "0. 退出"
    echo -n "请选择一个选项: "
    read option

    case $option in
        1)
            systemctl start ppeelink
            systemctl daemon-reload
            ;;
        2)
            systemctl stop ppeelink
            systemctl daemon-reload
            ;;
        3)
            # 停止服务之前检查服务是否存在
            if systemctl is-active --quiet ppeelink; then
                systemctl stop ppeelink
            fi
            if systemctl is-enabled --quiet ppeelink; then
                systemctl disable ppeelink
            fi
            # 删除服务文件
            if [ -f /etc/systemd/system/ppeelink.service ]; then
                sudo rm /etc/systemd/system/ppeelink.service
            fi
            # 删除相关文件和目录
            sudo rm -f "$INSTALL_DIR/ppeelink"
            sudo rm -f /usr/bin/ppeelink
            read -p "是否删除模板文件和数据库(y/n): " isDelete
            if [ "$isDelete" = "y" ]; then
                sudo rm -rf "$INSTALL_DIR/db"
                sudo rm -rf "$INSTALL_DIR/template"
                sudo rm -rf "$INSTALL_DIR/logs"
            fi
            echo "卸载完成"
            ;;
        4)
            systemctl status ppeelink
            ;;
        5)
            echo "运行目录: $INSTALL_DIR"
            echo "需要备份的目录为db, template目录为模版文件可备份可不备份"
            cd "$INSTALL_DIR"
            ;;
        6)
            SERVICE_FILE="/etc/systemd/system/ppeelink.service"
            read -p "请输入新的端口号: " Port
            echo "新的端口号: $Port"
            PARAMETER="run --port $Port"
            # 检查服务文件是否存在
            if [ ! -f "$SERVICE_FILE" ]; then
                echo "服务文件不存在: $SERVICE_FILE"
                exit 1
            fi

            # 检查 ExecStart 是否已经包含该参数
            if grep -q "run --port" "$SERVICE_FILE"; then
                echo "参数已存在，正在替换..."
                # 使用 sed 替换 ExecStart 行中的 -port 参数
                sudo sed -i "s/-port [0-9]\+/-port $Port/" "$SERVICE_FILE"
            else
                # 如果没有 -port 参数，添加新参数
                sudo sed -i "/^ExecStart=/ s|$| $PARAMETER|" "$SERVICE_FILE"
                echo "参数已添加到 ExecStart 行: $PARAMETER"
            fi

            # 重新加载 systemd 守护进程
            sudo systemctl daemon-reload
            # 重启 ppeelink 服务
            sudo systemctl restart ppeelink

            echo "服务已重启。"
            ;;
        7)
            Up
            ;;
        8)
            read -p "请输入新的账号: " User
            read -p "请输入新的密码: " Password
            # 运行二进制文件并传递启动参数，放在后台运行
            cd "$INSTALL_DIR"
            ./ppeelink setting --username "$User" --password "$Password" &
            pid=$!
            wait $pid
            systemctl restart ppeelink
            ;;
        0)
            exit 0
            ;;
        *)
            echo "无效的选项,请重新选择"
            Select
            ;;
    esac
}
Select