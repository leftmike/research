#!/bin/bash
# SessionStart hook for Claude Code on the web.
#
# Pre-warms the Go toolchain and module cache for the cypher modules so that
# `go test` / `go vet` run immediately in a fresh web session. The modules
# declare `go 1.25.0` while the base image may ship an older Go, so the first
# `go` command otherwise downloads the toolchain over the network at an
# inconvenient time. The container state is cached after this hook completes.
set -euo pipefail

# Only run in the remote (web) environment.
if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

root="${CLAUDE_PROJECT_DIR:-$(pwd)}"

for mod in "cypher" "cypher/graph"; do
  dir="$root/$mod"
  if [ -f "$dir/go.mod" ]; then
    echo "Warming Go module: $mod"
    (cd "$dir" && go mod download && go build ./...)
  fi
done

echo "session-start hook complete"
