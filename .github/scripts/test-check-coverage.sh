#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT HUP INT TERM

cat > "$scratch/exact.out" <<'EOF'
mode: atomic
example.go:1.1,2.1 95 1
example.go:3.1,4.1 5 0
EOF
sh "$root/check-coverage.sh" "$scratch/exact.out"

cat > "$scratch/rounded.out" <<'EOF'
mode: atomic
example.go:1.1,2.1 94999 1
example.go:3.1,4.1 5001 0
EOF
if sh "$root/check-coverage.sh" "$scratch/rounded.out"; then
    echo "coverage below 95% was accepted" >&2
    exit 1
fi

echo 'mode: atomic' > "$scratch/empty.out"
if sh "$root/check-coverage.sh" "$scratch/empty.out"; then
    echo "an empty coverage profile was accepted" >&2
    exit 1
fi
