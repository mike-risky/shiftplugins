#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# Builds com.shiftplugins.leave-<version>.tar.gz (server-only plugin, no webapp).
set -e
cd "$(dirname "$0")"
V=$(sed -n 's/.*"version": "\(.*\)".*/\1/p' plugin.json)
(cd server && gofmt -l . && go vet ./... && go test ./... && for a in amd64 arm64; do CGO_ENABLED=0 GOOS=linux GOARCH=$a go build -trimpath -ldflags="-s -w" -o dist/plugin-linux-$a .; done)
rm -rf pkg && mkdir -p pkg/com.shiftplugins.leave/server/dist
cp plugin.json pkg/com.shiftplugins.leave/ && cp server/dist/plugin-linux-* pkg/com.shiftplugins.leave/server/dist/
tar -C pkg -czf com.shiftplugins.leave-$V.tar.gz com.shiftplugins.leave && rm -rf pkg
echo "com.shiftplugins.leave-$V.tar.gz"
