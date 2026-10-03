#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
go test -race ./...
node ui/tests/detection-test.js
SLOPMETER_BACKEND="$PWD/build/slopmeter-capture" python3 scripts/test_live_refresh.py
QT_QPA_PLATFORM=offscreen QT_QUICK_BACKEND=software SLOPMETER_BACKEND="$PWD/build/slopmeter-capture" ./build/ui/slopmeter --ui-self-test -read tests/fixtures/boss.jsonl
