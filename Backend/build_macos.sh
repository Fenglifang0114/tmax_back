#!/bin/bash
set -e

# macOS 构建 Go 后端服务
echo "=== 开始编译 macOS 版 Go 后端服务 (tmaxsrv_mac) ==="

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"

# 必须开启 CGO（tinygo.org/x/bluetooth 依赖 CoreBluetooth/cbgo）
export CGO_ENABLED=1

ARCH=$(uname -m)
echo "当前架构: $ARCH"

go build -v -ldflags="-s -w -X 'main.Version=v1.82' -X 'tmaxsrv/build.User=$(id -u -n)' -X 'tmaxsrv/build.Time=$(date)'" -o tmaxsrv_mac .
chmod +x tmaxsrv_mac

echo "=== Go 后端服务编译成功: $DIR/tmaxsrv_mac ==="
