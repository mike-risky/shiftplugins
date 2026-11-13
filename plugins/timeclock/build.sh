#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
set -e
cd "$(dirname "$0")"
V=$(python3 -c "import json;print(json.load(open('plugin.json'))['version'])")
cd server
go vet ./...
for a in amd64 arm64; do GOOS=linux GOARCH=$a CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/plugin-linux-$a .; done
cd ..
rm -rf pkg && mkdir -p pkg/com.shiftplugins.timeclock/server
cp plugin.json pkg/com.shiftplugins.timeclock/ && cp -r server/dist pkg/com.shiftplugins.timeclock/server/
tar -C pkg -czf com.shiftplugins.timeclock-$V.tar.gz com.shiftplugins.timeclock && rm -rf pkg
echo built com.shiftplugins.timeclock-$V.tar.gz
