#!/usr/bin/env bash
set -euo pipefail

target="/usr/local/bin/"

go build -o pilot-bar ./cmd/waybar
go build -o pilot-bar-daemon ./cmd/daemon

sudo install -m755 pilot-bar pilot-bar-daemon "${target}"
