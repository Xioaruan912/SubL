#!/bin/bash
echo "Rebuilding backend..."
cd /root/sublinkX
pkill -f "ppeelink_amd64"
go build -tags "with_utls with_quic" -ldflags="-w -s" -o ppeelink_amd64 main.go cron.go
nohup ./ppeelink_amd64 > /tmp/ppeelink_backend.log 2>&1 &
echo "Rebuilding frontend..."
pkill -f "vite serve"
sleep 1
cd /root/sublinkX/webs
(setsid pnpm run dev > /tmp/ppeelink_frontend.log 2>&1 &)
echo "Done"