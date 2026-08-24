#!/bin/bash
# 二开编译脚本：需要 with_utls with_quic 标签（reality / hysteria2 解锁测试依赖）
GOOS=linux GOARCH=amd64 go build -tags "with_utls with_quic" -ldflags="-w -s" -o sublink_amd64 main.go
GOOS=linux GOARCH=arm64 go build -tags "with_utls with_quic" -ldflags="-w -s" -o sublink_arm64 main.go
GOOS=windows  GOARCH=amd64  go build -tags "with_utls with_quic" -ldflags="-w -s" -o sublink.exe main.go