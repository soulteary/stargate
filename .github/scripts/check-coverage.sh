#!/bin/sh
set -eu

# Compare statement counts directly: a rounded 95.0% can still be below 95%.
awk '
NR > 1 {
    total += $2
    if ($3 > 0) covered += $2
}
END {
    if (total == 0) {
        print "coverage profile contains no statements"
        exit 1
    }
    printf "coverage: %d/%d statements (%.4f%%), required: 95%%\n", covered, total, covered * 100 / total
    if (covered * 100 < total * 95) exit 1
}' "${1:?usage: check-coverage.sh coverage.out}"
