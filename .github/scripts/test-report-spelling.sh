#!/bin/sh
# Exercise the real pinned checker, including paths containing spaces.
set -eu
: "${STARGATE_REAL_MISSPELL:?set the path of the pinned misspell binary}"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT HUP INT TERM
printf 'package fixture\n// recieve a value\n' > "$fixture/example test.go"
printf 'Konfiguration, interfacce, exemple\n' > "$fixture/translated.md"
output=$(sh "$script_dir/report-tools/misspell" "$fixture")
printf '%s\n' "$output" | grep 'example test.go:2:' >/dev/null
if printf '%s\n' "$output" | grep 'translated.md' >/dev/null; then
    echo 'English checker scanned translated documentation' >&2
    exit 1
fi
printf 'package fixture\n// receive a value\n' > "$fixture/example test.go"
test -z "$(sh "$script_dir/report-tools/misspell" "$fixture")"
mkdir -p "$fixture/src/internal/i18n"
printf 'package i18n\n// recieve a value\nvar translated = "demandes impossibile momento configurato Konfiguration"\n' > "$fixture/src/internal/i18n/i18n.go"
output=$(sh "$script_dir/report-tools/misspell" "$fixture")
printf '%s\n' "$output" | grep 'i18n.go:2:' >/dev/null
if printf '%s\n' "$output" | grep 'i18n.go:3:' >/dev/null; then
    echo 'Valid localized message was marked as an English typo' >&2
    exit 1
fi
printf 'recieve a value\n' > "$fixture/english.md"
if "$STARGATE_REAL_MISSPELL" -error "$fixture/english.md" >/dev/null; then
    echo 'English documentation typo was not detected' >&2
    exit 1
fi
echo 'Report spelling scope checks passed'
