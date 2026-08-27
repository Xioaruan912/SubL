#!/bin/bash

echo "======================================"
echo "  启动 PPEELink (现代 UI 重构版) 🚀"
echo "======================================"

# 杀掉可能残留的进程
pkill -f "ppeelink_amd64"
pkill -f "vite serve"

echo "[1/2] 正在启动后端服务 (8000端口)..."
cd /root/sublinkX
nohup ./ppeelink_amd64 > /tmp/ppeelink_backend.log 2>&1 &
BACKEND_PID=$!
sleep 2

echo "[2/2] 正在启动前端服务 (8081端口)..."
cd /root/sublinkX/webs
nohup pnpm run dev > /tmp/ppeelink_frontend.log 2>&1 &
FRONTEND_PID=$!

echo ""
echo "✅ 服务已成功在后台启动！"
echo "--------------------------------------"
echo "👉 请在浏览器中访问: http://localhost:8081"
echo "--------------------------------------"
echo "查看后端日志: tail -f /tmp/ppeelink_backend.log"
echo "查看前端日志: tail -f /tmp/ppeelink_frontend.log"
echo "如需停止服务，请运行: pkill -f ppeelink_amd64 && pkill -f 'vite serve'"
echo "======================================"
