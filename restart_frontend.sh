#!/bin/bash
pkill -f "vite serve"
cd /root/sublinkX/webs && nohup pnpm run dev > /tmp/frontend.log 2>&1 &
echo "Frontend restarted"
