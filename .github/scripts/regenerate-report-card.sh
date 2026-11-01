#!/bin/sh
# Run after goreportcard-action installs its pinned CLI and native tools.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
STARGATE_REAL_MISSPELL=$(command -v misspell)
export STARGATE_REAL_MISSPELL
PATH="$script_dir/report-tools:$PATH"
export PATH

# The English dictionary is not suitable for translated documentation.
# English docs retain a separate, failing spelling check.
for document in ./*.md; do
    case "$document" in
        ./README.*.md) continue ;;
    esac
    "$STARGATE_REAL_MISSPELL" -error "$document"
done
"$STARGATE_REAL_MISSPELL" -error docs/enUS
sh "$script_dir/test-report-spelling.sh"
goreportcard-cli -d . -svg .github/goreportcard.svg -report .github/goreportcard-report.md -t 0
