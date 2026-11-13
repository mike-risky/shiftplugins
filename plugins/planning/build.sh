#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
# Builds com.shiftplugins.planning-<version>.tar.gz (Go server + plain-JS webapp, no JS build step).
set -e
cd "$(dirname "$0")"
V=$(sed -n 's/.*"version": "\(.*\)".*/\1/p' plugin.json)
(cd server && go vet ./... && for a in amd64 arm64; do CGO_ENABLED=0 GOOS=linux GOARCH=$a go build -trimpath -ldflags="-s -w" -o dist/plugin-linux-$a .; done)
rm -rf pkg && mkdir -p pkg/com.shiftplugins.planning/server/dist pkg/com.shiftplugins.planning/webapp/dist
cp plugin.json pkg/com.shiftplugins.planning/ && cp server/dist/plugin-linux-* pkg/com.shiftplugins.planning/server/dist/
cp webapp/dist/main.js pkg/com.shiftplugins.planning/webapp/dist/
tar -C pkg -czf com.shiftplugins.planning-$V.tar.gz com.shiftplugins.planning && rm -rf pkg
echo "com.shiftplugins.planning-$V.tar.gz"
