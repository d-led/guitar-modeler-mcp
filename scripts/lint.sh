#!/usr/bin/env bash
# Static analysis wrapper: run the full lint suite and fail fast on the first
# problem, so issues can be fixed one at a time.
#
# Usage:
#   scripts/lint.sh             # run every check
#   scripts/lint.sh --install   # install missing tools first, then run
#
# Checks (in order): gofmt, go vet, staticcheck, golangci-lint
# (cyclop + gocognit), gocyclo, revive, gosec, govulncheck, jscpd.
set -euo pipefail

# --- tunable thresholds -----------------------------------------------------
GOCYCLO_OVER="${GOCYCLO_OVER:-15}"      # cyclomatic complexity that fails the build
JSCPD_THRESHOLD="${JSCPD_THRESHOLD:-0.5}" # duplication % that fails the build

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ "${1:-}" == "--install" ]]; then
  echo "installing missing tools…"
  command -v staticcheck   >/dev/null || go install honnef.co/go/tools/cmd/staticcheck@latest
  command -v gocyclo       >/dev/null || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
  command -v revive        >/dev/null || go install github.com/mgechev/revive@latest
  command -v gosec         >/dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
  command -v govulncheck   >/dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
  command -v golangci-lint >/dev/null || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
  echo "done."
fi

RED=$'\033[31m'; GREEN=$'\033[32m'; BOLD=$'\033[1m'; RESET=$'\033[0m'

section() { printf '\n%s\n' "${BOLD}==> $*${RESET}"; }
fail()    { printf '%serror:%s %s\n' "$RED" "$RESET" "$*" >&2; exit 1; }
need()    { command -v "$1" >/dev/null 2>&1 || fail "$1 is not installed — run scripts/lint.sh --install (or: $2)"; }

section "gofmt"
files="$(gofmt -l .)"
[[ -z "$files" ]] || { printf '%s\n' "$files" >&2; fail "gofmt: the files above need formatting (gofmt -w .)"; }

section "go vet"
go vet ./...

section "staticcheck"
need staticcheck "go install honnef.co/go/tools/cmd/staticcheck@latest"
staticcheck ./...

section "golangci-lint"
need golangci-lint "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"
golangci-lint run --no-config --default=standard --enable=cyclop,gocognit --timeout=5m ./...

section "gocyclo (complexity > ${GOCYCLO_OVER})"
need gocyclo "go install github.com/fzipp/gocyclo/cmd/gocyclo@latest"
complex="$(gocyclo -over "${GOCYCLO_OVER}" . || true)"
[[ -z "$complex" ]] || { printf '%s\n' "$complex" >&2; fail "gocyclo: functions above cyclomatic complexity ${GOCYCLO_OVER}"; }

section "revive"
need revive "go install github.com/mgechev/revive@latest"
revive -config revive.toml ./...

section "gosec"
need gosec "go install github.com/securego/gosec/v2/cmd/gosec@latest"
# -exclude-generated skips the protoc-generated preset.pb.go (unsafe from
# protoc-gen-go, not our code). G304/G703 flag reading and writing paths that
# come from the user's own arguments — that is the tool's entire purpose, so
# the findings are expected rather than bugs.
gosec -quiet -exclude-generated \
  -exclude-rules="internal/install/install.go:G304;internal/golden/golden.go:G304;internal/mooer/file.go:G304;internal/gp200/file.go:G304;internal/waza/tsl.go:G304;internal/tools/tools.go:G304,G703;cmd/rigfile.go:G304" \
  ./...

section "govulncheck"
need govulncheck "go install golang.org/x/vuln/cmd/govulncheck@latest"
govulncheck ./...

section "jscpd (duplication ≤ ${JSCPD_THRESHOLD}%)"
need jscpd "npm install -g jscpd"
jscpd . --config .jscpd.json --threshold "${JSCPD_THRESHOLD}"

printf '\n%sall checks passed%s\n' "$GREEN" "$RESET"
