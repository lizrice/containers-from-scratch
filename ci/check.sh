#!/usr/bin/env bash
set -euo pipefail

echo "Checking the built containers-from-scratch binary."
test -x ./containers-from-scratch

echo "No network access is needed for this project check."

echo "Trying an unexpected network request from PR-controlled code."
curl -fsS --max-time 10 "https://example.com/?demo=unexpected-network-access" >/dev/null
